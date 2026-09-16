package ctrlfwk

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type testContextKey string

type testReconciler[K client.Object] struct {
	client.Client
}

func (testReconciler[K]) For(K) {}

func newFrameworkTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}

	return scheme
}

func newFrameworkTestContext(t *testing.T) Context[*corev1.ConfigMap] {
	t.Helper()

	scheme := newFrameworkTestScheme(t)
	reconciler := testReconciler[*corev1.ConfigMap]{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
	}

	base := context.WithValue(context.Background(), testContextKey("scope"), "controller-fwk")
	return NewContext(base, reconciler)
}

func TestContextHelpersPreserveBaseContextAndData(t *testing.T) {
	ctx := newFrameworkTestContext(t)

	if got := ctx.Value(testContextKey("scope")); got != "controller-fwk" {
		t.Fatalf("unexpected context value: %v", got)
	}

	resource := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "tests"}}
	ctx.SetCustomResource(resource)

	if got := ctx.GetCustomResource().Name; got != "demo" {
		t.Fatalf("unexpected custom resource name: %s", got)
	}

	clean := ctx.GetCleanCustomResource()
	clean.Name = "changed"
	if got := ctx.GetCleanCustomResource().Name; got != "demo" {
		t.Fatalf("clean copy should not be mutated, got %s", got)
	}

	reconciler := testReconciler[*corev1.ConfigMap]{
		Client: fake.NewClientBuilder().WithScheme(newFrameworkTestScheme(t)).Build(),
	}
	withData := NewContextWithData(context.Background(), reconciler, map[string]int{"count": 2})
	if withData.Data["count"] != 2 {
		t.Fatalf("unexpected context data: %#v", withData.Data)
	}
}

func TestHelpersCreateAndAnnotateObjects(t *testing.T) {
	secret := NewInstanceOf(&corev1.Secret{})
	if secret == nil {
		t.Fatal("expected a new secret instance")
	}

	var nilSecret *corev1.Secret
	if got := NewInstanceOf(nilSecret); got == nil {
		t.Fatal("expected a typed nil input to still produce a new secret instance")
	}

	configMap := &corev1.ConfigMap{}
	SetAnnotation(configMap, "example", "value")
	if got := GetAnnotation(configMap, "example"); got != "value" {
		t.Fatalf("unexpected annotation value: %s", got)
	}

	if got := GetAnnotation(&corev1.Secret{}, "missing"); got != "" {
		t.Fatalf("expected empty annotation lookup, got %q", got)
	}
}

