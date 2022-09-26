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

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PersistentVolumeClaim struct {
	*BaseCollector
}

var _ Collector = (*PersistentVolumeClaim)(nil)

func (p *PersistentVolumeClaim) Objects() (<-chan client.Object, error) {
	list := &corev1.PersistentVolumeClaimList{}
	err := p.Reader.List(context.Background(), list, p.opts...)
	if err != nil {
		return nil, err
	}

	ch := make(chan client.Object)
	go func() {
		for _, obj := range list.Items {
			ch <- &obj // nolint: gosec
		}
		close(ch)
	}()
	return ch, nil
}

func NewPVCCollector(cli client.Reader) Collector {
	addCorev1Scheme()
	return &PersistentVolumeClaim{
		BaseCollector: NewBaseCollector(cli),
	}
}
