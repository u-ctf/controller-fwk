package instrument

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/trace"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/priorityqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type testTracer struct{}

func (testTracer) StartSpan(_ *context.Context, localCtx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
	return localCtx, trace.SpanFromContext(localCtx)
}

type handlerInstrumenter struct {
	Tracer
	ctx       *context.Context
	spanNames []string
	logger    logr.Logger
}

func (h *handlerInstrumenter) InstrumentRequestHandler(inner handler.TypedEventHandler[client.Object, reconcile.Request]) handler.TypedEventHandler[client.Object, reconcile.Request] {
	return inner
}

func (h *handlerInstrumenter) GetContextForRequest(reconcile.Request) (*context.Context, bool) {
	return h.ctx, true
}

func (h *handlerInstrumenter) GetContextForEvent(any) *context.Context {
	return h.ctx
}

func (h *handlerInstrumenter) NewQueue(ctrl.Manager) func(string, workqueue.TypedRateLimiter[reconcile.Request]) workqueue.TypedRateLimitingInterface[reconcile.Request] {
	return nil
}

func (h *handlerInstrumenter) Cleanup(*context.Context, reconcile.Request) {}

func (h *handlerInstrumenter) NewLogger(context.Context) logr.Logger {
	return h.logger
}

func (h *handlerInstrumenter) StartSpan(globalCtx *context.Context, localCtx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	h.spanNames = append(h.spanNames, spanName)
	return h.Tracer.StartSpan(globalCtx, localCtx, spanName, opts...)
}

type configMapEventHandler struct {
	createFn  func(context.Context, event.TypedCreateEvent[*corev1.ConfigMap], workqueue.TypedRateLimitingInterface[reconcile.Request])
	updateFn  func(context.Context, event.TypedUpdateEvent[*corev1.ConfigMap], workqueue.TypedRateLimitingInterface[reconcile.Request])
	deleteFn  func(context.Context, event.TypedDeleteEvent[*corev1.ConfigMap], workqueue.TypedRateLimitingInterface[reconcile.Request])
	genericFn func(context.Context, event.TypedGenericEvent[*corev1.ConfigMap], workqueue.TypedRateLimitingInterface[reconcile.Request])
}

func (h configMapEventHandler) Create(ctx context.Context, e event.TypedCreateEvent[*corev1.ConfigMap], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if h.createFn != nil {
		h.createFn(ctx, e, q)
	}
}

func (h configMapEventHandler) Update(ctx context.Context, e event.TypedUpdateEvent[*corev1.ConfigMap], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if h.updateFn != nil {
		h.updateFn(ctx, e, q)
	}
}

func (h configMapEventHandler) Delete(ctx context.Context, e event.TypedDeleteEvent[*corev1.ConfigMap], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if h.deleteFn != nil {
		h.deleteFn(ctx, e, q)
	}
}

func (h configMapEventHandler) Generic(ctx context.Context, e event.TypedGenericEvent[*corev1.ConfigMap], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if h.genericFn != nil {
		h.genericFn(ctx, e, q)
	}
}

func testRequest(name string) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: "default"}}
}

