package collect

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Collector is an interface for collecting certain resource.
type Collector interface {
	// Objects returs a channel of kubernetes objects.
	Objects() (<-chan client.Object, error)
	// WithNamespace retrict collector to collect resources in specific
	// namespace.
	WithNamespace(ns string) Collector
	// WithLabel restrict collector to collect resources with specific labels.
	WithLabel(label map[string]string) Collector
}
