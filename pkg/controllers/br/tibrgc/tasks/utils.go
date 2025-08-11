// Copyright 2024 PingCAP, Inc.
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

package tasks

import (
	"fmt"

	v1alpha1br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/image"
)

func SecretName(tibrgc *v1alpha1br.TiBRGC) string {
	return fmt.Sprintf("%s-%s-cluster-secret", tibrgc.Name, ComponentName)
}
func TiBRGCSubResourceLabels(tibrgc *v1alpha1br.TiBRGC) map[string]string {
	return map[string]string{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyComponent: v1alpha1br.LabelValComponentTiBRGC,
		v1alpha1.LabelKeyCluster:   tibrgc.Spec.Cluster.Name,
		v1alpha1.LabelKeyInstance:  tibrgc.Name,
	}
}

func T2CronjobName(tibrgc *v1alpha1br.TiBRGC) string {
	return fmt.Sprintf("%s-%s", tibrgc.Name, v1alpha1br.TieredStorageStrategyNameToT2Storage)
}

func T3CronjobName(tibrgc *v1alpha1br.TiBRGC) string {
	return fmt.Sprintf("%s-%s", tibrgc.Name, v1alpha1br.TieredStorageStrategyNameToT3Storage)
}

func T2ConjobLabels(tibrgc *v1alpha1br.TiBRGC) map[string]string {
	labels := TiBRGCSubResourceLabels(tibrgc)
	labels[v1alpha1.LabelKeyName] = string(v1alpha1br.TieredStorageStrategyNameToT2Storage)
	return labels
}

func T3ConjobLabels(tibrgc *v1alpha1br.TiBRGC) map[string]string {
	labels := TiBRGCSubResourceLabels(tibrgc)
	labels[v1alpha1.LabelKeyName] = string(v1alpha1br.TieredStorageStrategyNameToT3Storage)
	return labels
}

func GetImage(tibrgc *v1alpha1br.TiBRGC) string {
	if tibrgc.Spec.Image != nil {
		return *tibrgc.Spec.Image
	}
	return string(image.TiKV)
}
