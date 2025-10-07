package tracing

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"weak"

	"github.com/getsentry/sentry-go"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/priorityqueue"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Tracer interface {
	InstrumentRequestHandler(handler handler.TypedEventHandler[client.Object, reconcile.Request]) handler.TypedEventHandler[client.Object, reconcile.Request]
	InstrumentPredicate(predicate predicate.Predicate) predicate.Predicate

	GetSentryHubForRequest(req reconcile.Request) (*sentry.Hub, bool)
	GetOrCreateSentryHubForEvent(event any) *sentry.Hub

	NewQueue(mgr ctrl.Manager) func(controllerName string, rateLimiter workqueue.TypedRateLimiter[reconcile.Request]) workqueue.TypedRateLimitingInterface[reconcile.Request]
}

type tracer struct {
	mgr ctrl.Manager

	queue *TracingQueue[reconcile.Request]

	lock     sync.Mutex
	hubCache map[string]weak.Pointer[sentry.Hub]
}

func NewTracer(mgr ctrl.Manager) Tracer {
	return &tracer{
		mgr:      mgr,
		hubCache: make(map[string]weak.Pointer[sentry.Hub]),
	}
}

func InstrumentRequestHandlerWithTracer[T client.Object](t Tracer, handler handler.TypedEventHandler[T, reconcile.Request]) handler.TypedEventHandler[T, reconcile.Request] {
	return NewTracingEventHandler(t, handler)
}

func (t *tracer) InstrumentRequestHandler(handler handler.TypedEventHandler[client.Object, reconcile.Request]) handler.TypedEventHandler[client.Object, reconcile.Request] {
	return NewTracingEventHandler(t, handler)
}

func (t *tracer) InstrumentPredicate(predicate predicate.Predicate) predicate.Predicate {
	return NewTracingPredicate(t, predicate)
}

func (t *tracer) NewQueue(mgr ctrl.Manager) func(controllerName string, rateLimiter workqueue.TypedRateLimiter[reconcile.Request]) workqueue.TypedRateLimitingInterface[reconcile.Request] {
	return func(controllerName string, _ workqueue.TypedRateLimiter[reconcile.Request]) workqueue.TypedRateLimitingInterface[reconcile.Request] {
		ratelimiter := workqueue.DefaultTypedControllerRateLimiter[*reconcile.Request]()

		if ptr.Deref(mgr.GetControllerOptions().UsePriorityQueue, false) {
			t.queue = NewTracingQueue(priorityqueue.New(controllerName, func(o *priorityqueue.Opts[*reconcile.Request]) {
				o.Log = mgr.GetLogger().WithValues("controller", controllerName)
				o.RateLimiter = ratelimiter
			}))

			return t.queue
		}

		t.queue = NewTracingQueue(workqueue.NewTypedRateLimitingQueueWithConfig(ratelimiter, workqueue.TypedRateLimitingQueueConfig[*reconcile.Request]{
			Name: controllerName,
		}))
		return t.queue
	}
}

func (t *tracer) GetSentryHubForRequest(req reconcile.Request) (*sentry.Hub, bool) {
	if t.queue.internalQueue == nil {
		newHub := sentry.CurrentHub().Clone()
		newHub.PushScope()
		return newHub, false
	}

	meta, ok := t.queue.GetMetaOf(req)
	if !ok {
		newHub := sentry.CurrentHub().Clone()
		newHub.PushScope()
		return newHub, false
	}

	return meta.Hub, true
}

func (t *tracer) GetOrCreateSentryHubForEvent(event any) *sentry.Hub {
	data, err := json.Marshal(event)
	if err != nil {
		ctrl.Log.Error(err, "failed to marshal event for tracing")
		newHub := sentry.CurrentHub().Clone()
		newHub.PushScope()
		return newHub
	}
	hash := md5.Sum(data)
	digest := fmt.Sprintf("%x", hash)

	t.lock.Lock()
	defer t.lock.Unlock()

	val, ok := t.hubCache[digest]
	if ok && val.Value() != nil {
		return val.Value()
	}

	hub := sentry.CurrentHub().Clone()
	hub.PushScope()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		ctx := sentry.NewPropagationContext()
		ctx.TraceID = hash
		scope.SetPropagationContext(ctx)
	})

	runtime.AddCleanup(hub, t.cleanupKey, digest)

	t.hubCache[digest] = weak.Make(hub)
	return hub
}

func (t *tracer) cleanupKey(key string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	delete(t.hubCache, key)
}
