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
	"context"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PV struct {
	*BaseCollector
}

var _ Collector = (*PV)(nil)

func (pv *PV) Objects() (<-chan client.Object, error) {
	list := &corev1.PersistentVolumeList{}
	err := pv.Reader.List(context.Background(), list, pv.opts...)
	if err != nil {
		return nil, err
	}

	ch := make(chan client.Object, 1)
	go func() {
		for _, obj := range list.Items {
			ch <- &obj
		}
		close(ch)
	}()
	return ch, nil
}

func NewPVCollector(cli client.Reader) Collector {
	addCorev1Scheme()
	return &PV{
		BaseCollector: NewBaseCollector(cli),
	}
}

var corev1Scheme sync.Once

func addCorev1Scheme() {
	corev1Scheme.Do(func() {
		corev1.AddToScheme(scheme)
	})
}
