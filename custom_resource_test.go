package ctrlfwk_test

import (
	"testing"

	ctrlfwk "github.com/u-ctf/controller-fwk"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestNewCustomResource(t *testing.T) {
	cr := &ctrlfwk.CustomResource[*unstructured.Unstructured]{}
	object := cr.GetCustomResource()
	if object == nil {
		t.Fatalf("expected non-nil custom resource")
	}

	object2 := cr.GetCustomResource()
	if object != object2 {
		t.Fatalf("expected same custom resource instance on multiple calls")
	}
}

func TestNewCustomResource_CleanResource(t *testing.T) {
	cr := &ctrlfwk.CustomResource[*unstructured.Unstructured]{}
	object := cr.GetCleanCustomResource()
	if object == nil {
		t.Fatalf("expected non-nil clean resource")
	}

	object2 := cr.GetCleanCustomResource()
	if object == object2 {
		t.Fatalf("expected different clean resource instance on multiple calls")
	}
}

func TestNewCustomResource_CleanIsImmutable(t *testing.T) {
	cr := &ctrlfwk.CustomResource[*unstructured.Unstructured]{}

	var resource unstructured.Unstructured
	resource.SetName("test-name")
	cr.SetCustomResource(&resource)

	cleanObject := cr.GetCleanCustomResource()
	if cleanObject.GetName() != "test-name" {
		t.Fatalf("expected clean resource name to be 'test-name', got '%s'", cleanObject.GetName())
	}

	cleanObject.SetName("modified-name")

	cleanObject2 := cr.GetCleanCustomResource()
	if cleanObject2.GetName() != "test-name" {
		t.Fatalf("expected clean resource name to be 'test-name', got '%s'", cleanObject2.GetName())
	}
}

func TestNewCustomResource_CleanIsResetable(t *testing.T) {
	cr := &ctrlfwk.CustomResource[*unstructured.Unstructured]{}

	var resource unstructured.Unstructured
	resource.SetName("test-name")
	cr.SetCustomResource(&resource)

	var resource2 unstructured.Unstructured
	resource2.SetName("test-name2")
	cr.SetCustomResource(&resource2)

	cleanObject := cr.GetCleanCustomResource()
	if cleanObject.GetName() != "test-name2" {
		t.Fatalf("expected clean resource name to be 'test-name2', got '%s'", cleanObject.GetName())
	}
}
