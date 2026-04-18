package ctrlfwk

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// NotPausedPredicate is a predicate that filters out paused resources from reconciliation.
// Resources with the ctrlfwk.uctf.io/pause label will not trigger reconciliation events.
type NotPausedPredicate = TypedNotPausedPredicate[client.Object]

// FinalizingPredicate allows reconciliation to continue when deletion starts so finalizers can run.
type FinalizingPredicate = TypedFinalizingPredicate[client.Object]

// TypedNotPausedPredicate filters reconciliation events for resources marked as paused.
// When applied to a controller, it prevents the controller from queuing reconciliation
// requests for resources that have the pause label set.
type TypedNotPausedPredicate[object client.Object] struct{}

func (p TypedNotPausedPredicate[object]) Create(e event.TypedCreateEvent[object]) bool {
	obj := e.Object
	labels := obj.GetLabels()
	if labels == nil {
		return true
	}
	if _, ok := labels[LabelReconciliationPaused]; ok {
		return false
	}
	return true
}

func (p TypedNotPausedPredicate[object]) Delete(e event.TypedDeleteEvent[object]) bool {
	return true
}

func (p TypedNotPausedPredicate[object]) Update(e event.TypedUpdateEvent[object]) bool {
	obj := e.ObjectNew
	labels := obj.GetLabels()
	if labels == nil {
		return true
	}
	if _, ok := labels[LabelReconciliationPaused]; ok {
		return false
	}
	return true
}

func (p TypedNotPausedPredicate[object]) Generic(e event.TypedGenericEvent[object]) bool {
	obj := e.Object
	labels := obj.GetLabels()
	if labels == nil {
		return true
	}
	if _, ok := labels[LabelReconciliationPaused]; ok {
		return false
	}
	return true
}

// TypedFinalizingPredicate matches the transition into the finalizing state.
// It should be ORed with stricter predicates such as generation-change filters
// so finalizer reconciliation is not skipped.
type TypedFinalizingPredicate[object client.Object] struct{}

func (p TypedFinalizingPredicate[object]) Create(e event.TypedCreateEvent[object]) bool {
	return isObjectFinalizing(e.Object)
}

func (p TypedFinalizingPredicate[object]) Delete(e event.TypedDeleteEvent[object]) bool {
	return false
}

func (p TypedFinalizingPredicate[object]) Update(e event.TypedUpdateEvent[object]) bool {
	return !hasDeletionTimestamp(e.ObjectOld) && isObjectFinalizing(e.ObjectNew)
}

func (p TypedFinalizingPredicate[object]) Generic(e event.TypedGenericEvent[object]) bool {
	return isObjectFinalizing(e.Object)
}

func hasDeletionTimestamp(obj client.Object) bool {
	if obj == nil {
		return false
	}

	return !obj.GetDeletionTimestamp().IsZero()
}

func isObjectFinalizing(obj client.Object) bool {
	if obj == nil {
		return false
	}

	deletionTimestamp := obj.GetDeletionTimestamp()
	return deletionTimestamp != nil && !deletionTimestamp.Equal(&metav1.Time{})
}
