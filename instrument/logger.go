package instrument

import (
	"context"

	"github.com/go-logr/logr"
)

type Logger interface {
	Enabled() bool
	Error(err error, msg string, keysAndValues ...any)
	GetSink() logr.LogSink
	GetV() int
	Info(msg string, keysAndValues ...any)
	IsZero() bool
	V(level int) logr.Logger
	WithCallDepth(depth int) logr.Logger
	WithCallStackHelper() (func(), logr.Logger)
	WithName(name string) logr.Logger
	WithSink(sink logr.LogSink) logr.Logger
	WithValues(keysAndValues ...any) logr.Logger
}

func NewLoggerFunc(logger logr.Logger) func(ctx context.Context) logr.Logger {
	return func(ctx context.Context) logr.Logger {
		return logger
	}
}
