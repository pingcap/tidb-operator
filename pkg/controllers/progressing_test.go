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

package controllers_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/pdgroup"
	pdtasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/pdgroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/resourcemanagergroup"
	resourcemanagertasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/resourcemanagergroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/routergroup"
	routertasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/routergroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/schedulergroup"
	schedulertasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/schedulergroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/schedulinggroup"
	schedulingtasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/schedulinggroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/ticdcgroup"
	ticdctasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/ticdcgroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tidbgroup"
	tidbtasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/tidbgroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tiflashgroup"
	tiflashtasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/tiflashgroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tikvgroup"
	tikvtasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/tikvgroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tikvworkergroup"
	tikvworkertasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/tikvworkergroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tiproxygroup"
	tiproxytasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/tiproxygroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tsogroup"
	tsotasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/tsogroup/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

type groupRunnerCase struct {
	name      string
	newGroup  func(*bool) client.Object
	newRunner func(client.Client, types.NamespacedName) task.TaskRunner
}

func groupRunnerCases() []groupRunnerCase {
	return []groupRunnerCase{
		{
			name: "PDGroup",
			newGroup: func(progressing *bool) client.Object {
				return &v1alpha1.PDGroup{Spec: v1alpha1.PDGroupSpec{
					Cluster: v1alpha1.ClusterReference{Name: "cluster"}, Progressing: progressing,
				}}
			},
			newRunner: func(c client.Client, key types.NamespacedName) task.TaskRunner {
				r := &pdgroup.Reconciler{Client: c}
				return r.NewRunner(&pdtasks.ReconcileContext{State: pdtasks.NewState(key)}, nil)
			},
		},
		{
			name: "TiKVGroup",
			newGroup: func(progressing *bool) client.Object {
				return &v1alpha1.TiKVGroup{Spec: v1alpha1.TiKVGroupSpec{
					Cluster: v1alpha1.ClusterReference{Name: "cluster"}, Progressing: progressing,
				}}
			},
			newRunner: func(c client.Client, key types.NamespacedName) task.TaskRunner {
				r := &tikvgroup.Reconciler{Client: c}
				return r.NewRunner(&tikvtasks.ReconcileContext{State: tikvtasks.NewState(key)}, nil)
			},
		},
		{
			name: "TiDBGroup",
			newGroup: func(progressing *bool) client.Object {
				return &v1alpha1.TiDBGroup{Spec: v1alpha1.TiDBGroupSpec{
					Cluster: v1alpha1.ClusterReference{Name: "cluster"}, Progressing: progressing,
				}}
			},
			newRunner: func(c client.Client, key types.NamespacedName) task.TaskRunner {
				r := &tidbgroup.Reconciler{Client: c}
				return r.NewRunner(&tidbtasks.ReconcileContext{State: tidbtasks.NewState(key)}, nil)
			},
		},
		{
			name: "TiFlashGroup",
			newGroup: func(progressing *bool) client.Object {
				return &v1alpha1.TiFlashGroup{Spec: v1alpha1.TiFlashGroupSpec{
					Cluster: v1alpha1.ClusterReference{Name: "cluster"}, Progressing: progressing,
				}}
			},
			newRunner: func(c client.Client, key types.NamespacedName) task.TaskRunner {
				r := &tiflashgroup.Reconciler{Client: c}
				return r.NewRunner(&tiflashtasks.ReconcileContext{State: tiflashtasks.NewState(key)}, nil)
			},
		},
		{
			name: "TiCDCGroup",
			newGroup: func(progressing *bool) client.Object {
				return &v1alpha1.TiCDCGroup{Spec: v1alpha1.TiCDCGroupSpec{
					Cluster: v1alpha1.ClusterReference{Name: "cluster"}, Progressing: progressing,
				}}
			},
			newRunner: func(c client.Client, key types.NamespacedName) task.TaskRunner {
				r := &ticdcgroup.Reconciler{Client: c}
				return r.NewRunner(&ticdctasks.ReconcileContext{State: ticdctasks.NewState(key)}, nil)
			},
		},
		{
			name: "TiProxyGroup",
			newGroup: func(progressing *bool) client.Object {
				return &v1alpha1.TiProxyGroup{Spec: v1alpha1.TiProxyGroupSpec{
					Cluster: v1alpha1.ClusterReference{Name: "cluster"}, Progressing: progressing,
				}}
			},
			newRunner: func(c client.Client, key types.NamespacedName) task.TaskRunner {
				r := &tiproxygroup.Reconciler{Client: c}
				return r.NewRunner(&tiproxytasks.ReconcileContext{State: tiproxytasks.NewState(key)}, nil)
			},
		},
		{
			name: "TSOGroup",
			newGroup: func(progressing *bool) client.Object {
				return &v1alpha1.TSOGroup{Spec: v1alpha1.TSOGroupSpec{
					Cluster: v1alpha1.ClusterReference{Name: "cluster"}, Progressing: progressing,
				}}
			},
			newRunner: func(c client.Client, key types.NamespacedName) task.TaskRunner {
				r := &tsogroup.Reconciler{Client: c}
				return r.NewRunner(&tsotasks.ReconcileContext{State: tsotasks.NewState(key)}, nil)
			},
		},
		{
			name: "SchedulingGroup",
			newGroup: func(progressing *bool) client.Object {
				return &v1alpha1.SchedulingGroup{Spec: v1alpha1.SchedulingGroupSpec{
					Cluster: v1alpha1.ClusterReference{Name: "cluster"}, Progressing: progressing,
				}}
			},
			newRunner: func(c client.Client, key types.NamespacedName) task.TaskRunner {
				r := &schedulinggroup.Reconciler{Client: c}
				return r.NewRunner(&schedulingtasks.ReconcileContext{State: schedulingtasks.NewState(key)}, nil)
			},
		},
		{
			name: "SchedulerGroup",
			newGroup: func(progressing *bool) client.Object {
				return &v1alpha1.SchedulerGroup{Spec: v1alpha1.SchedulerGroupSpec{
					Cluster: v1alpha1.ClusterReference{Name: "cluster"}, Progressing: progressing,
				}}
			},
			newRunner: func(c client.Client, key types.NamespacedName) task.TaskRunner {
				r := &schedulergroup.Reconciler{Client: c}
				return r.NewRunner(&schedulertasks.ReconcileContext{State: schedulertasks.NewState(key)}, nil)
			},
		},
		{
			name: "RouterGroup",
			newGroup: func(progressing *bool) client.Object {
				return &v1alpha1.RouterGroup{Spec: v1alpha1.RouterGroupSpec{
					Cluster: v1alpha1.ClusterReference{Name: "cluster"}, Progressing: progressing,
				}}
			},
			newRunner: func(c client.Client, key types.NamespacedName) task.TaskRunner {
				r := &routergroup.Reconciler{Client: c}
				return r.NewRunner(&routertasks.ReconcileContext{State: routertasks.NewState(key)}, nil)
			},
		},
		{
			name: "ResourceManagerGroup",
			newGroup: func(progressing *bool) client.Object {
				return &v1alpha1.ResourceManagerGroup{Spec: v1alpha1.ResourceManagerGroupSpec{
					Cluster: v1alpha1.ClusterReference{Name: "cluster"}, Progressing: progressing,
				}}
			},
			newRunner: func(c client.Client, key types.NamespacedName) task.TaskRunner {
				r := &resourcemanagergroup.Reconciler{Client: c}
				return r.NewRunner(&resourcemanagertasks.ReconcileContext{State: resourcemanagertasks.NewState(key)}, nil)
			},
		},
		{
			name: "TiKVWorkerGroup",
			newGroup: func(progressing *bool) client.Object {
				return &v1alpha1.TiKVWorkerGroup{Spec: v1alpha1.TiKVWorkerGroupSpec{
					Cluster: v1alpha1.ClusterReference{Name: "cluster"}, Progressing: progressing,
				}}
			},
			newRunner: func(c client.Client, key types.NamespacedName) task.TaskRunner {
				r := &tikvworkergroup.Reconciler{Client: c}
				return r.NewRunner(&tikvworkertasks.ReconcileContext{State: tikvworkertasks.NewState(key)}, nil)
			},
		},
	}
}

