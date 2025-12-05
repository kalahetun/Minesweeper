// Package discovery 实现服务发现功能
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/boifi/service-discovery/internal/types"
	"github.com/boifi/service-discovery/pkg/logger"
)

// JaegerDependency Jaeger Dependencies API 返回的依赖关系结构
type JaegerDependency struct {
	// Parent 调用方服务名
	Parent string `json:"parent"`
	// Child 被调用方服务名
	Child string `json:"child"`
	// CallCount 调用次数
	CallCount int `json:"callCount"`
}

// ToServiceEdge 转换为 ServiceEdge
func (d *JaegerDependency) ToServiceEdge() types.ServiceEdge {
	return types.ServiceEdge{
		Source:    d.Parent,
		Target:    d.Child,
		CallCount: d.CallCount,
	}
}

// JaegerDependenciesResponse Jaeger Dependencies API 响应结构
type JaegerDependenciesResponse struct {
	Data   []JaegerDependency `json:"data"`
	Errors []string           `json:"errors,omitempty"`
}

// JaegerClient Jaeger Query API 客户端
type JaegerClient struct {
	baseURL    string
	lookback   time.Duration
	timeout    time.Duration
	httpClient *http.Client
	log        logger.Logger
}

// NewJaegerClient 创建新的 Jaeger 客户端
func NewJaegerClient(baseURL string, lookback, timeout time.Duration, log logger.Logger) *JaegerClient {
	if log == nil {
		log = logger.NewDefault()
	}

	return &JaegerClient{
		baseURL:  baseURL,
		lookback: lookback,
		timeout:  timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log: log,
	}
}

