package log

import "go.uber.org/zap"

type Logger interface {
	Info(message, requestID string, fields ...zap.Field)
	Error(message, requestID string, fields ...zap.Field)
	Debug(message, requestID string, fields ...zap.Field)
	Warn(message, requestID string, fields ...zap.Field)
	Sync() error
}
