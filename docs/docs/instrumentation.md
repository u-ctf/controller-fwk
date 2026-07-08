# Instrumentation

The framework's instrumentation layer covers logging, tracing, and request-scoped context for your controllers. You wrap your reconciler once, and the logger it hands you carries the current trace context automatically.

It does not depend on the rest of the framework. There is no reliance on the framework context or the generic types, so you can use it in a plain controller-runtime controller.

## The Instrumenter interface

Everything is built around the `Instrumenter` interface. An implementation provides structured logging, tracing, request-scoped context, and a work queue that ties them together.

```go
type Instrumenter interface {
    InstrumentRequestHandler(handler handler.TypedEventHandler[client.Object, reconcile.Request]) handler.TypedEventHandler[client.Object, reconcile.Request]
    GetContextForRequest(req reconcile.Request) (*context.Context, bool)
    GetContextForEvent(event any) *context.Context
    NewQueue(mgr ctrl.Manager) func(controllerName string, rateLimiter workqueue.TypedRateLimiter[reconcile.Request]) workqueue.TypedRateLimitingInterface[reconcile.Request]
    Cleanup(ctx *context.Context, req reconcile.Request)
    NewLogger(ctx context.Context) logr.Logger
    Tracer
}

type Tracer interface {
    StartSpan(globalCtx *context.Context, localCtx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
}
```

`InstrumentedReconciler` wraps your reconciler and applies all of this on each request:

```go
type InstrumentedReconciler struct {
    internalReconciler reconcile.TypedReconciler[reconcile.Request]
    Instrumenter
}
```

## Setup

There are three steps: build an instrumenter, wrap your reconciler with it, and point it at a logging or tracing backend. Use the builder to assemble one:

```go
instrumenter := instrument.NewInstrumenter(mgr).
    WithLoggerFunc(loggerFunc).
    WithTracer(tracer).
    Build()
```

The rest of this page shows how to wire that instrumenter to Sentry or Jaeger, and how to attach it to a controller.

## Sentry integration

Sentry handles both error tracking and performance monitoring, using the logger for the former and the tracer for the latter.

Add the dependencies:

```bash
go get github.com/getsentry/sentry-go
go get github.com/getsentry/sentry-go/otel
go get github.com/getsentry/sentry-go/otel/otlp
```

If you are on sentry-go older than v0.47.0, the setup used `sentryotel.NewSentrySpanProcessor()` and `sentryotel.NewSentryPropagator()`. Both were removed in v0.47.0.

```go
package main

import (
    "context"

    "github.com/getsentry/sentry-go"
    sentryotel "github.com/getsentry/sentry-go/otel"
    sentryotlp "github.com/getsentry/sentry-go/otel/otlp"
    "go.opentelemetry.io/otel"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"

    "github.com/u-ctf/controller-fwk/instrument"
)

func main() {
    ctx := context.Background()
    dsn := "https://your-sentry-dsn@sentry.io/project-id"

    // Initialize Sentry
    sentry.Init(sentry.ClientOptions{
        Dsn:              dsn,
        EnableTracing:    true,
        TracesSampleRate: 1.0, // Adjust sampling rate for production
        Environment:      "production", // or "development", "staging"
        Release:          "v1.0.0",
        Integrations: func(i []sentry.Integration) []sentry.Integration {
            return append(i, sentryotel.NewOtelIntegration())
        },
    })

    // Set up OpenTelemetry with Sentry
    exporter, _ := sentryotlp.NewTraceExporter(ctx, dsn)
    tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
    otel.SetTracerProvider(tp)

    // Create instrumenter with Sentry logger and tracer
    instrumenter := instrument.NewInstrumenter(mgr).
        WithLoggerFunc(instrument.NewSentryLoggerFunc(ctrl.Log.WithName("controller"))).
        WithTracer(instrument.NewSentryTracer(tp.Tracer("controller"))).
        Build()

    // Use in your reconciler setup
    if err := (&YourReconciler{
        Client:        mgr.GetClient(),
        Instrumenter:  instrumenter,
        EventRecorder: mgr.GetEventRecorderFor("your-controller"),
        RuntimeScheme: mgr.GetScheme(),
    }).SetupWithManager(mgr); err != nil {
        setupLog.Error(err, "unable to create controller")
        os.Exit(1)
    }
}
```

### The logger and tracer come as a pair

The Sentry logger and tracer have to be configured together. This holds even when you set `EnableTracing: false` in the client options: leave one out and the instrumentation misbehaves.

```go
// Correct: configure both the logger and the tracer
instrumenter := instrument.NewInstrumenter(mgr).
    WithLoggerFunc(instrument.NewSentryLoggerFunc(baseLogger)).
    WithTracer(instrument.NewSentryTracer(tracer)).
    Build()

// Wrong: the tracer is missing, which will cause problems
instrumenter := instrument.NewInstrumenter(mgr).
    WithLoggerFunc(instrument.NewSentryLoggerFunc(baseLogger)).
    Build()
```

With Sentry wired up, reconciliation errors are captured with stack traces and tied to a release, spans record how long requests take, and each event carries the Kubernetes resource it came from.

## Jaeger integration

Jaeger gives you distributed tracing for visualizing how a request flows through your services.

Add the OpenTelemetry Jaeger dependencies:

```bash
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/exporters/jaeger
go get go.opentelemetry.io/otel/sdk/trace
```

