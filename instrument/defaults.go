package instrument

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

type NilTracer struct {
	embedded.Tracer
}

var _ Tracer = &NilTracer{}

func (nt *NilTracer) StartSpan(globalCtx *context.Context, localCtx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return localCtx, trace.SpanFromContext(localCtx)
}
