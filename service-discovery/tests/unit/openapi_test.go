package unit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/boifi/service-discovery/internal/discovery"
	"github.com/boifi/service-discovery/internal/types"
)

func TestNewOpenAPIFetcher(t *testing.T) {
	fetcher := discovery.NewOpenAPIFetcher(
		[]string{"/swagger.json", "/v3/api-docs"},
		5*time.Second,
		nil,
	)

	assert.NotNil(t, fetcher)
}

func TestFetchOpenAPI_Success(t *testing.T) {
	// 模拟 OpenAPI 响应
	openAPISpec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":   "Test API",
			"version": "1.0.0",
		},
		"paths": map[string]interface{}{
			"/users": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get users",
				},
				"post": map[string]interface{}{
					"summary": "Create user",
				},
			},
			"/users/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get user by ID",
				},
				"put": map[string]interface{}{
					"summary": "Update user",
				},
				"delete": map[string]interface{}{
					"summary": "Delete user",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/swagger.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(openAPISpec)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := discovery.NewOpenAPIFetcher(
		[]string{"/swagger.json", "/v3/api-docs"},
		5*time.Second,
		nil,
	)

	ctx := context.Background()
	spec, err := fetcher.FetchOpenAPI(ctx, server.URL)

	require.NoError(t, err)
	assert.NotNil(t, spec)
	assert.Equal(t, "3.0.0", spec.OpenAPI)
	assert.Equal(t, "Test API", spec.Info.Title)
	assert.Len(t, spec.Paths, 2)
}

func TestFetchOpenAPI_FallbackPath(t *testing.T) {
	// 第一个路径失败，第二个成功
	openAPISpec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":   "Fallback API",
			"version": "2.0.0",
		},
		"paths": map[string]interface{}{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/swagger.json":
			w.WriteHeader(http.StatusNotFound)
		case "/v3/api-docs":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(openAPISpec)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	fetcher := discovery.NewOpenAPIFetcher(
		[]string{"/swagger.json", "/v3/api-docs"},
		5*time.Second,
		nil,
	)

	ctx := context.Background()
	spec, err := fetcher.FetchOpenAPI(ctx, server.URL)

	require.NoError(t, err)
	assert.Equal(t, "Fallback API", spec.Info.Title)
}

func TestFetchOpenAPI_AllPathsFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := discovery.NewOpenAPIFetcher(
		[]string{"/swagger.json", "/v3/api-docs"},
		5*time.Second,
		nil,
	)

	ctx := context.Background()
	spec, err := fetcher.FetchOpenAPI(ctx, server.URL)

	assert.Error(t, err)
	assert.Nil(t, spec)
}

func TestFetchOpenAPI_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	fetcher := discovery.NewOpenAPIFetcher(
		[]string{"/swagger.json"},
		5*time.Second,
		nil,
	)

	ctx := context.Background()
	spec, err := fetcher.FetchOpenAPI(ctx, server.URL)

	assert.Error(t, err)
	assert.Nil(t, spec)
}

func TestFetchOpenAPI_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := discovery.NewOpenAPIFetcher(
		[]string{"/swagger.json"},
		50*time.Millisecond, // 很短的超时
		nil,
	)

	ctx := context.Background()
	spec, err := fetcher.FetchOpenAPI(ctx, server.URL)

	assert.Error(t, err)
	assert.Nil(t, spec)
}

func TestFetchOpenAPI_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := discovery.NewOpenAPIFetcher(
		[]string{"/swagger.json"},
		5*time.Second,
		nil,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	spec, err := fetcher.FetchOpenAPI(ctx, server.URL)

	assert.Error(t, err)
	assert.Nil(t, spec)
}

func TestParseOpenAPISpec_Swagger2(t *testing.T) {
	swagger2Spec := map[string]interface{}{
		"swagger": "2.0",
		"info": map[string]interface{}{
			"title":   "Swagger 2 API",
			"version": "1.0.0",
		},
		"paths": map[string]interface{}{
			"/items": map[string]interface{}{
				"get":  map[string]interface{}{},
				"post": map[string]interface{}{},
			},
		},
	}

	data, _ := json.Marshal(swagger2Spec)

	spec, err := discovery.ParseOpenAPISpec(data)

	require.NoError(t, err)
	assert.Equal(t, "2.0", spec.Swagger)
	assert.Equal(t, "Swagger 2 API", spec.Info.Title)
}

