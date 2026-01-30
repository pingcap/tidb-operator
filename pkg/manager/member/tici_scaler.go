// Copyright 2026 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package member

import (
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/pkg/controller"
)

type ticiScaler struct {
	generalScaler
}

// NewTiCIScaler returns a TiCI scaler.
func NewTiCIScaler(deps *controller.Dependencies) Scaler {
	return &ticiScaler{generalScaler: generalScaler{deps: deps}}
}

// NewFakeTiCIScaler returns a fake scaler for TiCI.
func NewFakeTiCIScaler() Scaler {
	return &ticiScaler{}
}

// Scale scales in or out of the statefulset.
func (s *ticiScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(oldSet, newSet)
	if scaling > 0 {
		return s.ScaleOut(meta, oldSet, newSet)
	} else if scaling < 0 {
		return s.ScaleIn(meta, oldSet, newSet)
	}
	return nil
}

// ScaleOut scales out of the statefulset.
func (s *ticiScaler) ScaleOut(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	_, _, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

// ScaleIn scales in of the statefulset.
func (s *ticiScaler) ScaleIn(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	_, _, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}
