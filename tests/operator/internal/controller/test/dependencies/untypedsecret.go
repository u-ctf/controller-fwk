package test_dependencies

import (
	ctrlfwk "github.com/u-ctf/controller-fwk"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"

	testv1 "operator/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// NewUntypedSecretDependency creates a new Dependency representing a Secret
func NewUntypedSecretDependency(ctx testv1.UntypedTestContext, reconciler ctrlfwk.ReconcilerWithEventRecorder[*testv1.UntypedTest]) testv1.UntypedTestDependency {
	cr := ctx.GetCustomResource()

	return ctrlfwk.NewUntypedDependencyBuilder(ctx, schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}).
		WithName(cr.Spec.Dependencies.Secret.Name).
		WithNamespace(cr.Spec.Dependencies.Secret.Namespace).
		WithOptional(false).
		WithIsReadyFunc(func(secret *unstructured.Unstructured) bool {
			return isUntypedSecretReady(secret)
		}).
		WithWaitForReady(true).
		WithAfterReconcile(func(ctx testv1.UntypedTestContext, resource *unstructured.Unstructured) error {
			if resource.GetName() == "" {
				reconciler.Eventf(cr, "Warning", "SecretNotFound", "The required Secret was not found")
				return SetConditionNotFoundOnUntypedTest(ctx, reconciler)
			}

			if !isUntypedSecretReady(resource) {
				reconciler.Eventf(cr, "Warning", "SecretNotReady", "The required Secret is not ready")
				return SetConditionNotReadyOnUntypedTest(ctx, reconciler)
			}

			return CleanupStatusOnOKOnUntypedTest(ctx, reconciler)
		}).
		Build()
}

func isUntypedSecretReady(secret *unstructured.Unstructured) bool {
	data, found, err := unstructured.NestedMap(secret.Object, "data")
	if err != nil || !found {
		return false
	}

	_, readyFound := data["ready"]
	return readyFound
}

func SetConditionNotFoundOnUntypedTest(
	ctx testv1.UntypedTestContext,
	reconciler ctrlfwk.Reconciler[*testv1.UntypedTest],
) error {
	cr := ctx.GetCustomResource()

	cond := meta.FindStatusCondition(cr.Status.Conditions, "SecretFound")
	if cond == nil {
		cond = &metav1.Condition{
			Type:               "SecretFound",
			Status:             metav1.ConditionFalse,
			Reason:             "SecretNotFound",
			Message:            "The required Secret was not found",
			ObservedGeneration: cr.Generation,
		}
	}
	cond.ObservedGeneration = cr.Generation

	changed := meta.SetStatusCondition(&cr.Status.Conditions, *cond)
	if changed {
		// If the condition changed, we need to update the status
		return ctrlfwk.PatchCustomResourceStatus(ctx, reconciler)
	}

	return nil
}

func SetConditionNotReadyOnUntypedTest(
	ctx testv1.UntypedTestContext,
	reconciler ctrlfwk.Reconciler[*testv1.UntypedTest],
) error {
	cr := ctx.GetCustomResource()

	changed := meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
		Type:               "SecretFound",
		Status:             metav1.ConditionFalse,
		Reason:             "SecretNotReady",
		Message:            "The required Secret is not ready",
		ObservedGeneration: cr.Generation,
	})
	if changed {
		// If the condition changed, we need to update the status
		return ctrlfwk.PatchCustomResourceStatus(ctx, reconciler)
	}

	return nil
}

func CleanupStatusOnOKOnUntypedTest(
	ctx testv1.UntypedTestContext,
	reconciler ctrlfwk.ReconcilerWithEventRecorder[*testv1.UntypedTest],
) error {
	cr := ctx.GetCustomResource()

	changed := meta.RemoveStatusCondition(&cr.Status.Conditions, "SecretFound")
	if changed {
		reconciler.Eventf(cr, "Normal", "SecretFound", "The required Secret was found")
		// If we removed the condition, we need to update the status
		return ctrlfwk.PatchCustomResourceStatus(ctx, reconciler)
	}

	return nil
}
