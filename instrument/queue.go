package instrument

import (
	"runtime"
	"sync"
	"time"
	"weak"

	"github.com/getsentry/sentry-go"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller/priorityqueue"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type encapsulatedItem[T comparable] struct {
	Hub *sentry.Hub

	Object weak.Pointer[T]
}

type InstrumentedQueue[T comparable] struct {
	lock *sync.Mutex

	currentHub    *sentry.Hub
	internalQueue workqueue.TypedRateLimitingInterface[*T]

	metamap map[T]*encapsulatedItem[T]
}

var _ priorityqueue.PriorityQueue[reconcile.Request] = InstrumentedQueue[reconcile.Request]{}

func NewInstrumentedQueue[T comparable](queue workqueue.TypedRateLimitingInterface[*T]) *InstrumentedQueue[T] {
	return &InstrumentedQueue[T]{
		lock:          &sync.Mutex{},
		internalQueue: queue,
		metamap:       make(map[T]*encapsulatedItem[T]),
	}
}

func (q InstrumentedQueue[T]) cleanupKey(key T) {
	q.lock.Lock()
	defer q.lock.Unlock()

	val, ok := q.metamap[key]
	if !ok {
		return
	}

	if val.Object.Value() != nil {
		return
	}

	delete(q.metamap, key)
}

func (q InstrumentedQueue[T]) WithHub(hub *sentry.Hub) *InstrumentedQueue[T] {
	return &InstrumentedQueue[T]{
		lock:          q.lock,
		currentHub:    hub,
		internalQueue: q.internalQueue,
		metamap:       q.metamap,
	}
}

func (q InstrumentedQueue[T]) GetMetaOf(item T) (*encapsulatedItem[T], bool) {
	val, ok := q.metamap[item]
	if !ok {
		return nil, false
	}

	return val, true
}

func (q InstrumentedQueue[T]) isInQueue(item T) bool {
	val, ok := q.metamap[item]
	if !ok {
		return false
	}

	if val.Object.Value() == nil {
		delete(q.metamap, item)
		return false
	}

	return true
}

func (q InstrumentedQueue[T]) Add(item T) {
	q.lock.Lock()
	defer q.lock.Unlock()

	pointerToItem := &item
	weakPointerToItem := weak.Make(pointerToItem)
	runtime.AddCleanup(pointerToItem, q.cleanupKey, item)

	if !q.isInQueue(item) {
		q.internalQueue.Add(pointerToItem)
		q.metamap[item] = &encapsulatedItem[T]{
			Hub:    q.currentHub,
			Object: weakPointerToItem,
		}
	}
}

func (q InstrumentedQueue[T]) AddAfter(item T, duration time.Duration) {
	q.lock.Lock()
	defer q.lock.Unlock()

	pointerToItem := &item
	weakPointerToItem := weak.Make(pointerToItem)
	runtime.AddCleanup(pointerToItem, q.cleanupKey, item)

	if !q.isInQueue(item) {
		q.internalQueue.AddAfter(pointerToItem, duration)
		q.metamap[item] = &encapsulatedItem[T]{
			Hub:    q.currentHub,
			Object: weakPointerToItem,
		}
	}
}

func (q InstrumentedQueue[T]) AddRateLimited(item T) {
	q.lock.Lock()
	defer q.lock.Unlock()

	pointerToItem := &item
	weakPointerToItem := weak.Make(pointerToItem)
	runtime.AddCleanup(pointerToItem, q.cleanupKey, item)

	if !q.isInQueue(item) {
		q.internalQueue.AddRateLimited(pointerToItem)
		q.metamap[item] = &encapsulatedItem[T]{
			Hub:    q.currentHub,
			Object: weakPointerToItem,
		}
	}
}

func (q InstrumentedQueue[T]) Done(item T) {
	capsule, ok := q.metamap[item]
	if !ok {
		return
	}

	q.internalQueue.Done(capsule.Object.Value())

	q.lock.Lock()
	defer q.lock.Unlock()
	delete(q.metamap, item)
}

func (q InstrumentedQueue[T]) Forget(item T) {
	capsule, ok := q.metamap[item]
	if !ok {
		return
	}

	q.internalQueue.Forget(capsule.Object.Value())

	q.lock.Lock()
	defer q.lock.Unlock()
	delete(q.metamap, item)
}

func (q InstrumentedQueue[T]) Get() (item T, shutdown bool) {
	pointerToItem, shutdown := q.internalQueue.Get()
	if pointerToItem == nil {
		var zero T
		return zero, shutdown
	}

	item = *pointerToItem
	return item, shutdown
}

func (q InstrumentedQueue[T]) Len() int {
	return q.internalQueue.Len()
}

func (q InstrumentedQueue[T]) NumRequeues(item T) int {
	capsule, ok := q.metamap[item]
	if !ok {
		return 0
	}

	return q.internalQueue.NumRequeues(capsule.Object.Value())
}

func (q InstrumentedQueue[T]) ShutDown() {
	q.internalQueue.ShutDown()
}

func (q InstrumentedQueue[T]) ShutDownWithDrain() {
	q.internalQueue.ShutDownWithDrain()
}

func (q InstrumentedQueue[T]) ShuttingDown() bool {
	return q.internalQueue.ShuttingDown()
}

func (q InstrumentedQueue[T]) AddWithOpts(o priorityqueue.AddOpts, Items ...T) {
	pq, ok := q.internalQueue.(priorityqueue.PriorityQueue[*T])
	if ok {
		q.lock.Lock()
		defer q.lock.Unlock()

		for _, item := range Items {
			pointerToItem := &item
			weakPointerToItem := weak.Make(pointerToItem)
			runtime.AddCleanup(pointerToItem, q.cleanupKey, item)

			if !q.isInQueue(item) {
				if o.After > 0 {
					pq.AddAfter(pointerToItem, o.After)
				} else if o.RateLimited {
					pq.AddRateLimited(pointerToItem)
				} else {
					pq.Add(pointerToItem)
				}

				q.metamap[item] = &encapsulatedItem[T]{
					Hub:    q.currentHub,
					Object: weakPointerToItem,
				}
			}
		}
		return
	}

	for _, item := range Items {
		if o.After > 0 {
			q.AddAfter(item, o.After)
		} else if o.RateLimited {
			q.AddRateLimited(item)
		} else {
			q.Add(item)
		}
	}
}

func (q InstrumentedQueue[T]) GetWithPriority() (item T, priority int, shutdown bool) {
	pq, ok := q.internalQueue.(priorityqueue.PriorityQueue[*T])
	if ok {
		pointerToItem, priority, shutdown := pq.GetWithPriority()
		if pointerToItem == nil {
			var zero T
			return zero, priority, shutdown
		}

		item = *pointerToItem
		return item, priority, shutdown
	}

	pointerToItem, shutdown := q.internalQueue.Get()
	if pointerToItem == nil {
		var zero T
		return zero, 0, shutdown
	}

	item = *pointerToItem
	return item, 0, shutdown
}
