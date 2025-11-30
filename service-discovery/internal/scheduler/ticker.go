// Package scheduler 提供定期服务发现调度功能
package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boifi/service-discovery/internal/types"
)

// Discoverer 服务发现接口
type Discoverer interface {
	Discover(ctx context.Context, namespace string) ([]types.ServiceInfo, error)
}

// TopologyBuilder 拓扑构建接口
type TopologyBuilder interface {
	BuildTopologyGraceful(ctx context.Context) ([]types.ServiceEdge, error)
}

// Publisher 发布接口
type Publisher interface {
	PublishAndNotify(ctx context.Context, sm *types.ServiceMap) error
}

// Scheduler 定期服务发现调度器
type Scheduler struct {
	discoverer  Discoverer
	topology    TopologyBuilder
	publisher   Publisher
	interval    time.Duration
	lookback    string
	lookbackStr string
	logger      *slog.Logger

	// 运行状态
	running int32
	stopCh  chan struct{}
	wg      sync.WaitGroup

	// 重叠保护
	discovering int32

	// 缓存
	lastMap   *types.ServiceMap
	lastMapMu sync.RWMutex
}

// NewScheduler 创建新的调度器
func NewScheduler(
	discoverer Discoverer,
	topology TopologyBuilder,
	publisher Publisher,
	interval time.Duration,
	lookback string,
	lookbackStr string,
	logger *slog.Logger,
) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Scheduler{
		discoverer:  discoverer,
		topology:    topology,
		publisher:   publisher,
		interval:    interval,
		lookback:    lookback,
		lookbackStr: lookbackStr,
		logger:      logger,
	}
}

// RunDiscovery 执行一次发现流程
func (s *Scheduler) RunDiscovery(ctx context.Context) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.logger.Info("starting discovery run")
	startTime := time.Now()

	// 1. 从 Kubernetes 发现服务
	services, err := s.discoverer.Discover(ctx, "")
	if err != nil {
		s.logger.Error("kubernetes discovery failed", "error", err)
		return err
	}
	s.logger.Info("kubernetes discovery completed", "service_count", len(services))

	// 2. 从 Jaeger 获取拓扑（优雅降级）
	edges, err := s.topology.BuildTopologyGraceful(ctx)
	if err != nil {
		s.logger.Warn("jaeger topology failed, continuing without edges", "error", err)
		edges = []types.ServiceEdge{} // 优雅降级
	} else {
		s.logger.Info("jaeger topology completed", "edge_count", len(edges))
	}

	// 3. 构建 ServiceMap
	sm := types.NewServiceMap()
	sm.Timestamp = time.Now()
	sm.Metadata.JaegerLookback = s.lookback
	sm.Metadata.DiscoveryInterval = s.lookbackStr
	sm.Metadata.Stale = false

	// 添加服务
	for _, svc := range services {
		sm.AddService(svc)
	}

	// 添加边
	for _, edge := range edges {
		sm.AddEdge(edge)
	}

	// 4. 发布到 Redis
	if err := s.publisher.PublishAndNotify(ctx, sm); err != nil {
		s.logger.Error("redis publish failed", "error", err)
		return err
	}

	// 5. 更新缓存
	s.lastMapMu.Lock()
	s.lastMap = sm
	s.lastMapMu.Unlock()

	duration := time.Since(startTime)
	s.logger.Info("discovery run completed",
		"duration", duration,
		"service_count", sm.ServiceCount(),
		"edge_count", sm.EdgeCount(),
	)

	return nil
}

// Start 启动调度器
func (s *Scheduler) Start(ctx context.Context) {
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		s.logger.Warn("scheduler already running")
		return
	}

	s.stopCh = make(chan struct{})
	s.wg.Add(1)

	go s.run(ctx)

	s.logger.Info("scheduler started", "interval", s.interval)
}

// run 运行调度循环
func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()
	defer atomic.StoreInt32(&s.running, 0)

	// 立即执行第一次
	s.executeWithOverlapProtection(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler context cancelled")
			return
		case <-s.stopCh:
			s.logger.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.executeWithOverlapProtection(ctx)
		}
	}
}

// executeWithOverlapProtection 带重叠保护的执行
func (s *Scheduler) executeWithOverlapProtection(ctx context.Context) {
	// 重叠保护：如果上一次还在运行，跳过本次
	if !atomic.CompareAndSwapInt32(&s.discovering, 0, 1) {
		s.logger.Warn("skipping discovery run, previous run still in progress")
		return
	}
	defer atomic.StoreInt32(&s.discovering, 0)

	if err := s.RunDiscovery(ctx); err != nil {
		s.logger.Error("discovery run failed", "error", err)
	}
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	if atomic.LoadInt32(&s.running) == 0 {
		return
	}

	if s.stopCh != nil {
		close(s.stopCh)
	}

	s.wg.Wait()
	s.logger.Info("scheduler stopped gracefully")
}

// IsRunning 返回调度器是否正在运行
func (s *Scheduler) IsRunning() bool {
	return atomic.LoadInt32(&s.running) == 1
}

// GetLastSuccessfulMap 返回上一次成功的 ServiceMap
func (s *Scheduler) GetLastSuccessfulMap() *types.ServiceMap {
	s.lastMapMu.RLock()
	defer s.lastMapMu.RUnlock()
	return s.lastMap
}
