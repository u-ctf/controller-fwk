package test_dependencies

import (
	ctrlfwk "github.com/u-ctf/controller-fwk"

	testv1 "operator/api/v1"

	corev1 "k8s.io/api/core/v1"
)

// NewOptionalSecretDependency creates a new Dependency representing a Secret
func NewOptionalSecretDependency(ctx testv1.TestContext, reconciler ctrlfwk.ReconcilerWithEventRecorder[*testv1.Test]) testv1.TestDependency {
	cr := ctx.GetCustomResource()

	return ctrlfwk.NewDependencyBuilder(ctx, &corev1.Secret{}).
		WithName(cr.Spec.Dependencies.Secret.Name + "-does-not-exist").
		WithNamespace(cr.Spec.Dependencies.Secret.Namespace).
		WithOptional(true).
		WithWaitForReady(true).
		WithIsReadyFunc(func(secret *corev1.Secret) bool {
			return isSecretReady(secret)
		}).
		WithAddManagedByAnnotation(true).
		Build()
}
