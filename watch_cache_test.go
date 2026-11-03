package ctrlfwk

import (
	"fmt"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestWatchCacheConcurrentAccess(t *testing.T) {
	cache := NewWatchCache(nil)

	var wg sync.WaitGroup
	for worker := range 8 {
		wg.Go(func() {
			for i := range 500 {
				var (
					shared  = NewWatchKey(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: fmt.Sprintf("kind-%d", i%4)}, CacheTypeEnqueueForOwner)
					private = NewWatchKey(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: fmt.Sprintf("kind-%d-%d", worker, i)}, CacheTypeEnqueueForOwner)
				)
				cache.AddWatchSource(shared)
				cache.AddWatchSource(private)

				if !cache.IsWatchingSource(shared) {
					t.Errorf("the watch source %q vanished from the cache", shared)
					return
				}
				_ = cache.IsWatchingSource(private)
			}
		})
	}
	wg.Wait()

	key := NewWatchKey(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "kind-0"}, CacheTypeEnqueueForOwner)
	if !cache.IsWatchingSource(key) {
		t.Fatalf("the watch source %q is missing from the cache after the run", key)
	}
}
