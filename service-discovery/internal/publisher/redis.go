// Package publisher 实现服务地图发布功能
package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/boifi/service-discovery/internal/types"
	"github.com/boifi/service-discovery/pkg/logger"
)

// RedisPublisher Redis 发布器
type RedisPublisher struct {
	client  *redis.Client
	key     string
	channel string
	log     logger.Logger
}

// NewRedisPublisher 创建新的 Redis 发布器
func NewRedisPublisher(addr, password string, db int, key, channel string, log logger.Logger) *RedisPublisher {
	if log == nil {
		log = logger.NewDefault()
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisPublisher{
		client:  client,
		key:     key,
		channel: channel,
		log:     log,
	}
}

// Ping 测试 Redis 连接
func (p *RedisPublisher) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}

// Close 关闭 Redis 连接
func (p *RedisPublisher) Close() error {
	return p.client.Close()
}

// SerializeServiceMap 将 ServiceMap 序列化为 JSON
func SerializeServiceMap(sm *types.ServiceMap) ([]byte, error) {
	return json.Marshal(sm)
}

// PublishServiceMap 将 ServiceMap 发布到 Redis
// 使用 SET 命令存储完整数据
func (p *RedisPublisher) PublishServiceMap(ctx context.Context, sm *types.ServiceMap) error {
	p.log.Debug("serializing service map",
		"services", sm.ServiceCount(),
		"edges", sm.EdgeCount(),
	)

	data, err := SerializeServiceMap(sm)
	if err != nil {
		p.log.Error("failed to serialize service map", "error", err.Error())
		return fmt.Errorf("failed to serialize service map: %w", err)
	}

	p.log.Debug("publishing service map to redis",
		"key", p.key,
		"size_bytes", len(data),
	)

	err = p.client.Set(ctx, p.key, data, 0).Err()
	if err != nil {
		p.log.Error("failed to publish service map to redis",
			"error", err.Error(),
			"key", p.key,
		)
		return fmt.Errorf("failed to publish to redis: %w", err)
	}

	p.log.Info("service map published to redis",
		"key", p.key,
		"services", sm.ServiceCount(),
		"edges", sm.EdgeCount(),
	)

	return nil
}

// NotifyUpdate 发送更新通知到 Redis channel
// 使用 PUBLISH 命令发送轻量通知
func (p *RedisPublisher) NotifyUpdate(ctx context.Context) error {
	p.log.Debug("sending update notification", "channel", p.channel)

	err := p.client.Publish(ctx, p.channel, "updated").Err()
	if err != nil {
		p.log.Error("failed to send update notification",
			"error", err.Error(),
			"channel", p.channel,
		)
		return fmt.Errorf("failed to publish notification: %w", err)
	}

	p.log.Debug("update notification sent", "channel", p.channel)
	return nil
}

// PublishAndNotify 发布 ServiceMap 并发送更新通知
func (p *RedisPublisher) PublishAndNotify(ctx context.Context, sm *types.ServiceMap) error {
	if err := p.PublishServiceMap(ctx, sm); err != nil {
		return err
	}
	return p.NotifyUpdate(ctx)
}

// PublishWithRetry 带重试的发布
func (p *RedisPublisher) PublishWithRetry(ctx context.Context, sm *types.ServiceMap, maxRetries int) error {
	return RetryWithBackoff(func() error {
		return p.PublishServiceMap(ctx, sm)
	}, maxRetries, time.Second)
}

// PublishAndNotifyWithRetry 带重试的发布和通知
func (p *RedisPublisher) PublishAndNotifyWithRetry(ctx context.Context, sm *types.ServiceMap, maxRetries int) error {
	return RetryWithBackoff(func() error {
		return p.PublishAndNotify(ctx, sm)
	}, maxRetries, time.Second)
}

// RetryWithBackoff 使用指数退避重试执行函数
// maxRetries: 最大重试次数
// initialDelay: 初始延迟时间
func RetryWithBackoff(fn func() error, maxRetries int, initialDelay time.Duration) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if attempt < maxRetries {
			// 计算退避时间: initialDelay * 2^(attempt-1)，最大 30 秒
			delay := time.Duration(float64(initialDelay) * math.Pow(2, float64(attempt-1)))
			maxDelay := 30 * time.Second
			if delay > maxDelay {
				delay = maxDelay
			}

			logger.Warn("retry attempt failed",
				"attempt", attempt,
				"max_retries", maxRetries,
				"error", err.Error(),
				"next_delay", delay.String(),
			)

			time.Sleep(delay)
		}
	}

	return fmt.Errorf("all %d retry attempts failed: %w", maxRetries, lastErr)
}

// GetKey 返回存储 key
func (p *RedisPublisher) GetKey() string {
	return p.key
}

// GetChannel 返回通知 channel
func (p *RedisPublisher) GetChannel() string {
	return p.channel
}

// GetStoredServiceMap 从 Redis 获取存储的 ServiceMap
func (p *RedisPublisher) GetStoredServiceMap(ctx context.Context) (*types.ServiceMap, error) {
	data, err := p.client.Get(ctx, p.key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // key 不存在
		}
		return nil, fmt.Errorf("failed to get service map from redis: %w", err)
	}

	sm, err := types.FromJSON(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize service map: %w", err)
	}

	return sm, nil
}

// IsConnected 检查 Redis 连接是否可用
func (p *RedisPublisher) IsConnected(ctx context.Context) bool {
	return p.Ping(ctx) == nil
}
