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

// merger merges a list of collectors
type merger struct {
	collectors []Collector
}

var _ Collector = (*merger)(nil)

func (m *merger) Objects() (<-chan client.Object, error) {
	objCh := make(chan client.Object)
	go func() {
		for len(m.collectors) > 0 {
			ch, err := m.collectors[0].Objects()
			if err != nil {
				panic(err)
			}
			for obj := range ch {
				objCh <- obj // nolint: gosec
			}
			m.collectors = m.collectors[1:]
		}
		close(objCh)
	}()
	return objCh, nil
}

func (m *merger) WithNamespace(ns string) Collector {
	panic("not implemented")
}

func (m *merger) WithLabel(label map[string]string) Collector {
	panic("not implemented")
}

// MergedCollectors combines a list of collectors.
func MergedCollectors(collectors ...Collector) Collector {
	return &merger{
		collectors: collectors,
	}
}
