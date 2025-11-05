package instrument

import (
	"context"
	"weak"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

type instrumenterBuilder struct {
	tracer    Tracer
	newLogger func(ctx context.Context) logr.Logger

	mgr ctrl.Manager
}

func NewInstrumenter(mgr ctrl.Manager) *instrumenterBuilder {
	return &instrumenterBuilder{
		mgr: mgr,

		tracer: &NilTracer{},
		newLogger: func(ctx context.Context) logr.Logger {
			return logr.New(nil)
		},
	}
}

func (b *instrumenterBuilder) WithTracer(t Tracer) *instrumenterBuilder {
	b.tracer = t
	return b
}

func (b *instrumenterBuilder) WithLoggerFunc(l func(ctx context.Context) logr.Logger) *instrumenterBuilder {
	b.newLogger = l
	return b
}

func (b *instrumenterBuilder) Build() Instrumenter {
	return &instrumenter{
		mgr:             b.mgr,
		ctxCache:        make(map[string]weak.Pointer[context.Context]),
		ctxCacheReverse: make(map[*context.Context]string),
		newLogger:       b.newLogger,

		Tracer: b.tracer,
	}
}
