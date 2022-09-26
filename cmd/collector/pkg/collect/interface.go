// Copyright 2022 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

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
