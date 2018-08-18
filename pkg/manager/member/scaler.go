// Copyright 2018 PingCAP, Inc.
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

package member

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	apps "k8s.io/api/apps/v1beta1"
)

// Scaler implements the logic for scaling out or scaling in the cluster.
type Scaler interface {
	// ScaleOut scales out the cluster
	ScaleOut(*v1alpha1.TidbCluster, *apps.StatefulSet, *apps.StatefulSet) error
	// ScaleIn scales in the cluster
	ScaleIn(*v1alpha1.TidbCluster, *apps.StatefulSet, *apps.StatefulSet) error
}

func resetReplicas(newSet *apps.StatefulSet, oldSet *apps.StatefulSet) {
	*newSet.Spec.Replicas = *oldSet.Spec.Replicas
}
func increaseReplicas(newSet *apps.StatefulSet, oldSet *apps.StatefulSet) {
	*newSet.Spec.Replicas = *oldSet.Spec.Replicas + 1
}
func decreaseReplicas(newSet *apps.StatefulSet, oldSet *apps.StatefulSet) {
	*newSet.Spec.Replicas = *oldSet.Spec.Replicas - 1
}
