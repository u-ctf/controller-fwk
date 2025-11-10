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

type ReconcilerWithDependencies[ControllerResourceType ControllerCustomResource, ContextType Context[ControllerResourceType]] interface {
	Reconciler[ControllerResourceType]

	GetDependencies(ctx ContextType, req ctrl.Request) ([]GenericDependency[ControllerResourceType, ContextType], error)
}

type ReconcilerWithResources[ControllerResourceType ControllerCustomResource, ContextType Context[ControllerResourceType]] interface {
	Reconciler[ControllerResourceType]

	GetResources(ctx ContextType, req ctrl.Request) ([]GenericResource[ControllerResourceType, ContextType], error)
}

type ReconcilerWithEventRecorder[ControllerResourceType ControllerCustomResource] interface {
	Reconciler[ControllerResourceType]

	record.EventRecorder
}
