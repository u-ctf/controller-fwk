/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	ctrlfwk "github.com/u-ctf/controller-fwk"
	"github.com/u-ctf/controller-fwk/instrument"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	testv1 "operator/api/v1"
	test_dependencies "operator/internal/controller/test/dependencies"
	test_resources "operator/internal/controller/test/resources"
)

// TestReconciler reconciles a Test object
type TestReconciler struct {
	client.Client
	ctrlfwk.WatchCache
	instrument.Instrumenter
	record.EventRecorder

	RuntimeScheme *runtime.Scheme
}

func (TestReconciler) For(*testv1.Test) {}

var _ ctrlfwk.Reconciler[*testv1.Test] = &TestReconciler{}
var _ ctrlfwk.ReconcilerWithDependencies[*testv1.Test, *ctrlfwk.ContextWithData[*testv1.Test, int]] = &TestReconciler{}
var _ ctrlfwk.ReconcilerWithResources[*testv1.Test, *ctrlfwk.ContextWithData[*testv1.Test, int]] = &TestReconciler{}
var _ ctrlfwk.ReconcilerWithWatcher[*testv1.Test] = &TestReconciler{}

// +kubebuilder:rbac:groups=test.example.com,resources=tests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=test.example.com,resources=tests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=test.example.com,resources=tests/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;patch

func (reconciler *TestReconciler) GetDependencies(ctx *ctrlfwk.ContextWithData[*testv1.Test, int], req ctrl.Request) (dependencies []ctrlfwk.GenericDependency[*testv1.Test, *ctrlfwk.ContextWithData[*testv1.Test, int]], err error) {
	return []ctrlfwk.GenericDependency[*testv1.Test, *ctrlfwk.ContextWithData[*testv1.Test, int]]{
		test_dependencies.NewSecretDependency(ctx, reconciler),
	}, nil
}

func (reconciler *TestReconciler) GetResources(ctx *ctrlfwk.ContextWithData[*testv1.Test, int], req ctrl.Request) ([]ctrlfwk.GenericResource[*testv1.Test, *ctrlfwk.ContextWithData[*testv1.Test, int]], error) {
	return []ctrlfwk.GenericResource[*testv1.Test, *ctrlfwk.ContextWithData[*testv1.Test, int]]{
		test_resources.NewConfigMapResource(ctx, reconciler),
	}, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Test object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (reconciler *TestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	context := ctrlfwk.NewContextWithData(ctx, reconciler, 12)

	stepper := ctrlfwk.NewStepperFor(context, logger).
		WithStep(ctrlfwk.NewFindControllerCustomResourceStep(context, reconciler)).
		WithStep(ctrlfwk.NewResolveDynamicDependenciesStep(context, reconciler)).
		WithStep(ctrlfwk.NewReconcileResourcesStep(context, reconciler)).
		WithStep(ctrlfwk.NewEndStep(context, reconciler, ctrlfwk.SetReadyCondition(reconciler))).
		Build()

	return stepper.Execute(context, req)
}

// SetupWithManager sets up the controller with the Manager.
func (reconciler *TestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrler, err := instrument.InstrumentedControllerManagedBy(reconciler, mgr).
		For(&testv1.Test{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("test").
		Build(reconciler)

	reconciler.WatchCache.SetController(ctrler)
	return err
}
