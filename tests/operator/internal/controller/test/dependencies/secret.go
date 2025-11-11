package test_dependencies

import (
	ctrlfwk "github.com/u-ctf/controller-fwk"
	"k8s.io/apimachinery/pkg/api/meta"

	testv1 "operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewSecretDependency creates a new Dependency representing a Secret
func NewSecretDependency(ctx testv1.TestContext, reconciler ctrlfwk.ReconcilerWithEventRecorder[*testv1.Test]) testv1.TestDependency {
	cr := ctx.GetCustomResource()

	return ctrlfwk.NewDependencyBuilder(ctx, &corev1.Secret{}).
		WithName(cr.Spec.Dependencies.Secret.Name).
		WithNamespace(cr.Spec.Dependencies.Secret.Namespace).
		WithOptional(false).
		WithIsReadyFunc(func(secret *corev1.Secret) bool {
			return isSecretReady(secret)
		}).
		WithWaitForReady(true).
		WithAfterReconcile(func(ctx testv1.TestContext, resource *corev1.Secret) error {
			if resource.Name == "" {
				reconciler.Eventf(cr, "Warning", "SecretNotFound", "The required Secret was not found")
				return SetConditionNotFound(ctx, reconciler)
			}

			if !isSecretReady(resource) {
				reconciler.Eventf(cr, "Warning", "SecretNotReady", "The required Secret is not ready")
				return SetConditionNotReady(ctx, reconciler)
			}

			return CleanupStatusOnOK(ctx, reconciler)
		}).
		Build()
}

func isSecretReady(secret *corev1.Secret) bool {
	return secret.Data["ready"] != nil
}

func SetConditionNotFound(
	ctx testv1.TestContext,
	reconciler ctrlfwk.Reconciler[*testv1.Test],
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

func SetConditionNotReady(
	ctx testv1.TestContext,
	reconciler ctrlfwk.Reconciler[*testv1.Test],
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

func CleanupStatusOnOK(
	ctx testv1.TestContext,
	reconciler ctrlfwk.ReconcilerWithEventRecorder[*testv1.Test],
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
