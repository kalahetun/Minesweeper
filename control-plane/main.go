package main

import (
	"flag"
	"fmt"
	"hfi/control-plane/api"
	"hfi/control-plane/logger"
	"hfi/control-plane/middleware"
	"hfi/control-plane/service"
	"hfi/control-plane/storage"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Initialize structured logging
	if err := logger.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	log := logger.WithComponent("main")

	// 解析命令行参数
	var (
		storageType = flag.String("storage", "memory", "Storage backend type: memory or etcd")
		etcdEndpoints = flag.String("etcd-endpoints", "localhost:2379", "Comma-separated list of etcd endpoints")
		listenAddr = flag.String("listen", "0.0.0.0:8080", "Address to listen on")
	)
	flag.Parse()

	// 支持环境变量覆盖命令行参数
	if envStorage := os.Getenv("STORAGE_BACKEND"); envStorage != "" {
		*storageType = envStorage
	}
	if envEtcdEndpoints := os.Getenv("ETCD_ENDPOINTS"); envEtcdEndpoints != "" {
		*etcdEndpoints = envEtcdEndpoints
	}
	if envListen := os.Getenv("LISTEN_ADDR"); envListen != "" {
		*listenAddr = envListen
	}

	log.Info("Starting Control Plane",
		zap.String("storage_backend", *storageType),
		zap.String("listen_addr", *listenAddr))

	if *storageType == "etcd" {
		log.Info("Using etcd storage backend",
			zap.String("etcd_endpoints", *etcdEndpoints))
	}

	// 1. 初始化存储
	store, err := initializeStore(*storageType, *etcdEndpoints)
	if err != nil {
		log.Fatal("Failed to initialize storage", zap.Error(err))
	}

	// 如果是 etcd store，确保在程序退出时清理资源
	if etcdStore, ok := store.(*storage.EtcdStore); ok {
		defer func() {
			if err := etcdStore.Close(); err != nil {
				log.Error("Failed to close etcd store", zap.Error(err))
			}
		}()
	}
	// 2. 初始化配置分发器
	distributor := NewConfigDistributor(store)

	// 3. 初始化服务层
	policyService := service.NewPolicyService(store)
	
	// 4. 初始化控制器
	policyController := api.NewPolicyController(policyService)

	// 5. 初始化 Gin 引擎
	router := gin.New()
	
	// 6. 添加中间件
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.LoggingMiddleware())
	router.Use(api.ErrorHandlerMiddleware())

	// 7. 定义 v1 路由组
	v1 := router.Group("/v1")
	{
		// 健康检查端点
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "healthy"})
		})

		// 策略管理端点
		v1.POST("/policies", policyController.CreateOrUpdate)      // 创建或更新策略 (原有行为)
		v1.POST("/policies/create", policyController.Create)       // 仅创建策略 (新增)
		v1.PUT("/policies/:id", policyController.Update)          // 仅更新策略 (新增)
		v1.GET("/policies/:id", policyController.Get)
		v1.GET("/policies", policyController.List)
		v1.DELETE("/policies/:id", policyController.Delete)

		// 将 SSE 端点处理器与分发器连接
		v1.GET("/config/stream", sseHandler(distributor))
	}

	// 8. 启动服务器
	log.Info("Control Plane server starting",
		zap.String("storage_backend", *storageType),
		zap.String("listen_addr", *listenAddr))

	log.Info("Server listening", zap.String("address", *listenAddr))
	
	if err := router.Run(*listenAddr); err != nil {
		log.Fatal("Failed to start server", zap.Error(err))
	}
}

// initializeStore 根据存储类型初始化相应的存储后端
func initializeStore(storageType, etcdEndpoints string) (storage.IPolicyStore, error) {
	log := logger.WithComponent("storage")
	
	switch strings.ToLower(storageType) {
	case "memory":
		log.Info("Using memory storage backend")
		return storage.NewMemoryStore(), nil
	case "etcd":
		log.Info("Using etcd storage backend", 
			zap.String("endpoints", etcdEndpoints))
		endpoints := strings.Split(etcdEndpoints, ",")
		for i, endpoint := range endpoints {
			endpoints[i] = strings.TrimSpace(endpoint)
		}
		return storage.NewEtcdStore(endpoints)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s (supported: memory, etcd)", storageType)
	}
}

// sseHandler 创建一个处理 SSE 连接的 Gin 处理器。
func sseHandler(distributor *ConfigDistributor) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置 SSE 所需的响应头
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

		// 为此客户端创建一个 channel
		clientChan := make(chan string, 1)
		distributor.RegisterClient(clientChan)
		defer distributor.UnregisterClient(clientChan)

		// 立即发送当前的全量配置
		initialConfig := distributor.GetCurrentConfig()
		fmt.Fprintf(c.Writer, "event: full_config\ndata: %s\n\n", initialConfig)
		c.Writer.Flush()

		ctx := c.Request.Context()
		for {
			select {
			case <-ctx.Done():
				// 客户端断开连接
				return
			case config := <-clientChan:
				// 收到新的配置更新
				fmt.Fprintf(c.Writer, "event: update\ndata: %s\n\n", config)
				c.Writer.Flush()
			}
		}
	}
}
