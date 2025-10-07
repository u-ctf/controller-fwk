package library

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewReconcileChildrenStep[
	ControllerResourceType ControllerResource,
](
	reconciler ReconcilerWithDynamicChildren[ControllerResourceType],
) Step {
	return Step{
		Name: StepReconcileChildren,
		Step: func(ctx context.Context, req ctrl.Request) StepResult {
			controller := reconciler.GetCustomResource()
			controllerStatus := controller.GetStatus()

			children, err := reconciler.GetChildren(ctx, req)
			if err != nil {
				return ResultInError(errors.Wrap(err, "failed to get children"))
			}

			var returnResults []StepResult
			var newChildrenRefs ObjectReferenceList

			for _, child := range children {
				subStep := NewReconcileChildStep(reconciler, child)
				result := subStep.Step(ctx, req)
				if result.ShouldReturn() {
					returnResults = append(returnResults, result)
					continue
				}

				output := child.Get()
				outputRef, err := EmptyObjectReference(reconciler, output)
				if err != nil {
					return ResultInError(errors.Wrap(err, "failed to create child resource ref"))
				}
				newChildrenRefs.Set(outputRef)
			}

			// Return result errors first
			for _, result := range returnResults {
				if result.err != nil {
					return result
				}
			}

			for _, result := range returnResults {
				if result.ShouldReturn() {
					return result
				}
			}

			missingItems := getItemsMissingFrom(newChildrenRefs, controllerStatus.ChildResources)
			for _, item := range missingItems {
				// Get the item from the cluster
				var object unstructured.Unstructured
				object.SetGroupVersionKind(item.GroupVersionKind())
				var key = types.NamespacedName{
					Name:      item.Name,
					Namespace: item.Namespace,
				}
				err := reconciler.Get(ctx, key, &object)
				if err != nil {
					// Ignore not found errors
					if client.IgnoreNotFound(err) != nil {
						return ResultInError(errors.Wrap(err, "failed to get child resource"))
					}
				}

				if err == nil {
					if err := reconciler.Delete(ctx, &object); err != nil {
						return ResultInError(err)
					}
				}

				// Remove the item from the status
				changed := controllerStatus.ChildResources.Remove(&item)
				if changed {
					if err := reconciler.Status().Update(ctx, controller); err != nil {
						return ResultInError(errors.Wrap(err, "failed to update status"))
					}
				}
			}

			return ResultSuccess()
		},
	}
}
