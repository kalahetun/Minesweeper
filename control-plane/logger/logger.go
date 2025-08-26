package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

// Init initializes the global logger
func Init() error {
	config := zap.NewProductionConfig()
	
	// Configure log level from environment
	logLevel := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	switch logLevel {
	case "DEBUG":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "INFO":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "WARN":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "ERROR":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Configure JSON encoding
	config.Encoding = "json"
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.StacktraceKey = "stacktrace"

	// Build logger
	logger, err := config.Build(zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	if err != nil {
		return err
	}

	Logger = logger
	return nil
}

// Sync flushes any buffered log entries
func Sync() {
	if Logger != nil {
		Logger.Sync()
	}
}

// WithFields creates a logger with additional fields
func WithFields(fields ...zap.Field) *zap.Logger {
	if Logger == nil {
		// Fallback to nop logger if not initialized
		return zap.NewNop()
	}
	return Logger.With(fields...)
}

// WithPolicyName creates a logger with policy name field
func WithPolicyName(policyName string) *zap.Logger {
	return WithFields(zap.String("policy_name", policyName))
}

// WithComponent creates a logger with component field
func WithComponent(component string) *zap.Logger {
	return WithFields(zap.String("component", component))
}

// WithRequestID creates a logger with request ID field
func WithRequestID(requestID string) *zap.Logger {
	return WithFields(zap.String("request_id", requestID))
}
