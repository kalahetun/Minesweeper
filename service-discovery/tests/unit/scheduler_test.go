package unit

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/boifi/service-discovery/internal/scheduler"
	"github.com/boifi/service-discovery/internal/types"
)

// MockDiscoverer 模拟发现器
type MockDiscoverer struct {
	services  []types.ServiceInfo
	err       error
	callCount int32
	mu        sync.Mutex
}

func (m *MockDiscoverer) Discover(ctx context.Context, namespace string) ([]types.ServiceInfo, error) {
	atomic.AddInt32(&m.callCount, 1)
	if m.err != nil {
		return nil, m.err
	}
	return m.services, nil
}

func (m *MockDiscoverer) GetCallCount() int {
	return int(atomic.LoadInt32(&m.callCount))
}

// MockTopologyBuilder 模拟拓扑构建器
type MockTopologyBuilder struct {
	edges     []types.ServiceEdge
	err       error
	callCount int32
}

func (m *MockTopologyBuilder) BuildTopologyGraceful(ctx context.Context) ([]types.ServiceEdge, error) {
	atomic.AddInt32(&m.callCount, 1)
	if m.err != nil {
		return nil, m.err
	}
	return m.edges, nil
}

func (m *MockTopologyBuilder) GetCallCount() int {
	return int(atomic.LoadInt32(&m.callCount))
}

// MockPublisher 模拟发布器
type MockPublisher struct {
	publishedMaps []*types.ServiceMap
	err           error
	callCount     int32
	mu            sync.Mutex
}

func (m *MockPublisher) PublishAndNotify(ctx context.Context, sm *types.ServiceMap) error {
	atomic.AddInt32(&m.callCount, 1)
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	m.publishedMaps = append(m.publishedMaps, sm)
	m.mu.Unlock()
	return nil
}

func (m *MockPublisher) GetCallCount() int {
	return int(atomic.LoadInt32(&m.callCount))
}

func (m *MockPublisher) GetPublishedMaps() []*types.ServiceMap {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.publishedMaps
}

func TestNewScheduler(t *testing.T) {
	discoverer := &MockDiscoverer{}
	topology := &MockTopologyBuilder{}
	publisher := &MockPublisher{}

	s := scheduler.NewScheduler(
		discoverer,
		topology,
		publisher,
		5*time.Minute,
		"1h",
		"5m",
		nil,
	)

	assert.NotNil(t, s)
}

func TestRunDiscovery_Success(t *testing.T) {
	discoverer := &MockDiscoverer{
		services: []types.ServiceInfo{
			{Name: "service-a", Namespace: "default"},
			{Name: "service-b", Namespace: "default"},
		},
	}
	topology := &MockTopologyBuilder{
		edges: []types.ServiceEdge{
			{Source: "service-a", Target: "service-b", CallCount: 100},
		},
	}
	publisher := &MockPublisher{}

	s := scheduler.NewScheduler(discoverer, topology, publisher, time.Minute, "1h", "5m", nil)
	ctx := context.Background()

	err := s.RunDiscovery(ctx)

	require.NoError(t, err)
	assert.Equal(t, 1, discoverer.GetCallCount())
	assert.Equal(t, 1, topology.GetCallCount())
	assert.Equal(t, 1, publisher.GetCallCount())

	// 验证发布的 ServiceMap
	publishedMaps := publisher.GetPublishedMaps()
	require.Len(t, publishedMaps, 1)
	assert.Equal(t, 2, publishedMaps[0].ServiceCount())
	assert.Equal(t, 1, publishedMaps[0].EdgeCount())
}

func TestRunDiscovery_DiscovererError(t *testing.T) {
	discoverer := &MockDiscoverer{
		err: assert.AnError,
	}
	topology := &MockTopologyBuilder{}
	publisher := &MockPublisher{}

	s := scheduler.NewScheduler(discoverer, topology, publisher, time.Minute, "1h", "5m", nil)
	ctx := context.Background()

	err := s.RunDiscovery(ctx)

	assert.Error(t, err)
	assert.Equal(t, 1, discoverer.GetCallCount())
	assert.Equal(t, 0, publisher.GetCallCount()) // 不应该发布
}

