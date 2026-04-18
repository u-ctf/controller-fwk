package ctrlfwk_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	ctrlfwk "github.com/u-ctf/controller-fwk"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWatchCacheOptionsBuilder_BuildsExpectedOptions(t *testing.T) {
	httpClient := &http.Client{}
	scheme := runtime.NewScheme()
	mapper := meta.NewDefaultRESTMapper(nil)
	defaultNamespaces := map[string]cache.Config{
		"default": {
			LabelSelector: labels.SelectorFromSet(labels.Set{"app": "demo"}),
		},
	}
	defaultLabelSelector := labels.SelectorFromSet(labels.Set{"managed-by": "ctrlfwk"})
	defaultFieldSelector := fields.OneTermEqualSelector("metadata.name", "demo")
	defaultTransform := func(obj interface{}) (interface{}, error) {
		return obj, nil
	}
	defaultWatchErrorHandler := func(ctx context.Context, reflector *toolscache.Reflector, err error) {
	}
	configMap := &corev1.ConfigMap{}
	byObject := map[client.Object]cache.ByObject{
		configMap: {
			Namespaces: map[string]cache.Config{
				"tenant-a": {
					FieldSelector: fields.OneTermEqualSelector("metadata.namespace", "tenant-a"),
				},
			},
		},
	}
	newInformer := func(toolscache.ListerWatcher, runtime.Object, time.Duration, toolscache.Indexers) toolscache.SharedIndexInformer {
		return nil
	}
	syncPeriod := 15 * time.Minute

	options := ctrlfwk.NewWatchCacheOptionsBuilder().
		WithHTTPClient(httpClient).
		WithScheme(scheme).
		WithMapper(mapper).
		WithSyncPeriod(syncPeriod).
		WithReaderFailOnMissingInformer(true).
		WithDefaultNamespaces(defaultNamespaces).
		WithDefaultLabelSelector(defaultLabelSelector).
		WithDefaultFieldSelector(defaultFieldSelector).
		WithDefaultTransform(defaultTransform).
		WithDefaultWatchErrorHandler(defaultWatchErrorHandler).
		WithDefaultUnsafeDisableDeepCopy(true).
		WithDefaultEnableWatchBookmarks(false).
		WithByObject(byObject).
		WithNewInformer(newInformer).
		Build()

	if options.HTTPClient != httpClient {
		t.Fatalf("expected HTTP client to be preserved")
	}

	if options.Scheme != scheme {
		t.Fatalf("expected scheme to be preserved")
	}

	if options.Mapper != mapper {
		t.Fatalf("expected mapper to be preserved")
	}

	if options.SyncPeriod == nil || *options.SyncPeriod != syncPeriod {
		t.Fatalf("expected sync period %s, got %v", syncPeriod, options.SyncPeriod)
	}

	if !options.ReaderFailOnMissingInformer {
		t.Fatalf("expected ReaderFailOnMissingInformer to be true")
	}

	if !reflect.DeepEqual(options.DefaultNamespaces, defaultNamespaces) {
		t.Fatalf("expected default namespaces to match")
	}

	if got := options.DefaultLabelSelector.String(); got != defaultLabelSelector.String() {
		t.Fatalf("expected default label selector %q, got %q", defaultLabelSelector.String(), got)
	}

	if got := options.DefaultFieldSelector.String(); got != defaultFieldSelector.String() {
		t.Fatalf("expected default field selector %q, got %q", defaultFieldSelector.String(), got)
	}

	transformed, err := options.DefaultTransform("value")
	if err != nil {
		t.Fatalf("expected default transform to succeed: %v", err)
	}
	if transformed != "value" {
		t.Fatalf("expected transformed value to be preserved, got %v", transformed)
	}

	watchErr := errors.New("watch failed")
	options.DefaultWatchErrorHandler(context.Background(), nil, watchErr)

	if options.DefaultUnsafeDisableDeepCopy == nil || !*options.DefaultUnsafeDisableDeepCopy {
		t.Fatalf("expected DefaultUnsafeDisableDeepCopy to be true")
	}

	if options.DefaultEnableWatchBookmarks == nil || *options.DefaultEnableWatchBookmarks {
		t.Fatalf("expected DefaultEnableWatchBookmarks to be false")
	}

	if !reflect.DeepEqual(options.ByObject, byObject) {
		t.Fatalf("expected by-object options to match")
	}

	if reflect.ValueOf(options.NewInformer).Pointer() != reflect.ValueOf(newInformer).Pointer() {
		t.Fatalf("expected new informer function to match")
	}
}

func TestWatchCacheOptionsBuilder_ResetPointerFields(t *testing.T) {
	options := ctrlfwk.NewWatchCacheOptionsBuilder().
		WithSyncPeriod(time.Minute).
		WithDefaultUnsafeDisableDeepCopy(true).
		WithDefaultEnableWatchBookmarks(true).
		WithoutSyncPeriod().
		WithoutDefaultUnsafeDisableDeepCopy().
		WithoutDefaultEnableWatchBookmarks().
		Build()

	if options.SyncPeriod != nil {
		t.Fatalf("expected sync period to be reset")
	}

	if options.DefaultUnsafeDisableDeepCopy != nil {
		t.Fatalf("expected DefaultUnsafeDisableDeepCopy to be reset")
	}

	if options.DefaultEnableWatchBookmarks != nil {
		t.Fatalf("expected DefaultEnableWatchBookmarks to be reset")
	}
}

