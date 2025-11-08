package ctrlfwk

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UntypedResource[K client.Object] struct {
	*Resource[K, *unstructured.Unstructured]
	gvk schema.GroupVersionKind
}

var _ GenericResource[client.Object] = &UntypedResource[client.Object]{}

func (c *UntypedResource[K]) Kind() string {
	return fmt.Sprintf("Untyped%s", c.gvk.Kind)
}

func (c *UntypedResource[K]) ObjectMetaGenerator() (obj client.Object, skip bool, err error) {
	obj, skip, err = c.Resource.ObjectMetaGenerator()
	if err != nil || skip {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(c.gvk)
		return obj, skip, err
	}

	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, false, fmt.Errorf("expected *unstructured.Unstructured, got %T", obj)
	}

	unstructuredObj.SetGroupVersionKind(c.gvk)
	return unstructuredObj, false, nil
}
