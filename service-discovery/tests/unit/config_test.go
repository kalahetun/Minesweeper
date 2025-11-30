package unit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/boifi/service-discovery/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	// 验证默认值
	assert.Equal(t, "", cfg.Kubernetes.Kubeconfig)
	assert.Equal(t, "http://jaeger-query:16686", cfg.Jaeger.URL)
	assert.Equal(t, time.Hour, cfg.Jaeger.Lookback)
	assert.Equal(t, 30*time.Second, cfg.Jaeger.Timeout)
	assert.Equal(t, "redis:6379", cfg.Redis.Address)
	assert.Equal(t, "", cfg.Redis.Password)
	assert.Equal(t, 0, cfg.Redis.DB)
	assert.Equal(t, "boifi:service-map", cfg.Redis.Key)
	assert.Equal(t, "boifi:service-map:updates", cfg.Redis.Channel)
	assert.Equal(t, 5*time.Minute, cfg.Discovery.Interval)
	assert.False(t, cfg.OpenAPI.Enabled)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
}

func TestLoadFromFile(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
kubernetes:
  kubeconfig: /path/to/kubeconfig

jaeger:
  url: http://jaeger:16686
  lookback: 2h
  timeout: 60s

redis:
  address: localhost:6379
  password: testpass
  db: 1
  key: test:service-map
  channel: test:updates

discovery:
  interval: 10m

openapi:
  enabled: true
  paths:
    - /api-docs
  timeout: 10s

log:
  level: debug
  format: text
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// 加载配置
	cfg, err := config.Load(configPath)
	require.NoError(t, err)

	// 验证配置值
	assert.Equal(t, "/path/to/kubeconfig", cfg.Kubernetes.Kubeconfig)
	assert.Equal(t, "http://jaeger:16686", cfg.Jaeger.URL)
	assert.Equal(t, 2*time.Hour, cfg.Jaeger.Lookback)
	assert.Equal(t, 60*time.Second, cfg.Jaeger.Timeout)
	assert.Equal(t, "localhost:6379", cfg.Redis.Address)
	assert.Equal(t, "testpass", cfg.Redis.Password)
	assert.Equal(t, 1, cfg.Redis.DB)
	assert.Equal(t, "test:service-map", cfg.Redis.Key)
	assert.Equal(t, "test:updates", cfg.Redis.Channel)
	assert.Equal(t, 10*time.Minute, cfg.Discovery.Interval)
	assert.True(t, cfg.OpenAPI.Enabled)
	assert.Equal(t, []string{"/api-docs"}, cfg.OpenAPI.Paths)
	assert.Equal(t, 10*time.Second, cfg.OpenAPI.Timeout)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "text", cfg.Log.Format)
}

func TestLoadFromEnv(t *testing.T) {
	// 设置环境变量
	envVars := map[string]string{
		"BOIFI_JAEGER_URL": "http://env-jaeger:16686",
		"BOIFI_REDIS_ADDR": "env-redis:6379",
		"BOIFI_LOG_LEVEL":  "warn",
		"BOIFI_LOG_FORMAT": "text",
	}

	// 设置环境变量
	for k, v := range envVars {
		os.Setenv(k, v)
	}

	// 清理环境变量
	t.Cleanup(func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	})

	// 加载配置
	cfg, err := config.LoadFromEnv()
	require.NoError(t, err)

	// 验证环境变量覆盖
	assert.Equal(t, "http://env-jaeger:16686", cfg.Jaeger.URL)
	assert.Equal(t, "env-redis:6379", cfg.Redis.Address)
	assert.Equal(t, "warn", cfg.Log.Level)
	assert.Equal(t, "text", cfg.Log.Format)
}

func TestLoadWithMissingFile(t *testing.T) {
	// 指定不存在的配置文件
	cfg, err := config.Load("/nonexistent/config.yaml")
	require.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoadWithEmptyPath(t *testing.T) {
	// 空路径应该使用默认值
	cfg, err := config.Load("")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// 应该使用默认值
	defaults := config.DefaultConfig()
	assert.Equal(t, defaults.Jaeger.URL, cfg.Jaeger.URL)
}

func TestLoadWithInvalidYAML(t *testing.T) {
	// 创建无效的 YAML 文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidContent := `
kubernetes:
  kubeconfig: [invalid: yaml: content
`
	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	// 加载应该失败
	cfg, err := config.Load(configPath)
	require.Error(t, err)
	assert.Nil(t, cfg)
}

func TestConfigValidate(t *testing.T) {
	cfg := config.DefaultConfig()
	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestMustLoadPanic(t *testing.T) {
	// 使用不存在的文件应该 panic
	assert.Panics(t, func() {
		config.MustLoad("/nonexistent/config.yaml")
	})
}

func TestMustLoadSuccess(t *testing.T) {
	// 创建有效配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte("log:\n  level: debug"), 0644)
	require.NoError(t, err)

	// 应该不会 panic
	assert.NotPanics(t, func() {
		cfg := config.MustLoad(configPath)
		assert.Equal(t, "debug", cfg.Log.Level)
	})
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "env set",
			envKey:       "TEST_VAR_SET",
			envValue:     "from_env",
			defaultValue: "default_value",
			expected:     "from_env",
		},
		{
			name:         "env not set",
			envKey:       "TEST_VAR_NOT_SET",
			envValue:     "",
			defaultValue: "default_value",
			expected:     "default_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				t.Cleanup(func() {
					os.Unsetenv(tt.envKey)
				})
			}

			result := config.GetEnvOrDefault(tt.envKey, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}
