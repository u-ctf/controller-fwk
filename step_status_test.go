package ctrlfwk_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	ctrlfwk "github.com/u-ctf/controller-fwk"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type readyStatusTestContext struct {
	context.Context
	ctrlfwk.CustomResource[*readyStatusTestObject]
}

type readyStatusReconciler struct {
	client.Client
}

func (readyStatusReconciler) For(*readyStatusTestObject) {}

func newReadyStatusScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	gv := schema.GroupVersion{Group: "tests.ctrlfwk", Version: "v1"}
	scheme.AddKnownTypes(gv, &readyStatusTestObject{})
	metav1.AddToGroupVersion(scheme, gv)
	return scheme
}

func newReadyStatusContext(obj *readyStatusTestObject) *readyStatusTestContext {
	ctx := &readyStatusTestContext{Context: context.Background()}
	ctx.SetCustomResource(obj)
	return ctx
}

func TestNewEndStep_PatchesReadyCondition(t *testing.T) {
	scheme := newReadyStatusScheme(t)
	obj := &readyStatusTestObject{TypeMeta: metav1.TypeMeta{APIVersion: "tests.ctrlfwk/v1", Kind: "ReadyStatusTestObject"}, ObjectMeta: metav1.ObjectMeta{Name: "sample", Namespace: "default", Generation: 9}}
	reconciler := readyStatusReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(obj).WithObjects(obj).Build()}
	ctx := newReadyStatusContext(obj)

	step := ctrlfwk.NewEndStep(ctx, reconciler, ctrlfwk.SetReadyCondition[*readyStatusTestObject](nil))
	result := step.Step(ctx, logr.Discard(), ctrl.Request{})
	if result.ShouldReturn() {
		t.Fatalf("expected successful end step, got %#v", result)
	}

	stored := &readyStatusTestObject{TypeMeta: obj.TypeMeta}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(obj), stored); err != nil {
		t.Fatalf("failed to get stored object: %v", err)
	}
	cond := meta.FindStatusCondition(stored.Status.Conditions, "Ready")
	if cond == nil || cond.Status != metav1.ConditionTrue {
		t.Fatalf("expected Ready=True condition to be patched, got %#v", stored.Status.Conditions)
	}

	result = step.Step(ctx, logr.Discard(), ctrl.Request{})
	if result.ShouldReturn() {
		t.Fatalf("expected unchanged condition to remain successful, got %#v", result)
	}
}

func TestNewEndStep_ReturnsErrorWhenReadySetterFails(t *testing.T) {
	scheme := newReadyStatusScheme(t)
	obj := &readyStatusTestObject{TypeMeta: metav1.TypeMeta{APIVersion: "tests.ctrlfwk/v1", Kind: "ReadyStatusTestObject"}, ObjectMeta: metav1.ObjectMeta{Name: "sample", Namespace: "default"}}
	reconciler := readyStatusReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(obj).WithObjects(obj).Build()}
	ctx := newReadyStatusContext(obj)

	step := ctrlfwk.NewEndStep(ctx, reconciler, func(*readyStatusTestObject) (bool, error) {
		return false, errors.New("boom")
	})
	result := step.Step(ctx, logr.Discard(), ctrl.Request{})
	if result.Error() == nil {
		t.Fatal("expected end step to return an error when the ready setter fails")
	}
}

type findCRReconciler struct {
	client.Client
}

func (findCRReconciler) For(*corev1.ConfigMap) {}

func TestFindControllerCustomResourceStep_HandlesSuccessPauseAndMissing(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add corev1 scheme: %v", err)
	}

	active := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "active", Namespace: "default"}}
	paused := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "paused", Namespace: "default", Labels: map[string]string{ctrlfwk.LabelReconciliationPaused: "true"}}}
	reconciler := findCRReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(active, paused).Build()}

	ctx := newResourceStepTestContext()
	step := ctrlfwk.NewFindControllerCustomResourceStep(ctx, reconciler)

	result := step.Step(ctx, logr.Discard(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(active)})
	if result.ShouldReturn() {
		t.Fatalf("expected active resource lookup to succeed, got %#v", result)
	}
	if got := ctx.GetCustomResource().GetName(); got != "active" {
		t.Fatalf("expected active resource to be stored in context, got %q", got)
	}

	ctx = newResourceStepTestContext()
	step = ctrlfwk.NewFindControllerCustomResourceStep(ctx, reconciler)
	result = step.Step(ctx, logr.Discard(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(paused)})
	if !result.IsEarlyReturn() {
		t.Fatalf("expected paused resource to return early, got %#v", result)
	}

	ctx = newResourceStepTestContext()
	step = ctrlfwk.NewFindControllerCustomResourceStep(ctx, reconciler)
	result = step.Step(ctx, logr.Discard(), ctrl.Request{NamespacedName: client.ObjectKey{Name: "missing", Namespace: "default"}})
	if !result.IsEarlyReturn() {
		t.Fatalf("expected missing resource to return early, got %#v", result)
	}
}
