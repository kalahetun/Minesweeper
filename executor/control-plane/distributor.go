package main

import (
	"context"
	"encoding/json"
	"hfi/control-plane/logger"
	"hfi/control-plane/storage"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
)

// ConfigDistributor 监视策略更改，并将编译后的配置分发给客户端。
type ConfigDistributor struct {
	store         storage.IPolicyStore
	currentConfig atomic.Value // 存储最新的配置字符串

	mu       sync.RWMutex
	clients  map[chan string]struct{}
	ctx      context.Context
	cancel   context.CancelFunc
	doneChan chan struct{}
}

// NewConfigDistributor 创建一个新的分发器并启动其监视循环。
func NewConfigDistributor(store storage.IPolicyStore) *ConfigDistributor {
	log := logger.WithComponent("distributor")

	ctx, cancel := context.WithCancel(context.Background())

	distributor := &ConfigDistributor{
		store:    store,
		clients:  make(map[chan string]struct{}),
		ctx:      ctx,
		cancel:   cancel,
		doneChan: make(chan struct{}),
	}

	// 仅初始化当前配置，不广播（此时还没有客户端连接）
	distributor.updateCurrentConfig()

	// 在后台开始监视变更
	go distributor.watchForChanges()

	log.Info("Config distributor initialized")
	return distributor
}

// Stop gracefully shuts down the distributor and closes all client connections.
func (d *ConfigDistributor) Stop() {
	log := logger.WithComponent("distributor")
	
	// Signal the watch goroutine to stop
	d.cancel()
	
	// Wait for watch to finish
	<-d.doneChan
	
	// Close all client channels
	d.mu.Lock()
	defer d.mu.Unlock()
	for clientChan := range d.clients {
		close(clientChan)
	}
	d.clients = make(map[chan string]struct{})
	
	log.Info("Config distributor stopped")
}

// updateCurrentConfig 仅更新内存中的配置而不广播
func (d *ConfigDistributor) updateCurrentConfig() {
	log := logger.WithComponent("distributor")

	policies := d.store.List()

	// 将策略列表编组为 JSON
	configBytes, err := json.Marshal(policies)
	if err != nil {
		log.Error("Error marshaling policies", zap.Error(err))
		return
	}

	newConfig := string(configBytes)
	d.currentConfig.Store(newConfig)

	log.Info("Config updated",
		zap.Int("policy_count", len(policies)))
}

// watchForChanges 监听来自存储的事件并触发配置更新。
// 包含错误处理和恢复逻辑。
// 现在使用 WatchWithContext 以支持优雅关闭。
func (d *ConfigDistributor) watchForChanges() {
	log := logger.WithComponent("distributor")

	defer func() {
		if r := recover(); r != nil {
			log.Error("Panic in watchForChanges, restarting", zap.Any("panic", r))
			// 重启监视 goroutine
			go d.watchForChanges()
			return
		}
		// 监视正常停止，发送完成信号
		close(d.doneChan)
	}()

	watchChan := d.store.WatchWithContext(d.ctx)

	for {
		select {
		case <-d.ctx.Done():
			log.Info("Watch context canceled, stopping watch")
			return
		case event, ok := <-watchChan:
			if !ok {
				// Watch channel closed (normal shutdown)
				log.Info("Watch channel closed")
				return
			}
			log.Info("Change detected, updating and broadcasting config",
				zap.String("event_type", string(event.Type)),
				zap.String("policy_name", event.Policy.Metadata.Name))
			d.updateAndBroadcast()
		}
	}
}

// updateAndBroadcast 获取所有策略，编译它们，并广播给客户端。
func (d *ConfigDistributor) updateAndBroadcast() {
	log := logger.WithComponent("distributor")

	policies := d.store.List()

	// 目前，"编译"只是将列表编组为 JSON。
	configBytes, err := json.Marshal(policies)
	if err != nil {
		log.Error("Error marshaling policies", zap.Error(err))
		return
	}

	newConfig := string(configBytes)
	d.currentConfig.Store(newConfig)

	log.Info("Broadcasting new config",
		zap.Int("policy_count", len(policies)),
		zap.String("config", newConfig))
	d.broadcast(newConfig)
}

// broadcast 将最新配置发送给所有已注册的客户端。
// 使用非阻塞发送，避免阻塞整个分发器。
func (d *ConfigDistributor) broadcast(config string) {
	log := logger.WithComponent("distributor")
	d.mu.RLock()

	// 创建客户端列表的副本，避免在持有锁的情况下发送数据
	clients := make([]chan string, 0, len(d.clients))
	for clientChan := range d.clients {
		clients = append(clients, clientChan)
	}
	d.mu.RUnlock()

	var failedClients []chan string

	for _, clientChan := range clients {
		select {
		case clientChan <- config:
			// 成功发送
		default:
			// 客户端 channel 已满或已关闭，标记为失败
			failedClients = append(failedClients, clientChan)
		}
	}

	log.Debug("Config broadcasted to clients",
		zap.Int("client_count", len(clients)),
		zap.Int("failed_count", len(failedClients)))

	// 清理失败的客户端（在主循环外进行）
	if len(failedClients) > 0 {
		d.removeFailedClients(failedClients)
	}
}

// removeFailedClients 从客户端列表中移除无法接收数据的客户端。
func (d *ConfigDistributor) removeFailedClients(failedClients []chan string) {
	log := logger.WithComponent("distributor")
	d.mu.Lock()
	defer d.mu.Unlock()

	removed := 0
	for _, failedChan := range failedClients {
		if _, exists := d.clients[failedChan]; exists {
			delete(d.clients, failedChan)
			removed++
		}
	}

	log.Warn("Removed failed clients",
		zap.Int("removed_count", removed),
		zap.Int("remaining_clients", len(d.clients)))
}

// RegisterClient 将新客户端添加到广播列表。
func (d *ConfigDistributor) RegisterClient(clientChan chan string) {
	log := logger.WithComponent("distributor")
	d.mu.Lock()
	defer d.mu.Unlock()
	d.clients[clientChan] = struct{}{}
	log.Info("Client registered", zap.Int("total_clients", len(d.clients)))
}

// UnregisterClient 从广播列表中移除客户端。
func (d *ConfigDistributor) UnregisterClient(clientChan chan string) {
	log := logger.WithComponent("distributor")
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.clients[clientChan]; ok {
		close(clientChan)
		delete(d.clients, clientChan)
		log.Info("Client unregistered", zap.Int("total_clients", len(d.clients)))
	}
}

// GetCurrentConfig 返回最新的已编译配置。
func (d *ConfigDistributor) GetCurrentConfig() string {
	config := d.currentConfig.Load()
	if config == nil {
		return "[]" // 如果尚未初始化，则返回空的 JSON 数组
	}
	return config.(string)
}
