// Package discovery 实现服务发现功能
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/boifi/service-discovery/internal/types"
	"github.com/boifi/service-discovery/pkg/logger"
)

// OpenAPISpec OpenAPI/Swagger 规范结构
type OpenAPISpec struct {
	// OpenAPI 3.x 版本号
	OpenAPI string `json:"openapi,omitempty"`
	// Swagger 2.x 版本号
	Swagger string `json:"swagger,omitempty"`
	// API 信息
	Info OpenAPIInfo `json:"info"`
	// API 路径定义
	Paths map[string]map[string]interface{} `json:"paths"`
}

// OpenAPIInfo API 元信息
type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

// OpenAPIFetcher OpenAPI 规范获取器
type OpenAPIFetcher struct {
	paths   []string
	timeout time.Duration
	client  *http.Client
	log     logger.Logger
}

// NewOpenAPIFetcher 创建新的 OpenAPI 获取器
func NewOpenAPIFetcher(paths []string, timeout time.Duration, log logger.Logger) *OpenAPIFetcher {
	if log == nil {
		log = logger.NewDefault()
	}

	if len(paths) == 0 {
		paths = []string{"/swagger.json", "/v3/api-docs", "/openapi.json"}
	}

	return &OpenAPIFetcher{
		paths:   paths,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
		log: log,
	}
}

// FetchOpenAPI 获取服务的 OpenAPI 规范
// 依次尝试配置的路径，返回第一个成功获取的规范
func (f *OpenAPIFetcher) FetchOpenAPI(ctx context.Context, baseURL string) (*OpenAPISpec, error) {
	var lastErr error

	for _, path := range f.paths {
		url := strings.TrimSuffix(baseURL, "/") + path

		f.log.Debug("trying openapi path", "url", url)

		spec, err := f.fetchFromURL(ctx, url)
		if err != nil {
			f.log.Debug("openapi path failed", "url", url, "error", err.Error())
			lastErr = err
			continue
		}

		f.log.Info("openapi spec fetched successfully",
			"url", url,
			"title", spec.Info.Title,
			"version", spec.Info.Version,
		)
		return spec, nil
	}

	return nil, fmt.Errorf("failed to fetch openapi from any path: %w", lastErr)
}

// FetchOpenAPIGraceful 优雅获取 OpenAPI 规范
// 如果获取失败，返回 nil 而不是错误（用于降级场景）
func (f *OpenAPIFetcher) FetchOpenAPIGraceful(ctx context.Context, baseURL string) (*OpenAPISpec, error) {
	spec, err := f.FetchOpenAPI(ctx, baseURL)
	if err != nil {
		f.log.Debug("openapi fetch failed (graceful degradation)",
			"base_url", baseURL,
			"error", err.Error(),
		)
		return nil, nil // 优雅降级
	}
	return spec, nil
}

// fetchFromURL 从指定 URL 获取 OpenAPI 规范
func (f *OpenAPIFetcher) fetchFromURL(ctx context.Context, url string) (*OpenAPISpec, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return ParseOpenAPISpec(body)
}

// ParseOpenAPISpec 解析 OpenAPI 规范 JSON
func ParseOpenAPISpec(data []byte) (*OpenAPISpec, error) {
	var spec OpenAPISpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parse openapi spec: %w", err)
	}
	return &spec, nil
}

// ExtractAPIsFromOpenAPI 从 OpenAPI 规范提取 API 端点
func ExtractAPIsFromOpenAPI(spec *OpenAPISpec) []types.APIEndpoint {
	if spec == nil {
		return nil
	}

	var apis []types.APIEndpoint

	// 支持的 HTTP 方法
	httpMethods := []string{"get", "post", "put", "delete", "patch", "options", "head"}

	for path, methods := range spec.Paths {
		for _, method := range httpMethods {
			if _, ok := methods[method]; ok {
				apis = append(apis, types.APIEndpoint{
					Path:      path,
					Method:    strings.ToUpper(method),
					MatchType: "exact",
				})
			}
		}
	}

	return apis
}

// MergeAPIs 合并 API 列表，OpenAPI 发现的信息优先
// 对于相同路径，用 OpenAPI 的详细方法替换通配符方法
func MergeAPIs(existing, openAPIAPIs []types.APIEndpoint) []types.APIEndpoint {
	if len(openAPIAPIs) == 0 {
		return existing
	}
	if len(existing) == 0 {
		return openAPIAPIs
	}

	// 建立 OpenAPI API 的路径索引
	openAPIPathSet := make(map[string]bool)
	for _, api := range openAPIAPIs {
		openAPIPathSet[api.Path] = true
	}

	// 构建合并结果
	var result []types.APIEndpoint

	// 1. 添加所有 OpenAPI 的 API
	result = append(result, openAPIAPIs...)

	// 2. 添加现有 API 中不被 OpenAPI 覆盖的
	for _, api := range existing {
		// 如果是通配符方法且 OpenAPI 有该路径的详细信息，跳过
		if api.Method == "*" && openAPIPathSet[api.Path] {
			continue
		}
		// 如果 OpenAPI 没有这个路径，保留现有 API
		if !openAPIPathSet[api.Path] {
			result = append(result, api)
		}
	}

	return result
}

// EnhanceServiceWithOpenAPI 使用 OpenAPI 规范增强服务信息
func (f *OpenAPIFetcher) EnhanceServiceWithOpenAPI(ctx context.Context, service types.ServiceInfo, baseURL string) (types.ServiceInfo, error) {
	spec, err := f.FetchOpenAPIGraceful(ctx, baseURL)
	if err != nil {
		return service, err
	}

	if spec == nil {
		// OpenAPI 不可用，返回原始服务信息
		return service, nil
	}

	// 提取 OpenAPI 中的 API
	openAPIAPIs := ExtractAPIsFromOpenAPI(spec)

	// 合并 API
	service.APIs = MergeAPIs(service.APIs, openAPIAPIs)

	f.log.Debug("service enhanced with openapi",
		"service", service.Name,
		"api_count", len(service.APIs),
	)

	return service, nil
}

// EnhanceServicesWithOpenAPI 批量增强服务信息
func (f *OpenAPIFetcher) EnhanceServicesWithOpenAPI(ctx context.Context, services []types.ServiceInfo, urlResolver func(types.ServiceInfo) string) []types.ServiceInfo {
	if urlResolver == nil {
		return services
	}

	enhanced := make([]types.ServiceInfo, len(services))

	for i, svc := range services {
		url := urlResolver(svc)
		if url == "" {
			enhanced[i] = svc
			continue
		}

		enhancedSvc, err := f.EnhanceServiceWithOpenAPI(ctx, svc, url)
		if err != nil {
			f.log.Debug("failed to enhance service with openapi",
				"service", svc.Name,
				"error", err.Error(),
			)
			enhanced[i] = svc
		} else {
			enhanced[i] = enhancedSvc
		}
	}

	return enhanced
}
