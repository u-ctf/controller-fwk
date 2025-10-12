package instrument

import (
	"context"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/wI2L/jsondiff"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type instrumentedEventHandler[ObjectType client.Object] struct {
	tracer    Instrumenter
	inner     handler.TypedEventHandler[ObjectType, reconcile.Request]
	innerName string
}

func NewInstrumentedEventHandler[ObjectType client.Object](tracer Instrumenter, inner handler.TypedEventHandler[ObjectType, reconcile.Request]) handler.TypedEventHandler[ObjectType, reconcile.Request] {
	return &instrumentedEventHandler[ObjectType]{
		tracer:    tracer,
		inner:     inner,
		innerName: fmt.Sprintf("%T", inner),
	}
}

// Ensure tracingEventHandler implements the handler.TypedEventHandler interface
func (t *instrumentedEventHandler[ObjectType]) Create(
	ctx context.Context,
	e event.TypedCreateEvent[ObjectType],
	q workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	hub := t.tracer.GetOrCreateSentryHubForEvent(e)
	ctx = sentry.SetHubOnContext(ctx, hub)

	tx := sentry.StartTransaction(ctx, fmt.Sprintf("event.create.handler.%T", t.inner))
	defer tx.Finish()

	// Add breadcrumb: beginning of the create event
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "controller.event",
		Message:  "Processing create event",
		Level:    sentry.LevelInfo,
		Data: map[string]interface{}{
			"object": map[string]interface{}{
				"type":            fmt.Sprintf("%T", e.Object),
				"gvk":             e.Object.GetObjectKind().GroupVersionKind().String(),
				"name":            e.Object.GetName(),
				"namespace":       e.Object.GetNamespace(),
				"generation":      e.Object.GetGeneration(),
				"resourceVersion": e.Object.GetResourceVersion(),
			},
		},
	}, nil)

	tracingQueue, ok := q.(*InstrumentedQueue[reconcile.Request])
	if !ok {
		// If the provided queue is not a TracingQueue, we cannot proceed with tracing
		t.inner.Create(ctx, e, q)
		return
	}

	// Create a temporary queue with the current hub
	tempQueue := tracingQueue.WithHub(hub)
	t.inner.Create(ctx, e, tempQueue)

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "controller.event",
		Message:  "Finished create event processing",
		Level:    sentry.LevelInfo,
	}, nil)
}

// Update is called in response to an update event -  e.g. Pod Updated.
func (t *instrumentedEventHandler[ObjectType]) Update(ctx context.Context, e event.TypedUpdateEvent[ObjectType], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	hub := t.tracer.GetOrCreateSentryHubForEvent(e)
	ctx = sentry.SetHubOnContext(ctx, hub)

	tx := sentry.StartTransaction(ctx, fmt.Sprintf("event.update.handler.%T", t.inner))
	defer tx.Finish()

	patch, _ := jsondiff.Compare(e.ObjectOld, e.ObjectNew,
		jsondiff.Ignores("/metadata/managedFields", "/kind", "/apiVersion"),
	)

	// Add breadcrumb: beginning of the update event
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "controller.event",
		Message:  "Processing update event",
		Level:    sentry.LevelInfo,
		Data: map[string]interface{}{
			"object_old": map[string]interface{}{
				"type":            fmt.Sprintf("%T", e.ObjectOld),
				"gvk":             e.ObjectOld.GetObjectKind().GroupVersionKind().String(),
				"name":            e.ObjectOld.GetName(),
				"namespace":       e.ObjectOld.GetNamespace(),
				"generation":      e.ObjectOld.GetGeneration(),
				"resourceVersion": e.ObjectOld.GetResourceVersion(),
			},
			"object_new": map[string]interface{}{
				"type":            fmt.Sprintf("%T", e.ObjectNew),
				"gvk":             e.ObjectNew.GetObjectKind().GroupVersionKind().String(),
				"name":            e.ObjectNew.GetName(),
				"namespace":       e.ObjectNew.GetNamespace(),
				"generation":      e.ObjectNew.GetGeneration(),
				"resourceVersion": e.ObjectNew.GetResourceVersion(),
			},
			"patch": patch,
		},
	}, nil)

	tracingQueue, ok := q.(*InstrumentedQueue[reconcile.Request])
	if !ok {
		// If the provided queue is not a TracingQueue, we cannot proceed with tracing
		t.inner.Update(ctx, e, q)
		return
	}

	// Create a temporary queue with the current hub
	tempQueue := tracingQueue.WithHub(hub)
	t.inner.Update(ctx, e, tempQueue)

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "controller.event",
		Message:  "Finished update event processing",
		Level:    sentry.LevelInfo,
	}, nil)
}

