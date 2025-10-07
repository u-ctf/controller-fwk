package library_test

import (
	"encoding/json"
	"library"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ExampleObjectContract struct {
	Test2 string      `json:"test2"`
	Time  metav1.Time `json:"time"`
}

type ExampleObjectStatus struct {
	Test     string                `json:"test"`
	Contract ExampleObjectContract `json:"contract"`
}

type ExampleObject struct {
	Status ExampleObjectStatus `json:"status"`
}

func TestContractDecoding(t *testing.T) {
	// Example of a contract that is not a pointer
	contract := ExampleObject{
		Status: ExampleObjectStatus{
			Test: "test",
			Contract: ExampleObjectContract{
				Test2: "test2",
				Time:  metav1.Now(),
			},
		},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("Failed to marshal contract: %v", err)
	}

	// Put JSON back in a unstructured
	var object unstructured.Unstructured
	json.Unmarshal(jsonData, &object)

	c, err := library.GetContract[ExampleObjectContract](&object, "contract")
	if err != nil {
		t.Fatalf("Failed to marshal contract: %v", err)
	}

	if c.Time.IsZero() {
		t.Fatal("contract time should not be zero")
	}
}
