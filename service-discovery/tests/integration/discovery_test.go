package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/boifi/service-discovery/internal/discovery"
	"github.com/boifi/service-discovery/internal/publisher"
	"github.com/boifi/service-discovery/internal/scheduler"
	"github.com/boifi/service-discovery/internal/types"
)

// MockK8sDiscoverer 模拟 K8s 发现器
type MockK8sDiscoverer struct {
	services []types.ServiceInfo
}

func (m *MockK8sDiscoverer) Discover(ctx context.Context, namespace string) ([]types.ServiceInfo, error) {
	return m.services, nil
}

// TestEndToEndDiscoveryFlow 端到端发现流程测试
func TestEndToEndDiscoveryFlow(t *testing.T) {
	// 1. 启动 miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// 2. 创建模拟 Jaeger 服务器
	jaegerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/dependencies" {
			deps := []map[string]interface{}{
				{"parent": "frontend", "child": "backend", "callCount": 1000},
				{"parent": "backend", "child": "database", "callCount": 500},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"data": deps})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer jaegerServer.Close()

	// 3. 创建组件
	k8sDiscoverer := &MockK8sDiscoverer{
		services: []types.ServiceInfo{
			{Name: "frontend", Namespace: "default", APIs: []types.APIEndpoint{{Path: "/", Method: "*"}}},
			{Name: "backend", Namespace: "default", APIs: []types.APIEndpoint{{Path: "/api", Method: "*"}}},
			{Name: "database", Namespace: "default", APIs: []types.APIEndpoint{}},
		},
	}

	jaegerClient := discovery.NewJaegerClient(
		jaegerServer.URL,
		5*time.Second,
		time.Hour,
		nil,
	)

	redisPublisher := publisher.NewRedisPublisher(
		mr.Addr(),
		"",
		0,
		"test:service-map",
		"test:updates",
		nil,
	)
	defer redisPublisher.Close()

	// 4. 创建并运行调度器
	sched := scheduler.NewScheduler(
		k8sDiscoverer,
		jaegerClient,
		redisPublisher,
		time.Minute,
		"1h",
		"1m",
		nil,
	)

	ctx := context.Background()

	// 执行一次发现
	err = sched.RunDiscovery(ctx)
	require.NoError(t, err)

	// 5. 验证结果
	// 检查 Redis 中的数据
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	data, err := client.Get(ctx, "test:service-map").Bytes()
	require.NoError(t, err)

	var sm types.ServiceMap
	err = json.Unmarshal(data, &sm)
	require.NoError(t, err)

	// 验证服务数量
	assert.Equal(t, 3, sm.ServiceCount())

	// 验证边的数量
	assert.Equal(t, 2, sm.EdgeCount())

	// 验证元数据
	assert.Equal(t, "1h", sm.Metadata.JaegerLookback)
	assert.False(t, sm.Metadata.Stale)
}