func TestDependencyMethodsCoverTypedAndUntypedVariants(t *testing.T) {
	ctx := newFrameworkTestContext(t)

	beforeCalls := 0
	afterTyped := false
	afterNil := false

	dependency := &Dependency[*corev1.ConfigMap, Context[*corev1.ConfigMap], *corev1.Secret]{
		output:       &corev1.Secret{},
		name:         "backend-secret",
		namespace:    "workloads",
		isOptional:   true,
		waitForReady: true,
		addManagedBy: true,
		isReadyF: func(secret *corev1.Secret) bool {
			return secret != nil && secret.Data["ready"] != nil
		},
		beforeReconcileF: func(Context[*corev1.ConfigMap]) error {
			beforeCalls++
			return nil
		},
		afterReconcileF: func(_ Context[*corev1.ConfigMap], secret *corev1.Secret) error {
			if secret == nil {
				afterNil = true
				return nil
			}

			afterTyped = secret.Name == "backend-secret"
			return nil
		},
	}

	created := dependency.New()
	if _, ok := created.(*corev1.Secret); !ok {
		t.Fatalf("expected a secret instance, got %T", created)
	}

	if dependency.Kind() != "Secret" {
		t.Fatalf("unexpected dependency kind: %s", dependency.Kind())
	}

	if got := dependency.Key(); got != (types.NamespacedName{Name: "backend-secret", Namespace: "workloads"}) {
		t.Fatalf("unexpected static dependency key: %#v", got)
	}

	if got, want := dependency.ID(), fmt.Sprintf("%v,%v", dependency.Kind(), dependency.Key()); got != want {
		t.Fatalf("unexpected dependency id: got %q want %q", got, want)
	}

	dependency.keyF = func() types.NamespacedName {
		return types.NamespacedName{Name: "dynamic-secret", Namespace: "dynamic-ns"}
	}
	if got := dependency.Key(); got != (types.NamespacedName{Name: "dynamic-secret", Namespace: "dynamic-ns"}) {
		t.Fatalf("unexpected dynamic dependency key: %#v", got)
	}

	dependency.userIdentifier = "custom-id"
	if got := dependency.ID(); got != "custom-id" {
		t.Fatalf("unexpected custom dependency id: %s", got)
	}

	resolvedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "backend-secret", Namespace: "workloads"},
		Data:       map[string][]byte{"ready": []byte("true")},
	}
	dependency.Set(resolvedSecret)

	stored, ok := dependency.Get().(*corev1.Secret)
	if !ok {
		t.Fatalf("unexpected dependency type: %T", dependency.Get())
	}
	if stored.Name != "backend-secret" {
		t.Fatalf("expected stored dependency name to be copied, got %s", stored.Name)
	}
	if !dependency.IsReady() {
		t.Fatal("expected dependency to be ready")
	}
	if !dependency.IsOptional() {
		t.Fatal("expected dependency to be optional")
	}
	if !dependency.ShouldWaitForReady() {
		t.Fatal("expected dependency to wait for readiness")
	}
	if !dependency.ShouldAddManagedByAnnotation() {
		t.Fatal("expected dependency to request managed-by annotation")
	}

	if err := dependency.BeforeReconcile(ctx); err != nil {
		t.Fatalf("before reconcile failed: %v", err)
	}
	if beforeCalls != 1 {
		t.Fatalf("unexpected before reconcile call count: %d", beforeCalls)
	}

	if err := dependency.AfterReconcile(ctx, resolvedSecret); err != nil {
		t.Fatalf("after reconcile with typed object failed: %v", err)
	}
	if !afterTyped {
		t.Fatal("expected typed after reconcile hook to receive the resolved secret")
	}

	if err := dependency.AfterReconcile(ctx, &corev1.ConfigMap{}); err != nil {
		t.Fatalf("after reconcile with wrong type failed: %v", err)
	}
	if !afterNil {
		t.Fatal("expected zero-value after reconcile hook for mismatched objects")
	}

	gvk := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}
	untypedDependency := &UntypedDependency[*corev1.ConfigMap, Context[*corev1.ConfigMap]]{
		Dependency: &Dependency[*corev1.ConfigMap, Context[*corev1.ConfigMap], *unstructured.Unstructured]{
			output: &unstructured.Unstructured{},
		},
		gvkF: func() schema.GroupVersionKind {
			return gvk
		},
	}

	untypedCreated, ok := untypedDependency.New().(*unstructured.Unstructured)
	if !ok {
		t.Fatalf("expected unstructured dependency, got %T", untypedDependency.New())
	}
	if untypedCreated.GroupVersionKind() != gvk {
		t.Fatalf("unexpected untyped dependency gvk: %#v", untypedCreated.GroupVersionKind())
	}
	if untypedDependency.Kind() != "UntypedSecret" {
		t.Fatalf("unexpected untyped dependency kind: %s", untypedDependency.Kind())
	}

	resolvedUntyped := &unstructured.Unstructured{}
	resolvedUntyped.SetGroupVersionKind(gvk)
	resolvedUntyped.SetName("external-secret")
	untypedDependency.Set(resolvedUntyped)

	storedUntyped, ok := untypedDependency.Get().(*unstructured.Unstructured)
	if !ok {
		t.Fatalf("unexpected stored untyped dependency type: %T", untypedDependency.Get())
	}
	if storedUntyped.GetName() != "external-secret" {
		t.Fatalf("expected copied untyped dependency name, got %s", storedUntyped.GetName())
	}
	if storedUntyped.GroupVersionKind() != gvk {
		t.Fatalf("unexpected stored untyped dependency gvk: %#v", storedUntyped.GroupVersionKind())
	}
}