```go
package main

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

    "github.com/u-ctf/controller-fwk/instrument"
)

func setupJaeger() (*sdktrace.TracerProvider, error) {
    // Create Jaeger exporter
    jaegerExporter, err := jaeger.New(
        jaeger.WithCollectorEndpoint(
            jaeger.WithEndpoint("http://localhost:14268/api/traces"),
        ),
    )
    if err != nil {
        return nil, err
    }

    // Create resource with service information
    res, err := resource.New(context.Background(),
        resource.WithAttributes(
            semconv.ServiceNameKey.String("my-controller"),
            semconv.ServiceVersionKey.String("v1.0.0"),
            semconv.DeploymentEnvironmentKey.String("production"),
        ),
    )
    if err != nil {
        return nil, err
    }

    // Create tracer provider
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(jaegerExporter),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.AlwaysSample()), // Adjust for production
    )

    return tp, nil
}

func main() {
    // Set up Jaeger tracing
    tp, err := setupJaeger()
    if err != nil {
        setupLog.Error(err, "failed to setup Jaeger")
        os.Exit(1)
    }
    defer func() { _ = tp.Shutdown(context.Background()) }()

    otel.SetTracerProvider(tp)

    // Create instrumenter with Jaeger tracer
    instrumenter := instrument.NewInstrumenter(mgr).
        WithLoggerFunc(func(ctx context.Context) logr.Logger {
            return ctrl.Log.WithName("controller")
        }).
        WithTracer(instrument.NewOtelTracer(tp.Tracer("controller"))).
        Build()

    // Use in your reconciler
    if err := (&YourReconciler{
        Client:        mgr.GetClient(),
        Instrumenter:  instrumenter,
        EventRecorder: mgr.GetEventRecorderFor("your-controller"),
        RuntimeScheme: mgr.GetScheme(),
    }).SetupWithManager(mgr); err != nil {
        setupLog.Error(err, "unable to create controller")
        os.Exit(1)
    }
}
```

Once traces reach Jaeger you can follow a request across services, spot where latency builds up, and add your own spans at points you care about.

## Attaching it to a controller

Swap the standard controller builder for the instrumented one:

```go
func (r *YourReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return instrument.InstrumentedControllerManagedBy(r.Instrumenter, mgr).
        For(&yourv1.YourResource{}).
        Named("your-controller").
        Build(r)
}
```

Embed the `Instrumenter` in your reconciler so its methods are available:

```go
type YourReconciler struct {
    client.Client
    instrument.Instrumenter
    record.EventRecorder
    RuntimeScheme *runtime.Scheme
    // ... other fields
}

func (r *YourReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // InstrumentedReconciler applies the instrumentation for you.
    // The logger from logf.FromContext(ctx) already carries the trace context.

    // Your reconciliation logic here

    return ctrl.Result{}, nil
}
```

## Custom logger functions

The logger function receives the request context, so you can pull values out of it and attach them to every log line. Adding the trace ID is a common case:

```go
func customLoggerFunc(ctx context.Context) logr.Logger {
    baseLogger := ctrl.Log.WithName("my-controller")

    if traceID := getTraceIDFromContext(ctx); traceID != "" {
        baseLogger = baseLogger.WithValues("traceId", traceID)
    }

    return baseLogger
}

instrumenter := instrument.NewInstrumenter(mgr).
    WithLoggerFunc(customLoggerFunc).
    Build()
```

## One instrumenter per controller

Nothing forces you to share a single instrumenter. You can build one per controller and, for example, sample a high-volume controller more aggressively than a critical one:

```go
// High-volume controller with sampling
highVolumeInstrumenter := instrument.NewInstrumenter(mgr).
    WithLoggerFunc(instrument.NewSentryLoggerFunc(ctrl.Log)).
    WithTracer(instrument.NewSentryTracer(sampledTracer)).
    Build()

// Critical controller with full tracing
criticalInstrumenter := instrument.NewInstrumenter(mgr).
    WithLoggerFunc(instrument.NewSentryLoggerFunc(ctrl.Log)).
    WithTracer(instrument.NewSentryTracer(fullTracer)).
    Build()
```

## Notes on running in production

Set the trace sampling rate to something sensible for your volume. Sampling every request is fine in development but expensive under load. The framework cleans up request context for you, so you do not need to manage it by hand.

Keep sensitive values out of logs and spans, since they end up in whichever backend you configured. Traces also travel over the network, so secure the connection and authenticate to the backend the same way you would any other service.

## Troubleshooting

If traces never show up, check that the tracer provider is set with `otel.SetTracerProvider` and that the exporter endpoint is reachable. If spans appear but the logger has no trace context, the context is being dropped somewhere between the request and your logging call. High memory use usually traces back to sampling everything or large batch sizes.

To see what the instrumentation is doing, turn on debug logging:

```go
ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
```

## Migrating from manual instrumentation

If you already set up loggers and tracers by hand inside your reconcilers, move that configuration up to the manager level:

1. Remove the manual logger and tracer setup from the reconcilers.
2. Build the instrumenter once, near the manager.
3. Use `InstrumentedControllerManagedBy` in place of the standard builder, passing the instrumenter as its first argument.
4. Embed `instrument.Instrumenter` in the reconciler struct.

The instrumentation targets controller-runtime v0.15+, OpenTelemetry Go v1.19+, and Go 1.19+.
