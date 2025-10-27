package ctrlfwk

import "sigs.k8s.io/controller-runtime/pkg/client"

func IsFinalizing(obj client.Object) bool {
	return !obj.GetDeletionTimestamp().IsZero()
}
