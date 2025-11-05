package instrument

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"weak"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/priorityqueue"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

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

type instrumenter struct {
	mgr ctrl.Manager

	queue *InstrumentedQueue[reconcile.Request]

	lock            sync.Mutex
	ctxCache        map[string]weak.Pointer[context.Context]
	ctxCacheReverse map[*context.Context]string
	newLogger       func(ctx context.Context) logr.Logger

	Tracer
}

func InstrumentRequestHandlerWithTracer[T client.Object](t Instrumenter, handler handler.TypedEventHandler[T, reconcile.Request]) handler.TypedEventHandler[T, reconcile.Request] {
	return NewInstrumentedEventHandler(t, handler)
}

func (t *instrumenter) InstrumentRequestHandler(handler handler.TypedEventHandler[client.Object, reconcile.Request]) handler.TypedEventHandler[client.Object, reconcile.Request] {
	return NewInstrumentedEventHandler(t, handler)
}

func (t *instrumenter) NewQueue(mgr ctrl.Manager) func(controllerName string, rateLimiter workqueue.TypedRateLimiter[reconcile.Request]) workqueue.TypedRateLimitingInterface[reconcile.Request] {
	return func(controllerName string, _ workqueue.TypedRateLimiter[reconcile.Request]) workqueue.TypedRateLimitingInterface[reconcile.Request] {
		ratelimiter := workqueue.DefaultTypedControllerRateLimiter[*reconcile.Request]()

		if ptr.Deref(mgr.GetControllerOptions().UsePriorityQueue, false) {
			t.queue = NewInstrumentedQueue(priorityqueue.New(controllerName, func(o *priorityqueue.Opts[*reconcile.Request]) {
				o.Log = mgr.GetLogger().WithValues("controller", controllerName)
				o.RateLimiter = ratelimiter
			}))

			return t.queue
		}

		t.queue = NewInstrumentedQueue(workqueue.NewTypedRateLimitingQueueWithConfig(ratelimiter, workqueue.TypedRateLimitingQueueConfig[*reconcile.Request]{
			Name: controllerName,
		}))
		return t.queue
	}
}

func (t *instrumenter) GetContextForRequest(req reconcile.Request) (*context.Context, bool) {
	var defaultContext = context.Background()
	if t.queue.internalQueue == nil {
		return &defaultContext, false
	}

	meta, ok := t.queue.GetMetaOf(req)
	if !ok {
		return &defaultContext, false
	}

	return meta.Context, true
}

func (t *instrumenter) GetContextForEvent(event any) *context.Context {
	var digest string

	data, err := json.Marshal(event)
	if err != nil {
		hash := md5.Sum([]byte(fmt.Sprintf("%#+v\n", event)))
		digest = fmt.Sprintf("%x", hash)
	} else {
		hash := md5.Sum(data)
		digest = fmt.Sprintf("%x", hash)
	}

	t.lock.Lock()
	defer t.lock.Unlock()

	val, ok := t.ctxCache[digest]
	if ok && val.Value() != nil {
		return val.Value()
	}

	ctx := context.Background()
	ctxPtr := &ctx

	runtime.AddCleanup(ctxPtr, t.cleanupKey, digest)

	t.ctxCache[digest] = weak.Make(ctxPtr)
	t.ctxCacheReverse[ctxPtr] = digest
	return ctxPtr
}

func (t *instrumenter) cleanupKey(key string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	delete(t.ctxCache, key)
}

func (t *instrumenter) Cleanup(ctx *context.Context, req reconcile.Request) {
	if t.queue == nil {
		return
	}

	t.queue.cleanupKey(req)

	t.lock.Lock()
	defer t.lock.Unlock()

	digest, ok := t.ctxCacheReverse[ctx]
	if !ok {
		return
	}

	delete(t.ctxCache, digest)
	delete(t.ctxCacheReverse, ctx)
}

func (t *instrumenter) NewLogger(ctx context.Context) logr.Logger {
	return t.newLogger(ctx)
}
