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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	testv1 "operator/api/v1"
	test_children "operator/internal/controller/test/children"
)

// TestReconciler reconciles a Test object
type TestReconciler struct {
	ctrl.Manager
	client.Client
	ctrlfwk.WatchCache
	ctrlfwk.CustomResource[*testv1.Test]

	RuntimeScheme *runtime.Scheme
}

var _ ctrlfwk.Reconciler[*testv1.Test] = &TestReconciler{}

// +kubebuilder:rbac:groups=test.example.com,resources=tests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=test.example.com,resources=tests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=test.example.com,resources=tests/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

func (reconciler *TestReconciler) GetDependencies(ctx context.Context, req ctrl.Request) (dependencies []ctrlfwk.GenericDependencyResource, err error) {
	return nil, nil
}

func (reconciler *TestReconciler) GetChildren(ctx context.Context, req ctrl.Request) ([]ctrlfwk.GenericChildResource, error) {
	return []ctrlfwk.GenericChildResource{
		test_children.NewConfigMapChild(reconciler),
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

	var newReconciler = &TestReconciler{
		Manager:       reconciler.Manager,
		Client:        reconciler.Client,
		WatchCache:    reconciler.WatchCache,
		RuntimeScheme: reconciler.RuntimeScheme,
	}

	stepper := ctrlfwk.NewStepper(logger,
		ctrlfwk.WithStep(ctrlfwk.NewFindControllerCustomResourceStep(newReconciler)),
		ctrlfwk.WithStep(ctrlfwk.NewResolveDynamicDependenciesStep(newReconciler)),
		ctrlfwk.WithStep(ctrlfwk.NewReconcileChildrenStep(newReconciler)),
		ctrlfwk.WithStep(ctrlfwk.NewEndStep(newReconciler, ctrlfwk.SetReadyCondition(newReconciler))),
	)

	return stepper.Execute(ctx, req)
}

// SetupWithManager sets up the controller with the Manager.
func (reconciler *TestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrler, err := ctrl.NewControllerManagedBy(mgr).
		For(&testv1.Test{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("test").
		Build(reconciler)

	reconciler.WatchCache.SetController(ctrler)

	return err
}
