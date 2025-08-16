package main

import (
	"encoding/json"
	"hfi/control-plane/storage"
	"log"
	"sync"
	"sync/atomic"
)

// ConfigDistributor 监视策略更改，并将编译后的配置分发给客户端。
type ConfigDistributor struct {
	store         storage.IPolicyStore
	currentConfig atomic.Value // 存储最新的配置字符串

	mu      sync.RWMutex
	clients map[chan string]struct{}
}

// NewConfigDistributor 创建一个新的分发器并启动其监视循环。
func NewConfigDistributor(store storage.IPolicyStore) *ConfigDistributor {
	distributor := &ConfigDistributor{
		store:   store,
		clients: make(map[chan string]struct{}),
	}

	// 使用当前状态进行初始化
	distributor.updateAndBroadcast()

	// 在后台开始监视变更
	go distributor.watchForChanges()

	return distributor
}

// watchForChanges 监听来自存储的事件并触发配置更新。
func (d *ConfigDistributor) watchForChanges() {
	watchChan := d.store.Watch()
	for event := range watchChan {
		log.Printf("Change detected (type: %s, policy: %s), updating and broadcasting config...", event.Type, event.Policy.Metadata.Name)
		d.updateAndBroadcast()
	}
}

// updateAndBroadcast 获取所有策略，编译它们，并广播给客户端。
func (d *ConfigDistributor) updateAndBroadcast() {
	policies := d.store.List()

	// 目前，“编译”只是将列表编组为 JSON。
	configBytes, err := json.Marshal(policies)
	if err != nil {
		log.Printf("Error marshaling policies: %v", err)
		return
	}

	newConfig := string(configBytes)
	d.currentConfig.Store(newConfig)

	log.Printf("Broadcasting new config: %s", newConfig)
	d.broadcast(newConfig)
}

// broadcast 将最新配置发送给所有已注册的客户端。
func (d *ConfigDistributor) broadcast(config string) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for clientChan := range d.clients {
		select {
		case clientChan <- config:
		default:
			// 如果客户端的 channel 已满，则不阻塞。
			// sseHandler 负责关闭和注销。
		}
	}
}

// RegisterClient 将新客户端添加到广播列表。
func (d *ConfigDistributor) RegisterClient(clientChan chan string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.clients[clientChan] = struct{}{}
	log.Printf("Client registered. Total clients: %d", len(d.clients))
}

// UnregisterClient 从广播列表中移除客户端。
func (d *ConfigDistributor) UnregisterClient(clientChan chan string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.clients[clientChan]; ok {
		close(clientChan)
		delete(d.clients, clientChan)
		log.Printf("Client unregistered. Total clients: %d", len(d.clients))
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
