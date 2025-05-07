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

package scope

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type Scheduler struct{}

func (Scheduler) From(f *v1alpha1.Scheduler) *runtime.Scheduler {
	return runtime.FromScheduler(f) // TODO: Define runtime.FromScheduler
}

func (Scheduler) To(t *runtime.Scheduler) *v1alpha1.Scheduler {
	return runtime.ToScheduler(t) // TODO: Define runtime.ToScheduler
}

func (Scheduler) Component() string {
	return v1alpha1.LabelValComponentScheduler // TODO: Define v1alpha1.LabelValComponentScheduler
}

func (Scheduler) NewList() client.ObjectList {
	return &v1alpha1.SchedulerList{} // TODO: Define v1alpha1.SchedulerList
}

type SchedulerGroup struct{}

func (SchedulerGroup) From(f *v1alpha1.SchedulerGroup) *runtime.SchedulerGroup {
	return runtime.FromSchedulerGroup(f) // TODO: Define runtime.FromSchedulerGroup
}

func (SchedulerGroup) To(t *runtime.SchedulerGroup) *v1alpha1.SchedulerGroup {
	return runtime.ToSchedulerGroup(t) // TODO: Define runtime.ToSchedulerGroup
}

func (SchedulerGroup) NewInstanceList() *v1alpha1.SchedulerList { // TODO: Define v1alpha1.SchedulerList
	return &v1alpha1.SchedulerList{}
}

func (SchedulerGroup) GetInstanceItems(l *v1alpha1.SchedulerList) []*v1alpha1.Scheduler { // TODO: Define v1alpha1.SchedulerList and v1alpha1.Scheduler
	items := make([]*v1alpha1.Scheduler, 0, len(l.Items))
	for i := range l.Items {
		items = append(items, &l.Items[i])
	}
	return items
}

func (SchedulerGroup) Component() string {
	return v1alpha1.LabelValComponentScheduler // TODO: Define v1alpha1.LabelValComponentScheduler
}

func (SchedulerGroup) NewList() client.ObjectList {
	return &v1alpha1.SchedulerGroupList{} // TODO: Define v1alpha1.SchedulerGroupList
}
