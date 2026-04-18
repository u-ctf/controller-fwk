package ctrlfwk_test

import (
	"errors"
	"testing"

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

func TestResourceBuilder_ConfiguresBuiltResource(t *testing.T) {
	ctx := newTestContext()
	output := &corev1.ConfigMap{}

	beforeCalled := false
	afterCalled := false
	createdCalled := false
	updatedCalled := false
	deletedCalled := false
	finalizedCalled := false

	resource := ctrlfwk.NewResourceBuilder(ctx, &corev1.ConfigMap{}).
		WithKey(types.NamespacedName{Name: "managed", Namespace: "default"}).
		WithOutput(output).
		WithMutator(func(cm *corev1.ConfigMap) error {
			cm.Data = map[string]string{"ready": "true"}
			return nil
		}).
		WithReadinessCondition(func(cm *corev1.ConfigMap) bool {
			return cm != nil && cm.Data["ready"] == "true"
		}).
		WithSkipAndDeleteOnCondition(func() bool { return true }).
		WithRequireManualDeletionForFinalize(func(cm *corev1.ConfigMap) bool {
			return cm != nil && cm.Name == "managed"
		}).
		WithBeforeReconcile(func(*testContext) error {
			beforeCalled = true
			return nil
		}).
		WithAfterReconcile(func(_ *testContext, resource *corev1.ConfigMap) error {
			afterCalled = resource != nil
			return nil
		}).
		WithAfterCreate(func(_ *testContext, resource *corev1.ConfigMap) error {
			createdCalled = resource != nil
			return nil
		}).
		WithAfterUpdate(func(_ *testContext, resource *corev1.ConfigMap) error {
			updatedCalled = resource != nil
			return nil
		}).
		WithAfterDelete(func(_ *testContext, resource *corev1.ConfigMap) error {
			deletedCalled = resource != nil
			return nil
		}).
		WithAfterFinalize(func(_ *testContext, resource *corev1.ConfigMap) error {
			finalizedCalled = resource != nil
			return nil
		}).
		WithUserIdentifier("resource-id").
		WithCanBePaused(true).
		Build()

	if got := resource.ID(); got != "resource-id" {
		t.Fatalf("unexpected resource id: %s", got)
	}

	obj, deleteNow, err := resource.ObjectMetaGenerator()
	if err != nil {
		t.Fatalf("object meta generation failed: %v", err)
	}
	if !deleteNow {
		t.Fatal("expected delete-now condition to be true")
	}

	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		t.Fatalf("expected ConfigMap object, got %T", obj)
	}
	if cm.Name != "managed" || cm.Namespace != "default" {
		t.Fatalf("unexpected object identity: %s/%s", cm.Namespace, cm.Name)
	}
	if !resource.ShouldDeleteNow() {
		t.Fatal("expected resource to request deletion")
	}
	if !resource.CanBePaused() {
		t.Fatal("expected resource to be pausable")
	}

	if err := resource.BeforeReconcile(ctx); err != nil {
		t.Fatalf("before reconcile failed: %v", err)
	}
	if !beforeCalled {
		t.Fatal("expected before reconcile hook to be called")
	}

	mutator := resource.GetMutator(cm)
	if err := mutator(); err != nil {
		t.Fatalf("mutator failed: %v", err)
	}
	if !resource.IsReady(cm) {
		t.Fatal("expected resource to be ready after mutation")
	}
	if !resource.RequiresManualDeletion(cm) {
		t.Fatal("expected resource to require manual deletion")
	}

	resource.Set(cm)
	stored, ok := resource.Get().(*corev1.ConfigMap)
	if !ok {
		t.Fatalf("unexpected stored resource type: %T", resource.Get())
	}
	if stored.Data["ready"] != "true" {
		t.Fatalf("expected stored resource data to be copied, got %#v", stored.Data)
	}
	if output.Data["ready"] != "true" {
		t.Fatalf("expected output resource to be populated, got %#v", output.Data)
	}

	if err := resource.AfterReconcile(ctx, cm); err != nil {
		t.Fatalf("after reconcile failed: %v", err)
	}
	if err := resource.OnCreate(ctx, cm); err != nil {
		t.Fatalf("after create failed: %v", err)
	}
	if err := resource.OnUpdate(ctx, cm); err != nil {
		t.Fatalf("after update failed: %v", err)
	}
	if err := resource.OnDelete(ctx, cm); err != nil {
		t.Fatalf("after delete failed: %v", err)
	}
	if err := resource.OnFinalize(ctx, cm); err != nil {
		t.Fatalf("after finalize failed: %v", err)
	}
	if !afterCalled || !createdCalled || !updatedCalled || !deletedCalled || !finalizedCalled {
		t.Fatalf("expected all hooks to be called: after=%v create=%v update=%v delete=%v finalize=%v", afterCalled, createdCalled, updatedCalled, deletedCalled, finalizedCalled)
	}
}

