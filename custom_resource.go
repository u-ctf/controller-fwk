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

func (cr *CustomResource[K]) SetCustomResource(key K) {
	cr.CR = key
	cr.crInitialized = true
	if reflect.ValueOf(cr.cleanObject).IsNil() {
		cr.cleanObject = cr.CR.DeepCopyObject().(K)
	}
}
