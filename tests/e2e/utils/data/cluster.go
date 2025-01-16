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

// TODO: migrate to tests/e2e/data pkg
package data

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

const (
	imageRegistry = "gcr.io/pingcap-public/dbaas/"
	helperImage   = "gcr.io/pingcap-public/dbaas/busybox:1.36.0"

	version = "v8.1.0"
	GiB     = 1024 * 1024 * 1024
)

func StorageSizeGi2quantity(sizeGi uint32) resource.Quantity {
	return *resource.NewQuantity(int64(sizeGi)*GiB, resource.BinarySI)
}

func NewCluster(namespace, name string) *v1alpha1.Cluster {
	return &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: v1alpha1.ClusterSpec{
			UpgradePolicy: v1alpha1.UpgradePolicyDefault,
		},
	}
}

func NewPDGroup(namespace, name, clusterName string, replicas *int32, apply func(*v1alpha1.PDGroup)) *v1alpha1.PDGroup {
	pdg := &v1alpha1.PDGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				v1alpha1.LabelKeyGroup:     name,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
				v1alpha1.LabelKeyCluster:   clusterName,
			},
		},
		Spec: v1alpha1.PDGroupSpec{
			Cluster:  v1alpha1.ClusterReference{Name: clusterName},
			Replicas: replicas,
			Version:  version,
			Template: v1alpha1.PDTemplate{
				Spec: v1alpha1.PDTemplateSpec{
					Image: ptr.To(imageRegistry + "pd"),
					Volumes: []v1alpha1.Volume{
						{
							Name:    "data",
							Mounts:  []v1alpha1.VolumeMount{{Type: "data"}},
							Storage: StorageSizeGi2quantity(1),
						},
					},
				},
			},
		},
	}
	if apply != nil {
		apply(pdg)
	}
	return pdg
}

func NewTiKVGroup(namespace, name, clusterName string, replicas *int32, apply func(*v1alpha1.TiKVGroup)) *v1alpha1.TiKVGroup {
	kvg := &v1alpha1.TiKVGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				v1alpha1.LabelKeyGroup:     name,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
				v1alpha1.LabelKeyCluster:   clusterName,
			},
		},
		Spec: v1alpha1.TiKVGroupSpec{
			Cluster:  v1alpha1.ClusterReference{Name: clusterName},
			Version:  version,
			Replicas: replicas,
			Template: v1alpha1.TiKVTemplate{
				Spec: v1alpha1.TiKVTemplateSpec{
					Image: ptr.To(imageRegistry + "tikv"),
					Volumes: []v1alpha1.Volume{
						{
							Name:    "data",
							Mounts:  []v1alpha1.VolumeMount{{Type: "data"}},
							Storage: StorageSizeGi2quantity(1),
						},
					},
				},
			},
		},
	}
	if apply != nil {
		apply(kvg)
	}
	return kvg
}

func NewTiDBGroup(namespace, name, clusterName string, replicas *int32, apply func(group *v1alpha1.TiDBGroup)) *v1alpha1.TiDBGroup {
	tg := &v1alpha1.TiDBGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				v1alpha1.LabelKeyGroup:     name,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
				v1alpha1.LabelKeyCluster:   clusterName,
			},
		},
		Spec: v1alpha1.TiDBGroupSpec{
			Cluster:  v1alpha1.ClusterReference{Name: clusterName},
			Version:  version,
			Replicas: replicas,
			Template: v1alpha1.TiDBTemplate{
				Spec: v1alpha1.TiDBTemplateSpec{
					Image: ptr.To(imageRegistry + "tidb"),
					SlowLog: &v1alpha1.TiDBSlowLog{
						Image: ptr.To(helperImage),
					},
				},
			},
		},
	}
	if apply != nil {
		apply(tg)
	}
	return tg
}

func NewTiFlashGroup(namespace, name, clusterName string, replicas *int32,
	apply func(group *v1alpha1.TiFlashGroup),
) *v1alpha1.TiFlashGroup {
	flashg := &v1alpha1.TiFlashGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				v1alpha1.LabelKeyGroup:     name,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
				v1alpha1.LabelKeyCluster:   clusterName,
			},
		},
		Spec: v1alpha1.TiFlashGroupSpec{
			Cluster:  v1alpha1.ClusterReference{Name: clusterName},
			Version:  version,
			Replicas: replicas,
			Template: v1alpha1.TiFlashTemplate{
				Spec: v1alpha1.TiFlashTemplateSpec{
					Image: ptr.To(imageRegistry + "tiflash"),
					Volumes: []v1alpha1.Volume{
						{
							Name:    "data",
							Mounts:  []v1alpha1.VolumeMount{{Type: "data"}},
							Storage: StorageSizeGi2quantity(1),
						},
					},
					LogTailer: &v1alpha1.TiFlashLogTailer{
						Image: ptr.To(helperImage),
					},
				},
			},
		},
	}
	if apply != nil {
		apply(flashg)
	}
	return flashg
}
