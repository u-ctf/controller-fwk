package ctrlfwk

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

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
	// GetController returns the controller for the watch cache
	GetController() controller.TypedController[reconcile.Request]
}

type WatchCache struct {
	cache      map[WatchCacheKey]bool
	controller controller.TypedController[reconcile.Request]
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

func (w *WatchCache) GetController() controller.TypedController[reconcile.Request] {
	return w.controller
}

func (w *WatchCache) SetController(ctrler controller.TypedController[reconcile.Request]) {
	w.controller = ctrler
}
