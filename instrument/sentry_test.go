package instrument

import (
	"context"
	"errors"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/trace/noop"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestSentryLoggerHelpers(t *testing.T) {
	mapped := keysAndValuesToMap("key", 1, "odd")
	if mapped["key"] != "1" || mapped["odd"] != "unknown" {
		t.Fatalf("unexpected mapped key-values: %#v", mapped)
	}

	hub := sentry.NewHub(nil, sentry.NewScope())
	ctx := sentry.SetHubOnContext(context.Background(), hub)
	baseLogger := logr.New(logf.NullLogSink{})
	sink := NewSentrySink(ctx, baseLogger.GetSink())
	_ = sink.Enabled(0)

	logger := NewSentryLoggerFunc(baseLogger)(ctx)
	if logger.GetSink() == nil {
		t.Fatal("expected sentry logger helper to return a logger with a sink")
	}

	sink.Info(0, "hello", "key", "value")
	sink.Error(errors.New("boom"), "failed", "step", "sync")

	withValues, ok := sink.WithValues("controller", "demo").(*sentrySink)
	if !ok || withValues.values["controller"] != "demo" {
		t.Fatalf("expected WithValues to preserve sentry sink values, got %#v", withValues)
	}
	withName, ok := sink.WithName("named").(*sentrySink)
	if !ok || withName.LogSink == nil {
		t.Fatalf("expected WithName to preserve sink wrapping, got %#v", withName)
	}
}

func TestSentryTracer_StartSpanUsesHubFromContext(t *testing.T) {
	tracer := NewSentryTracer(noop.NewTracerProvider().Tracer("tests"))

	hub := sentry.NewHub(nil, sentry.NewScope())
	globalCtx := sentry.SetHubOnContext(context.Background(), hub)
	localCtx, span := tracer.StartSpan(&globalCtx, context.Background(), "span")
	defer span.End()
	if sentry.GetHubFromContext(localCtx) != hub {
		t.Fatal("expected local context to inherit the existing hub")
	}

	withoutHub := context.Background()
	localCtx, span = tracer.StartSpan(&withoutHub, context.Background(), "span-with-clone")
	defer span.End()
	if sentry.GetHubFromContext(withoutHub) == nil || sentry.GetHubFromContext(localCtx) == nil {
		t.Fatal("expected tracer to clone and attach a hub when one is missing")
	}
}
