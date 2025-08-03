package log

import (
	"os"
	"runtime/debug"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LoggerZap struct {
	*zap.Logger
	env string
}

// NewLogger creates a Zap logger writing JSON to stdout, suitable for Promtail/Loki
func NewLogger(cfg Config) Logger {
	if cfg.Env == "" {
		cfg.Env = "dev"
	}

	encoder := newJSONEncoder(cfg.Env)
	level := parseLevel(cfg.Level)

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)

	return &LoggerZap{
		Logger: zap.New(core, zap.AddCaller()),
		env:    cfg.Env,
	}
}

func (l *LoggerZap) Info(message, requestID string, fields ...zap.Field) {
	l.Logger.Info(message, append([]zap.Field{
		zap.String("request_id", requestID),
		zap.String("env", l.env),
	}, fields...)...)
}

func (l *LoggerZap) Warn(message, requestID string, fields ...zap.Field) {
	l.Logger.Warn(message, append([]zap.Field{
		zap.String("request_id", requestID),
		zap.String("env", l.env),
	}, fields...)...)
}

func (l *LoggerZap) Error(message, requestID string, fields ...zap.Field) {
	if l.env == "dev" {
		sugar := l.Logger.Sugar()
		sugar.Errorf("Error: %s, RequestID: %s, Fields: %+v\nStack Trace:\n%s",
			message, requestID, fields, string(debug.Stack()))
		return
	}

	l.Logger.Error(message, append([]zap.Field{
		zap.String("request_id", requestID),
		zap.String("env", l.env),
		zap.Stack("stack_trace"),
	}, fields...)...)
}

func (l *LoggerZap) Debug(message, requestID string, fields ...zap.Field) {
	l.Logger.Debug(message, append([]zap.Field{
		zap.String("request_id", requestID),
		zap.String("env", l.env),
	}, fields...)...)
}

func (l *LoggerZap) Sync(wg *sync.WaitGroup) error {
	wg.Add(1)
	defer wg.Done()
	return l.Logger.Sync()
}

func newJSONEncoder(env string) zapcore.Encoder {
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "time"
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.LevelKey = "level"
	cfg.EncodeLevel = zapcore.LowercaseLevelEncoder
	cfg.CallerKey = "caller"
	cfg.EncodeCaller = zapcore.ShortCallerEncoder
	cfg.MessageKey = "message"
	cfg.StacktraceKey = "stack_trace"

	if env == "dev" {
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncodeTime = zapcore.ISO8601TimeEncoder
		return zapcore.NewConsoleEncoder(cfg)
	}

	return zapcore.NewJSONEncoder(cfg)
}

func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