func TestGroupRunnerPause(t *testing.T) {
	ctx := context.Background()
	for _, groupCase := range groupRunnerCases() {
		t.Run(groupCase.name, func(t *testing.T) {
			for _, mode := range []struct {
				name          string
				progressing   *bool
				clusterPaused bool
				paused        bool
			}{
				{name: "omitted"},
				{name: "enabled", progressing: ptr.To(true)},
				{name: "group paused", progressing: ptr.To(false), paused: true},
				{name: "cluster paused", progressing: ptr.To(true), clusterPaused: true, paused: true},
				{name: "both paused", progressing: ptr.To(false), clusterPaused: true, paused: true},
			} {
				for _, deleting := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/deleting=%t", mode.name, deleting), func(t *testing.T) {
						group := groupCase.newGroup(mode.progressing)
						group.SetName("group")
						group.SetNamespace("default")
						group.SetFinalizers([]string{meta.Finalizer})
						if deleting {
							now := metav1.Now()
							group.SetDeletionTimestamp(&now)
						}
						before := group.DeepCopyObject()
						cluster := &v1alpha1.Cluster{
							ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "default"},
							Spec:       v1alpha1.ClusterSpec{Paused: mode.clusterPaused},
						}
						c := client.NewFakeClient(group, cluster)
						// The first task after the pause boundary lists instances. Reject
						// all work except context reads to catch late guards and any writes.
						const unexpectedWork = "reconciliation proceeded beyond context reads"
						for _, verb := range []string{"list", "create", "update", "patch", "delete", "delete-collection"} {
							c.WithError(verb, "*", fmt.Errorf("%s", unexpectedWork))
						}
						key := types.NamespacedName{Namespace: group.GetNamespace(), Name: group.GetName()}
						result, err := groupCase.newRunner(c, key).Run(ctx)
						if mode.paused {
							require.NoError(t, err)
							require.False(t, result.Requeue)
							require.Zero(t, result.RequeueAfter)
						} else {
							require.ErrorContains(t, err, unexpectedWork)
						}
						require.NoError(t, c.Get(ctx, key, group))
						require.Equal(t, before, group, "context loading must not change the group")
					})
				}
			}
		})
	}
}