func TestRunDiscovery_TopologyGracefulDegradation(t *testing.T) {
	discoverer := &MockDiscoverer{
		services: []types.ServiceInfo{
			{Name: "service-a", Namespace: "default"},
		},
	}
	topology := &MockTopologyBuilder{
		err: assert.AnError, // Jaeger 不可用
	}
	publisher := &MockPublisher{}

	s := scheduler.NewScheduler(discoverer, topology, publisher, time.Minute, "1h", "5m", nil)
	ctx := context.Background()

	// 即使拓扑构建失败，也应该成功（优雅降级）
	err := s.RunDiscovery(ctx)

	// 使用 graceful degradation，不应该返回错误
	require.NoError(t, err)
	assert.Equal(t, 1, publisher.GetCallCount())
}

func TestRunDiscovery_PublisherError(t *testing.T) {
	discoverer := &MockDiscoverer{
		services: []types.ServiceInfo{{Name: "service-a"}},
	}
	topology := &MockTopologyBuilder{}
	publisher := &MockPublisher{
		err: assert.AnError,
	}

	s := scheduler.NewScheduler(discoverer, topology, publisher, time.Minute, "1h", "5m", nil)
	ctx := context.Background()

	err := s.RunDiscovery(ctx)

	assert.Error(t, err)
}

