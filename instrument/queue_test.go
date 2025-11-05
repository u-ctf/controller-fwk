package instrument

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestNewInstrumentedQueue(t *testing.T) {
	internalQueue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[*reconcile.Request]())

	instrumentedQueue := NewInstrumentedQueue(internalQueue)

	if instrumentedQueue.internalQueue != internalQueue {
		t.Errorf("expected internal queue to be set correctly")
	}

	if instrumentedQueue.metamap == nil {
		t.Errorf("expected metamap to be initialized")
	}

	if instrumentedQueue.lock == nil {
		t.Errorf("expected lock to be initialized")
	}
}

func TestInstrumentedQueue_AddAndGet(t *testing.T) {
	internalQueue := workqueue.NewTypedRateLimitingQueue[*reconcile.Request](workqueue.DefaultTypedControllerRateLimiter[*reconcile.Request]())
	instrumentedQueue := NewInstrumentedQueue(internalQueue)

	// Create a context and set it on the queue
	ctx := context.Background()
	ctxPtr := &ctx
	queueWithContext := instrumentedQueue.WithContext(ctxPtr)

	testRequest := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "test-name",
		},
	}

	// Test Add
	queueWithContext.Add(testRequest)

	// Verify metadata is stored
	meta, exists := queueWithContext.GetMetaOf(testRequest)
	if !exists {
		t.Errorf("expected metadata to exist for added item")
	}

	if meta.Context != ctxPtr {
		t.Errorf("expected context to be stored in metadata")
	}

	// Test Get
	retrievedItem, shutdown := queueWithContext.Get()
	if shutdown {
		t.Errorf("expected queue not to be shut down")
	}

	if retrievedItem != testRequest {
		t.Errorf("expected retrieved item to match added item")
	}

	// Test queue length
	if queueWithContext.Len() != 0 {
		t.Errorf("expected queue length to be 0, got %d", queueWithContext.Len())
	}
}
