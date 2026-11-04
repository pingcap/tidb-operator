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

// Package image is defined to return image of components
package common_test

import (
	"context"
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	dmtasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/dm/tasks"
	dmworkertasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/dmworker/tasks"
	pdtasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/pd/tasks"
	resourcemanagertasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/resourcemanager/tasks"
	routertasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/router/tasks"
	schedulertasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/scheduler/tasks"
	schedulingtasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/scheduling/tasks"
	ticdctasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/ticdc/tasks"
	tidbtasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/tidb/tasks"
	tiflashtasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/tiflash/tasks"
	tikvtasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/tikv/tasks"
	tikvworkertasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/tikvworker/tasks"
	tiproxytasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/tiproxy/tasks"
	tsotasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/tso/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"testing"
)

// With minimal cluster context and no client, invalid input must fail before rendering or changing a Pod.
func TestComponentPodRejectsInvalidImage(t *testing.T) {
	factories := map[string]func(*string, string) task.Task{
		"pd": func(img *string, version string) task.Task {
			state := pdtasks.NewState(types.NamespacedName{})
			obj := &v1alpha1.PD{}
			obj.Name = "component-abc"
			obj.Spec.Image, obj.Spec.Version = img, version
			state.SetObject(obj)
			state.SetCluster(&v1alpha1.Cluster{})
			return pdtasks.TaskPod(&pdtasks.ReconcileContext{State: state}, nil)
		},
		"tidb": func(img *string, version string) task.Task {
			state := tidbtasks.NewState(types.NamespacedName{})
			obj := &v1alpha1.TiDB{}
			obj.Name = "component-abc"
			obj.Spec.Image, obj.Spec.Version = img, version
			state.SetObject(obj)
			state.SetCluster(&v1alpha1.Cluster{})
			return tidbtasks.TaskPod(&tidbtasks.ReconcileContext{State: state}, nil)
		},
		"tikv": func(img *string, version string) task.Task {
			state := tikvtasks.NewState(types.NamespacedName{})
			obj := &v1alpha1.TiKV{}
			obj.Name = "component-abc"
			obj.Spec.Image, obj.Spec.Version = img, version
			state.SetObject(obj)
			state.SetCluster(&v1alpha1.Cluster{})
			return tikvtasks.TaskPod(&tikvtasks.ReconcileContext{State: state}, nil, nil)
		},
		"tiflash": func(img *string, version string) task.Task {
			state := tiflashtasks.NewState(types.NamespacedName{})
			obj := &v1alpha1.TiFlash{}
			obj.Name = "component-abc"
			obj.Spec.Image, obj.Spec.Version = img, version
			state.SetObject(obj)
			state.SetCluster(&v1alpha1.Cluster{})
			return tiflashtasks.TaskPod(&tiflashtasks.ReconcileContext{State: state}, nil)
		},
		"ticdc": func(img *string, version string) task.Task {
			state := ticdctasks.NewState(types.NamespacedName{})
			obj := &v1alpha1.TiCDC{}
			obj.Name = "component-abc"
			obj.Spec.Image, obj.Spec.Version = img, version
			state.SetObject(obj)
			state.SetCluster(&v1alpha1.Cluster{})
			return ticdctasks.TaskPod(&ticdctasks.ReconcileContext{State: state}, nil)
		},
		"tiproxy": func(img *string, version string) task.Task {
			state := tiproxytasks.NewState(types.NamespacedName{})
			obj := &v1alpha1.TiProxy{}
			obj.Name = "component-abc"
			obj.Spec.Image, obj.Spec.Version = img, version
			state.SetObject(obj)
			state.SetCluster(&v1alpha1.Cluster{})
			return tiproxytasks.TaskPod(&tiproxytasks.ReconcileContext{State: state}, nil)
		},
		"tikvworker": func(img *string, version string) task.Task {
			state := tikvworkertasks.NewState(types.NamespacedName{})
			obj := &v1alpha1.TiKVWorker{}
			obj.Name = "component-abc"
			obj.Spec.Image, obj.Spec.Version = img, version
			state.SetObject(obj)
			state.SetCluster(&v1alpha1.Cluster{})
			return tikvworkertasks.TaskPod(state, nil)
		},
		"dm": func(img *string, version string) task.Task {
			state := dmtasks.NewState(types.NamespacedName{})
			obj := &v1alpha1.DM{}
			obj.Name = "component-abc"
			obj.Spec.Image, obj.Spec.Version = img, version
			state.SetObject(obj)
			state.SetCluster(&v1alpha1.Cluster{})
			return dmtasks.TaskPod(&dmtasks.ReconcileContext{State: state}, nil)
		},
		"dmworker": func(img *string, version string) task.Task {
			state := dmworkertasks.NewState(types.NamespacedName{})
			obj := &v1alpha1.DMWorker{}
			obj.Name = "component-abc"
			obj.Spec.Image, obj.Spec.Version = img, version
			state.SetObject(obj)
			state.SetCluster(&v1alpha1.Cluster{})
			return dmworkertasks.TaskPod(&dmworkertasks.ReconcileContext{State: state}, nil)
		},
		"tso": func(img *string, version string) task.Task {
			state := tsotasks.NewState(types.NamespacedName{})
			obj := &v1alpha1.TSO{}
			obj.Name = "component-abc"
			obj.Spec.Image, obj.Spec.Version = img, version
			state.SetObject(obj)
			state.SetCluster(&v1alpha1.Cluster{})
			return tsotasks.TaskPod(&tsotasks.ReconcileContext{State: state}, nil)
		},
		"scheduling": func(img *string, version string) task.Task {
			state := schedulingtasks.NewState(types.NamespacedName{})
			obj := &v1alpha1.Scheduling{}
			obj.Name = "component-abc"
			obj.Spec.Image, obj.Spec.Version = img, version
			state.SetObject(obj)
			state.SetCluster(&v1alpha1.Cluster{})
			return schedulingtasks.TaskPod(&schedulingtasks.ReconcileContext{State: state}, nil)
		},
		"scheduler": func(img *string, version string) task.Task {
			state := schedulertasks.NewState(types.NamespacedName{})
			obj := &v1alpha1.Scheduler{}
			obj.Name = "component-abc"
			obj.Spec.Image, obj.Spec.Version = img, version
			state.SetObject(obj)
			state.SetCluster(&v1alpha1.Cluster{})
			return schedulertasks.TaskPod(&schedulertasks.ReconcileContext{State: state}, nil)
		},
		"router": func(img *string, version string) task.Task {
			state := routertasks.NewState(types.NamespacedName{})
			obj := &v1alpha1.Router{}
			obj.Name = "component-abc"
			obj.Spec.Image, obj.Spec.Version = img, version
			state.SetObject(obj)
			state.SetCluster(&v1alpha1.Cluster{})
			return routertasks.TaskPod(&routertasks.ReconcileContext{State: state}, nil)
		},
		"resourcemanager": func(img *string, version string) task.Task {
			state := resourcemanagertasks.NewState(types.NamespacedName{})
			obj := &v1alpha1.ResourceManager{}
			obj.Name = "component-abc"
			obj.Spec.Image, obj.Spec.Version = img, version
			state.SetObject(obj)
			state.SetCluster(&v1alpha1.Cluster{})
			return resourcemanagertasks.TaskPod(&resourcemanagertasks.ReconcileContext{State: state}, nil)
		},
	}
	for component, factory := range factories {
		t.Run(component, func(t *testing.T) {
			for _, tc := range []struct {
				name    string
				image   *string
				version string
			}{
				{"invalid image", ptr.To("bad:image:tag"), "v8.5.0"},
				{"empty image", ptr.To(""), "v8.5.0"},
				{"invalid version", nil, "bad/tag"},
				{"empty version", nil, ""},
			} {
				t.Run(tc.name, func(t *testing.T) {
					require.NotPanics(t, func() {
						result, _ := task.RunTask(context.Background(), factory(tc.image, tc.version))
						require.Equal(t, task.SFail, result.Status(), result.Message())
						require.Contains(t, result.Message(), "invalid component image")
					})
				})
			}
		})
	}
}