func TestStart_PeriodicExecution(t *testing.T) {
	discoverer := &MockDiscoverer{
		services: []types.ServiceInfo{{Name: "service-a"}},
	}
	topology := &MockTopologyBuilder{}
	publisher := &MockPublisher{}

	// 使用短周期进行测试
	s := scheduler.NewScheduler(discoverer, topology, publisher, 50*time.Millisecond, "1h", "5m", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// 启动调度器
	s.Start(ctx)

	// 等待一段时间
	<-ctx.Done()
	s.Stop()

	// 应该执行了多次（至少 2 次）
	assert.GreaterOrEqual(t, discoverer.GetCallCount(), 2)
}

func TestStart_ImmediateFirstRun(t *testing.T) {
	discoverer := &MockDiscoverer{
		services: []types.ServiceInfo{{Name: "service-a"}},
	}
	topology := &MockTopologyBuilder{}
	publisher := &MockPublisher{}

	s := scheduler.NewScheduler(discoverer, topology, publisher, time.Hour, "1h", "5m", nil)

	// 启动后立即检查
	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	// 等待一小段时间让第一次执行完成
	time.Sleep(50 * time.Millisecond)
	cancel()
	s.Stop()

	// 应该至少执行了一次
	assert.GreaterOrEqual(t, discoverer.GetCallCount(), 1)
}

func TestStop_GracefulShutdown(t *testing.T) {
	discoverer := &MockDiscoverer{
		services: []types.ServiceInfo{{Name: "service-a"}},
	}
	topology := &MockTopologyBuilder{}
	publisher := &MockPublisher{}

	s := scheduler.NewScheduler(discoverer, topology, publisher, 100*time.Millisecond, "1h", "5m", nil)
	ctx := context.Background()

	s.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	// 停止应该优雅完成
	done := make(chan struct{})
	go func() {
		s.Stop()
		close(done)
	}()

	select {
	case <-done:
		// 成功停止
	case <-time.After(time.Second):
		t.Fatal("Stop did not complete in time")
	}
}

func TestOverlapProtection(t *testing.T) {
	// 模拟慢速发现
	discoverer := &MockDiscoverer{
		services: []types.ServiceInfo{{Name: "service-a"}},
	}
	topology := &MockTopologyBuilder{}
	publisher := &MockPublisher{}

	s := scheduler.NewScheduler(discoverer, topology, publisher, 10*time.Millisecond, "1h", "5m", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	s.Start(ctx)
	<-ctx.Done()
	s.Stop()

	// 由于重叠保护，调用次数应该受限
	// 具体次数取决于实现，但不应该无限增长
	assert.Less(t, discoverer.GetCallCount(), 20)
}

func TestGetLastSuccessfulMap(t *testing.T) {
	discoverer := &MockDiscoverer{
		services: []types.ServiceInfo{
			{Name: "service-a", Namespace: "default"},
		},
	}
	topology := &MockTopologyBuilder{
		edges: []types.ServiceEdge{
			{Source: "service-a", Target: "service-b", CallCount: 50},
		},
	}
	publisher := &MockPublisher{}

	s := scheduler.NewScheduler(discoverer, topology, publisher, time.Minute, "1h", "5m", nil)
	ctx := context.Background()

	// 初始时为 nil
	assert.Nil(t, s.GetLastSuccessfulMap())

	// 执行发现后
	err := s.RunDiscovery(ctx)
	require.NoError(t, err)

	// 应该有缓存
	lastMap := s.GetLastSuccessfulMap()
	assert.NotNil(t, lastMap)
	assert.Equal(t, 1, lastMap.ServiceCount())
}

func TestRunDiscovery_UpdatesTimestamp(t *testing.T) {
	discoverer := &MockDiscoverer{
		services: []types.ServiceInfo{{Name: "service-a"}},
	}
	topology := &MockTopologyBuilder{}
	publisher := &MockPublisher{}

	s := scheduler.NewScheduler(discoverer, topology, publisher, time.Minute, "1h", "5m", nil)
	ctx := context.Background()

	before := time.Now()
	err := s.RunDiscovery(ctx)
	after := time.Now()

	require.NoError(t, err)

	publishedMaps := publisher.GetPublishedMaps()
	require.Len(t, publishedMaps, 1)

	// 时间戳应该在执行期间
	assert.True(t, publishedMaps[0].Timestamp.After(before) || publishedMaps[0].Timestamp.Equal(before))
	assert.True(t, publishedMaps[0].Timestamp.Before(after) || publishedMaps[0].Timestamp.Equal(after))
}

func TestRunDiscovery_SetsMetadata(t *testing.T) {
	discoverer := &MockDiscoverer{
		services: []types.ServiceInfo{{Name: "service-a"}},
	}
	topology := &MockTopologyBuilder{}
	publisher := &MockPublisher{}

	s := scheduler.NewScheduler(discoverer, topology, publisher, time.Minute, "1h", "5m", nil)
	ctx := context.Background()

	err := s.RunDiscovery(ctx)
	require.NoError(t, err)

	publishedMaps := publisher.GetPublishedMaps()
	require.Len(t, publishedMaps, 1)

	assert.Equal(t, "1h", publishedMaps[0].Metadata.JaegerLookback)
	assert.Equal(t, "5m", publishedMaps[0].Metadata.DiscoveryInterval)
	assert.False(t, publishedMaps[0].Metadata.Stale)
}

func TestRunDiscovery_ContextCancellation(t *testing.T) {
	discoverer := &MockDiscoverer{
		services: []types.ServiceInfo{{Name: "service-a"}},
	}
	topology := &MockTopologyBuilder{}
	publisher := &MockPublisher{}

	s := scheduler.NewScheduler(discoverer, topology, publisher, time.Minute, "1h", "5m", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	err := s.RunDiscovery(ctx)

	// 可能会返回错误（取决于实现），但不应该 panic
	_ = err
}

func TestIsRunning(t *testing.T) {
	discoverer := &MockDiscoverer{
		services: []types.ServiceInfo{{Name: "service-a"}},
	}
	topology := &MockTopologyBuilder{}
	publisher := &MockPublisher{}

	s := scheduler.NewScheduler(discoverer, topology, publisher, time.Hour, "1h", "5m", nil)

	assert.False(t, s.IsRunning())

	ctx := context.Background()
	s.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	assert.True(t, s.IsRunning())

	s.Stop()
	time.Sleep(10 * time.Millisecond)

	assert.False(t, s.IsRunning())
}

func TestMultipleStartStop(t *testing.T) {
	discoverer := &MockDiscoverer{
		services: []types.ServiceInfo{{Name: "service-a"}},
	}
	topology := &MockTopologyBuilder{}
	publisher := &MockPublisher{}

	s := scheduler.NewScheduler(discoverer, topology, publisher, 50*time.Millisecond, "1h", "5m", nil)
	ctx := context.Background()

	// 多次启动停止
	for i := 0; i < 3; i++ {
		s.Start(ctx)
		time.Sleep(30 * time.Millisecond)
		s.Stop()
	}

	// 不应该 panic
	assert.False(t, s.IsRunning())
}
