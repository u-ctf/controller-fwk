package ctrlfwk

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// NotPausedPredicate is a predicate that filters out paused resources from reconciliation.
// Resources with the ctrlfwk.com/pause label will not trigger reconciliation events.
type NotPausedPredicate = TypedNotPausedPredicate[client.Object]

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
