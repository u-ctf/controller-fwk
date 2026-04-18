package ctrlfwk_test

import (
	"testing"
	"time"

	"github.com/go-logr/logr"
	ctrlfwk "github.com/u-ctf/controller-fwk"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newResourceScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add corev1 scheme: %v", err)
	}
	return scheme
}

func TestNewReconcileResourceStep_CreatesManagedResource(t *testing.T) {
	scheme := newResourceScheme(t)
	cr := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "controller", Namespace: "default"}}
	reconciler := fakeResourceReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).Build()}
	ctx := newResourceStepTestContext()
	ctx.SetCustomResource(cr)

	createdCalled := false
	afterCalled := false
	resource := ctrlfwk.NewResourceBuilder(ctx, &corev1.ConfigMap{}).
		WithKey(types.NamespacedName{Name: "managed", Namespace: "default"}).
		WithMutator(func(resource *corev1.ConfigMap) error {
			resource.Data = map[string]string{"key": "value"}
			return nil
		}).
		WithReadinessCondition(func(resource *corev1.ConfigMap) bool { return resource.Data["key"] == "value" }).
		WithAfterCreate(func(_ *resourceStepTestContext, resource *corev1.ConfigMap) error {
			createdCalled = resource != nil
			return nil
		}).
		WithAfterReconcile(func(_ *resourceStepTestContext, resource *corev1.ConfigMap) error {
			afterCalled = resource != nil
			return nil
		}).
		Build()

	result := ctrlfwk.NewReconcileResourceStep(ctx, reconciler, resource).Step(ctx, logr.Discard(), ctrl.Request{})
	if result.ShouldReturn() {
		t.Fatalf("expected resource create to succeed, got %#v", result)
	}
	if !createdCalled || !afterCalled {
		t.Fatalf("expected create and after hooks to run, got created=%v after=%v", createdCalled, afterCalled)
	}

	stored := &corev1.ConfigMap{}
	if err := reconciler.Get(ctx, client.ObjectKey{Name: "managed", Namespace: "default"}, stored); err != nil {
		t.Fatalf("failed to fetch created resource: %v", err)
	}
	if stored.Data["key"] != "value" {
		t.Fatalf("unexpected created resource data: %#v", stored.Data)
	}
}

func TestNewReconcileResourceStep_UpdatesResourceAndReturnsEarlyWhenUnready(t *testing.T) {
	scheme := newResourceScheme(t)
	cr := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "controller", Namespace: "default"}}
	existing := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "managed", Namespace: "default"}, Data: map[string]string{"key": "old"}}
	reconciler := fakeResourceReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, existing).Build()}
	ctx := newResourceStepTestContext()
	ctx.SetCustomResource(cr)

	updatedCalled := false
	resource := ctrlfwk.NewResourceBuilder(ctx, &corev1.ConfigMap{}).
		WithKey(types.NamespacedName{Name: "managed", Namespace: "default"}).
		WithMutator(func(resource *corev1.ConfigMap) error {
			resource.Data = map[string]string{"key": "new"}
			return nil
		}).
		WithReadinessCondition(func(resource *corev1.ConfigMap) bool { return false }).
		WithAfterUpdate(func(_ *resourceStepTestContext, resource *corev1.ConfigMap) error {
			updatedCalled = resource != nil
			return nil
		}).
		Build()

	result := ctrlfwk.NewReconcileResourceStep(ctx, reconciler, resource).Step(ctx, logr.Discard(), ctrl.Request{})
	if !result.IsEarlyReturn() {
		t.Fatalf("expected unready updated resource to early return, got %#v", result)
	}
	if !updatedCalled {
		t.Fatal("expected update hook to run")
	}

	stored := &corev1.ConfigMap{}
	if err := reconciler.Get(ctx, client.ObjectKey{Name: "managed", Namespace: "default"}, stored); err != nil {
		t.Fatalf("failed to fetch updated resource: %v", err)
	}
	if stored.Data["key"] != "new" {
		t.Fatalf("expected updated data to be stored, got %#v", stored.Data)
	}
}

