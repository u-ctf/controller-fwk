package ctrlfwk

import (
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SetupWatch[
	ControllerResourceType ControllerCustomResource,
](
	reconciler ReconcilerWithWatcher[ControllerResourceType],
	object client.Object,
	isDependency bool,
) func(ctx context.Context, req ctrl.Request) StepResult {
	return func(ctx context.Context, req ctrl.Request) StepResult {
		// Setup watch if not already set
		var partialObject metav1.PartialObjectMetadata
		var partialObjectInterface client.Object = &partialObject
		partialObject.SetGroupVersionKind(object.GetObjectKind().GroupVersionKind())

		watchSource := NewWatchKey(partialObjectInterface, CacheTypeEnqueueForOwner)
		if !reconciler.IsWatchingSource(watchSource) {
			requestHandler := handler.EnqueueRequestForOwner(reconciler.GetScheme(), reconciler.GetRESTMapper(), reconciler.GetCustomResource())
			if isDependency {
				managedByHandler, err := GetManagedByReconcileRequests(reconciler.GetCustomResource(), reconciler.GetScheme())
				if err != nil {
					return ResultInError(errors.Wrap(err, "failed to add watch source"))
				}

				requestHandler = handler.EnqueueRequestsFromMapFunc(managedByHandler)
			}

			// Add the watch source to the reconciler
			err := reconciler.GetController().Watch(
				source.Kind(
					reconciler.GetCache(),
					partialObjectInterface,
					requestHandler,
					ResourceChangedPredicate{},
				),
			)
			if err != nil {
				return ResultInError(errors.Wrap(err, "failed to add watch source"))
			}

			reconciler.AddWatchSource(watchSource)
		}

		return ResultSuccess()
	}
}

type ResourceChangedPredicate struct {
	predicate.Funcs
}

func (ResourceChangedPredicate) Update(e event.UpdateEvent) bool {
	return e.ObjectOld.GetResourceVersion() != e.ObjectNew.GetResourceVersion()
}

func (ResourceChangedPredicate) Create(e event.CreateEvent) bool {
	return false
}

func (ResourceChangedPredicate) Delete(e event.DeleteEvent) bool {
	return true
}

func (ResourceChangedPredicate) Generic(e event.GenericEvent) bool {
	return true
}
