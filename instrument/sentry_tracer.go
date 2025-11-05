package instrument

import (
	"context"

	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/trace"
)

type SentryTracer struct {
	tracer trace.Tracer
}

func NewSentryTracer(t trace.Tracer) *SentryTracer {
	return &SentryTracer{
		tracer: t,
	}
}

func (t *SentryTracer) StartSpan(globalCtx *context.Context, localCtx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	hub := sentry.GetHubFromContext(*globalCtx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
		*globalCtx = sentry.SetHubOnContext(*globalCtx, hub)
	}
	localCtx = sentry.SetHubOnContext(localCtx, hub)

	return t.tracer.Start(localCtx, spanName, opts...)
}
