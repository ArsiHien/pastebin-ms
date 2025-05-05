package shared

import (
	"go.uber.org/zap"
)

type Logger struct {
	zap *zap.Logger
}

func NewLogger() *Logger {
	logger, err := zap.NewProduction()
	if err != nil {
		logger = zap.NewNop()
	}
	return &Logger{zap: logger}
}

func (l *Logger) Infof(msg string, args ...interface{}) {
	fields := make([]zap.Field, 0, len(args)/2)
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			fields = append(fields, zap.Any(key, args[i+1]))
		}
	}
	l.zap.Info(msg, fields...)
}

func (l *Logger) Errorf(msg string, args ...interface{}) {
	fields := make([]zap.Field, 0, len(args)/2)
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			fields = append(fields, zap.Any(key, args[i+1]))
		}
	}
	l.zap.Error(msg, fields...)
}

func (l *Logger) Fatalf(msg string, args ...interface{}) {
	fields := make([]zap.Field, 0, len(args)/2)
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			fields = append(fields, zap.Any(key, args[i+1]))
		}
	}
	l.zap.Fatal(msg, fields...)
}

func (l *Logger) With(args ...interface{}) *Logger {
	fields := make([]zap.Field, 0, len(args)/2)
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			fields = append(fields, zap.Any(key, args[i+1]))
		}
	}
	return &Logger{zap: l.zap.With(fields...)}
}

func (l *Logger) Sync() {
	_ = l.zap.Sync()
}