func TestInstrumenterBuilderAndContextLifecycle(t *testing.T) {
	mgr := newTestManager(t)
	loggerFuncCalled := false
	built := NewInstrumenter(mgr).
		WithTracer(testTracer{}).
		WithLoggerFunc(func(ctx context.Context) logr.Logger {
			loggerFuncCalled = ctx != nil
			return logr.Discard()
		}).
		Build()

	inst, ok := built.(*instrumenter)
	if !ok {
		t.Fatalf("expected concrete instrumenter, got %T", built)
	}

	wrapped := InstrumentRequestHandlerWithTracer[client.Object](inst, nil)
	if _, ok := wrapped.(*instrumentedEventHandler[client.Object]); !ok {
		t.Fatalf("expected helper to wrap handler, got %T", wrapped)
	}
	wrapped = inst.InstrumentRequestHandler(nil)
	if _, ok := wrapped.(*instrumentedEventHandler[client.Object]); !ok {
		t.Fatalf("expected instrumenter method to wrap handler, got %T", wrapped)
	}

	ctx := context.Background()
	if got := inst.NewLogger(ctx); got.GetSink() != logr.Discard().GetSink() || !loggerFuncCalled {
		t.Fatal("expected custom logger func to be used")
	}

	req := testRequest("queued")
	ctxWithValue := context.WithValue(context.Background(), struct{}{}, "value")
	queueFactory := inst.NewQueue(mgr)
	queue := queueFactory("test-controller", nil)
	queue.(*InstrumentedQueue[reconcile.Request]).WithContext(&ctxWithValue).Add(req)

	ctxPtr, ok := inst.GetContextForRequest(req)
	if !ok || ctxPtr != &ctxWithValue {
		t.Fatalf("expected request context to be recovered from queue, got ok=%v ctx=%p want=%p", ok, ctxPtr, &ctxWithValue)
	}

	missingCtx, found := inst.GetContextForRequest(testRequest("missing"))
	if found || missingCtx == nil {
		t.Fatalf("expected missing request to fall back to default context, got found=%v ctx=%v", found, missingCtx)
	}

	first := inst.GetContextForEvent(struct{ Name string }{Name: "same"})
	second := inst.GetContextForEvent(struct{ Name string }{Name: "same"})
	if first != second {
		t.Fatal("expected equal events to reuse cached contexts")
	}

	fallbackFirst := inst.GetContextForEvent(struct{ F func() }{})
	fallbackSecond := inst.GetContextForEvent(struct{ F func() }{})
	if fallbackFirst != fallbackSecond {
		t.Fatal("expected fmt-based digest path to reuse cached contexts")
	}

	inst.Cleanup(first, req)
	if _, ok := inst.queue.getMetaOf(req); ok {
		t.Fatal("expected cleanup to remove queued request metadata")
	}
	if _, ok := inst.ctxCacheReverse[first]; ok {
		t.Fatal("expected cleanup to remove reverse context cache entry")
	}

	noQueue := NewInstrumenter(mgr).Build().(*instrumenter)
	defaultCtx, found := noQueue.GetContextForRequest(req)
	if found || defaultCtx == nil {
		t.Fatalf("expected nil queue to return default context, got found=%v ctx=%v", found, defaultCtx)
	}
	noQueue.Cleanup(first, req)
}

func TestInstrumentedEventHandler_DelegatesAllEventTypes(t *testing.T) {
	baseCtx := context.WithValue(context.Background(), struct{}{}, "event")
	tracker := &handlerInstrumenter{Tracer: testTracer{}, ctx: &baseCtx, logger: logr.Discard()}

	internalQueue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[*reconcile.Request]())
	queue := NewInstrumentedQueue(internalQueue)

	createReq := testRequest("create")
	updateReq := testRequest("update")
	deleteReq := testRequest("delete")
	genericReq := testRequest("generic")

	handler := NewInstrumentedEventHandler[*corev1.ConfigMap](tracker, configMapEventHandler{
		createFn: func(_ context.Context, _ event.TypedCreateEvent[*corev1.ConfigMap], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			q.Add(createReq)
		},
		updateFn: func(_ context.Context, _ event.TypedUpdateEvent[*corev1.ConfigMap], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			q.Add(updateReq)
		},
		deleteFn: func(_ context.Context, _ event.TypedDeleteEvent[*corev1.ConfigMap], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			q.Add(deleteReq)
		},
		genericFn: func(_ context.Context, _ event.TypedGenericEvent[*corev1.ConfigMap], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			q.Add(genericReq)
		},
	})

	obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "obj", Namespace: "default"}}
	updated := obj.DeepCopy()
	updated.Data = map[string]string{"changed": "true"}

	handler.Create(context.Background(), event.TypedCreateEvent[*corev1.ConfigMap]{Object: obj}, queue)
	handler.Update(context.Background(), event.TypedUpdateEvent[*corev1.ConfigMap]{ObjectOld: obj, ObjectNew: updated}, queue)
	handler.Delete(context.Background(), event.TypedDeleteEvent[*corev1.ConfigMap]{Object: obj}, queue)
	handler.Generic(context.Background(), event.TypedGenericEvent[*corev1.ConfigMap]{Object: obj}, queue)

	if meta, ok := queue.getMetaOf(createReq); !ok || meta.Context != &baseCtx {
		t.Fatalf("expected create event to preserve event context, got ok=%v meta=%#v", ok, meta)
	}
	if meta, ok := queue.getMetaOf(updateReq); !ok || meta.Context == nil {
		t.Fatalf("expected update event to enqueue with context, got ok=%v meta=%#v", ok, meta)
	}
	if meta, ok := queue.getMetaOf(deleteReq); !ok || meta.Context != &baseCtx {
		t.Fatalf("expected delete event to preserve event context, got ok=%v meta=%#v", ok, meta)
	}
	if meta, ok := queue.getMetaOf(genericReq); !ok || meta.Context != &baseCtx {
		t.Fatalf("expected generic event to preserve event context, got ok=%v meta=%#v", ok, meta)
	}
	if len(tracker.spanNames) != 4 {
		t.Fatalf("expected spans for all event types, got %v", tracker.spanNames)
	}
}