func TestGroupDeletionResumes(t *testing.T) {
	ctx := context.Background()
	for _, groupCase := range groupRunnerCases() {
		t.Run(groupCase.name, func(t *testing.T) {
			for _, resume := range []struct {
				name        string
				progressing *bool
			}{
				{name: "omitted"},
				{name: "enabled", progressing: ptr.To(true)},
			} {
				t.Run(resume.name, func(t *testing.T) {
					group := groupCase.newGroup(ptr.To(false))
					group.SetName("group")
					group.SetNamespace("default")
					group.SetFinalizers([]string{meta.Finalizer})
					now := metav1.Now()
					group.SetDeletionTimestamp(&now)
					cluster := &v1alpha1.Cluster{
						ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "default"},
					}
					c := client.NewFakeClient(group, cluster)
					key := types.NamespacedName{Namespace: group.GetNamespace(), Name: group.GetName()}
					_, err := groupCase.newRunner(c, key).Run(ctx)
					require.NoError(t, err)
					before := group.DeepCopyObject()
					require.NoError(t, c.Get(ctx, key, group))
					require.Equal(t, before, group, "paused deletion must preserve finalizers and status")

					resumed := groupCase.newGroup(resume.progressing)
					resumed.SetName(group.GetName())
					resumed.SetNamespace(group.GetNamespace())
					resumed.SetResourceVersion(group.GetResourceVersion())
					resumed.SetFinalizers(group.GetFinalizers())
					resumed.SetDeletionTimestamp(group.GetDeletionTimestamp())
					require.NoError(t, c.Update(ctx, resumed))
					_, err = groupCase.newRunner(c, key).Run(ctx)
					require.NoError(t, err)
					require.NoError(t, c.Get(ctx, key, resumed))
					require.NotContains(t, resumed.GetFinalizers(), meta.Finalizer,
						"resuming must allow deletion of an empty group to finish")
				})
			}
		})
	}
}
