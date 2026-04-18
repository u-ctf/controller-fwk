package ctrlfwk

import (
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WatchCacheOptionsBuilder provides a fluent builder for controller-runtime cache options.
type WatchCacheOptionsBuilder struct {
	Options cache.Options
}

// WatchCacheByObjectBuilder provides a fluent builder for controller-runtime per-object cache options.
type WatchCacheByObjectBuilder struct {
	ByObject cache.ByObject
}

// NewWatchCacheOptionsBuilder creates a new builder with empty cache options.
func NewWatchCacheOptionsBuilder() *WatchCacheOptionsBuilder {
	return &WatchCacheOptionsBuilder{
		Options: cache.Options{},
	}
}

// NewCacheOptionsBuilder is kept as a compatibility alias.
func NewCacheOptionsBuilder() *WatchCacheOptionsBuilder {
	return NewWatchCacheOptionsBuilder()
}

// NewWatchCacheByObjectBuilder creates a new builder with empty per-object cache options.
func NewWatchCacheByObjectBuilder() *WatchCacheByObjectBuilder {
	return &WatchCacheByObjectBuilder{
		ByObject: cache.ByObject{},
	}
}

// NewByObjectBuilder is kept as a short compatibility alias.
func NewByObjectBuilder() *WatchCacheByObjectBuilder {
	return NewWatchCacheByObjectBuilder()
}

func (b *WatchCacheOptionsBuilder) WithHTTPClient(httpClient *http.Client) *WatchCacheOptionsBuilder {
	b.Options.HTTPClient = httpClient
	return b
}

func (b *WatchCacheOptionsBuilder) WithScheme(scheme *runtime.Scheme) *WatchCacheOptionsBuilder {
	b.Options.Scheme = scheme
	return b
}

func (b *WatchCacheOptionsBuilder) WithMapper(mapper meta.RESTMapper) *WatchCacheOptionsBuilder {
	b.Options.Mapper = mapper
	return b
}

func (b *WatchCacheOptionsBuilder) WithSyncPeriod(syncPeriod time.Duration) *WatchCacheOptionsBuilder {
	b.Options.SyncPeriod = &syncPeriod
	return b
}

func (b *WatchCacheOptionsBuilder) WithoutSyncPeriod() *WatchCacheOptionsBuilder {
	b.Options.SyncPeriod = nil
	return b
}

func (b *WatchCacheOptionsBuilder) WithReaderFailOnMissingInformer(enabled bool) *WatchCacheOptionsBuilder {
	b.Options.ReaderFailOnMissingInformer = enabled
	return b
}

func (b *WatchCacheOptionsBuilder) WithDefaultNamespaces(namespaces map[string]cache.Config) *WatchCacheOptionsBuilder {
	b.Options.DefaultNamespaces = namespaces
	return b
}

func (b *WatchCacheOptionsBuilder) WithDefaultLabelSelector(selector labels.Selector) *WatchCacheOptionsBuilder {
	b.Options.DefaultLabelSelector = selector
	return b
}

func (b *WatchCacheOptionsBuilder) WithDefaultFieldSelector(selector fields.Selector) *WatchCacheOptionsBuilder {
	b.Options.DefaultFieldSelector = selector
	return b
}

func (b *WatchCacheOptionsBuilder) WithDefaultTransform(transform toolscache.TransformFunc) *WatchCacheOptionsBuilder {
	b.Options.DefaultTransform = transform
	return b
}

func (b *WatchCacheOptionsBuilder) WithDefaultWatchErrorHandler(handler toolscache.WatchErrorHandlerWithContext) *WatchCacheOptionsBuilder {
	b.Options.DefaultWatchErrorHandler = handler
	return b
}

func (b *WatchCacheOptionsBuilder) WithDefaultUnsafeDisableDeepCopy(disable bool) *WatchCacheOptionsBuilder {
	b.Options.DefaultUnsafeDisableDeepCopy = &disable
	return b
}

func (b *WatchCacheOptionsBuilder) WithoutDefaultUnsafeDisableDeepCopy() *WatchCacheOptionsBuilder {
	b.Options.DefaultUnsafeDisableDeepCopy = nil
	return b
}

func (b *WatchCacheOptionsBuilder) WithDefaultEnableWatchBookmarks(enabled bool) *WatchCacheOptionsBuilder {
	b.Options.DefaultEnableWatchBookmarks = &enabled
	return b
}

func (b *WatchCacheOptionsBuilder) WithoutDefaultEnableWatchBookmarks() *WatchCacheOptionsBuilder {
	b.Options.DefaultEnableWatchBookmarks = nil
	return b
}

func (b *WatchCacheOptionsBuilder) WithByObject(byObject map[client.Object]cache.ByObject) *WatchCacheOptionsBuilder {
	b.Options.ByObject = byObject
	return b
}

func (b *WatchCacheOptionsBuilder) WithByObjectFor(object client.Object, byObject cache.ByObject) *WatchCacheOptionsBuilder {
	if b.Options.ByObject == nil {
		b.Options.ByObject = make(map[client.Object]cache.ByObject)
	}

	b.Options.ByObject[object] = byObject
	return b
}

func (b *WatchCacheOptionsBuilder) WithNewInformer(newInformer func(toolscache.ListerWatcher, runtime.Object, time.Duration, toolscache.Indexers) toolscache.SharedIndexInformer) *WatchCacheOptionsBuilder {
	b.Options.NewInformer = newInformer
	return b
}

func (b *WatchCacheOptionsBuilder) Build() cache.Options {
	return b.Options
}

func (b *WatchCacheByObjectBuilder) WithNamespaces(namespaces map[string]cache.Config) *WatchCacheByObjectBuilder {
	b.ByObject.Namespaces = namespaces
	return b
}

func (b *WatchCacheByObjectBuilder) WithNamespace(namespace string, config cache.Config) *WatchCacheByObjectBuilder {
	if b.ByObject.Namespaces == nil {
		b.ByObject.Namespaces = make(map[string]cache.Config)
	}

	b.ByObject.Namespaces[namespace] = config
	return b
}

func (b *WatchCacheByObjectBuilder) WithLabelSelector(selector labels.Selector) *WatchCacheByObjectBuilder {
	b.ByObject.Label = selector
	return b
}

func (b *WatchCacheByObjectBuilder) WithFieldSelector(selector fields.Selector) *WatchCacheByObjectBuilder {
	b.ByObject.Field = selector
	return b
}

func (b *WatchCacheByObjectBuilder) WithTransform(transform toolscache.TransformFunc) *WatchCacheByObjectBuilder {
	b.ByObject.Transform = transform
	return b
}

func (b *WatchCacheByObjectBuilder) WithUnsafeDisableDeepCopy(disable bool) *WatchCacheByObjectBuilder {
	b.ByObject.UnsafeDisableDeepCopy = &disable
	return b
}

func (b *WatchCacheByObjectBuilder) WithoutUnsafeDisableDeepCopy() *WatchCacheByObjectBuilder {
	b.ByObject.UnsafeDisableDeepCopy = nil
	return b
}

func (b *WatchCacheByObjectBuilder) WithEnableWatchBookmarks(enabled bool) *WatchCacheByObjectBuilder {
	b.ByObject.EnableWatchBookmarks = &enabled
	return b
}

func (b *WatchCacheByObjectBuilder) WithoutEnableWatchBookmarks() *WatchCacheByObjectBuilder {
	b.ByObject.EnableWatchBookmarks = nil
	return b
}

func (b *WatchCacheByObjectBuilder) WithSyncPeriod(syncPeriod time.Duration) *WatchCacheByObjectBuilder {
	b.ByObject.SyncPeriod = &syncPeriod
	return b
}

func (b *WatchCacheByObjectBuilder) WithoutSyncPeriod() *WatchCacheByObjectBuilder {
	b.ByObject.SyncPeriod = nil
	return b
}

func (b *WatchCacheByObjectBuilder) Build() cache.ByObject {
	return b.ByObject
}