// Delete is called in response to a delete event - e.g. Pod Deleted.
func (t *instrumentedEventHandler[ObjectType]) Delete(ctx context.Context, e event.TypedDeleteEvent[ObjectType], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	hub := t.tracer.GetOrCreateSentryHubForEvent(e)
	ctx = sentry.SetHubOnContext(ctx, hub)

	tx := sentry.StartTransaction(ctx, fmt.Sprintf("event.delete.handler.%T", t.inner))
	defer tx.Finish()

	// Add breadcrumb: beginning of the delete event
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "controller.event",
		Message:  "Processing delete event",
		Level:    sentry.LevelInfo,
		Data: map[string]interface{}{
			"object": map[string]interface{}{
				"type":            fmt.Sprintf("%T", e.Object),
				"gvk":             e.Object.GetObjectKind().GroupVersionKind().String(),
				"name":            e.Object.GetName(),
				"namespace":       e.Object.GetNamespace(),
				"generation":      e.Object.GetGeneration(),
				"resourceVersion": e.Object.GetResourceVersion(),
			},
		},
	}, nil)

	tracingQueue, ok := q.(*InstrumentedQueue[reconcile.Request])
	if !ok {
		// If the provided queue is not a TracingQueue, we cannot proceed with tracing
		t.inner.Delete(ctx, e, q)
		return
	}

	// Create a temporary queue with the current hub
	tempQueue := tracingQueue.WithHub(hub)
	t.inner.Delete(ctx, e, tempQueue)

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "controller.event",
		Message:  "Finished delete event processing",
		Level:    sentry.LevelInfo,
	}, nil)
}

// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request - e.g. reconcile Autoscaling, or a Webhook.
func (t *instrumentedEventHandler[ObjectType]) Generic(ctx context.Context, e event.TypedGenericEvent[ObjectType], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	hub := t.tracer.GetOrCreateSentryHubForEvent(e)
	ctx = sentry.SetHubOnContext(ctx, hub)

	tx := sentry.StartTransaction(ctx, fmt.Sprintf("event.generic.handler.%T", t.inner))
	defer tx.Finish()

	// Add breadcrumb: beginning of the generic event
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "controller.event",
		Message:  "Processing generic event",
		Level:    sentry.LevelInfo,
		Data: map[string]interface{}{
			"object": map[string]interface{}{
				"type":            fmt.Sprintf("%T", e.Object),
				"gvk":             e.Object.GetObjectKind().GroupVersionKind().String(),
				"name":            e.Object.GetName(),
				"namespace":       e.Object.GetNamespace(),
				"generation":      e.Object.GetGeneration(),
				"resourceVersion": e.Object.GetResourceVersion(),
			},
		},
	}, nil)

	tracingQueue, ok := q.(*InstrumentedQueue[reconcile.Request])
	if !ok {
		// If the provided queue is not a TracingQueue, we cannot proceed with tracing
		t.inner.Generic(ctx, e, q)
		return
	}

	// Create a temporary queue with the current hub
	tempQueue := tracingQueue.WithHub(hub)
	t.inner.Generic(ctx, e, tempQueue)

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "controller.event",
		Message:  "Finished generic event processing",
		Level:    sentry.LevelInfo,
	}, nil)
}
