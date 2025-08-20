package global

import (
	"context"
	"os"
	"sync"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	Sub *zap.Logger
}

func (log *Logger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	log.Sub.Info(msg, decaps(ctx, fields...)...)
}

func (log *Logger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	log.Sub.Error(msg, decaps(ctx, fields...)...)
}

func (log *Logger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	log.Sub.Debug(msg, decaps(ctx, fields...)...)
}

func (log *Logger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	log.Sub.Warn(msg, decaps(ctx, fields...)...)
}

func decaps(_ context.Context, fields ...zap.Field) []zap.Field {
	// TODO add specific keys from context
	return fields
}

var (
	logger  *Logger
	logOnce sync.Once
)

func Log() *Logger {
	logOnce.Do(func() {
		sub, _ := zap.NewProduction()
		if Conf.Otlp.Tracing && loggerProvider != nil {
			lvl, _ := zapcore.ParseLevel(Conf.LogLevel)
			core := zapcore.NewTee(
				zapcore.NewCore(
					zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
					zapcore.AddSync(os.Stdout),
					lvl,
				),
				otelzap.NewCore(
					Conf.Otlp.ServiceName,
					otelzap.WithLoggerProvider(loggerProvider),
				),
			)
			sub = zap.New(core)
		}

		logger = &Logger{
			Sub: sub,
		}
	})
	return logger
}
