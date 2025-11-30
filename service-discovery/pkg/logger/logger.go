package logger

// Package logger 提供结构化日志功能

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Logger 日志器接口
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
	WithContext(ctx context.Context) Logger
}

// slogLogger 基于 slog 的日志实现
type slogLogger struct {
	logger *slog.Logger
}

// Options 日志配置选项
type Options struct {
	// Level 日志级别: debug, info, warn, error
	Level string

	// Format 日志格式: json, text
	Format string

	// Output 输出目标，默认为 os.Stdout
	Output io.Writer

	// AddSource 是否添加源代码位置
	AddSource bool
}

// DefaultOptions 返回默认选项
func DefaultOptions() *Options {
	return &Options{
		Level:     "info",
		Format:    "json",
		Output:    os.Stdout,
		AddSource: false,
	}
}

// New 创建新的日志器
func New(opts *Options) Logger {
	if opts == nil {
		opts = DefaultOptions()
	}

	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	// 解析日志级别
	level := parseLevel(opts.Level)

	// 创建 handler options
	handlerOpts := &slog.HandlerOptions{
		Level:     level,
		AddSource: opts.AddSource,
	}

	// 根据格式创建 handler
	var handler slog.Handler
	switch strings.ToLower(opts.Format) {
	case "text":
		handler = slog.NewTextHandler(opts.Output, handlerOpts)
	default:
		handler = slog.NewJSONHandler(opts.Output, handlerOpts)
	}

	return &slogLogger{
		logger: slog.New(handler),
	}
}

// NewDefault 创建默认配置的日志器
func NewDefault() Logger {
	return New(DefaultOptions())
}

// NewFromConfig 从配置创建日志器
func NewFromConfig(level, format string) Logger {
	return New(&Options{
		Level:  level,
		Format: format,
	})
}

// parseLevel 解析日志级别字符串
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Debug 记录 debug 级别日志
func (l *slogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Info 记录 info 级别日志
func (l *slogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn 记录 warn 级别日志
func (l *slogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error 记录 error 级别日志
func (l *slogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// With 返回带有额外属性的日志器
func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{
		logger: l.logger.With(args...),
	}
}

// WithContext 返回带有 context 的日志器
func (l *slogLogger) WithContext(ctx context.Context) Logger {
	// 可以从 context 中提取 trace ID 等信息
	return l
}

// 全局日志器
var defaultLogger Logger = NewDefault()

// SetDefault 设置默认日志器
func SetDefault(l Logger) {
	defaultLogger = l
}

// Default 获取默认日志器
func Default() Logger {
	return defaultLogger
}

// 便捷的全局日志函数
func Debug(msg string, args ...any) { defaultLogger.Debug(msg, args...) }
func Info(msg string, args ...any)  { defaultLogger.Info(msg, args...) }
func Warn(msg string, args ...any)  { defaultLogger.Warn(msg, args...) }
func Error(msg string, args ...any) { defaultLogger.Error(msg, args...) }

// WithFields 创建带有字段的日志器
func WithFields(fields ...any) Logger {
	return defaultLogger.With(fields...)
}

// LogDiscoveryStart 记录发现流程开始
func LogDiscoveryStart(l Logger) {
	l.Info("discovery started")
}

// LogDiscoveryComplete 记录发现流程完成
func LogDiscoveryComplete(l Logger, servicesCount, edgesCount int, durationMs int64) {
	l.Info("discovery completed",
		"services_count", servicesCount,
		"edges_count", edgesCount,
		"duration_ms", durationMs,
	)
}

// LogDiscoveryError 记录发现流程错误
func LogDiscoveryError(l Logger, component string, err error) {
	l.Error("discovery error",
		"component", component,
		"error", err.Error(),
	)
}

// LogPublishSuccess 记录发布成功
func LogPublishSuccess(l Logger, key string) {
	l.Info("service map published",
		"redis_key", key,
	)
}

// LogPublishError 记录发布错误
func LogPublishError(l Logger, err error, attempt int) {
	l.Error("publish failed",
		"error", err.Error(),
		"attempt", attempt,
	)
}