func TestWatchCacheByObjectBuilder_BuildsExpectedOptions(t *testing.T) {
	labelSelector := labels.SelectorFromSet(labels.Set{"component": "api"})
	fieldSelector := fields.OneTermEqualSelector("metadata.name", "demo")
	transform := func(obj interface{}) (interface{}, error) {
		return obj, nil
	}
	syncPeriod := 5 * time.Minute
	defaultConfig := cache.Config{
		LabelSelector: labels.SelectorFromSet(labels.Set{"tenant": "a"}),
	}

	byObject := ctrlfwk.NewWatchCacheByObjectBuilder().
		WithNamespace("default", defaultConfig).
		WithLabelSelector(labelSelector).
		WithFieldSelector(fieldSelector).
		WithTransform(transform).
		WithUnsafeDisableDeepCopy(true).
		WithEnableWatchBookmarks(false).
		WithSyncPeriod(syncPeriod).
		Build()

	if !reflect.DeepEqual(byObject.Namespaces, map[string]cache.Config{"default": defaultConfig}) {
		t.Fatalf("expected namespaces to match")
	}

	if got := byObject.Label.String(); got != labelSelector.String() {
		t.Fatalf("expected label selector %q, got %q", labelSelector.String(), got)
	}

	if got := byObject.Field.String(); got != fieldSelector.String() {
		t.Fatalf("expected field selector %q, got %q", fieldSelector.String(), got)
	}

	transformed, err := byObject.Transform("value")
	if err != nil {
		t.Fatalf("expected transform to succeed: %v", err)
	}
	if transformed != "value" {
		t.Fatalf("expected transformed value to be preserved, got %v", transformed)
	}

	if byObject.UnsafeDisableDeepCopy == nil || !*byObject.UnsafeDisableDeepCopy {
		t.Fatalf("expected UnsafeDisableDeepCopy to be true")
	}

	if byObject.EnableWatchBookmarks == nil || *byObject.EnableWatchBookmarks {
		t.Fatalf("expected EnableWatchBookmarks to be false")
	}

	if byObject.SyncPeriod == nil || *byObject.SyncPeriod != syncPeriod {
		t.Fatalf("expected sync period %s, got %v", syncPeriod, byObject.SyncPeriod)
	}
}

func TestWatchCacheByObjectBuilder_ResetPointerFields(t *testing.T) {
	byObject := ctrlfwk.NewWatchCacheByObjectBuilder().
		WithUnsafeDisableDeepCopy(true).
		WithEnableWatchBookmarks(true).
		WithSyncPeriod(time.Minute).
		WithoutUnsafeDisableDeepCopy().
		WithoutEnableWatchBookmarks().
		WithoutSyncPeriod().
		Build()

	if byObject.UnsafeDisableDeepCopy != nil {
		t.Fatalf("expected UnsafeDisableDeepCopy to be reset")
	}

	if byObject.EnableWatchBookmarks != nil {
		t.Fatalf("expected EnableWatchBookmarks to be reset")
	}

	if byObject.SyncPeriod != nil {
		t.Fatalf("expected SyncPeriod to be reset")
	}
}

func TestWatchCacheOptionsBuilder_WithByObjectFor_AddsEntry(t *testing.T) {
	configMap := &corev1.ConfigMap{}
	byObject := ctrlfwk.NewWatchCacheByObjectBuilder().
		WithLabelSelector(labels.SelectorFromSet(labels.Set{"component": "worker"})).
		Build()

	options := ctrlfwk.NewWatchCacheOptionsBuilder().
		WithByObjectFor(configMap, byObject).
		Build()

	if len(options.ByObject) != 1 {
		t.Fatalf("expected one by-object entry, got %d", len(options.ByObject))
	}

	stored, ok := options.ByObject[configMap]
	if !ok {
		t.Fatalf("expected configmap entry to exist")
	}

	if got := stored.Label.String(); got != byObject.Label.String() {
		t.Fatalf("expected stored label selector %q, got %q", byObject.Label.String(), got)
	}
}

func TestNewCacheOptionsBuilder_CompatibilityAlias(t *testing.T) {
	options := ctrlfwk.NewCacheOptionsBuilder().WithReaderFailOnMissingInformer(true).Build()

	if !options.ReaderFailOnMissingInformer {
		t.Fatalf("expected compatibility constructor to return a working builder")
	}
}

func TestNewByObjectBuilder_CompatibilityAlias(t *testing.T) {
	byObject := ctrlfwk.NewByObjectBuilder().WithEnableWatchBookmarks(true).Build()

	if byObject.EnableWatchBookmarks == nil || !*byObject.EnableWatchBookmarks {
		t.Fatalf("expected compatibility constructor to return a working by-object builder")
	}
}
