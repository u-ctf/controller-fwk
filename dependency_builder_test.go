package ctrlfwk_test

import (
	"context"
	"testing"

	ctrlfwk "github.com/u-ctf/controller-fwk"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

type testContext struct {
	context.Context
	ctrlfwk.CustomResource[*unstructured.Unstructured]
}

func newTestContext() *testContext {
	return &testContext{Context: context.Background()}
}

func TestDependencyBuilder_WithKeyFunc_UsesDynamicKey(t *testing.T) {
	ctx := newTestContext()

	name := "dep-a"
	namespace := "ns-a"
	dep := ctrlfwk.NewDependencyBuilder(ctx, &corev1.ConfigMap{}).
		WithKeyFunc(func() types.NamespacedName {
			return types.NamespacedName{Name: name, Namespace: namespace}
		}).
		Build()

	firstKey := dep.Key()
	if firstKey.Name != "dep-a" || firstKey.Namespace != "ns-a" {
		t.Fatalf("unexpected first key: got %s/%s", firstKey.Namespace, firstKey.Name)
	}

	firstID := dep.ID()

	name = "dep-b"
	namespace = "ns-b"

	secondKey := dep.Key()
	if secondKey.Name != "dep-b" || secondKey.Namespace != "ns-b" {
		t.Fatalf("unexpected second key: got %s/%s", secondKey.Namespace, secondKey.Name)
	}

	secondID := dep.ID()
	if firstID == secondID {
		t.Fatalf("expected ID to change when key changes, got same ID: %s", firstID)
	}
}

func TestUntypedDependencyBuilder_WithKeyFunc_UsesDynamicKey(t *testing.T) {
	ctx := newTestContext()

	gvk := schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Widget"}
	name := "widget-a"
	namespace := "widgets"

	dep := ctrlfwk.NewUntypedDependencyBuilder(ctx, gvk).
		WithKeyFunc(func() types.NamespacedName {
			return types.NamespacedName{Name: name, Namespace: namespace}
		}).
		Build()

	firstKey := dep.Key()
	if firstKey.Name != "widget-a" || firstKey.Namespace != "widgets" {
		t.Fatalf("unexpected first key: got %s/%s", firstKey.Namespace, firstKey.Name)
	}

	firstID := dep.ID()

	name = "widget-b"
	namespace = "widgets-2"

	secondKey := dep.Key()
	if secondKey.Name != "widget-b" || secondKey.Namespace != "widgets-2" {
		t.Fatalf("unexpected second key: got %s/%s", secondKey.Namespace, secondKey.Name)
	}

	secondID := dep.ID()
	if firstID == secondID {
		t.Fatalf("expected ID to change when key changes, got same ID: %s", firstID)
	}
}
