package library

import "sigs.k8s.io/controller-runtime/pkg/client"

type WatchCacheKey string
type WatchCacheType string

const (
	CacheTypeEnqueueForOwner WatchCacheType = "enqueueForOwner"
)

type Watcher interface {
	// AddWatchSource adds a watch source to the cache
	AddWatchSource(key WatchCacheKey)
	// IsWatchSource checks if the key is a watch source
	IsWatchingSource(key WatchCacheKey) bool
}

type WatchCache struct {
	cache map[WatchCacheKey]bool
}

func NewWatchKey(obj client.Object, watchType WatchCacheType) WatchCacheKey {
	return WatchCacheKey(obj.GetName() + "/" + string(watchType))
}

func (w *WatchCache) AddWatchSource(key WatchCacheKey) {
	if w.cache == nil {
		w.cache = make(map[WatchCacheKey]bool)
	}
	w.cache[key] = true
}

func (w *WatchCache) IsWatchingSource(key WatchCacheKey) bool {
	if w.cache == nil {
		return false
	}
	_, ok := w.cache[key]
	return ok
}