// FetchDependencies 从 Jaeger 获取服务依赖关系
// 使用 /api/dependencies 端点
func (c *JaegerClient) FetchDependencies(ctx context.Context) ([]JaegerDependency, error) {
	c.log.Debug("fetching dependencies from jaeger",
		"url", c.baseURL,
		"lookback", c.lookback.String(),
	)

	// 构建请求 URL
	endTs := time.Now().UnixMilli()
	lookbackMs := c.lookback.Milliseconds()

	reqURL, err := url.Parse(c.baseURL + "/api/dependencies")
	if err != nil {
		return nil, fmt.Errorf("invalid jaeger url: %w", err)
	}

	query := reqURL.Query()
	query.Set("endTs", strconv.FormatInt(endTs, 10))
	query.Set("lookback", strconv.FormatInt(lookbackMs, 10))
	reqURL.RawQuery = query.Encode()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("jaeger request failed", "error", err.Error(), "url", reqURL.String())
		return nil, fmt.Errorf("jaeger request failed: %w", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.Error("jaeger returned non-200 status",
			"status", resp.StatusCode,
			"body", string(body),
		)
		return nil, fmt.Errorf("jaeger returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response JaegerDependenciesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.log.Error("failed to decode jaeger response", "error", err.Error())
		return nil, fmt.Errorf("failed to decode jaeger response: %w", err)
	}

	// 检查 API 错误
	if len(response.Errors) > 0 {
		c.log.Warn("jaeger returned errors", "errors", response.Errors)
	}

	c.log.Debug("fetched dependencies from jaeger",
		"count", len(response.Data),
	)

	return response.Data, nil
}

// BuildTopology 从 Jaeger 构建服务拓扑
// 返回 ServiceEdge 列表
func (c *JaegerClient) BuildTopology(ctx context.Context) ([]types.ServiceEdge, error) {
	c.log.Info("building service topology from jaeger")

	deps, err := c.FetchDependencies(ctx)
	if err != nil {
		return nil, err
	}

	edges := make([]types.ServiceEdge, 0, len(deps))
	for _, dep := range deps {
		edges = append(edges, dep.ToServiceEdge())
	}

	c.log.Info("built service topology",
		"edges_count", len(edges),
	)

	return edges, nil
}

// BuildTopologyGraceful 从 Jaeger 构建服务拓扑（优雅降级）
// 如果 Jaeger 不可用，返回空的拓扑而不是错误
func (c *JaegerClient) BuildTopologyGraceful(ctx context.Context) ([]types.ServiceEdge, error) {
	edges, err := c.BuildTopology(ctx)
	if err != nil {
		c.log.Warn("jaeger unavailable, returning empty topology",
			"error", err.Error(),
		)
		return []types.ServiceEdge{}, nil
	}
	return edges, nil
}

// IsAvailable 检查 Jaeger 是否可用
func (c *JaegerClient) IsAvailable(ctx context.Context) bool {
	reqURL := c.baseURL + "/api/services"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return false
	}

	// 使用较短的超时进行健康检查
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.log.Debug("jaeger health check failed", "error", err.Error())
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetLookback 返回配置的 lookback 时间
func (c *JaegerClient) GetLookback() time.Duration {
	return c.lookback
}

// GetBaseURL 返回 Jaeger base URL
func (c *JaegerClient) GetBaseURL() string {
	return c.baseURL
}

// JaegerTracesResponse Jaeger Traces API 响应结构
type JaegerTracesResponse struct {
	Data   []JaegerTrace `json:"data"`
	Errors []string      `json:"errors,omitempty"`
}

// JaegerTrace 单个 trace
type JaegerTrace struct {
	TraceID   string       `json:"traceID"`
	Spans     []JaegerSpan `json:"spans"`
	Processes map[string]struct {
		ServiceName string `json:"serviceName"`
	} `json:"processes"`
}

// JaegerSpan 单个 span
type JaegerSpan struct {
	TraceID       string          `json:"traceID"`
	SpanID        string          `json:"spanID"`
	OperationName string          `json:"operationName"`
	ProcessID     string          `json:"processID"`
	Tags          []JaegerSpanTag `json:"tags"`
	References    []struct {
		RefType string `json:"refType"`
		TraceID string `json:"traceID"`
		SpanID  string `json:"spanID"`
	} `json:"references"`
}

// JaegerSpanTag span 标签
type JaegerSpanTag struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

// EdgeAPIInfo API 调用信息（用于聚合）
type EdgeAPIInfo struct {
	Source    string
	Target    string
	Path      string
	Method    string
	CallCount int
}

// BuildTopologyWithAPIs 从 Jaeger 构建服务拓扑，包含 API 调用详情
func (c *JaegerClient) BuildTopologyWithAPIs(ctx context.Context) ([]types.ServiceEdge, error) {
	c.log.Info("building service topology with APIs from jaeger")

	// 1. 获取基础依赖关系
	deps, err := c.FetchDependencies(ctx)
	if err != nil {
		return nil, err
	}

	if len(deps) == 0 {
		return []types.ServiceEdge{}, nil
	}

	// 2. 获取所有服务列表
	services, err := c.FetchServices(ctx)
	if err != nil {
		c.log.Warn("failed to fetch services, using dependencies only", "error", err.Error())
		// 回退到只有依赖信息
		edges := make([]types.ServiceEdge, 0, len(deps))
		for _, dep := range deps {
			edges = append(edges, dep.ToServiceEdge())
		}
		return edges, nil
	}

	// 3. 为每个服务获取 traces 并提取 API 调用信息
	// edgeAPIs: "source->target" -> path -> EdgeAPIInfo
	edgeAPIs := make(map[string]map[string]*EdgeAPIInfo)

	for _, service := range services {
		traces, err := c.FetchTraces(ctx, service, 50) // 限制每个服务 50 个 traces
		if err != nil {
			c.log.Debug("failed to fetch traces for service", "service", service, "error", err.Error())
			continue
		}

		// 从 traces 中提取 API 调用信息
		c.extractAPIInfoFromTracesV2(traces, edgeAPIs)
	}

	c.log.Debug("extracted API info from traces", "edge_count", len(edgeAPIs))

	// 4. 合并依赖关系和 API 信息
	edges := make([]types.ServiceEdge, 0, len(deps))
	for _, dep := range deps {
		edge := dep.ToServiceEdge()

		// 构建边的 key
		edgeKey := dep.Parent + "->" + dep.Child

		// 查找此边的所有 API 调用信息
		if pathAPIs, exists := edgeAPIs[edgeKey]; exists {
			for _, apiInfo := range pathAPIs {
				edge.APIs = append(edge.APIs, types.EdgeAPI{
					Path:      apiInfo.Path,
					Method:    apiInfo.Method,
					CallCount: apiInfo.CallCount,
				})
			}
			c.log.Debug("added APIs to edge",
				"source", dep.Parent,
				"target", dep.Child,
				"api_count", len(edge.APIs),
			)
		}

		edges = append(edges, edge)
	}

	c.log.Info("built service topology with APIs",
		"edges_count", len(edges),
	)

	return edges, nil
}

// extractAPIInfoFromTracesV2 从 traces 中提取 API 调用信息（改进版）
func (c *JaegerClient) extractAPIInfoFromTracesV2(traces []JaegerTrace, edgeAPIs map[string]map[string]*EdgeAPIInfo) {
	for _, trace := range traces {
		// 建立 processID -> serviceName 映射
		processToService := make(map[string]string)
		for pid, proc := range trace.Processes {
			processToService[pid] = proc.ServiceName
		}

		for _, span := range trace.Spans {
			// 只关注 client 类型的 span（代表对外调用）
			spanKind := c.getTagValue(span.Tags, "span.kind")
			if spanKind != "client" {
				continue
			}

			// 获取调用方服务
			source := processToService[span.ProcessID]
			if source == "" {
				continue
			}

			// 获取目标服务（从 upstream_cluster 或 http.url 中提取）
			target := c.extractTargetServiceFull(span.Tags)
			if target == "" {
				continue
			}

			// 获取 API 路径
			path := c.getTagValue(span.Tags, "grpc.path")
			if path == "" {
				path = c.getTagValue(span.Tags, "http.url")
				if path != "" {
					// 从完整 URL 中提取路径
					if u, err := url.Parse(path); err == nil {
						path = u.Path
					}
				}
			}

			if path == "" {
				path = span.OperationName
			}

			// 获取 HTTP 方法
			method := c.getTagValue(span.Tags, "http.method")
			if method == "" {
				method = "POST" // gRPC 默认为 POST
			}

			// 构建边的 key
			edgeKey := source + "->" + target

			// 聚合 API 信息
			if edgeAPIs[edgeKey] == nil {
				edgeAPIs[edgeKey] = make(map[string]*EdgeAPIInfo)
			}

			if info, exists := edgeAPIs[edgeKey][path]; exists {
				info.CallCount++
			} else {
				edgeAPIs[edgeKey][path] = &EdgeAPIInfo{
					Source:    source,
					Target:    target,
					Path:      path,
					Method:    method,
					CallCount: 1,
				}
			}
		}
	}
}

// extractTargetServiceFull 从 span tags 中提取目标服务名（保留命名空间）
func (c *JaegerClient) extractTargetServiceFull(tags []JaegerSpanTag) string {
	// 尝试从 upstream_cluster 提取
	upstream := c.getTagValue(tags, "upstream_cluster")
	if upstream != "" {
		// 格式: outbound|3550||productcatalogservice.demo.svc.cluster.local
		parts := strings.Split(upstream, "|")
		if len(parts) >= 4 {
			target := parts[3]
			// 移除 .svc.cluster.local 后缀，保留 .namespace
			target = strings.TrimSuffix(target, ".svc.cluster.local")
			return target
		}
	}

	// 尝试从 grpc.authority 或 http.url 提取
	authority := c.getTagValue(tags, "grpc.authority")
	if authority != "" {
		// 格式: productcatalogservice:3550
		// 这里没有命名空间信息，需要从其他地方获取
		parts := strings.Split(authority, ":")
		if len(parts) >= 1 {
			// 尝试从 http.url 获取完整信息
			httpURL := c.getTagValue(tags, "http.url")
			if httpURL != "" {
				if u, err := url.Parse(httpURL); err == nil {
					host := u.Hostname()
					host = strings.TrimSuffix(host, ".svc.cluster.local")
					if host != "" {
						return host
					}
				}
			}
			return parts[0]
		}
	}

	return ""
}

// FetchServices 从 Jaeger 获取所有服务列表
func (c *JaegerClient) FetchServices(ctx context.Context) ([]string, error) {
	reqURL := c.baseURL + "/api/services"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jaeger request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jaeger returned status %d", resp.StatusCode)
	}

	var response struct {
		Data   []string `json:"data"`
		Errors []string `json:"errors,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Data, nil
}

// FetchTraces 从 Jaeger 获取指定服务的 traces
func (c *JaegerClient) FetchTraces(ctx context.Context, service string, limit int) ([]JaegerTrace, error) {
	reqURL, err := url.Parse(c.baseURL + "/api/traces")
	if err != nil {
		return nil, fmt.Errorf("invalid jaeger url: %w", err)
	}

	query := reqURL.Query()
	query.Set("service", service)
	query.Set("limit", strconv.Itoa(limit))
	query.Set("lookback", c.lookback.String())
	reqURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jaeger request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jaeger returned status %d: %s", resp.StatusCode, string(body))
	}

	var response JaegerTracesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Data, nil
}

// getTagValue 获取 tag 的字符串值
func (c *JaegerClient) getTagValue(tags []JaegerSpanTag, key string) string {
	for _, tag := range tags {
		if tag.Key == key {
			switch v := tag.Value.(type) {
			case string:
				return v
			case float64:
				return strconv.FormatFloat(v, 'f', -1, 64)
			default:
				return fmt.Sprintf("%v", v)
			}
		}
	}
	return ""
}

// BuildTopologyWithAPIsGraceful 从 Jaeger 构建服务拓扑（包含 API，优雅降级）
func (c *JaegerClient) BuildTopologyWithAPIsGraceful(ctx context.Context) ([]types.ServiceEdge, error) {
	edges, err := c.BuildTopologyWithAPIs(ctx)
	if err != nil {
		c.log.Warn("jaeger unavailable, returning empty topology",
			"error", err.Error(),
		)
		return []types.ServiceEdge{}, nil
	}
	return edges, nil
}