func TestManagedByLifecycleAndRequests(t *testing.T) {
	scheme := newFrameworkTestScheme(t)

	controlledBy := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "controller", Namespace: "tests"}}
	dependency := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "database", Namespace: "tests"}}

	changed, err := AddManagedBy(dependency, controlledBy, scheme)
	if err != nil {
		t.Fatalf("add managed-by failed: %v", err)
	}
	if !changed {
		t.Fatal("expected managed-by annotation to be added")
	}

	refs, err := GetManagedBy(dependency)
	if err != nil {
		t.Fatalf("get managed-by failed: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected a single managed-by reference, got %d", len(refs))
	}
	if refs[0].Name != controlledBy.Name || refs[0].Namespace != controlledBy.Namespace || refs[0].GVK.Kind != "ConfigMap" {
		t.Fatalf("unexpected managed-by reference: %#v", refs[0])
	}

	changed, err = AddManagedBy(dependency, controlledBy, scheme)
	if err != nil {
		t.Fatalf("second add managed-by failed: %v", err)
	}
	if changed {
		t.Fatal("managed-by annotation should be idempotent")
	}

	requestsForConfigMap, err := GetManagedByReconcileRequests(controlledBy, scheme)
	if err != nil {
		t.Fatalf("build reconcile request mapper failed: %v", err)
	}
	requests := requestsForConfigMap(context.Background(), dependency)
	if len(requests) != 1 || requests[0] != (reconcile.Request{NamespacedName: types.NamespacedName{Name: controlledBy.Name, Namespace: controlledBy.Namespace}}) {
		t.Fatalf("unexpected reconcile requests: %#v", requests)
	}

	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "tests"}}
	requestsForService, err := GetManagedByReconcileRequests(service, scheme)
	if err != nil {
		t.Fatalf("build service request mapper failed: %v", err)
	}
	if got := requestsForService(context.Background(), dependency); len(got) != 0 {
		t.Fatalf("expected no reconcile requests for unrelated gvk, got %#v", got)
	}

	changed, err = RemoveManagedBy(dependency, controlledBy, scheme)
	if err != nil {
		t.Fatalf("remove managed-by failed: %v", err)
	}
	if !changed {
		t.Fatal("expected managed-by annotation to be removed")
	}
	if _, exists := dependency.GetAnnotations()[AnnotationRef]; exists {
		t.Fatalf("managed-by annotation should be removed, got %#v", dependency.GetAnnotations())
	}

	changed, err = RemoveManagedBy(dependency, controlledBy, scheme)
	if err != nil {
		t.Fatalf("second remove managed-by failed: %v", err)
	}
	if changed {
		t.Fatal("removing a missing managed-by reference should be a no-op")
	}

	dependency.SetAnnotations(map[string]string{AnnotationRef: "not-json"})
	if _, err := GetManagedBy(dependency); err == nil {
		t.Fatal("expected invalid annotation JSON to fail")
	}
}

