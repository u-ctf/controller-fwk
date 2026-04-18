package ctrlfwk_test

import (
	"testing"

	ctrlfwk "github.com/u-ctf/controller-fwk"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func TestUntypedResourceBuilder_ConfiguresWrappers(t *testing.T) {
	ctx := newTestContext()
	gvk := schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Widget"}
	output := &unstructured.Unstructured{}
	beforeCalled := false
	afterCalled := false
	createdCalled := false
	updatedCalled := false
	deletedCalled := false
	finalizedCalled := false

	resource := ctrlfwk.NewUntypedResourceBuilder(ctx, gvk).
		WithKeyFunc(func() types.NamespacedName {
			return types.NamespacedName{Name: "widget", Namespace: "default"}
		}).
		WithOutput(output).
		WithMutator(func(obj *unstructured.Unstructured) error {
			obj.Object = map[string]any{"status": map[string]any{"phase": "Ready"}}
			return nil
		}).
		WithReadinessCondition(func(obj *unstructured.Unstructured) bool { return obj != nil }).
		WithSkipAndDeleteOnCondition(func() bool { return true }).
		WithRequireManualDeletionForFinalize(func(obj *unstructured.Unstructured) bool { return obj != nil }).
		WithBeforeReconcile(func(*testContext) error {
			beforeCalled = true
			return nil
		}).
		WithAfterReconcile(func(_ *testContext, obj *unstructured.Unstructured) error {
			afterCalled = obj != nil
			return nil
		}).
		WithAfterCreate(func(_ *testContext, obj *unstructured.Unstructured) error {
			createdCalled = obj != nil
			return nil
		}).
		WithAfterUpdate(func(_ *testContext, obj *unstructured.Unstructured) error {
			updatedCalled = obj != nil
			return nil
		}).
		WithAfterDelete(func(_ *testContext, obj *unstructured.Unstructured) error {
			deletedCalled = obj != nil
			return nil
		}).
		WithAfterFinalize(func(_ *testContext, obj *unstructured.Unstructured) error {
			finalizedCalled = obj != nil
			return nil
		}).
		WithUserIdentifier("widget-resource").
		WithCanBePaused(true).
		WithCanBePausedFunc(func() bool { return true }).
		Build()

	if got := resource.Kind(); got != "UntypedWidget" {
		t.Fatalf("unexpected untyped resource kind: %s", got)
	}
	if got := resource.ID(); got != "widget-resource" {
		t.Fatalf("unexpected untyped resource id: %s", got)
	}
	if !resource.CanBePaused() || !resource.ShouldDeleteNow() {
		t.Fatal("expected untyped resource pause and delete flags to be configured")
	}

	obj, deleteNow, err := resource.ObjectMetaGenerator()
	if err != nil {
		t.Fatalf("untyped object meta generation failed: %v", err)
	}
	if !deleteNow {
		t.Fatal("expected untyped resource to request deletion")
	}
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		t.Fatalf("expected unstructured object, got %T", obj)
	}
	if unstructuredObj.GroupVersionKind() != gvk {
		t.Fatalf("unexpected gvk on untyped object: %#v", unstructuredObj.GroupVersionKind())
	}

	reconciled := &unstructured.Unstructured{}
	reconciled.SetGroupVersionKind(gvk)
	reconciled.SetName("widget")
	reconciled.SetNamespace("default")
	reconciled.Object = map[string]any{"status": map[string]any{"phase": "Ready"}}

	resource.Set(reconciled)
	if !resource.IsReady(reconciled) || !resource.RequiresManualDeletion(reconciled) {
		t.Fatal("expected untyped resource readiness and manual deletion checks to succeed")
	}
	if got := resource.Get(); got == nil {
		t.Fatalf("expected untyped resource get to return the reconciled object, got %#v", got)
	}
	if output == nil {
		t.Fatal("expected configured untyped output target to be retained")
	}

	if err := resource.BeforeReconcile(ctx); err != nil {
		t.Fatalf("before reconcile failed: %v", err)
	}
	if err := resource.AfterReconcile(ctx, reconciled); err != nil {
		t.Fatalf("after reconcile failed: %v", err)
	}
	if err := resource.OnCreate(ctx, reconciled); err != nil {
		t.Fatalf("after create failed: %v", err)
	}
	if err := resource.OnUpdate(ctx, reconciled); err != nil {
		t.Fatalf("after update failed: %v", err)
	}
	if err := resource.OnDelete(ctx, reconciled); err != nil {
		t.Fatalf("after delete failed: %v", err)
	}
	if err := resource.OnFinalize(ctx, reconciled); err != nil {
		t.Fatalf("after finalize failed: %v", err)
	}
	if !beforeCalled || !afterCalled || !createdCalled || !updatedCalled || !deletedCalled || !finalizedCalled {
		t.Fatalf("expected all untyped hooks to run: before=%v after=%v create=%v update=%v delete=%v finalize=%v", beforeCalled, afterCalled, createdCalled, updatedCalled, deletedCalled, finalizedCalled)
	}

	static := ctrlfwk.NewUntypedResourceBuilder(ctx, gvk).
		WithKey(types.NamespacedName{Name: "fixed", Namespace: "other"}).
		Build()
	staticObj, _, err := static.ObjectMetaGenerator()
	if err != nil {
		t.Fatalf("static untyped object generation failed: %v", err)
	}
	if got := staticObj.(*unstructured.Unstructured); got.GetName() != "fixed" || got.GetNamespace() != "other" {
		t.Fatalf("unexpected static untyped key: %s/%s", got.GetNamespace(), got.GetName())
	}
}
