package testlabels

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	WatchLabelKey   = "e2e.ctrlfwk.uctf.io/watch"
	WatchLabelValue = "true"
)

func Labels() map[string]string {
	return map[string]string{
		WatchLabelKey: WatchLabelValue,
	}
}

func Selector() labels.Selector {
	return labels.SelectorFromSet(Labels())
}

func ApplyToObject(object metav1.Object) {
	currentLabels := object.GetLabels()
	if currentLabels == nil {
		currentLabels = make(map[string]string)
	}

	for key, value := range Labels() {
		currentLabels[key] = value
	}

	object.SetLabels(currentLabels)
}