func TestResourceMethods_HandleZeroAndMismatchedObjects(t *testing.T) {
	ctx := newTestContext()
	afterCalled := false
	onCreateCalled := false

	resource := ctrlfwk.NewResourceBuilder(ctx, &corev1.ConfigMap{}).
		WithKey(types.NamespacedName{Name: "managed", Namespace: "default"}).
		WithOutput(&corev1.ConfigMap{}).
		WithReadinessCondition(func(cm *corev1.ConfigMap) bool {
			return cm != nil && cm.Name == "ready"
		}).
		WithRequireManualDeletionForFinalize(func(cm *corev1.ConfigMap) bool {
			return cm != nil && cm.Name == "manual"
		}).
		WithAfterReconcile(func(_ *testContext, cm *corev1.ConfigMap) error {
			afterCalled = cm == nil
			return nil
		}).
		WithAfterCreate(func(_ *testContext, cm *corev1.ConfigMap) error {
			onCreateCalled = cm == nil
			return nil
		}).
		Build()

	if resource.IsReady(nil) {
		t.Fatal("expected nil object to be treated as not ready")
	}
	if resource.IsReady(&corev1.Secret{}) {
		t.Fatal("expected mismatched object type to be treated as not ready")
	}
	if !resource.IsReady(&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "ready"}}) {
		t.Fatal("expected typed object to be evaluated for readiness")
	}

	if resource.RequiresManualDeletion(nil) {
		t.Fatal("expected nil object to not require manual deletion")
	}
	if resource.RequiresManualDeletion(&corev1.Secret{}) {
		t.Fatal("expected mismatched object type to not require manual deletion")
	}
	if !resource.RequiresManualDeletion(&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "manual"}}) {
		t.Fatal("expected typed object to be evaluated for manual deletion")
	}

	if err := resource.AfterReconcile(ctx, &corev1.Secret{}); err != nil {
		t.Fatalf("after reconcile with mismatched type failed: %v", err)
	}
	if err := resource.OnCreate(ctx, &corev1.Secret{}); err != nil {
		t.Fatalf("on create with mismatched type failed: %v", err)
	}
	if afterCalled || onCreateCalled {
		t.Fatalf("mismatched object types should not invoke typed hooks, got after=%v create=%v", afterCalled, onCreateCalled)
	}
	if err := resource.AfterReconcile(ctx, nil); err != nil {
		t.Fatalf("after reconcile with nil resource failed: %v", err)
	}
	if err := resource.OnCreate(ctx, nil); err != nil {
		t.Fatalf("on create with nil resource failed: %v", err)
	}
	if !afterCalled || !onCreateCalled {
		t.Fatalf("expected nil-resource hooks to run with zero values, got after=%v create=%v", afterCalled, onCreateCalled)
	}
	if err := resource.GetMutator(nil)(); err != nil {
		t.Fatalf("nil mutator invocation should succeed, got %v", err)
	}
}

type fakeResourcesReconciler struct {
	client.Client
	resources []ctrlfwk.GenericResource[*corev1.ConfigMap, *resourceStepTestContext]
	err       error
}

func (fakeResourcesReconciler) For(*corev1.ConfigMap) {}

func (r fakeResourcesReconciler) GetResources(ctx *resourceStepTestContext, req ctrl.Request) ([]ctrlfwk.GenericResource[*corev1.ConfigMap, *resourceStepTestContext], error) {
	return r.resources, r.err
}

type earlyReturnResource struct{}

func (earlyReturnResource) ID() string                                                   { return "early-return" }
func (earlyReturnResource) ObjectMetaGenerator() (client.Object, bool, error)            { return nil, true, nil }
func (earlyReturnResource) ShouldDeleteNow() bool                                        { return true }
func (earlyReturnResource) GetMutator(client.Object) func() error                        { return func() error { return nil } }
func (earlyReturnResource) Set(client.Object)                                            {}
func (earlyReturnResource) Get() client.Object                                           { return nil }
func (earlyReturnResource) Kind() string                                                 { return "ConfigMap" }
func (earlyReturnResource) IsReady(client.Object) bool                                   { return true }
func (earlyReturnResource) RequiresManualDeletion(client.Object) bool                    { return false }
func (earlyReturnResource) CanBePaused() bool                                            { return false }
func (earlyReturnResource) BeforeReconcile(*resourceStepTestContext) error               { return nil }
func (earlyReturnResource) AfterReconcile(*resourceStepTestContext, client.Object) error { return nil }
func (earlyReturnResource) OnCreate(*resourceStepTestContext, client.Object) error       { return nil }
func (earlyReturnResource) OnUpdate(*resourceStepTestContext, client.Object) error       { return nil }
func (earlyReturnResource) OnDelete(*resourceStepTestContext, client.Object) error       { return nil }
func (earlyReturnResource) OnFinalize(*resourceStepTestContext, client.Object) error     { return nil }

func TestReconcileResourcesStep_HandlesErrorsAndEarlyReturn(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add corev1 scheme: %v", err)
	}

	cr := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "controller", Namespace: "default"}}
	ctx := newResourceStepTestContext()
	ctx.SetCustomResource(cr)

	reconciler := fakeResourcesReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).Build(),
		err:    errors.New("boom"),
	}
	step := ctrlfwk.NewReconcileResourcesStep(ctx, reconciler)
	result := step.Step(ctx, logr.Discard(), ctrl.Request{})
	if result.Error() == nil {
		t.Fatal("expected GetResources error to be returned")
	}

	reconciler.err = nil
	reconciler.resources = []ctrlfwk.GenericResource[*corev1.ConfigMap, *resourceStepTestContext]{
		earlyReturnResource{},
	}

	step = ctrlfwk.NewReconcileResourcesStep(ctx, reconciler)
	result = step.Step(ctx, logr.Discard(), ctrl.Request{})
	if result.ShouldReturn() {
		t.Fatalf("expected delete-now substeps to normalize to success at resources-step level, got %#v", result)
	}
}