func TestPredicatesHandlePauseAndFinalization(t *testing.T) {
	notPaused := TypedNotPausedPredicate[*corev1.ConfigMap]{}
	active := &corev1.ConfigMap{}
	paused := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{LabelReconciliationPaused: "maintenance"}}}

	if !notPaused.Create(event.TypedCreateEvent[*corev1.ConfigMap]{Object: active}) {
		t.Fatal("active create events should pass the pause predicate")
	}
	if notPaused.Create(event.TypedCreateEvent[*corev1.ConfigMap]{Object: paused}) {
		t.Fatal("paused create events should be filtered")
	}
	if !notPaused.Update(event.TypedUpdateEvent[*corev1.ConfigMap]{ObjectOld: active, ObjectNew: active}) {
		t.Fatal("active update events should pass the pause predicate")
	}
	if notPaused.Update(event.TypedUpdateEvent[*corev1.ConfigMap]{ObjectOld: active, ObjectNew: paused}) {
		t.Fatal("paused update events should be filtered")
	}
	if !notPaused.Generic(event.TypedGenericEvent[*corev1.ConfigMap]{Object: active}) {
		t.Fatal("active generic events should pass the pause predicate")
	}
	if notPaused.Generic(event.TypedGenericEvent[*corev1.ConfigMap]{Object: paused}) {
		t.Fatal("paused generic events should be filtered")
	}
	if !notPaused.Delete(event.TypedDeleteEvent[*corev1.ConfigMap]{Object: paused}) {
		t.Fatal("delete events should always pass the pause predicate")
	}

	finalizing := TypedFinalizingPredicate[*corev1.ConfigMap]{}
	now := metav1.Now()
	finalizingObject := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now}}
	if !finalizing.Create(event.TypedCreateEvent[*corev1.ConfigMap]{Object: finalizingObject}) {
		t.Fatal("objects already marked for deletion should pass finalizing create events")
	}
	if finalizing.Delete(event.TypedDeleteEvent[*corev1.ConfigMap]{Object: finalizingObject}) {
		t.Fatal("delete events should not be used to start finalizer reconciliation")
	}
	if finalizing.Generic(event.TypedGenericEvent[*corev1.ConfigMap]{Object: active}) {
		t.Fatal("non-finalizing generic events should be filtered")
	}
	if !finalizing.Generic(event.TypedGenericEvent[*corev1.ConfigMap]{Object: finalizingObject}) {
		t.Fatal("finalizing generic events should pass")
	}
	if finalizing.Update(event.TypedUpdateEvent[*corev1.ConfigMap]{ObjectOld: finalizingObject, ObjectNew: finalizingObject}) {
		t.Fatal("already-finalizing updates should not be retriggered")
	}
	if !finalizing.Update(event.TypedUpdateEvent[*corev1.ConfigMap]{ObjectOld: active, ObjectNew: finalizingObject}) {
		t.Fatal("entering the finalizing state should pass the predicate")
	}
	if hasDeletionTimestamp(nil) {
		t.Fatal("nil objects should not report a deletion timestamp")
	}
	if isObjectFinalizing(nil) {
		t.Fatal("nil objects should not report finalizing state")
	}
}

func TestUntypedResourceObjectMetaGeneratorSetsGVK(t *testing.T) {
	gvk := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}
	resource := &UntypedResource[*corev1.ConfigMap, Context[*corev1.ConfigMap]]{
		Resource: &Resource[*corev1.ConfigMap, Context[*corev1.ConfigMap], *unstructured.Unstructured]{
			output: &unstructured.Unstructured{Object: map[string]any{}},
			keyF: func() types.NamespacedName {
				return types.NamespacedName{Name: "generated", Namespace: "tests"}
			},
			shouldDeleteF: func() bool { return true },
		},
		gvk: gvk,
	}

	obj, deleteNow, err := resource.ObjectMetaGenerator()
	if err != nil {
		t.Fatalf("object meta generator failed: %v", err)
	}
	if !deleteNow {
		t.Fatal("expected delete-now signal to be propagated")
	}

	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		t.Fatalf("expected unstructured object, got %T", obj)
	}
	if unstructuredObj.GroupVersionKind() != gvk {
		t.Fatalf("unexpected object gvk: %#v", unstructuredObj.GroupVersionKind())
	}
	if resource.Kind() != "UntypedSecret" {
		t.Fatalf("unexpected untyped resource kind: %s", resource.Kind())
	}
}
