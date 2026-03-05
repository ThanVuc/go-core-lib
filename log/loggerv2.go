package log

import (
	"runtime/debug"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogOption func(*logOptions)

type logOptions struct {
	requestID string
	fields    []zap.Field
}

func WithRequestID(requestID string) LogOption {
	return func(o *logOptions) {
		o.requestID = requestID
	}
}

func WithFields(fields ...zap.Field) LogOption {
	return func(o *logOptions) {
		o.fields = append(o.fields, fields...)
	}
}

type LoggerZapV2 struct {
	*zap.Logger
	env string
}

func NewLoggerZapV2(env string) (*LoggerZapV2, error) {
	var cfg zap.Config

	if env == "production" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
	}

	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return &LoggerZapV2{
		Logger: logger,
		env:    env,
	}, nil
}

func (l *LoggerZapV2) buildFields(opts ...LogOption) []zap.Field {
	options := &logOptions{}

	for _, opt := range opts {
		opt(options)
	}

	fields := []zap.Field{
		zap.String("env", l.env),
	}

	if options.requestID != "" {
		fields = append(fields, zap.String("request_id", options.requestID))
	}

	fields = append(fields, options.fields...)

	return fields
}

func (l *LoggerZapV2) Info(message string, opts ...LogOption) {
	l.Logger.Info(message, l.buildFields(opts...)...)
}

func (l *LoggerZapV2) Debug(message string, opts ...LogOption) {
	l.Logger.Debug(message, l.buildFields(opts...)...)
}

func (l *LoggerZapV2) Warn(message string, opts ...LogOption) {
	l.Logger.Warn(message, l.buildFields(opts...)...)
}

func (l *LoggerZapV2) Error(message string, opts ...LogOption) {
	fields := append(
		l.buildFields(opts...),
		zap.String("stack_trace", string(debug.Stack())),
	)

	l.Logger.Error(message, fields...)
}
