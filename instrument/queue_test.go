package instrument

import (
	"context"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type queueGetResult struct {
	item     reconcile.Request
	shutdown bool
	timedOut bool
}

func getFromWorkqueueWithTimeout(q workqueue.TypedRateLimitingInterface[*reconcile.Request], timeout time.Duration) queueGetResult {
	resultCh := make(chan queueGetResult, 1)

	go func() {
		item, shutdown := q.Get()
		if item == nil {
			resultCh <- queueGetResult{shutdown: shutdown}
			return
		}

		resultCh <- queueGetResult{item: *item, shutdown: shutdown}
	}()

	select {
	case result := <-resultCh:
		return result
	case <-time.After(timeout):
		q.ShutDown()
		<-resultCh
		return queueGetResult{timedOut: true}
	}
}

func getFromInstrumentedQueueWithTimeout(q *InstrumentedQueue[reconcile.Request], timeout time.Duration) queueGetResult {
	resultCh := make(chan queueGetResult, 1)

	go func() {
		item, shutdown := q.Get()
		resultCh <- queueGetResult{item: item, shutdown: shutdown}
	}()

	select {
	case result := <-resultCh:
		return result
	case <-time.After(timeout):
		q.ShutDown()
		<-resultCh
		return queueGetResult{timedOut: true}
	}
}

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
	meta, exists := queueWithContext.getMetaOf(testRequest)
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

func TestInstrumentedQueue_AddWhileDelayedItemPending_IsProcessedImmediately(t *testing.T) {
	internalQueue := workqueue.NewTypedRateLimitingQueue[*reconcile.Request](workqueue.DefaultTypedControllerRateLimiter[*reconcile.Request]())
	instrumentedQueue := NewInstrumentedQueue(internalQueue)

	ctx := context.Background()
	queueWithContext := instrumentedQueue.WithContext(&ctx)

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "test-name",
		},
	}

	queueWithContext.AddAfter(req, time.Hour)
	queueWithContext.Add(req)

	type getResult struct {
		item     reconcile.Request
		shutdown bool
	}

	resultCh := make(chan getResult, 1)
	go func() {
		item, shutdown := queueWithContext.Get()
		resultCh <- getResult{item: item, shutdown: shutdown}
	}()

	select {
	case result := <-resultCh:
		if result.shutdown {
			t.Fatalf("expected queue to return pending item, got shutdown")
		}

		if result.item != req {
			t.Fatalf("expected dequeued item to match request")
		}
	case <-time.After(250 * time.Millisecond):
		queueWithContext.ShutDown()
		<-resultCh
		t.Fatalf("expected immediate dequeue when same item is added while delayed")
	}
}

func TestInstrumentedQueue_MatchesWorkqueueBehavior_ForDelayedThenImmediateAdd(t *testing.T) {
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "test-namespace",
			Name:      "test-name",
		},
	}

	plainQueue := workqueue.NewTypedRateLimitingQueue[*reconcile.Request](workqueue.DefaultTypedControllerRateLimiter[*reconcile.Request]())
	plainQueue.AddAfter(&req, time.Hour)
	plainQueue.Add(&req)
	plainResult := getFromWorkqueueWithTimeout(plainQueue, 250*time.Millisecond)

	instrumentedInternal := workqueue.NewTypedRateLimitingQueue[*reconcile.Request](workqueue.DefaultTypedControllerRateLimiter[*reconcile.Request]())
	instrumentedQueue := NewInstrumentedQueue(instrumentedInternal)
	ctx := context.Background()
	queueWithContext := instrumentedQueue.WithContext(&ctx)
	queueWithContext.AddAfter(req, time.Hour)
	queueWithContext.Add(req)
	instrumentedResult := getFromInstrumentedQueueWithTimeout(queueWithContext, 250*time.Millisecond)

	if plainResult.timedOut != instrumentedResult.timedOut {
		t.Fatalf("expected same timeout behavior, workqueue timedOut=%v instrumented timedOut=%v", plainResult.timedOut, instrumentedResult.timedOut)
	}

	if plainResult.shutdown != instrumentedResult.shutdown {
		t.Fatalf("expected same shutdown behavior, workqueue shutdown=%v instrumented shutdown=%v", plainResult.shutdown, instrumentedResult.shutdown)
	}

	if plainResult.item != instrumentedResult.item {
		t.Fatalf("expected same dequeued item, workqueue item=%+v instrumented item=%+v", plainResult.item, instrumentedResult.item)
	}

	if plainQueue.Len() != queueWithContext.Len() {
		t.Fatalf("expected same queue length after dequeue, workqueue len=%d instrumented len=%d", plainQueue.Len(), queueWithContext.Len())
	}
}
