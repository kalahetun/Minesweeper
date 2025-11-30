package main

// Package main 是 Service Discovery 服务的入口点

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/boifi/service-discovery/internal/config"
	"github.com/boifi/service-discovery/internal/discovery"
	"github.com/boifi/service-discovery/internal/publisher"
	"github.com/boifi/service-discovery/internal/scheduler"
	"github.com/boifi/service-discovery/internal/types"
	"github.com/boifi/service-discovery/pkg/logger"
)

var (
	// Version 版本号，由构建时注入
	Version = "dev"

	// Commit Git commit hash，由构建时注入
	Commit = "unknown"

	// BuildTime 构建时间，由构建时注入
	BuildTime = "unknown"
)

// 命令行参数
var (
	configPath string
	logLevel   string
	logFormat  string
	runOnce    bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "service-discovery",
	Short: "BOIFI Service Discovery - 构建服务拓扑图",
	Long: `BOIFI Service Discovery 是一个服务发现组件，
用于聚合 Kubernetes、Istio 和 Jaeger 的数据，
构建服务拓扑图 (ServiceMap) 并发布到 Redis。`,
	RunE: run,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("service-discovery %s\n", Version)
		fmt.Printf("  commit: %s\n", Commit)
		fmt.Printf("  built:  %s\n", BuildTime)
	},
}

func init() {
	// 全局参数
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "配置文件路径")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "日志级别 (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "", "日志格式 (json, text)")
	rootCmd.Flags().BoolVar(&runOnce, "once", false, "执行一次发现后退出 (不启动周期调度)")

	// 子命令
	rootCmd.AddCommand(versionCmd)
}

func run(cmd *cobra.Command, args []string) error {
	// 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 命令行参数覆盖配置文件
	if logLevel != "" {
		cfg.Log.Level = logLevel
	}
	if logFormat != "" {
		cfg.Log.Format = logFormat
	}

	// 初始化日志
	log := logger.NewFromConfig(cfg.Log.Level, cfg.Log.Format)
	logger.SetDefault(log)

	log.Info("starting service-discovery",
		"version", Version,
		"config", configPath,
	)

	// 创建可取消的 context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务
	errCh := make(chan error, 1)
	go func() {
		errCh <- runService(ctx, cfg, log, runOnce)
	}()

	// 等待退出
	select {
	case sig := <-sigCh:
		log.Info("received signal, shutting down", "signal", sig.String())
		cancel()
		// 等待服务优雅关闭
		<-errCh
	case err := <-errCh:
		if err != nil {
			log.Error("service error", "error", err.Error())
			return err
		}
	}

	log.Info("service stopped")
	return nil
}

// runService 运行主服务逻辑
func runService(ctx context.Context, cfg *config.Config, log logger.Logger, once bool) error {
	log.Info("initializing service components",
		"discovery_interval", cfg.Discovery.Interval.String(),
		"jaeger_url", cfg.Jaeger.URL,
		"redis_addr", cfg.Redis.Address,
		"run_once", once,
	)

	// 1. 创建 Kubernetes 发现器
	k8sDiscoverer, err := discovery.NewKubernetesDiscoverer(
		cfg.Kubernetes.Kubeconfig,
		log.With("component", "kubernetes"),
	)
	if err != nil {
		return fmt.Errorf("create kubernetes discoverer: %w", err)
	}
	log.Info("kubernetes discoverer initialized")

	// 2. 创建 Jaeger 客户端
	jaegerClient := discovery.NewJaegerClient(
		cfg.Jaeger.URL,
		cfg.Jaeger.Timeout,
		cfg.Jaeger.Lookback,
		log.With("component", "jaeger"),
	)
	log.Info("jaeger client initialized", "url", cfg.Jaeger.URL)

	// 3. 创建 Redis 发布器
	redisPublisher := publisher.NewRedisPublisher(
		cfg.Redis.Address,
		cfg.Redis.Password,
		cfg.Redis.DB,
		cfg.Redis.Key,
		cfg.Redis.Channel,
		log.With("component", "redis"),
	)
	defer redisPublisher.Close()

	// Ping Redis 确保连接正常
	if err := redisPublisher.Ping(ctx); err != nil {
		log.Warn("redis ping failed, will retry on first publish", "error", err.Error())
	} else {
		log.Info("redis publisher initialized", "addr", cfg.Redis.Address)
	}

	// 4. 创建 Scheduler
	sched := scheduler.NewScheduler(
		&k8sDiscovererAdapter{k8sDiscoverer},
		jaegerClient,
		redisPublisher,
		cfg.Discovery.Interval,
		formatDuration(cfg.Jaeger.Lookback),
		cfg.Discovery.Interval.String(),
		slog.Default(),
	)

	// 5. 根据模式运行
	if once {
		// 单次执行模式
		log.Info("running single discovery (--once mode)")
		if err := sched.RunDiscovery(ctx); err != nil {
			return fmt.Errorf("discovery failed: %w", err)
		}
		log.Info("single discovery completed successfully")
		return nil
	}

	// 周期执行模式
	sched.Start(ctx)
	log.Info("scheduler started", "interval", cfg.Discovery.Interval.String())

	// 等待 context 取消
	<-ctx.Done()

	// 优雅关闭
	log.Info("stopping scheduler...")
	sched.Stop()

	return nil
}

// k8sDiscovererAdapter 适配 Kubernetes 发现器到 Scheduler 接口
type k8sDiscovererAdapter struct {
	discoverer *discovery.KubernetesDiscoverer
}

func (a *k8sDiscovererAdapter) Discover(ctx context.Context, namespace string) ([]types.ServiceInfo, error) {
	return a.discoverer.AggregateServices(ctx, namespace)
}

// formatDuration 格式化 duration 为可读字符串
func formatDuration(d time.Duration) string {
	if d >= time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	if d >= time.Minute {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return d.String()
}
