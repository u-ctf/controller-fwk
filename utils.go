package ctrlfwk

import (
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsFinalizing(obj client.Object) bool {
	return !obj.GetDeletionTimestamp().IsZero()
}

func IsNamespaceTerminating(err error) bool {
	return apierrors.HasStatusCause(err, corev1.NamespaceTerminatingCause)
}
