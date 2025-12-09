package config

// Package config 定义服务配置数据结构

import "time"

// Config 服务配置顶层结构
type Config struct {
	Kubernetes KubernetesConfig `mapstructure:"kubernetes" yaml:"kubernetes"`
	Jaeger     JaegerConfig     `mapstructure:"jaeger" yaml:"jaeger"`
	Redis      RedisConfig      `mapstructure:"redis" yaml:"redis"`
	Discovery  DiscoveryConfig  `mapstructure:"discovery" yaml:"discovery"`
	OpenAPI    OpenAPIConfig    `mapstructure:"openapi" yaml:"openapi"`
	Log        LogConfig        `mapstructure:"log" yaml:"log"`
}

// KubernetesConfig Kubernetes 连接配置
type KubernetesConfig struct {
	// Kubeconfig kubeconfig 文件路径
	// 空字符串表示使用 in-cluster 配置
	Kubeconfig string `mapstructure:"kubeconfig" yaml:"kubeconfig"`
}

// JaegerConfig Jaeger Query API 配置
type JaegerConfig struct {
	// URL Jaeger Query API 地址
	URL string `mapstructure:"url" yaml:"url"`

	// Lookback 查询时间范围
	Lookback time.Duration `mapstructure:"lookback" yaml:"lookback"`

	// Timeout API 请求超时
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout"`
}

// RedisConfig Redis 连接配置
type RedisConfig struct {
	// Address Redis 服务器地址 (host:port)
	Address string `mapstructure:"address" yaml:"address"`

	// Password Redis 密码 (可选)
	Password string `mapstructure:"password" yaml:"password"`

	// DB Redis 数据库编号
	DB int `mapstructure:"db" yaml:"db"`

	// Key 存储 ServiceMap 的 key
	Key string `mapstructure:"key" yaml:"key"`

	// Channel 发布更新通知的 channel
	Channel string `mapstructure:"channel" yaml:"channel"`
}

// DiscoveryConfig 发现配置
type DiscoveryConfig struct {
	// Interval 发现周期
	Interval time.Duration `mapstructure:"interval" yaml:"interval"`
}

// OpenAPIConfig OpenAPI 增强配置
type OpenAPIConfig struct {
	// Enabled 是否启用 OpenAPI 规范获取
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`

	// Paths 尝试访问的 OpenAPI 路径列表
	Paths []string `mapstructure:"paths" yaml:"paths"`

	// Timeout 单个请求超时
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout"`
}

// LogConfig 日志配置
type LogConfig struct {
	// Level 日志级别: debug, info, warn, error
	Level string `mapstructure:"level" yaml:"level"`

	// Format 日志格式: json, text
	Format string `mapstructure:"format" yaml:"format"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Kubernetes: KubernetesConfig{
			Kubeconfig: "", // in-cluster
		},
		Jaeger: JaegerConfig{
			URL:      "http://jaeger-query:16686",
			Lookback: time.Hour,
			Timeout:  30 * time.Second,
		},
		Redis: RedisConfig{
			Address:  "redis:6379",
			Password: "",
			DB:       0,
			Key:      "boifi:service-map",
			Channel:  "boifi:service-map:updates",
		},
		Discovery: DiscoveryConfig{
			Interval: 5 * time.Minute,
		},
		OpenAPI: OpenAPIConfig{
			Enabled: false,
			Paths: []string{
				"/swagger.json",
				"/v3/api-docs",
				"/openapi.json",
			},
			Timeout: 5 * time.Second,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	// TODO: 添加详细的配置验证
	return nil
}
