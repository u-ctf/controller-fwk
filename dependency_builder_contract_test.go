package ctrlfwk_test

import (
	"testing"

	ctrlfwk "github.com/u-ctf/controller-fwk"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func TestDependencyBuilders_ConfigureOptions(t *testing.T) {
	ctx := newTestContext()
	output := &corev1.Secret{}
	beforeCalled := false
	afterCalled := false

	dep := ctrlfwk.NewDependencyBuilder(ctx, &corev1.Secret{}).
		WithName("database").
		WithNamespace("default").
		WithOutput(output).
		WithOptional(true).
		WithWaitForReady(true).
		WithUserIdentifier("db-secret").
		WithBeforeReconcile(func(*testContext) error {
			beforeCalled = true
			return nil
		}).
		WithAfterReconcile(func(_ *testContext, secret *corev1.Secret) error {
			afterCalled = secret != nil
			return nil
		}).
		WithAddManagedByAnnotation(true).
		WithReadinessCondition(func(secret *corev1.Secret) bool {
			return secret != nil && secret.Data["ready"] != nil
		}).
		Build()

	if got := dep.Key(); got != (types.NamespacedName{Name: "database", Namespace: "default"}) {
		t.Fatalf("unexpected dependency key: %#v", got)
	}
	if got := dep.ID(); got != "db-secret" {
		t.Fatalf("unexpected dependency id: %s", got)
	}
	if !dep.IsOptional() || !dep.ShouldWaitForReady() || !dep.ShouldAddManagedByAnnotation() {
		t.Fatal("expected optional, wait-for-ready, and managed-by flags to be set")
	}

	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "database", Namespace: "default"}, Data: map[string][]byte{"ready": []byte("true")}}
	dep.Set(secret)
	if !dep.IsReady() {
		t.Fatal("expected dependency readiness condition to evaluate to true")
	}
	if output.Name != "database" {
		t.Fatalf("expected output to be populated, got %#v", output)
	}
	if err := dep.BeforeReconcile(ctx); err != nil {
		t.Fatalf("before reconcile failed: %v", err)
	}
	if err := dep.AfterReconcile(ctx, secret); err != nil {
		t.Fatalf("after reconcile failed: %v", err)
	}
	if !beforeCalled || !afterCalled {
		t.Fatalf("expected dependency hooks to run, got before=%v after=%v", beforeCalled, afterCalled)
	}

	static := ctrlfwk.NewDependencyBuilder(ctx, &corev1.Secret{}).
		WithKey(types.NamespacedName{Name: "fixed", Namespace: "other"}).
		Build()
	if got := static.Key(); got != (types.NamespacedName{Name: "fixed", Namespace: "other"}) {
		t.Fatalf("unexpected static key: %#v", got)
	}
}

func TestUntypedDependencyBuilder_ConfigureWrappers(t *testing.T) {
	ctx := newTestContext()
	gvk := schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Widget"}
	output := &unstructured.Unstructured{}
	beforeCalled := false
	afterCalled := false

	dep := ctrlfwk.NewUntypedDependencyBuilder(ctx, gvk).
		WithName("widget").
		WithNamespace("default").
		WithOutput(output).
		WithOptional(true).
		WithWaitForReady(true).
		WithUserIdentifier("widget-dep").
		WithBeforeReconcile(func(*testContext) error {
			beforeCalled = true
			return nil
		}).
		WithAfterReconcile(func(_ *testContext, obj *unstructured.Unstructured) error {
			afterCalled = obj != nil
			return nil
		}).
		WithAddManagedByAnnotation(true).
		WithReadinessCondition(func(obj *unstructured.Unstructured) bool {
			phase, found, _ := unstructured.NestedString(obj.Object, "status", "phase")
			return found && phase == "Ready"
		}).
		Build()

	obj := &unstructured.Unstructured{Object: map[string]any{"status": map[string]any{"phase": "Ready"}}}
	obj.SetGroupVersionKind(gvk)
	obj.SetName("widget")
	obj.SetNamespace("default")
	dep.Set(obj)

	if got := dep.Kind(); got != "UntypedWidget" {
		t.Fatalf("unexpected untyped dependency kind: %s", got)
	}
	if got := dep.ID(); got != "widget-dep" {
		t.Fatalf("unexpected untyped dependency id: %s", got)
	}
	if !dep.IsOptional() || !dep.ShouldWaitForReady() || !dep.ShouldAddManagedByAnnotation() || !dep.IsReady() {
		t.Fatal("expected all untyped dependency flags to be configured")
	}
	if output.GetName() != "widget" || output.GroupVersionKind() != gvk {
		t.Fatalf("expected untyped output to be populated, got %s %#v", output.GetName(), output.GroupVersionKind())
	}
	if err := dep.BeforeReconcile(ctx); err != nil {
		t.Fatalf("before reconcile failed: %v", err)
	}
	if err := dep.AfterReconcile(ctx, obj); err != nil {
		t.Fatalf("after reconcile failed: %v", err)
	}
	if !beforeCalled || !afterCalled {
		t.Fatalf("expected untyped dependency hooks to run, got before=%v after=%v", beforeCalled, afterCalled)
	}

	static := ctrlfwk.NewUntypedDependencyBuilder(ctx, gvk).
		WithKey(types.NamespacedName{Name: "fixed", Namespace: "other"}).
		Build()
	if got := static.Key(); got != (types.NamespacedName{Name: "fixed", Namespace: "other"}) {
		t.Fatalf("unexpected untyped static key: %#v", got)
	}
}

func TestGetContract_ErrorPaths(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{}}
	if _, err := ctrlfwk.GetContract[ExampleObjectContract](obj, "contract"); err == nil {
		t.Fatal("expected missing contract path to return an error")
	}

	obj.Object = map[string]any{
		"status": map[string]any{
			"contract": map[string]any{
				"test2": "ok",
				"time":  true,
			},
		},
	}
	if _, err := ctrlfwk.GetContract[ExampleObjectContract](obj, "contract"); err == nil {
		t.Fatal("expected invalid time type to return an error")
	}

	obj.Object["status"].(map[string]any)["contract"].(map[string]any)["time"] = "not-a-time"
	if _, err := ctrlfwk.GetContract[ExampleObjectContract](obj, "contract"); err == nil {
		t.Fatal("expected invalid RFC3339 string to return an error")
	}
}
