package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/boifi/service-discovery/internal/publisher"
	"github.com/boifi/service-discovery/internal/types"
)

// createTestServiceMap 创建测试用的 ServiceMap
func createTestServiceMap() *types.ServiceMap {
	sm := types.NewServiceMap()
	sm.AddService(types.ServiceInfo{
		Name:      "productpage",
		Namespace: "default",
		APIs: []types.APIEndpoint{
			{Method: "GET", Path: "/productpage", MatchType: types.MatchTypeExact},
		},
		Source: types.SourceVirtualService,
	})
	sm.AddService(types.ServiceInfo{
		Name:      "reviews",
		Namespace: "default",
		APIs: []types.APIEndpoint{
			{Method: "*", Path: "/reviews", MatchType: types.MatchTypePrefix},
		},
		Source: types.SourceVirtualService,
	})
	sm.AddEdge(types.ServiceEdge{
		Source:    "productpage",
		Target:    "reviews",
		CallCount: 1000,
	})
	sm.Metadata.DiscoveryInterval = "5m"
	sm.Metadata.JaegerLookback = "1h"
	return sm
}

func TestNewRedisPublisher(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	pub := publisher.NewRedisPublisher(s.Addr(), "", 0, "test:key", "test:channel", nil)
	assert.NotNil(t, pub)
}

func TestNewRedisPublisher_WithPassword(t *testing.T) {
	s := miniredis.RunT(t)
	s.RequireAuth("testpassword")
	defer s.Close()

	pub := publisher.NewRedisPublisher(s.Addr(), "testpassword", 0, "test:key", "test:channel", nil)
	assert.NotNil(t, pub)

	// 验证可以连接
	ctx := context.Background()
	err := pub.Ping(ctx)
	assert.NoError(t, err)
}

func TestPublishServiceMap_Success(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	pub := publisher.NewRedisPublisher(s.Addr(), "", 0, "boifi:service-map", "boifi:updates", nil)
	ctx := context.Background()

	sm := createTestServiceMap()

	err := pub.PublishServiceMap(ctx, sm)
	require.NoError(t, err)

	// 验证数据已存储
	stored, err := s.Get("boifi:service-map")
	require.NoError(t, err)
	assert.NotEmpty(t, stored)

	// 验证 JSON 格式正确
	var decoded types.ServiceMap
	err = json.Unmarshal([]byte(stored), &decoded)
	require.NoError(t, err)
	assert.Equal(t, 2, len(decoded.Services))
	assert.Equal(t, 1, len(decoded.Topology))
}

func TestPublishServiceMap_OverwriteExisting(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	pub := publisher.NewRedisPublisher(s.Addr(), "", 0, "boifi:service-map", "boifi:updates", nil)
	ctx := context.Background()

	// 第一次发布
	sm1 := createTestServiceMap()
	err := pub.PublishServiceMap(ctx, sm1)
	require.NoError(t, err)

	// 第二次发布（覆盖）
	sm2 := types.NewServiceMap()
	sm2.AddService(types.ServiceInfo{
		Name:      "new-service",
		Namespace: "production",
		Source:    types.SourceVirtualService,
	})
	err = pub.PublishServiceMap(ctx, sm2)
	require.NoError(t, err)

	// 验证数据已被覆盖
	stored, err := s.Get("boifi:service-map")
	require.NoError(t, err)

	var decoded types.ServiceMap
	err = json.Unmarshal([]byte(stored), &decoded)
	require.NoError(t, err)
	assert.Equal(t, 1, len(decoded.Services))
	assert.Contains(t, decoded.Services, "new-service")
}

func TestNotifyUpdate_Success(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	pub := publisher.NewRedisPublisher(s.Addr(), "", 0, "boifi:service-map", "boifi:updates", nil)
	ctx := context.Background()

	err := pub.NotifyUpdate(ctx)
	require.NoError(t, err)
}

func TestPublishAndNotify_Success(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	pub := publisher.NewRedisPublisher(s.Addr(), "", 0, "boifi:service-map", "boifi:updates", nil)
	ctx := context.Background()

	sm := createTestServiceMap()

	err := pub.PublishAndNotify(ctx, sm)
	require.NoError(t, err)

	// 验证数据已存储
	stored, err := s.Get("boifi:service-map")
	require.NoError(t, err)
	assert.NotEmpty(t, stored)
}

func TestPublishServiceMap_ConnectionError(t *testing.T) {
	// 使用不存在的 Redis 地址
	pub := publisher.NewRedisPublisher("localhost:19999", "", 0, "test:key", "test:channel", nil)
	ctx := context.Background()

	sm := createTestServiceMap()

	err := pub.PublishServiceMap(ctx, sm)
	assert.Error(t, err)
}

