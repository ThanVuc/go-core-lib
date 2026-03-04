package log

import (
	"sync"

	"go.uber.org/zap"
)

type Logger interface {
	Info(message, requestID string, fields ...zap.Field)
	Error(message, requestID string, fields ...zap.Field)
	Debug(message, requestID string, fields ...zap.Field)
	Warn(message, requestID string, fields ...zap.Field)
	Sync(wg *sync.WaitGroup) error
}

type LoggerV2 interface {
	Info(message string, opts ...LogOption)
	Error(message string, opts ...LogOption)
	Debug(message string, opts ...LogOption)
	Warn(message string, opts ...LogOption)
	Sync(wg *sync.WaitGroup) error
}
