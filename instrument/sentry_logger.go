package instrument

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-logr/logr"
)

func NewSentryLoggerFunc(logger logr.Logger) func(ctx context.Context) logr.Logger {
	return func(ctx context.Context) logr.Logger {
		return logger.WithSink(NewSentrySink(ctx, logger.GetSink()))
	}
}

type sentrySink struct {
	context.Context
	logr.LogSink

	values map[string]any
}

var _ logr.LogSink = &sentrySink{}

func NewSentrySink(ctx context.Context, logSink logr.LogSink) *sentrySink {
	return &sentrySink{
		Context: ctx,
		LogSink: logSink,
		values:  make(map[string]any),
	}
}

func keysAndValuesToMap(keysAndValues ...any) sentry.BreadcrumbHint {
	if len(keysAndValues)%2 != 0 {
		keysAndValues = append(keysAndValues, "unknown")
	}

	m := make(map[string]any)
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		m[fmt.Sprint(keysAndValues[i])] = fmt.Sprint(keysAndValues[i+1])
	}
	return m
}

func (s *sentrySink) Info(level int, msg string, keysAndValues ...any) {
	s.LogSink.Info(level, msg, keysAndValues...)

	hub := sentry.GetHubFromContext(s.Context)
	if hub == nil {
		return
	}

	data := keysAndValuesToMap(keysAndValues...)
	maps.Copy(data, s.values)

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Level:     sentry.LevelInfo,
		Message:   msg,
		Data:      data,
		Type:      "log",
		Timestamp: time.Now(),
	}, &data)
}

func (s *sentrySink) Error(err error, msg string, keysAndValues ...any) {
	s.LogSink.Error(err, msg, keysAndValues...)

	hub := sentry.GetHubFromContext(s.Context)
	if hub == nil {
		return
	}

	hub.CaptureException(err)
}

func (s *sentrySink) Enabled(level int) bool {
	return s.LogSink.Enabled(level)
}

func (s *sentrySink) WithValues(keysAndValues ...any) logr.LogSink {
	newValues := keysAndValuesToMap(keysAndValues...)
	maps.Copy(newValues, s.values)

	return &sentrySink{
		Context: s.Context,
		LogSink: s.LogSink.WithValues(keysAndValues...),
		values:  newValues,
	}
}

func (s *sentrySink) WithName(name string) logr.LogSink {
	return &sentrySink{
		Context: s.Context,
		LogSink: s.LogSink.WithName(name),
		values:  s.values,
	}
}