func TestPublishServiceMap_ContextCanceled(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	pub := publisher.NewRedisPublisher(s.Addr(), "", 0, "test:key", "test:channel", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	sm := createTestServiceMap()

	err := pub.PublishServiceMap(ctx, sm)
	assert.Error(t, err)
}

func TestSerializeServiceMap(t *testing.T) {
	sm := createTestServiceMap()

	data, err := publisher.SerializeServiceMap(sm)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// 验证可以反序列化
	var decoded types.ServiceMap
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, sm.ServiceCount(), decoded.ServiceCount())
	assert.Equal(t, sm.EdgeCount(), decoded.EdgeCount())
}

func TestSerializeServiceMap_EmptyMap(t *testing.T) {
	sm := types.NewServiceMap()

	data, err := publisher.SerializeServiceMap(sm)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded types.ServiceMap
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Empty(t, decoded.Services)
	assert.Empty(t, decoded.Topology)
}

func TestPing_Success(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	pub := publisher.NewRedisPublisher(s.Addr(), "", 0, "test:key", "test:channel", nil)
	ctx := context.Background()

	err := pub.Ping(ctx)
	assert.NoError(t, err)
}

func TestPing_ConnectionError(t *testing.T) {
	pub := publisher.NewRedisPublisher("localhost:19999", "", 0, "test:key", "test:channel", nil)
	ctx := context.Background()

	err := pub.Ping(ctx)
	assert.Error(t, err)
}

func TestClose(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	pub := publisher.NewRedisPublisher(s.Addr(), "", 0, "test:key", "test:channel", nil)

	err := pub.Close()
	assert.NoError(t, err)
}

func TestRetryWithBackoff_Success(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 3 {
			return assert.AnError
		}
		return nil
	}

	err := publisher.RetryWithBackoff(fn, 5, 10*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestRetryWithBackoff_AllFail(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		return assert.AnError
	}

	err := publisher.RetryWithBackoff(fn, 3, 10*time.Millisecond)
	assert.Error(t, err)
	assert.Equal(t, 3, attempts)
}

func TestRetryWithBackoff_ImmediateSuccess(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		return nil
	}

	err := publisher.RetryWithBackoff(fn, 5, 10*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestPublishWithRetry_Success(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	pub := publisher.NewRedisPublisher(s.Addr(), "", 0, "test:key", "test:channel", nil)
	ctx := context.Background()

	sm := createTestServiceMap()

	err := pub.PublishWithRetry(ctx, sm, 3)
	require.NoError(t, err)

	// 验证数据已存储
	stored, err := s.Get("test:key")
	require.NoError(t, err)
	assert.NotEmpty(t, stored)
}

func TestSubscribe(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	// 创建发布者
	pub := publisher.NewRedisPublisher(s.Addr(), "", 0, "test:key", "test:channel", nil)

	// 创建独立的 Redis 客户端用于订阅
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 订阅频道
	pubsub := client.Subscribe(ctx, "test:channel")
	defer pubsub.Close()

	// 等待订阅确认
	_, err := pubsub.Receive(ctx)
	require.NoError(t, err)

	// 发布消息
	go func() {
		time.Sleep(100 * time.Millisecond)
		pub.NotifyUpdate(context.Background())
	}()

	// 接收消息
	msg, err := pubsub.ReceiveMessage(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test:channel", msg.Channel)
	assert.Equal(t, "updated", msg.Payload)
}

func TestLargeServiceMap(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	pub := publisher.NewRedisPublisher(s.Addr(), "", 0, "test:key", "test:channel", nil)
	ctx := context.Background()

	// 创建大型 ServiceMap
	sm := types.NewServiceMap()
	for i := 0; i < 100; i++ {
		sm.AddService(types.ServiceInfo{
			Name:      "service-" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			Namespace: "default",
			APIs: []types.APIEndpoint{
				{Method: "GET", Path: "/api/v1/resource", MatchType: types.MatchTypePrefix},
				{Method: "POST", Path: "/api/v1/resource", MatchType: types.MatchTypePrefix},
			},
			Source: types.SourceVirtualService,
		})
	}
	for i := 0; i < 200; i++ {
		sm.AddEdge(types.ServiceEdge{
			Source:    "service-" + string(rune('a'+i%26)),
			Target:    "service-" + string(rune('a'+(i+1)%26)),
			CallCount: i * 10,
		})
	}

	err := pub.PublishServiceMap(ctx, sm)
	require.NoError(t, err)

	// 验证数据完整性
	stored, err := s.Get("test:key")
	require.NoError(t, err)

	var decoded types.ServiceMap
	err = json.Unmarshal([]byte(stored), &decoded)
	require.NoError(t, err)
	assert.Equal(t, 100, len(decoded.Services))
	assert.Equal(t, 200, len(decoded.Topology))
}

func TestGetKey(t *testing.T) {
	pub := publisher.NewRedisPublisher("localhost:6379", "", 0, "my:custom:key", "my:channel", nil)
	assert.Equal(t, "my:custom:key", pub.GetKey())
}

func TestGetChannel(t *testing.T) {
	pub := publisher.NewRedisPublisher("localhost:6379", "", 0, "my:key", "my:custom:channel", nil)
	assert.Equal(t, "my:custom:channel", pub.GetChannel())
}
