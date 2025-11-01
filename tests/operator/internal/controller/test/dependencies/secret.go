package test_dependencies

import (
	"context"

	ctrlfwk "github.com/u-ctf/controller-fwk"
	"k8s.io/apimachinery/pkg/api/meta"

	testv1 "operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewSecretDependency creates a new Dependency representing a Secret
func NewSecretDependency(reconciler ctrlfwk.Reconciler[*testv1.Test]) *ctrlfwk.Dependency[*corev1.Secret] {
	cr := reconciler.GetCustomResource()

	return ctrlfwk.NewDependencyBuilder(&corev1.Secret{}).
		WithName(cr.Spec.Dependencies.Secret.Name).
		WithNamespace(cr.Spec.Dependencies.Secret.Namespace).
		WithOptional(false).
		WithIsReadyFunc(func(secret *corev1.Secret) bool {
			return isSecretReady(secret)
		}).
		WithWaitForReady(true).
		WithAfterReconcile(func(ctx context.Context, resource *corev1.Secret) error {
			if !isSecretReady(resource) {
				return SetConditionNotFoundWhenNotFound(ctx, reconciler)
			}

			return CleanupStatusOnOK(ctx, reconciler)
		}).
		Build()
}

func isSecretReady(secret *corev1.Secret) bool {
	return secret.Data["ready"] != nil
}

func SetConditionNotFoundWhenNotFound(
	ctx context.Context,
	reconciler ctrlfwk.Reconciler[*testv1.Test],
) error {
	cr := reconciler.GetCustomResource()

	cond := meta.FindStatusCondition(cr.Status.Conditions, "SecretFound")
	if cond == nil {
		cond = &metav1.Condition{
			Type:               "SecretFound",
			Status:             metav1.ConditionFalse,
			Reason:             "NotFound",
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

func CleanupStatusOnOK(
	ctx context.Context,
	reconciler ctrlfwk.Reconciler[*testv1.Test],
) error {
	cr := reconciler.GetCustomResource()
	changed := meta.RemoveStatusCondition(&cr.Status.Conditions, "SecretFound")
	if changed {
		// If we removed the condition, we need to update the status
		return ctrlfwk.PatchCustomResourceStatus(ctx, reconciler)
	}

	return nil
}
