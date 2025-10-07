package library

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func NewResolveDynamicDependenciesStep[
	ControllerResourceType ControllerResource,
](
	reconciler ReconcilerWithDynamicDependencies[ControllerResourceType],
) Step {
	return Step{
		Name: StepResolveDependencies,
		Step: func(ctx context.Context, req ctrl.Request) StepResult {
			controller := reconciler.GetCustomResource()
			controllerStatus := controller.GetStatus()

			dependencies, err := reconciler.GetDependencies(ctx, req)
			if err != nil {
				return ResultInError(errors.Wrap(err, "failed to get dependencies"))
			}

			var returnResults []StepResult
			var newDependenciesRef ObjectReferenceList

			for _, dependency := range dependencies {
				subStep := newResolveDependencyStep(reconciler, dependency)
				result := subStep.Step(ctx, req)
				if result.ShouldReturn() {
					returnResults = append(returnResults, result)
					continue
				}

				if isFinalizing(reconciler) {
					continue
				}

				output := dependency.Get()
				outputRef, err := EmptyObjectReference(reconciler, output)
				if err != nil {
					return ResultInError(errors.Wrap(err, "failed to create dependency resource ref"))
				}
				newDependenciesRef.Set(outputRef)
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

			missingItems := getItemsMissingFrom(newDependenciesRef, controllerStatus.Dependencies)
			for _, item := range missingItems {
				// Get the item from the cluster
				var object unstructured.Unstructured
				object.SetAPIVersion(item.GroupVersionKind().GroupVersion().String())
				object.SetKind(item.GroupVersionKind().Kind)
				var key = types.NamespacedName{
					Name:      item.Name,
					Namespace: item.Namespace,
				}
				err := reconciler.Get(ctx, key, &object)
				if err != nil {
					// Ignore not found errors
					if client.IgnoreNotFound(err) != nil {
						return ResultInError(errors.Wrap(err, "failed to get dependency resource"))
					}
				}

				if err == nil {
					// Remove its owner reference
					hasOwnerRef, err := controllerutil.HasOwnerReference(object.GetOwnerReferences(), controller, reconciler.GetScheme())
					if err != nil {
						return ResultInError(errors.Wrap(err, "failed to check owner reference"))
					}

					if hasOwnerRef {
						if err := controllerutil.RemoveOwnerReference(controller, &object, reconciler.GetScheme()); err != nil {
							return ResultInError(errors.Wrap(err, "failed to remove owner reference"))
						}

						if err := reconciler.Update(ctx, &object); err != nil {
							return ResultInError(errors.Wrap(err, "failed to update dependency resource"))
						}
					}
				}

				// Remove the item from the status
				changed := controllerStatus.Dependencies.Remove(&item)
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

func getItemsMissingFrom(
	newList,
	oldList ObjectReferenceList,
) ObjectReferenceList {
	missing := ObjectReferenceList{}
	for _, oldItem := range oldList {
		found := false
		for _, newItem := range newList {
			if oldItem.Equal(&newItem) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, oldItem)
		}
	}
	return missing
}
