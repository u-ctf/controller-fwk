/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// UntypedTestSpec defines the desired state of UntypedTest
type UntypedTestSpec struct {
	// dependencies specifies the dependencies required by the Test resource
	Dependencies TestDependencies `json:"dependencies,omitempty"`

	// configMap specifies the configuration for the ConfigMap resource
	ConfigMap ConfigMapSpec `json:"configMap,omitempty"`
}

// UntypedTestStatus defines the observed state of UntypedTest.
type UntypedTestStatus struct {
	// conditions represent the current state of the Test resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// configMapStatus provides status information about the ConfigMap resource if it has been created.
	ConfigMapStatus *ConfigMapStatus `json:"configMapStatus,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// UntypedTest is the Schema for the untypedtests API
type UntypedTest struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of UntypedTest
	// +required
	Spec UntypedTestSpec `json:"spec"`

	// status defines the observed state of UntypedTest
	// +optional
	Status UntypedTestStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// UntypedTestList contains a list of UntypedTest
type UntypedTestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UntypedTest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&UntypedTest{}, &UntypedTestList{})
}
