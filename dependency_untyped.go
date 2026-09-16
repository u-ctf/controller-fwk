package ctrlfwk

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UntypedDependency[CustomResourceType client.Object, ContextType Context[CustomResourceType]] struct {
	*Dependency[CustomResourceType, ContextType, *unstructured.Unstructured]
	gvkF func() schema.GroupVersionKind
}

var _ GenericDependency[client.Object, Context[client.Object]] = &UntypedDependency[client.Object, Context[client.Object]]{}

func (c *UntypedDependency[CustomResourceType, ContextType]) New() client.Object {
	out := &unstructured.Unstructured{}
	out.SetGroupVersionKind(c.gvkF())
	return out
}

func (c *UntypedDependency[CustomResourceType, ContextType]) Kind() string {
	return fmt.Sprintf("Untyped%s", c.gvkF().Kind)
}

func (c *UntypedDependency[CustomResourceType, ContextType]) Set(obj client.Object) {
	if c.output == nil {
		c.output = &unstructured.Unstructured{}
		c.output.SetGroupVersionKind(c.gvkF())
	}

	unstructuredObj := obj.(*unstructured.Unstructured)
	*c.output = *unstructuredObj
	c.output.SetGroupVersionKind(c.gvkF())
}