func TestInstrumentedEventHandler_FallsBackToPlainQueue(t *testing.T) {
	baseCtx := context.Background()
	tracker := &handlerInstrumenter{Tracer: testTracer{}, ctx: &baseCtx, logger: logr.Discard()}
	fallbackCalled := false

	handler := NewInstrumentedEventHandler[*corev1.ConfigMap](tracker, configMapEventHandler{
		createFn: func(_ context.Context, _ event.TypedCreateEvent[*corev1.ConfigMap], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			fallbackCalled = true
			q.Add(testRequest("plain"))
		},
	})

	plainQueue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
	obj := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "obj", Namespace: "default"}}
	handler.Create(context.Background(), event.TypedCreateEvent[*corev1.ConfigMap]{Object: obj}, plainQueue)

	if !fallbackCalled {
		t.Fatal("expected plain queue branch to delegate directly")
	}
	if item, shutdown := plainQueue.Get(); shutdown || item != testRequest("plain") {
		t.Fatalf("expected plain queue to receive enqueued request, got item=%+v shutdown=%v", item, shutdown)
	}
}

func TestInstrumentedQueue_AdditionalMethods(t *testing.T) {
	t.Run("done forget and shutdown", func(t *testing.T) {
		internalQueue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[*reconcile.Request]())
		queue := NewInstrumentedQueue(internalQueue)
		ctx := context.Background()
		withCtx := queue.WithContext(&ctx)
		req := testRequest("rate-limited")

		withCtx.AddRateLimited(req)
		if !withCtx.isInQueue(req) {
			t.Fatal("expected rate-limited item to be tracked")
		}
		if count := withCtx.NumRequeues(req); count < 0 {
			t.Fatalf("expected non-negative requeue count, got %d", count)
		}
		withCtx.Forget(req)
		if _, ok := withCtx.getMetaOf(req); ok {
			t.Fatal("expected forget to remove item metadata")
		}

		withCtx.Add(req)
		got, shutdown := withCtx.Get()
		if shutdown || got != req {
			t.Fatalf("expected queued item before done, got item=%+v shutdown=%v", got, shutdown)
		}
		withCtx.Done(req)
		if _, ok := withCtx.getMetaOf(req); ok {
			t.Fatal("expected done to remove item metadata")
		}

		withCtx.ShutDownWithDrain()
		if !withCtx.ShuttingDown() {
			t.Fatal("expected queue to report shutting down")
		}
		zero, shutdown := withCtx.Get()
		if !shutdown || zero != (reconcile.Request{}) {
			t.Fatalf("expected zero request after shutdown, got item=%+v shutdown=%v", zero, shutdown)
		}
	})

	t.Run("add with opts fallback", func(t *testing.T) {
		internalQueue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[*reconcile.Request]())
		queue := NewInstrumentedQueue(internalQueue)
		ctx := context.Background()
		withCtx := queue.WithContext(&ctx)
		req := testRequest("opts")

		withCtx.AddWithOpts(priorityqueue.AddOpts{}, req)
		item, priority, shutdown := withCtx.GetWithPriority()
		if shutdown || priority != 0 || item != req {
			t.Fatalf("expected fallback GetWithPriority to return queued item, got item=%+v priority=%d shutdown=%v", item, priority, shutdown)
		}

		withCtx.AddWithOpts(priorityqueue.AddOpts{After: time.Millisecond}, req)
		if _, ok := withCtx.getMetaOf(req); !ok {
			t.Fatal("expected delayed AddWithOpts to record metadata")
		}
	})

	t.Run("priority queue path", func(t *testing.T) {
		pq := priorityqueue.New("test", func(o *priorityqueue.Opts[*reconcile.Request]) {
			o.Log = logr.Discard()
			o.RateLimiter = workqueue.DefaultTypedControllerRateLimiter[*reconcile.Request]()
		})
		queue := NewInstrumentedQueue[reconcile.Request](pq)
		ctx := context.Background()
		withCtx := queue.WithContext(&ctx)
		req := testRequest("priority")

		withCtx.AddWithOpts(priorityqueue.AddOpts{}, req)
		item, _, shutdown := withCtx.GetWithPriority()
		if shutdown || item != req {
			t.Fatalf("expected priority queue path to return queued item, got item=%+v shutdown=%v", item, shutdown)
		}
	})
}
