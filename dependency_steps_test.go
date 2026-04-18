package ctrlfwk_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	ctrlfwk "github.com/u-ctf/controller-fwk"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type dependencyStepTestContext struct {
	context.Context
	ctrlfwk.CustomResource[*corev1.ConfigMap]
}

func newDependencyStepTestContext(cr *corev1.ConfigMap) *dependencyStepTestContext {
	ctx := &dependencyStepTestContext{Context: context.Background()}
	ctx.SetCustomResource(cr)
	return ctx
}

type fakeDependencyReconciler struct {
	client.Client
	dependencies []ctrlfwk.GenericDependency[*corev1.ConfigMap, *dependencyStepTestContext]
	err          error
}

func (fakeDependencyReconciler) For(*corev1.ConfigMap) {}

func (r fakeDependencyReconciler) GetDependencies(ctx *dependencyStepTestContext, req ctrl.Request) ([]ctrlfwk.GenericDependency[*corev1.ConfigMap, *dependencyStepTestContext], error) {
	return r.dependencies, r.err
}

func newDependencyScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add corev1 scheme: %v", err)
	}

	return scheme
}

func TestResolveDynamicDependenciesStep_AddsFinalizerAndManagedByForReadyDependency(t *testing.T) {
	scheme := newDependencyScheme(t)

	cr := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "controller", Namespace: "default"}}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "backend", Namespace: "default"},
		Data:       map[string][]byte{"ready": []byte("true")},
	}

	reconciler := fakeDependencyReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, secret).Build(),
	}
	ctx := newDependencyStepTestContext(cr)

	dependency := ctrlfwk.NewDependencyBuilder(ctx, &corev1.Secret{}).
		WithName(secret.Name).
		WithNamespace(secret.Namespace).
		WithWaitForReady(true).
		WithAddManagedByAnnotation(true).
		WithIsReadyFunc(func(obj *corev1.Secret) bool {
			return obj != nil && obj.Data["ready"] != nil
		}).
		Build()
	reconciler.dependencies = []ctrlfwk.GenericDependency[*corev1.ConfigMap, *dependencyStepTestContext]{dependency}

	step := ctrlfwk.NewResolveDynamicDependenciesStep(ctx, reconciler)
	result := step.Step(ctx, logr.Discard(), ctrl.Request{})
	if result.ShouldReturn() {
		t.Fatalf("expected success result, got %#v", result)
	}

	storedCR := &corev1.ConfigMap{}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(cr), storedCR); err != nil {
		t.Fatalf("failed to get reconciled CR: %v", err)
	}
	if len(storedCR.Finalizers) != 1 || storedCR.Finalizers[0] != ctrlfwk.FinalizerDependenciesManagedBy {
		t.Fatalf("expected dependency cleanup finalizer to be added, got %#v", storedCR.Finalizers)
	}

	storedSecret := &corev1.Secret{}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(secret), storedSecret); err != nil {
		t.Fatalf("failed to get dependency secret: %v", err)
	}
	managedBy, err := ctrlfwk.GetManagedBy(storedSecret)
	if err != nil {
		t.Fatalf("failed to read managed-by annotation: %v", err)
	}
	if len(managedBy) != 1 {
		t.Fatalf("expected one managed-by reference, got %#v", managedBy)
	}
	if managedBy[0].Name != cr.Name || managedBy[0].Namespace != cr.Namespace || managedBy[0].GVK.Kind != "ConfigMap" {
		t.Fatalf("unexpected managed-by reference: %#v", managedBy[0])
	}
	if dependency.Get() == nil {
		t.Fatal("expected resolved dependency to be stored on the dependency object")
	}
}

