package ctrlfwk_test

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	ctrlfwk "github.com/u-ctf/controller-fwk"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type fakeResourceReconciler struct {
	client.Client
}

func (fakeResourceReconciler) For(*corev1.ConfigMap) {}

type resourceStepTestContext struct {
	context.Context
	ctrlfwk.CustomResource[*corev1.ConfigMap]
}

func newResourceStepTestContext() *resourceStepTestContext {
	ctx := &resourceStepTestContext{Context: context.Background()}
	ctx.SetCustomResource(&corev1.ConfigMap{})
	return ctx
}

func TestNewReconcileResourceStep_SkipsCreateOrPatchWhenManagedResourceIsPaused(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add corev1 scheme: %v", err)
	}

	cr := &corev1.ConfigMap{}
	cr.SetName("controller")
	cr.SetNamespace("default")

	existing := &corev1.ConfigMap{}
	existing.SetName("managed")
	existing.SetNamespace("default")
	existing.SetLabels(map[string]string{ctrlfwk.LabelReconciliationPaused: "true"})
	existing.Data = map[string]string{"key": "original"}

	reconciler := fakeResourceReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr, existing).Build(),
	}

	ctx := newResourceStepTestContext()
	ctx.SetCustomResource(cr)

	mutatorCalled := false
	resource := ctrlfwk.NewResourceBuilder(ctx, &corev1.ConfigMap{}).
		WithKey(types.NamespacedName{Name: "managed", Namespace: "default"}).
		WithCanBePaused(true).
		WithMutator(func(resource *corev1.ConfigMap) error {
			mutatorCalled = true
			resource.Data = map[string]string{"key": "updated"}
			return nil
		}).
		WithReadinessCondition(func(resource *corev1.ConfigMap) bool { return true }).
		Build()

	step := ctrlfwk.NewReconcileResourceStep(ctx, reconciler, resource)
	result := step.Step(ctx, logr.Discard(), ctrl.Request{})
	if result.ShouldReturn() {
		t.Fatalf("expected success result, got %#v", result)
	}
	if mutatorCalled {
		t.Fatal("expected mutator not to be called when managed resource is paused")
	}

	stored := &corev1.ConfigMap{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: "managed", Namespace: "default"}, stored); err != nil {
		t.Fatalf("failed to get stored resource: %v", err)
	}
	if stored.Data["key"] != "original" {
		t.Fatalf("expected ConfigMap data to remain unchanged, got %#v", stored.Data)
	}
	if stored.Labels[ctrlfwk.LabelReconciliationPaused] != "true" {
		t.Fatalf("expected pause label to be preserved, got %#v", stored.Labels)
	}
}
