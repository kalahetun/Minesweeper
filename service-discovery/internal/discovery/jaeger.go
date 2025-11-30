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
