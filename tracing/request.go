package tracing

import "k8s.io/apimachinery/pkg/types"

type Request struct {
	TraceID string
	types.NamespacedName
}