func TestNewReconcileResourceStep_DeleteNowRunsOnDeleteAndNormalizesResult(t *testing.T) {
	scheme := newResourceScheme(t)
	cr := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "controller", Namespace: "default"}}
	existing := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "managed", Namespace: "default"}}
	reconciler := fakeResourceReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, existing).Build()}
	ctx := newResourceStepTestContext()
	ctx.SetCustomResource(cr)

	deletedCalled := false
	resource := ctrlfwk.NewResourceBuilder(ctx, &corev1.ConfigMap{}).
		WithKey(types.NamespacedName{Name: "managed", Namespace: "default"}).
		WithSkipAndDeleteOnCondition(func() bool { return true }).
		WithAfterDelete(func(_ *resourceStepTestContext, resource *corev1.ConfigMap) error {
			deletedCalled = resource != nil
			return nil
		}).
		Build()

	result := ctrlfwk.NewReconcileResourceStep(ctx, reconciler, resource).Step(ctx, logr.Discard(), ctrl.Request{})
	if result.ShouldReturn() {
		t.Fatalf("expected delete-now branch to normalize to success, got %#v", result)
	}
	if !deletedCalled {
		t.Fatal("expected delete hook to run")
	}

	stored := &corev1.ConfigMap{}
	err := reconciler.Get(ctx, client.ObjectKey{Name: "managed", Namespace: "default"}, stored)
	if client.IgnoreNotFound(err) != nil {
		t.Fatalf("expected resource to be deleted, got error %v", err)
	}
}

func TestNewReconcileResourceStep_FinalizationPaths(t *testing.T) {
	scheme := newResourceScheme(t)
	now := metav1.NewTime(time.Now())

	t.Run("skip manual deletion", func(t *testing.T) {
		cr := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "controller", Namespace: "default", DeletionTimestamp: &now, Finalizers: []string{"tests.ctrlfwk/finalizer"}}}
		reconciler := fakeResourceReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).Build()}
		ctx := newResourceStepTestContext()
		ctx.SetCustomResource(cr)

		finalizedCalled := false
		resource := ctrlfwk.NewResourceBuilder(ctx, &corev1.ConfigMap{}).
			WithAfterFinalize(func(_ *resourceStepTestContext, resource *corev1.ConfigMap) error {
				finalizedCalled = true
				return nil
			}).
			Build()

		result := ctrlfwk.NewReconcileResourceStep(ctx, reconciler, resource).Step(ctx, logr.Discard(), ctrl.Request{})
		if result.ShouldReturn() {
			t.Fatalf("expected finalization without manual deletion to succeed, got %#v", result)
		}
		if !finalizedCalled {
			t.Fatal("expected finalize hook to run")
		}
	})

	t.Run("manual deletion", func(t *testing.T) {
		cr := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "controller", Namespace: "default", DeletionTimestamp: &now, Finalizers: []string{"tests.ctrlfwk/finalizer"}}}
		existing := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "managed", Namespace: "default"}}
		reconciler := fakeResourceReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, existing).Build()}
		ctx := newResourceStepTestContext()
		ctx.SetCustomResource(cr)

		finalizedCalled := false
		resource := ctrlfwk.NewResourceBuilder(ctx, &corev1.ConfigMap{}).
			WithKey(types.NamespacedName{Name: "managed", Namespace: "default"}).
			WithRequireManualDeletionForFinalize(func(*corev1.ConfigMap) bool { return true }).
			WithAfterFinalize(func(_ *resourceStepTestContext, resource *corev1.ConfigMap) error {
				finalizedCalled = resource != nil
				return nil
			}).
			Build()

		result := ctrlfwk.NewReconcileResourceStep(ctx, reconciler, resource).Step(ctx, logr.Discard(), ctrl.Request{})
		if result.ShouldReturn() {
			t.Fatalf("expected finalization with manual deletion to succeed, got %#v", result)
		}
		if !finalizedCalled {
			t.Fatal("expected finalize hook to run after deletion")
		}

		stored := &corev1.ConfigMap{}
		err := reconciler.Get(ctx, client.ObjectKey{Name: "managed", Namespace: "default"}, stored)
		if client.IgnoreNotFound(err) != nil {
			t.Fatalf("expected managed resource to be deleted during finalization, got %v", err)
		}
	})
}