func TestParseOpenAPISpec_OpenAPI3(t *testing.T) {
	openapi3Spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":   "OpenAPI 3 API",
			"version": "3.0.0",
		},
		"paths": map[string]interface{}{
			"/products": map[string]interface{}{
				"get": map[string]interface{}{},
			},
		},
	}

	data, _ := json.Marshal(openapi3Spec)

	spec, err := discovery.ParseOpenAPISpec(data)

	require.NoError(t, err)
	assert.Equal(t, "3.0.0", spec.OpenAPI)
	assert.Equal(t, "OpenAPI 3 API", spec.Info.Title)
}

func TestExtractAPIsFromOpenAPI(t *testing.T) {
	spec := &discovery.OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: discovery.OpenAPIInfo{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: map[string]map[string]interface{}{
			"/users": {
				"get":  map[string]interface{}{},
				"post": map[string]interface{}{},
			},
			"/users/{id}": {
				"get":    map[string]interface{}{},
				"put":    map[string]interface{}{},
				"delete": map[string]interface{}{},
			},
		},
	}

	apis := discovery.ExtractAPIsFromOpenAPI(spec)

	assert.Len(t, apis, 5)

	// 验证 API 路径和方法
	methodPathMap := make(map[string]bool)
	for _, api := range apis {
		key := api.Method + " " + api.Path
		methodPathMap[key] = true
	}

	assert.True(t, methodPathMap["GET /users"])
	assert.True(t, methodPathMap["POST /users"])
	assert.True(t, methodPathMap["GET /users/{id}"])
	assert.True(t, methodPathMap["PUT /users/{id}"])
	assert.True(t, methodPathMap["DELETE /users/{id}"])
}

func TestMergeAPIs_OpenAPIPriority(t *testing.T) {
	// 现有 API (来自 VirtualService)
	existingAPIs := []types.APIEndpoint{
		{Path: "/users", Method: "*", MatchType: "prefix"},
		{Path: "/products", Method: "*", MatchType: "exact"},
	}

	// OpenAPI 发现的 API
	openAPIAPIs := []types.APIEndpoint{
		{Path: "/users", Method: "GET", MatchType: "exact"},
		{Path: "/users", Method: "POST", MatchType: "exact"},
		{Path: "/orders", Method: "GET", MatchType: "exact"},
	}

	merged := discovery.MergeAPIs(existingAPIs, openAPIAPIs)

	// /users 应该被 OpenAPI 的详细信息替换
	// /products 保留
	// /orders 新增
	assert.GreaterOrEqual(t, len(merged), 4)

	// 验证合并后的结果包含 OpenAPI 的详细 API
	hasGetUsers := false
	hasPostUsers := false
	hasProducts := false
	hasOrders := false

	for _, api := range merged {
		if api.Path == "/users" && api.Method == "GET" {
			hasGetUsers = true
		}
		if api.Path == "/users" && api.Method == "POST" {
			hasPostUsers = true
		}
		if api.Path == "/products" {
			hasProducts = true
		}
		if api.Path == "/orders" {
			hasOrders = true
		}
	}

	assert.True(t, hasGetUsers, "should have GET /users from OpenAPI")
	assert.True(t, hasPostUsers, "should have POST /users from OpenAPI")
	assert.True(t, hasProducts, "should keep /products from existing")
	assert.True(t, hasOrders, "should add /orders from OpenAPI")
}

func TestMergeAPIs_EmptyOpenAPI(t *testing.T) {
	existingAPIs := []types.APIEndpoint{
		{Path: "/users", Method: "*", MatchType: "prefix"},
	}

	merged := discovery.MergeAPIs(existingAPIs, nil)

	assert.Equal(t, existingAPIs, merged)
}

