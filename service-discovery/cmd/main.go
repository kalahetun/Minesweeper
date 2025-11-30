package main

// Package main 是 Service Discovery 服务的入口点

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/boifi/service-discovery/internal/config"
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
		errCh <- runService(ctx, cfg, log)
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
// TODO: 在后续 Phase 中实现具体的服务逻辑
func runService(ctx context.Context, cfg *config.Config, log logger.Logger) error {
	log.Info("service is running",
		"discovery_interval", cfg.Discovery.Interval.String(),
		"jaeger_url", cfg.Jaeger.URL,
		"redis_addr", cfg.Redis.Address,
	)

	// 等待 context 取消
	<-ctx.Done()
	return nil
}
