package instrument

import (
	"context"
	"fmt"

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
	ctxPtr := t.tracer.GetContextForEvent(e)
	logger := t.tracer.NewLogger(*ctxPtr)

	ctx, span := t.tracer.StartSpan(ctxPtr, ctx, fmt.Sprintf("event.create.handler.%T", t.inner))
	defer span.End()

	logger.Info("Received create event", "object_type", e.Object.GetObjectKind().GroupVersionKind(), "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())

	tracingQueue, ok := q.(*InstrumentedQueue[reconcile.Request])
	if !ok {
		// If the provided queue is not a TracingQueue, we cannot proceed with tracing
		t.inner.Create(ctx, e, q)
		return
	}

	// Create a temporary queue with the current context
	tempQueue := tracingQueue.WithContext(ctxPtr)
	t.inner.Create(ctx, e, tempQueue)
}

// Update is called in response to an update event -  e.g. Pod Updated.
func (t *instrumentedEventHandler[ObjectType]) Update(ctx context.Context, e event.TypedUpdateEvent[ObjectType], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	ctxPtr := t.tracer.GetContextForEvent(e)
	logger := t.tracer.NewLogger(*ctxPtr)

	ctx, span := t.tracer.StartSpan(ctxPtr, ctx, fmt.Sprintf("event.update.handler.%T", t.inner))
	defer span.End()

	patch, _ := jsondiff.Compare(e.ObjectOld, e.ObjectNew,
		jsondiff.Ignores("/metadata/managedFields", "/kind", "/apiVersion"),
	)

	logger.Info("Received update event", "object_type", e.ObjectOld.GetObjectKind().GroupVersionKind(), "name", e.ObjectOld.GetName(), "namespace", e.ObjectOld.GetNamespace(), "patch_data", patch)

	tracingQueue, ok := q.(*InstrumentedQueue[reconcile.Request])
	if !ok {
		// If the provided queue is not a TracingQueue, we cannot proceed with tracing
		t.inner.Update(ctx, e, q)
		return
	}

	// Create a temporary queue with the current context
	tempQueue := tracingQueue.WithContext(&ctx)
	t.inner.Update(ctx, e, tempQueue)

}

// Delete is called in response to a delete event - e.g. Pod Deleted.
func (t *instrumentedEventHandler[ObjectType]) Delete(ctx context.Context, e event.TypedDeleteEvent[ObjectType], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	ctxPtr := t.tracer.GetContextForEvent(e)
	logger := t.tracer.NewLogger(*ctxPtr)

	ctx, span := t.tracer.StartSpan(ctxPtr, ctx, fmt.Sprintf("event.delete.handler.%T", t.inner))
	defer span.End()

	logger.Info("Received delete event", "object_type", e.Object.GetObjectKind().GroupVersionKind(), "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())

	tracingQueue, ok := q.(*InstrumentedQueue[reconcile.Request])
	if !ok {
		// If the provided queue is not a TracingQueue, we cannot proceed with tracing
		t.inner.Delete(ctx, e, q)
		return
	}

	// Create a temporary queue with the current context
	tempQueue := tracingQueue.WithContext(ctxPtr)
	t.inner.Delete(ctx, e, tempQueue)
}

// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request - e.g. reconcile Autoscaling, or a Webhook.
func (t *instrumentedEventHandler[ObjectType]) Generic(ctx context.Context, e event.TypedGenericEvent[ObjectType], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	ctxPtr := t.tracer.GetContextForEvent(e)
	logger := t.tracer.NewLogger(*ctxPtr)

	ctx, span := t.tracer.StartSpan(ctxPtr, *ctxPtr, fmt.Sprintf("event.generic.handler.%T", t.inner))
	defer span.End()

	logger.Info("Received generic event", "object_type", e.Object.GetObjectKind().GroupVersionKind(), "name", e.Object.GetName(), "namespace", e.Object.GetNamespace())

	tracingQueue, ok := q.(*InstrumentedQueue[reconcile.Request])
	if !ok {
		// If the provided queue is not a TracingQueue, we cannot proceed with tracing
		t.inner.Generic(ctx, e, q)
		return
	}

	// Create a temporary queue with the current context
	tempQueue := tracingQueue.WithContext(ctxPtr)
	t.inner.Generic(ctx, e, tempQueue)
}
