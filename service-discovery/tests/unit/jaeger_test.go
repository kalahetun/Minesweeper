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

// mockJaegerDependenciesResponse 创建模拟的 Jaeger dependencies 响应
func mockJaegerDependenciesResponse(deps []discovery.JaegerDependency) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		if r.URL.Path != "/api/dependencies" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		// 验证必要参数
		if r.URL.Query().Get("endTs") == "" {
			http.Error(w, "missing endTs parameter", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"data": deps,
		}
		json.NewEncoder(w).Encode(resp)
	}
}

func TestNewJaegerClient(t *testing.T) {
	client := discovery.NewJaegerClient("http://jaeger:16686", time.Hour, 30*time.Second, nil)
	assert.NotNil(t, client)
}

func TestFetchDependencies_Success(t *testing.T) {
	// 模拟 Jaeger API 响应
	deps := []discovery.JaegerDependency{
		{Parent: "productpage", Child: "reviews", CallCount: 1000},
		{Parent: "productpage", Child: "details", CallCount: 980},
		{Parent: "reviews", Child: "ratings", CallCount: 950},
	}

	server := httptest.NewServer(mockJaegerDependenciesResponse(deps))
	defer server.Close()

	client := discovery.NewJaegerClient(server.URL, time.Hour, 30*time.Second, nil)
	ctx := context.Background()

	result, err := client.FetchDependencies(ctx)

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "productpage", result[0].Parent)
	assert.Equal(t, "reviews", result[0].Child)
	assert.Equal(t, 1000, result[0].CallCount)
}

func TestFetchDependencies_EmptyResponse(t *testing.T) {
	deps := []discovery.JaegerDependency{}

	server := httptest.NewServer(mockJaegerDependenciesResponse(deps))
	defer server.Close()

	client := discovery.NewJaegerClient(server.URL, time.Hour, 30*time.Second, nil)
	ctx := context.Background()

	result, err := client.FetchDependencies(ctx)

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestFetchDependencies_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := discovery.NewJaegerClient(server.URL, time.Hour, 30*time.Second, nil)
	ctx := context.Background()

	result, err := client.FetchDependencies(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestFetchDependencies_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := discovery.NewJaegerClient(server.URL, time.Hour, 30*time.Second, nil)
	ctx := context.Background()

	result, err := client.FetchDependencies(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestFetchDependencies_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟慢响应
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 设置非常短的超时
	client := discovery.NewJaegerClient(server.URL, time.Hour, 50*time.Millisecond, nil)
	ctx := context.Background()

	result, err := client.FetchDependencies(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestFetchDependencies_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := discovery.NewJaegerClient(server.URL, time.Hour, 30*time.Second, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	result, err := client.FetchDependencies(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestBuildTopology_Success(t *testing.T) {
	deps := []discovery.JaegerDependency{
		{Parent: "productpage", Child: "reviews", CallCount: 1000},
		{Parent: "productpage", Child: "details", CallCount: 980},
		{Parent: "reviews", Child: "ratings", CallCount: 950},
	}

	server := httptest.NewServer(mockJaegerDependenciesResponse(deps))
	defer server.Close()

	client := discovery.NewJaegerClient(server.URL, time.Hour, 30*time.Second, nil)
	ctx := context.Background()

	edges, err := client.BuildTopology(ctx)

	require.NoError(t, err)
	assert.Len(t, edges, 3)

	// 验证边的转换
	edgeMap := make(map[string]types.ServiceEdge)
	for _, edge := range edges {
		key := edge.Source + "->" + edge.Target
		edgeMap[key] = edge
	}

	assert.Contains(t, edgeMap, "productpage->reviews")
	assert.Contains(t, edgeMap, "productpage->details")
	assert.Contains(t, edgeMap, "reviews->ratings")

	assert.Equal(t, 1000, edgeMap["productpage->reviews"].CallCount)
	assert.Equal(t, 980, edgeMap["productpage->details"].CallCount)
	assert.Equal(t, 950, edgeMap["reviews->ratings"].CallCount)
}

func TestBuildTopology_EmptyDependencies(t *testing.T) {
	deps := []discovery.JaegerDependency{}

	server := httptest.NewServer(mockJaegerDependenciesResponse(deps))
	defer server.Close()

	client := discovery.NewJaegerClient(server.URL, time.Hour, 30*time.Second, nil)
	ctx := context.Background()

	edges, err := client.BuildTopology(ctx)

	require.NoError(t, err)
	assert.Empty(t, edges)
}

func TestBuildTopology_JaegerUnavailable(t *testing.T) {
	// 不启动服务器，模拟 Jaeger 不可达
	client := discovery.NewJaegerClient("http://localhost:19999", time.Hour, 1*time.Second, nil)
	ctx := context.Background()

	edges, err := client.BuildTopology(ctx)

	// 应该返回错误，但不崩溃
	assert.Error(t, err)
	assert.Nil(t, edges)
}

func TestBuildTopologyWithGracefulDegradation(t *testing.T) {
	// 不启动服务器，模拟 Jaeger 不可达
	client := discovery.NewJaegerClient("http://localhost:19999", time.Hour, 1*time.Second, nil)
	ctx := context.Background()

	// 使用 graceful degradation 方法
	edges, err := client.BuildTopologyGraceful(ctx)

	// 应该返回空的 edges 而不是错误
	assert.NoError(t, err)
	assert.Empty(t, edges)
}

func TestJaegerDependency_Conversion(t *testing.T) {
	dep := discovery.JaegerDependency{
		Parent:    "service-a",
		Child:     "service-b",
		CallCount: 500,
	}

	edge := dep.ToServiceEdge()

	assert.Equal(t, "service-a", edge.Source)
	assert.Equal(t, "service-b", edge.Target)
	assert.Equal(t, 500, edge.CallCount)
}

func TestFetchDependencies_WithLookback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 lookback 参数
		lookback := r.URL.Query().Get("lookback")
		assert.NotEmpty(t, lookback)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []discovery.JaegerDependency{},
		})
	}))
	defer server.Close()

	// 使用 2 小时 lookback
	client := discovery.NewJaegerClient(server.URL, 2*time.Hour, 30*time.Second, nil)
	ctx := context.Background()

	_, err := client.FetchDependencies(ctx)
	require.NoError(t, err)
}

func TestIsAvailable(t *testing.T) {
	// 可用的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := discovery.NewJaegerClient(server.URL, time.Hour, 30*time.Second, nil)
	ctx := context.Background()

	available := client.IsAvailable(ctx)
	assert.True(t, available)
}

func TestIsAvailable_Unavailable(t *testing.T) {
	client := discovery.NewJaegerClient("http://localhost:19999", time.Hour, 1*time.Second, nil)
	ctx := context.Background()

	available := client.IsAvailable(ctx)
	assert.False(t, available)
}

func TestLargeDependencySet(t *testing.T) {
	// 创建大量依赖关系
	deps := make([]discovery.JaegerDependency, 100)
	for i := 0; i < 100; i++ {
		deps[i] = discovery.JaegerDependency{
			Parent:    "service-" + string(rune('a'+i%26)),
			Child:     "service-" + string(rune('a'+(i+1)%26)),
			CallCount: (i + 1) * 10,
		}
	}

	server := httptest.NewServer(mockJaegerDependenciesResponse(deps))
	defer server.Close()

	client := discovery.NewJaegerClient(server.URL, time.Hour, 30*time.Second, nil)
	ctx := context.Background()

	edges, err := client.BuildTopology(ctx)

	require.NoError(t, err)
	assert.Len(t, edges, 100)
}
