package library

import (
	"fmt"
	"reflect"
	"time"

	"github.com/go-viper/mapstructure/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func GetContract[K any](object *unstructured.Unstructured, path ...string) (*K, error) {
	path = append([]string{"status"}, path...)

	// Get the contract from the object using the provided path
	contractMap, found, err := unstructured.NestedMap(object.Object, path...)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("contract not found at path: %s", path)
	}

	// Convert using mapstructure
	var result K

	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: DecodeMetaTime(),
		TagName:    "json",
		Result:     &result,
	})
	if err != nil {
		return nil, err
	}

	err = dec.Decode(contractMap)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

var metaTime = reflect.TypeOf(metav1.Time{})

func DecodeMetaTime() mapstructure.DecodeHookFuncType {
	return func(from, to reflect.Type, i interface{}) (interface{}, error) {
		if to == metaTime {
			if t, ok := i.(string); ok {
				realTime, err := time.Parse(time.RFC3339, t)
				if err != nil {
					return nil, err
				}
				return metav1.NewTime(realTime), nil
			}
			return nil, fmt.Errorf("expected string, got %T", i)
		}

		return i, nil
	}
}
