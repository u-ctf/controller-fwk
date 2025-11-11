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

// UntypedTestReconciler reconciles a UntypedTest object
type UntypedTestReconciler struct {
	client.Client
	ctrlfwk.WatchCache
	instrument.Instrumenter
	record.EventRecorder

	RuntimeScheme *runtime.Scheme
}

func (UntypedTestReconciler) For(*testv1.UntypedTest) {}

var _ ctrlfwk.Reconciler[*testv1.UntypedTest] = &UntypedTestReconciler{}
var _ ctrlfwk.ReconcilerWithDependencies[*testv1.UntypedTest, testv1.UntypedTestContext] = &UntypedTestReconciler{}
var _ ctrlfwk.ReconcilerWithResources[*testv1.UntypedTest, testv1.UntypedTestContext] = &UntypedTestReconciler{}
var _ ctrlfwk.ReconcilerWithWatcher[*testv1.UntypedTest] = &UntypedTestReconciler{}

// +kubebuilder:rbac:groups=test.example.com,resources=untypedtests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=test.example.com,resources=untypedtests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=test.example.com,resources=untypedtests/finalizers,verbs=update

func (reconciler *UntypedTestReconciler) GetDependencies(ctx testv1.UntypedTestContext, req ctrl.Request) (dependencies []testv1.UntypedTestDependency, err error) {
	return []testv1.UntypedTestDependency{
		test_dependencies.NewUntypedSecretDependency(ctx, reconciler),
	}, nil
}

func (reconciler *UntypedTestReconciler) GetResources(ctx testv1.UntypedTestContext, req ctrl.Request) ([]testv1.UntypedTestResource, error) {
	return []testv1.UntypedTestResource{
		test_resources.NewUntypedConfigMapResource(ctx, reconciler),
	}, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the UntypedTest object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (reconciler *UntypedTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	context := ctrlfwk.NewContext(ctx, reconciler)

	stepper := ctrlfwk.NewStepperFor(context, logger).
		WithStep(ctrlfwk.NewFindControllerCustomResourceStep(context, reconciler)).
		WithStep(ctrlfwk.NewResolveDynamicDependenciesStep(context, reconciler)).
		WithStep(ctrlfwk.NewReconcileResourcesStep(context, reconciler)).
		WithStep(ctrlfwk.NewEndStep(context, reconciler, ctrlfwk.SetReadyCondition(reconciler))).
		Build()

	return stepper.Execute(context, req)
}

// SetupWithManager sets up the controller with the Manager.
func (reconciler *UntypedTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrler, err := instrument.InstrumentedControllerManagedBy(reconciler, mgr).
		For(&testv1.UntypedTest{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("untypedtest").
		Build(reconciler)

	reconciler.WatchCache.SetController(ctrler)
	return err
}
