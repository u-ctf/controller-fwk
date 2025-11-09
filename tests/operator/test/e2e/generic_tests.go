//go:build e2e
// +build e2e

package e2e

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	testv1 "operator/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TestableResource defines a common interface for both Test and UntypedTest resources
type TestableResource interface {
	client.Object
	GetSpec() GenericTestSpec
	GetStatus() GenericTestStatus
	SetSpec(GenericTestSpec)
	SetStatus(GenericTestStatus)
}

// GenericTestSpec represents the common spec structure
type GenericTestSpec struct {
	Dependencies testv1.TestDependencies
	ConfigMap    testv1.ConfigMapSpec
}

// GenericTestStatus represents the common status structure
type GenericTestStatus struct {
	Conditions      []metav1.Condition
	ConfigMapStatus *testv1.ConfigMapStatus
}

// TestWrapper wraps a Test resource to implement TestableResource
type TestWrapper struct {
	*testv1.Test
}

func (tw *TestWrapper) GetSpec() GenericTestSpec {
	return GenericTestSpec{
		Dependencies: tw.Test.Spec.Dependencies,
		ConfigMap:    tw.Test.Spec.ConfigMap,
	}
}

func (tw *TestWrapper) GetStatus() GenericTestStatus {
	return GenericTestStatus{
		Conditions:      tw.Test.Status.Conditions,
		ConfigMapStatus: tw.Test.Status.ConfigMapStatus,
	}
}

func (tw *TestWrapper) SetSpec(spec GenericTestSpec) {
	tw.Test.Spec.Dependencies = spec.Dependencies
	tw.Test.Spec.ConfigMap = spec.ConfigMap
}

func (tw *TestWrapper) SetStatus(status GenericTestStatus) {
	tw.Test.Status.Conditions = status.Conditions
	tw.Test.Status.ConfigMapStatus = status.ConfigMapStatus
}

func (tw *TestWrapper) DeepCopyObject() runtime.Object {
	dc := tw.Test.DeepCopy()
	return &TestWrapper{Test: dc}
}

// UntypedTestWrapper wraps an UntypedTest resource to implement TestableResource
type UntypedTestWrapper struct {
	*testv1.UntypedTest
}

func (utw *UntypedTestWrapper) GetSpec() GenericTestSpec {
	return GenericTestSpec{
		Dependencies: utw.UntypedTest.Spec.Dependencies,
		ConfigMap:    utw.UntypedTest.Spec.ConfigMap,
	}
}

func (utw *UntypedTestWrapper) GetStatus() GenericTestStatus {
	return GenericTestStatus{
		Conditions:      utw.UntypedTest.Status.Conditions,
		ConfigMapStatus: utw.UntypedTest.Status.ConfigMapStatus,
	}
}

func (utw *UntypedTestWrapper) SetSpec(spec GenericTestSpec) {
	utw.UntypedTest.Spec.Dependencies = spec.Dependencies
	utw.UntypedTest.Spec.ConfigMap = spec.ConfigMap
}

func (utw *UntypedTestWrapper) SetStatus(status GenericTestStatus) {
	utw.UntypedTest.Status.Conditions = status.Conditions
	utw.UntypedTest.Status.ConfigMapStatus = status.ConfigMapStatus
}

func (utw *UntypedTestWrapper) DeepCopyObject() runtime.Object {
	dc := utw.UntypedTest.DeepCopy()
	return &UntypedTestWrapper{UntypedTest: dc}
}

// ResourceFactory defines how to create new instances of each resource type
type ResourceFactory func(name, namespace string) TestableResource

// CreateTestResource creates a new Test resource wrapped as TestableResource
func CreateTestResource(name, namespace string) TestableResource {
	return &TestWrapper{
		Test: &testv1.Test{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		},
	}
}

// CreateUntypedTestResource creates a new UntypedTest resource wrapped as TestableResource
func CreateUntypedTestResource(name, namespace string) TestableResource {
	return &UntypedTestWrapper{
		UntypedTest: &testv1.UntypedTest{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		},
	}
}
