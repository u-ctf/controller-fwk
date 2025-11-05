package instrument

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type OtelTracer struct {
	trace.Tracer
}

func NewOtelTracer(t trace.Tracer) *OtelTracer {
	return &OtelTracer{
		Tracer: t,
	}
}

func (t *OtelTracer) StartSpan(globalCtx *context.Context, localCtx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.Start(localCtx, spanName, opts...)
}
