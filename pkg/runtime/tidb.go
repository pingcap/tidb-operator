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

package runtime

import (
	"unsafe"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

type (
	TiDB      v1alpha1.TiDB
	TiDBGroup v1alpha1.TiDBGroup
)

func FromTiDB(db *v1alpha1.TiDB) *TiDB {
	return (*TiDB)(db)
}

func FromTiDBSlice(dbs []*v1alpha1.TiDB) []*TiDB {
	return *(*[]*TiDB)(unsafe.Pointer(&dbs))
}

var _ instance = &TiDB{}

func (db *TiDB) DeepCopyObject() runtime.Object {
	return (*v1alpha1.TiDB)(db).DeepCopyObject()
}

func (db *TiDB) To() client.Object {
	return (*v1alpha1.TiDB)(db)
}

func (db *TiDB) GetTopology() v1alpha1.Topology {
	return db.Spec.Topology
}

func (db *TiDB) SetTopology(t v1alpha1.Topology) {
	db.Spec.Topology = t
}

func (db *TiDB) GetUpdateRevision() string {
	if db.Labels == nil {
		return ""
	}
	return db.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
}

func (db *TiDB) IsHealthy() bool {
	return meta.IsStatusConditionTrue(db.Status.Conditions, v1alpha1.TiKVCondHealth)
}

func (db *TiDB) IsUpToDate() bool {
	return db.Status.ObservedGeneration == db.GetGeneration() && db.GetUpdateRevision() == db.Status.CurrentRevision
}

func (db *TiDB) Conditions() []metav1.Condition {
	return db.Status.Conditions
}
