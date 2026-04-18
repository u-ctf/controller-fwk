package instrument

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/trace"
	traceembedded "go.opentelemetry.io/otel/trace/embedded"
)

type fakeTracer struct {
	traceembedded.Tracer
	called bool
}

func (t *fakeTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	t.called = true
	return context.WithValue(ctx, "span", spanName), trace.SpanFromContext(ctx)
}

func TestNilTracerAndOtelTracer(t *testing.T) {
	ctx := context.Background()
	nilTracer := &NilTracer{}
	returnedCtx, span := nilTracer.StartSpan(nil, ctx, "noop")
	if returnedCtx != ctx {
		t.Fatal("expected nil tracer to return the local context unchanged")
	}
	if span == nil {
		t.Fatal("expected nil tracer to return a non-nil span handle")
	}

	fake := &fakeTracer{}
	otelTracer := NewOtelTracer(fake)
	returnedCtx, _ = otelTracer.StartSpan(nil, ctx, "demo")
	if !fake.called {
		t.Fatal("expected otel tracer to delegate to the wrapped tracer")
	}
	if returnedCtx.Value("span") != "demo" {
		t.Fatalf("expected delegated context to contain span marker, got %v", returnedCtx.Value("span"))
	}
}

func TestNewLoggerFuncAndMergedContextBehaviors(t *testing.T) {
	logger := logr.Discard()
	if got := NewLoggerFunc(logger)(context.Background()); got.GetSink() != logger.GetSink() {
		t.Fatal("expected logger factory to return the provided logger")
	}

	deadlineSoon := time.Now().Add(10 * time.Millisecond)
	deadlineLater := time.Now().Add(50 * time.Millisecond)
	ctx1, cancel1 := context.WithDeadline(context.Background(), deadlineLater)
	defer cancel1()
	ctx2, cancel2 := context.WithDeadline(context.Background(), deadlineSoon)
	defer cancel2()

	merged := NewMergedContext(context.WithValue(ctx1, "a", 1), context.WithValue(ctx2, "b", 2))
	deadline, ok := merged.Deadline()
	if !ok || deadline.After(deadlineSoon.Add(5*time.Millisecond)) {
		t.Fatalf("expected earliest deadline, got %v ok=%v", deadline, ok)
	}
	if merged.Value("a") != 1 || merged.Value("b") != 2 {
		t.Fatalf("expected merged context values, got a=%v b=%v", merged.Value("a"), merged.Value("b"))
	}

	ctxDone1, stop1 := context.WithCancel(context.Background())
	ctxDone2, stop2 := context.WithCancel(context.Background())
	defer stop2()
	mergedDone := NewMergedContext(ctxDone1, ctxDone2)
	stop1()
	select {
	case <-mergedDone.Done():
	case <-time.After(time.Second):
		t.Fatal("expected merged Done channel to close when either context is cancelled")
	}

	errCtx1 := context.WithValue(context.Background(), "x", "y")
	errCtx2 := context.WithValue(context.Background(), "z", "w")
	errCtx1, cancelErr1 := context.WithCancelCause(errCtx1)
	errCtx2, cancelErr2 := context.WithCancelCause(errCtx2)
	defer cancelErr2(nil)
	cause := errors.New("boom")
	cancelErr1(cause)
	mergedErr := NewMergedContext(errCtx1, errCtx2)
	if !errors.Is(mergedErr.Err(), context.Canceled) {
		t.Fatalf("expected merged context error to prioritize ctx1 cancellation, got %v", mergedErr.Err())
	}
}
