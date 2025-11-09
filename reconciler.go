package ctrlfwk

import (
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ControllerCustomResource interface {
	client.Object
}

type Reconciler[ControllerResourceType ControllerCustomResource] interface {
	client.Client

	For(ControllerResourceType)
}

type ReconcilerWithWatcher[ControllerResourceType ControllerCustomResource] interface {
	Reconciler[ControllerResourceType]

	Watcher
}

type ReconcilerWithDependencies[ControllerResourceType ControllerCustomResource] interface {
	Reconciler[ControllerResourceType]

	GetDependencies(ctx Context[ControllerResourceType], req ctrl.Request) ([]GenericDependency[ControllerResourceType], error)
}

type ReconcilerWithResources[ControllerResourceType ControllerCustomResource] interface {
	Reconciler[ControllerResourceType]

	GetResources(ctx Context[ControllerResourceType], req ctrl.Request) ([]GenericResource[ControllerResourceType], error)
}

type ReconcilerWithEventRecorder[ControllerResourceType ControllerCustomResource] interface {
	Reconciler[ControllerResourceType]

	record.EventRecorder
}
