package ctrlfwk

import (
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CustomResource[K client.Object] struct {
	cleanObject K
	CR          K

	cleanObjectInitialized bool
	crInitialized          bool
}

// GetCleanCustomResource gives back the resource that was stored previously unedited of any changes that the resource may have went through,
// It is especially useful to generate patches between the time it was first seen and the second time.
func (cr *CustomResource[K]) GetCleanCustomResource() K {
	if cr.cleanObjectInitialized {
		return cr.cleanObject
	}

	if reflect.ValueOf(cr.cleanObject).IsNil() {
		cr.cleanObject = cr.GetCustomResource().DeepCopyObject().(K)
	}
	cr.cleanObjectInitialized = true
	return cr.cleanObject.DeepCopyObject().(K)
}

// GetCustomResource gives back the resource that was stored previously,
// This resource can be edited as it should always be a client.Object which is a pointer to something
func (cr *CustomResource[K]) GetCustomResource() K {
	if cr.crInitialized {
		return cr.CR
	}

	if reflect.ValueOf(cr.CR).IsNil() {
		cr.CR = reflect.New(reflect.TypeOf(cr.CR).Elem()).Interface().(K)
	}
	cr.crInitialized = true
	return cr.CR
}

// SetCustomResource sets the resource and also the base resource,
// It should only be used once per reconciliation.
func (cr *CustomResource[K]) SetCustomResource(key K) {
	cr.CR = key
	cr.cleanObject = cr.CR.DeepCopyObject().(K)

	cr.crInitialized = true
	cr.cleanObjectInitialized = true
}