func TestMergeAPIs_EmptyExisting(t *testing.T) {
	openAPIAPIs := []types.APIEndpoint{
		{Path: "/users", Method: "GET", MatchType: "exact"},
	}

	merged := discovery.MergeAPIs(nil, openAPIAPIs)

	assert.Equal(t, openAPIAPIs, merged)
}

func TestFetchOpenAPIGraceful_Unavailable(t *testing.T) {
	fetcher := discovery.NewOpenAPIFetcher(
		[]string{"/swagger.json"},
		time.Second,
		nil,
	)

	ctx := context.Background()
	// 不存在的服务
	spec, err := fetcher.FetchOpenAPIGraceful(ctx, "http://localhost:19999")

	// 优雅降级 - 不返回错误，返回 nil spec
	assert.NoError(t, err)
	assert.Nil(t, spec)
}

func TestFetchOpenAPIGraceful_Success(t *testing.T) {
	openAPISpec := map[string]interface{}{
		"openapi": "3.0.0",
		"info":    map[string]interface{}{"title": "Test"},
		"paths":   map[string]interface{}{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAPISpec)
	}))
	defer server.Close()

	fetcher := discovery.NewOpenAPIFetcher(
		[]string{"/swagger.json"},
		5*time.Second,
		nil,
	)

	ctx := context.Background()
	spec, err := fetcher.FetchOpenAPIGraceful(ctx, server.URL)

	require.NoError(t, err)
	assert.NotNil(t, spec)
}

func TestEnhanceServiceWithOpenAPI(t *testing.T) {
	openAPISpec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":   "User Service API",
			"version": "2.0.0",
		},
		"paths": map[string]interface{}{
			"/users": map[string]interface{}{
				"get":  map[string]interface{}{},
				"post": map[string]interface{}{},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAPISpec)
	}))
	defer server.Close()

	fetcher := discovery.NewOpenAPIFetcher(
		[]string{"/swagger.json"},
		5*time.Second,
		nil,
	)

	service := types.ServiceInfo{
		Name:      "user-service",
		Namespace: "default",
		APIs: []types.APIEndpoint{
			{Path: "/users", Method: "*", MatchType: "prefix"},
		},
	}

	ctx := context.Background()
	enhanced, err := fetcher.EnhanceServiceWithOpenAPI(ctx, service, server.URL)

	require.NoError(t, err)
	// API 应该被增强
	assert.GreaterOrEqual(t, len(enhanced.APIs), 2)
}

func TestOpenAPISpec_HTTPMethods(t *testing.T) {
	// 测试支持的 HTTP 方法
	spec := &discovery.OpenAPISpec{
		Paths: map[string]map[string]interface{}{
			"/resource": {
				"get":     map[string]interface{}{},
				"post":    map[string]interface{}{},
				"put":     map[string]interface{}{},
				"delete":  map[string]interface{}{},
				"patch":   map[string]interface{}{},
				"options": map[string]interface{}{},
				"head":    map[string]interface{}{},
			},
		},
	}

	apis := discovery.ExtractAPIsFromOpenAPI(spec)

	assert.Len(t, apis, 7)

	methods := make(map[string]bool)
	for _, api := range apis {
		methods[api.Method] = true
	}

	assert.True(t, methods["GET"])
	assert.True(t, methods["POST"])
	assert.True(t, methods["PUT"])
	assert.True(t, methods["DELETE"])
	assert.True(t, methods["PATCH"])
	assert.True(t, methods["OPTIONS"])
	assert.True(t, methods["HEAD"])
}

func TestExtractAPIsFromOpenAPI_EmptyPaths(t *testing.T) {
	spec := &discovery.OpenAPISpec{
		Paths: map[string]map[string]interface{}{},
	}

	apis := discovery.ExtractAPIsFromOpenAPI(spec)

	assert.Empty(t, apis)
}

func TestExtractAPIsFromOpenAPI_NilSpec(t *testing.T) {
	apis := discovery.ExtractAPIsFromOpenAPI(nil)

	assert.Empty(t, apis)
}
