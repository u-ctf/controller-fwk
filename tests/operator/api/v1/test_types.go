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

type ConfigMapSpec struct {
	// enabled indicates whether the ConfigMap should be created
	Enabled bool `json:"enabled,omitempty"`

	// name is the name of the ConfigMap to be created
	Name string `json:"name,omitempty"`

	// data contains the key-value pairs to be stored in the ConfigMap
	Data map[string]string `json:"data,omitempty"`
}

// TestSpec defines the desired state of Test
type TestSpec struct {
	// configMap specifies the configuration for the ConfigMap child resource
	ConfigMap ConfigMapSpec `json:"configMap,omitempty"`
}

type ConfigMapStatus struct {
	// name is the name of the ConfigMap created
	Name string `json:"name,omitempty"`
}

// TestStatus defines the observed state of Test.
type TestStatus struct {
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

	// configMapStatus provides status information about the ConfigMap child resource if it has been created.
	ConfigMapStatus *ConfigMapStatus `json:"configMapStatus,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Test is the Schema for the tests API
type Test struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of Test
	// +required
	Spec TestSpec `json:"spec"`

	// status defines the observed state of Test
	// +optional
	Status TestStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// TestList contains a list of Test
type TestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Test `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Test{}, &TestList{})
}
