package config

// Package config 实现配置加载

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Load 加载配置
// 配置优先级（从高到低）：
// 1. 命令行参数
// 2. 环境变量 (BOIFI_* 前缀)
// 3. 配置文件
// 4. 默认值
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 配置环境变量
	v.SetEnvPrefix("BOIFI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 如果指定了配置文件路径，加载配置文件
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			// 配置文件不存在时使用默认配置
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
		}
	} else {
		// 尝试从默认位置加载
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("/app")
		v.AddConfigPath("$HOME/.boifi")

		// 忽略找不到配置文件的错误
		_ = v.ReadInConfig()
	}

	// 绑定环境变量
	bindEnvVars(v)

	// 解析配置
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// LoadFromEnv 仅从环境变量加载配置
func LoadFromEnv() (*Config, error) {
	return Load("")
}

// setDefaults 设置默认值
func setDefaults(v *viper.Viper) {
	defaults := DefaultConfig()

	// Kubernetes
	v.SetDefault("kubernetes.kubeconfig", defaults.Kubernetes.Kubeconfig)

	// Jaeger
	v.SetDefault("jaeger.url", defaults.Jaeger.URL)
	v.SetDefault("jaeger.lookback", defaults.Jaeger.Lookback)
	v.SetDefault("jaeger.timeout", defaults.Jaeger.Timeout)

	// Redis
	v.SetDefault("redis.address", defaults.Redis.Address)
	v.SetDefault("redis.password", defaults.Redis.Password)
	v.SetDefault("redis.db", defaults.Redis.DB)
	v.SetDefault("redis.key", defaults.Redis.Key)
	v.SetDefault("redis.channel", defaults.Redis.Channel)

	// Discovery
	v.SetDefault("discovery.interval", defaults.Discovery.Interval)

	// OpenAPI
	v.SetDefault("openapi.enabled", defaults.OpenAPI.Enabled)
	v.SetDefault("openapi.paths", defaults.OpenAPI.Paths)
	v.SetDefault("openapi.timeout", defaults.OpenAPI.Timeout)

	// Log
	v.SetDefault("log.level", defaults.Log.Level)
	v.SetDefault("log.format", defaults.Log.Format)
}

// bindEnvVars 绑定环境变量
func bindEnvVars(v *viper.Viper) {
	// 显式绑定环境变量以确保正确映射
	envBindings := map[string]string{
		"kubernetes.kubeconfig": "KUBECONFIG",
		"jaeger.url":            "BOIFI_JAEGER_URL",
		"jaeger.lookback":       "BOIFI_JAEGER_LOOKBACK",
		"jaeger.timeout":        "BOIFI_JAEGER_TIMEOUT",
		"redis.address":         "BOIFI_REDIS_ADDR",
		"redis.password":        "BOIFI_REDIS_PASSWORD",
		"redis.db":              "BOIFI_REDIS_DB",
		"redis.key":             "BOIFI_REDIS_KEY",
		"redis.channel":         "BOIFI_REDIS_CHANNEL",
		"discovery.interval":    "BOIFI_DISCOVERY_INTERVAL",
		"openapi.enabled":       "BOIFI_OPENAPI_ENABLED",
		"openapi.timeout":       "BOIFI_OPENAPI_TIMEOUT",
		"log.level":             "BOIFI_LOG_LEVEL",
		"log.format":            "BOIFI_LOG_FORMAT",
	}

	for key, env := range envBindings {
		_ = v.BindEnv(key, env)
	}
}

// MustLoad 加载配置，失败时 panic
func MustLoad(configPath string) *Config {
	cfg, err := Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// GetEnvOrDefault 获取环境变量或返回默认值
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
