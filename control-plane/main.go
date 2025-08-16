package main

import (
	"fmt"
	"hfi/control-plane/storage"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 初始化存储
	store := storage.NewMemoryStore()

	// 2. 初始化配置分发器
	distributor := NewConfigDistributor(store)

	// 3. 初始化 Gin 引擎
	router := gin.Default()

	// 4. 定义 v1 路由组
	v1 := router.Group("/v1")
	{
		// 健康检查端点
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "healthy"})
		})

		// 实现策略创建/更新的 POST 端点
		v1.POST("/policies", createPolicyHandler(store))

		// 将 SSE 端点处理器与分发器连接
		v1.GET("/config/stream", sseHandler(distributor))
	}

	// 5. 启动服务器
	log.Println("Control Plane server listening on 0.0.0.0:8080")
	if err := router.Run("0.0.0.0:8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// createPolicyHandler 创建一个处理策略创建/更新请求的 Gin 处理器。
func createPolicyHandler(store storage.IPolicyStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var policy storage.FaultInjectionPolicy
		if err := c.ShouldBindJSON(&policy); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}

		if err := store.CreateOrUpdate(&policy); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save policy: " + err.Error()})
			return
		}

		log.Printf("Policy '%s' created/updated successfully.", policy.Metadata.Name)
		c.JSON(http.StatusCreated, policy)
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