func TestResolveDynamicDependenciesStep_FinalizingRemovesManagedByAndFinalizer(t *testing.T) {
	scheme := newDependencyScheme(t)
	now := metav1.Now()
	cr := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "controller",
			Namespace:         "default",
			Finalizers:        []string{ctrlfwk.FinalizerDependenciesManagedBy},
			DeletionTimestamp: &now,
		},
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "backend", Namespace: "default"}}
	if changed, err := ctrlfwk.AddManagedBy(secret, cr, scheme); err != nil || !changed {
		t.Fatalf("failed to seed managed-by annotation: changed=%v err=%v", changed, err)
	}

	reconciler := fakeDependencyReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, secret).Build(),
	}
	ctx := newDependencyStepTestContext(cr)

	dependency := ctrlfwk.NewDependencyBuilder(ctx, &corev1.Secret{}).
		WithName(secret.Name).
		WithNamespace(secret.Namespace).
		WithAddManagedByAnnotation(true).
		Build()
	reconciler.dependencies = []ctrlfwk.GenericDependency[*corev1.ConfigMap, *dependencyStepTestContext]{dependency}

	step := ctrlfwk.NewResolveDynamicDependenciesStep(ctx, reconciler)
	result := step.Step(ctx, logr.Discard(), ctrl.Request{})
	if result.ShouldReturn() {
		t.Fatalf("expected success result while finalizing, got %#v", result)
	}

	if len(ctx.GetCustomResource().Finalizers) != 0 {
		t.Fatalf("expected dependency cleanup finalizer to be removed, got %#v", ctx.GetCustomResource().Finalizers)
	}

	storedSecret := &corev1.Secret{}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(secret), storedSecret); err != nil {
		t.Fatalf("failed to get dependency secret: %v", err)
	}
	managedBy, err := ctrlfwk.GetManagedBy(storedSecret)
	if err != nil {
		t.Fatalf("failed to read managed-by annotation after finalization: %v", err)
	}
	if len(managedBy) != 0 {
		t.Fatalf("expected managed-by references to be removed, got %#v", managedBy)
	}
}

func TestResolveDependencyStep_RequeuesWhenRequiredDependencyIsMissing(t *testing.T) {
	scheme := newDependencyScheme(t)
	cr := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "controller", Namespace: "default"}}
	reconciler := fakeDependencyReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).Build(),
	}
	ctx := newDependencyStepTestContext(cr)

	afterCalled := false
	dependency := ctrlfwk.NewDependencyBuilder(ctx, &corev1.Secret{}).
		WithName("missing").
		WithNamespace("default").
		WithWaitForReady(true).
		WithAfterReconcile(func(ctx *dependencyStepTestContext, resource *corev1.Secret) error {
			afterCalled = true
			if resource == nil {
				t.Fatal("expected zero-value typed resource in AfterReconcile for missing dependency")
			}
			if resource.Name != "" || resource.Namespace != "" || len(resource.Data) != 0 {
				t.Fatalf("expected empty resource in AfterReconcile for missing dependency, got %#v", resource)
			}
			return nil
		}).
		Build()

	step := ctrlfwk.NewResolveDependencyStep(ctx, reconciler, dependency)
	result := step.Step(ctx, logr.Discard(), ctrl.Request{})
	if result.Error() != nil {
		t.Fatalf("expected requeue result, got error %v", result.Error())
	}
	if result.RequeueAfter() != 30*time.Second {
		t.Fatalf("expected 30s requeue, got %s", result.RequeueAfter())
	}
	if !afterCalled {
		t.Fatal("expected AfterReconcile to be called for missing dependencies")
	}
}

func TestWatchCacheAndResourceVersionPredicate(t *testing.T) {
	watchCache := ctrlfwk.NewWatchCache(nil)
	key := ctrlfwk.NewWatchKey(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}, ctrlfwk.CacheTypeEnqueueForOwner)

	if watchCache.IsWatchingSource(key) {
		t.Fatal("expected watch source cache to be empty initially")
	}
	watchCache.AddWatchSource(key)
	if !watchCache.IsWatchingSource(key) {
		t.Fatal("expected added watch source to be cached")
	}
	watchCache.SetController(nil)
	if watchCache.GetController() != nil {
		t.Fatal("expected nil controller to round-trip through watch cache")
	}

	predicate := ctrlfwk.ResourceVersionChangedPredicate{}
	oldObject := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "1"}}
	newObject := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "2"}}
	if !predicate.Update(event.UpdateEvent{ObjectOld: oldObject, ObjectNew: newObject}) {
		t.Fatal("expected update events with different resource versions to pass")
	}
	if predicate.Update(event.UpdateEvent{ObjectOld: newObject, ObjectNew: newObject.DeepCopy()}) {
		t.Fatal("expected update events with identical resource versions to be filtered")
	}
	if predicate.Create(event.CreateEvent{Object: newObject}) {
		t.Fatal("expected create events to be filtered")
	}
	if !predicate.Delete(event.DeleteEvent{Object: newObject}) {
		t.Fatal("expected delete events to pass")
	}
	if !predicate.Generic(event.GenericEvent{Object: newObject}) {
		t.Fatal("expected generic events to pass")
	}
	if key != ctrlfwk.WatchCacheKey("/v1, Kind=Secret/enqueueForOwner") {
		t.Fatalf("unexpected watch cache key: %s", key)
	}
}
