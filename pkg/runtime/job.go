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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	brv1alpha1 "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
)

// Job is the interface for a job. for example, backup, restore, etc.
type Job interface {
	Object

	Object() client.Object
}

// br
type (
	Backup brv1alpha1.Backup
)

var _ Job = &Backup{}

func (b *Backup) SetCluster(cluster string) {
	b.Spec.BR.Cluster = cluster
}

func (b *Backup) Cluster() string {
	return b.Spec.BR.Cluster
}

func (b *Backup) Component() string {
	return v1alpha1.LabelValComponentBackup
}

func (b *Backup) Conditions() []metav1.Condition {
	return b.Status.Conditions
}

func (b *Backup) SetConditions(conds []metav1.Condition) {
	b.Status.Conditions = conds
}

func (b *Backup) ObservedGeneration() int64 {
	// return b.Status.ObservedGeneration
	// fixme(ideascf): do we need this?
	return 0
}

func (b *Backup) SetObservedGeneration(g int64) {
	// fixme(ideascf): do we need this?
	// b.Status.ObservedGeneration = g
}

func (b *Backup) Object() client.Object {
	return (*brv1alpha1.Backup)(b)
}

type (
	Restore brv1alpha1.Restore
)

var _ Job = &Restore{}

func (r *Restore) SetCluster(cluster string) {
	r.Spec.BR.Cluster = cluster
}

func (r *Restore) Cluster() string {
	return r.Spec.BR.Cluster
}

func (r *Restore) Component() string {
	return v1alpha1.LabelValComponentRestore
}

func (r *Restore) Conditions() []metav1.Condition {
	return r.Status.Conditions
}

func (r *Restore) SetConditions(conds []metav1.Condition) {
	r.Status.Conditions = conds
}

func (r *Restore) ObservedGeneration() int64 {
	// return r.Status.ObservedGeneration
	// fixme(ideascf): do we need this?
	return 0
}

func (r *Restore) SetObservedGeneration(g int64) {
	// fixme(ideascf): do we need this?
	// r.Status.ObservedGeneration = g
}

func (r *Restore) Object() client.Object {
	return (*brv1alpha1.Restore)(r)
}
