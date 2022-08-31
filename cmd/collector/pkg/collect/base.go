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
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BaseCollector is a base collector that implements the Collector interface.
// It's used to host common objects required by all collectors.
type BaseCollector struct {
	// Reader is a read-only client for retrieving objects information.
	client.Reader
	// opts is a set of customized options for listing information.
	opts []client.ListOption
}

var _ Collector = (*BaseCollector)(nil)

func (*BaseCollector) Objects() (<-chan client.Object, error) {
	panic("not implemented")
}

// NewBaseCollector returns an instance of the BaseCollector.
func NewBaseCollector(cli client.Reader) *BaseCollector {

	return &BaseCollector{
		Reader: cli,
		opts:   []client.ListOption{},
	}
}

// WithNamespace add option to base collector to select resources from specific
// namespace.
func (b *BaseCollector) WithNamespace(ns string) Collector {
	b.opts = append(b.opts, (client.InNamespace)(ns))
	return b
}

// WithLabel add option to base collector to select resources with specific
// labels.
func (b *BaseCollector) WithLabel(label map[string]string) Collector {
	b.opts = append(b.opts, client.MatchingLabelsSelector{
		Selector: (labels.Set)(label).AsSelector(),
	})
	return b
}