// TestDiscoveryWithOpenAPIEnhancement 带 OpenAPI 增强的发现流程
func TestDiscoveryWithOpenAPIEnhancement(t *testing.T) {
	// 1. 启动 miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// 2. 创建模拟 OpenAPI 服务器
	openapiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/swagger.json" {
			spec := map[string]interface{}{
				"openapi": "3.0.0",
				"info":    map[string]interface{}{"title": "Test API", "version": "1.0.0"},
				"paths": map[string]interface{}{
					"/users":      map[string]interface{}{"get": map[string]interface{}{}, "post": map[string]interface{}{}},
					"/users/{id}": map[string]interface{}{"get": map[string]interface{}{}, "delete": map[string]interface{}{}},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(spec)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer openapiServer.Close()

	// 3. 测试 OpenAPI 获取
	fetcher := discovery.NewOpenAPIFetcher(
		[]string{"/swagger.json"},
		5*time.Second,
		nil,
	)

	ctx := context.Background()
	spec, err := fetcher.FetchOpenAPI(ctx, openapiServer.URL)

	require.NoError(t, err)
	assert.NotNil(t, spec)
	assert.Equal(t, "Test API", spec.Info.Title)

	// 提取 API
	apis := discovery.ExtractAPIsFromOpenAPI(spec)
	assert.Len(t, apis, 4) // GET /users, POST /users, GET /users/{id}, DELETE /users/{id}
}

// TestGracefulDegradation 测试优雅降级
func TestGracefulDegradation(t *testing.T) {
	// 1. 启动 miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// 2. 创建 Jaeger 客户端（指向不存在的服务）
	jaegerClient := discovery.NewJaegerClient(
		"http://localhost:19999",
		time.Second,
		time.Hour,
		nil,
	)

	// 3. 创建组件
	k8sDiscoverer := &MockK8sDiscoverer{
		services: []types.ServiceInfo{
			{Name: "service-a", Namespace: "default"},
		},
	}

	redisPublisher := publisher.NewRedisPublisher(
		mr.Addr(),
		"",
		0,
		"test:service-map",
		"test:updates",
		nil,
	)
	defer redisPublisher.Close()

	// 4. 创建调度器
	sched := scheduler.NewScheduler(
		k8sDiscoverer,
		jaegerClient,
		redisPublisher,
		time.Minute,
		"1h",
		"1m",
		nil,
	)

	ctx := context.Background()

	// 执行发现 - 应该成功（Jaeger 不可用时优雅降级）
	err = sched.RunDiscovery(ctx)
	require.NoError(t, err)

	// 验证服务被发布
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	data, err := client.Get(ctx, "test:service-map").Bytes()
	require.NoError(t, err)

	var sm types.ServiceMap
	err = json.Unmarshal(data, &sm)
	require.NoError(t, err)

	assert.Equal(t, 1, sm.ServiceCount())
	assert.Equal(t, 0, sm.EdgeCount()) // 没有边（Jaeger 不可用）
}

// TestPeriodicExecution 测试周期执行
func TestPeriodicExecution(t *testing.T) {
	// 1. 启动 miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// 2. 创建模拟 Jaeger
	jaegerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
	}))
	defer jaegerServer.Close()

	// 3. 创建组件
	k8sDiscoverer := &MockK8sDiscoverer{
		services: []types.ServiceInfo{{Name: "test-service"}},
	}

	jaegerClient := discovery.NewJaegerClient(
		jaegerServer.URL,
		5*time.Second,
		time.Hour,
		nil,
	)

	redisPublisher := publisher.NewRedisPublisher(
		mr.Addr(),
		"",
		0,
		"test:service-map",
		"test:updates",
		nil,
	)
	defer redisPublisher.Close()

	// 4. 创建调度器（短周期）
	sched := scheduler.NewScheduler(
		k8sDiscoverer,
		jaegerClient,
		redisPublisher,
		50*time.Millisecond,
		"1h",
		"50ms",
		nil,
	)

	// 5. 启动调度器
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	sched.Start(ctx)

	// 等待一段时间
	<-ctx.Done()
	sched.Stop()

	// 验证多次执行 - 使用新的 context
	verifyCtx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	// 数据应该存在
	exists, err := client.Exists(verifyCtx, "test:service-map").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists)
}

// TestSubscribeToUpdates 测试订阅更新通知
func TestSubscribeToUpdates(t *testing.T) {
	// 1. 启动 miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// 2. 创建 Redis 客户端用于订阅
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 订阅通道
	pubsub := client.Subscribe(ctx, "test:updates")
	defer pubsub.Close()

	// 等待订阅确认
	_, err = pubsub.Receive(ctx)
	require.NoError(t, err)

	// 3. 创建发布器并发布
	redisPublisher := publisher.NewRedisPublisher(
		mr.Addr(),
		"",
		0,
		"test:service-map",
		"test:updates",
		nil,
	)
	defer redisPublisher.Close()

	sm := types.NewServiceMap()
	sm.AddService(types.ServiceInfo{Name: "test-service"})

	err = redisPublisher.PublishAndNotify(ctx, sm)
	require.NoError(t, err)

	// 4. 接收通知
	msgCh := pubsub.Channel()
	select {
	case msg := <-msgCh:
		assert.Equal(t, "test:updates", msg.Channel)
		assert.Contains(t, msg.Payload, "updated")
	case <-time.After(time.Second):
		t.Fatal("did not receive update notification")
	}
}
