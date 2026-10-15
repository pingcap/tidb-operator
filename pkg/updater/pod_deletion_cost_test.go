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

package updater

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
)

func TestPodDeletionCost(t *testing.T) {
	assert.Zero(t, podDeletionCost(nil))
	for _, tc := range []struct {
		value string
		want  int32
	}{
		{"0", 0}, {"10", 10}, {"-10", -10},
		{"2147483647", 2147483647}, {"-2147483648", -2147483648},
		{"2147483648", 0}, {"-2147483649", 0},
		{"", 0}, {"invalid", 0}, {"1.5", 0}, {"+10", 0},
		{"01", 0}, {"00", 0}, {" 10", 0}, {"10 ", 0},
		{"-01", -1}, {"-0", 0},
	} {
		t.Run(tc.value, func(t *testing.T) {
			assert.Equal(t, tc.want, podDeletionCost(map[string]string{corev1.PodDeletionCost: tc.value}))
		})
	}
}

func podWithDeletionCost(in runtime.Instance, cost string) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		// Deliberately use a non-default name to cover adopted Pods.
		Name: "adopted-" + in.GetName(), Namespace: in.GetNamespace(),
		Labels: map[string]string{
			v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
			v1alpha1.LabelKeyComponent: in.Component(),
			v1alpha1.LabelKeyInstance:  in.GetName(),
		},
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: v1alpha1.SchemeGroupVersion.String(), Kind: "PD",
			Name: in.GetName(), UID: in.GetUID(), Controller: ptr.To(true),
		}},
		Annotations: map[string]string{corev1.PodDeletionCost: cost},
	}}
}

func TestBuilderPodDeletionCostOrder(t *testing.T) {
	for _, tc := range []struct {
		name       string
		older      *runtime.PD
		olderCost  string
		newerCost  string
		missingPod bool
		custom     bool
		want       string
	}{
		{name: "lower cost beats newer", older: fakePD("older", true, true), olderCost: "-1", newerCost: "0", want: "older"},
		{name: "higher cost retains older", older: fakePD("older", true, true), olderCost: "10", newerCost: "0", want: "newer"},
		{name: "equal costs prefer newer", older: fakePD("older", true, true), olderCost: "10", newerCost: "10", want: "newer"},
		{name: "missing pod costs zero", older: fakePD("older", true, true), missingPod: true, newerCost: "10", want: "older"},
		{name: "invalid cost is zero", older: fakePD("older", true, true), olderCost: "invalid", newerCost: "10", want: "older"},
		{name: "instance priority beats cost", older: fakePDWithPriority("older", true, true, "0"), olderCost: "100", newerCost: "-100", want: "older"},
		{name: "unready beats cost", older: fakePD("older", true, false), olderCost: "100", newerCost: "-100", want: "older"},
		{name: "not running beats cost", older: fakePD("older", false, true), olderCost: "100", newerCost: "-100", want: "older"},
		{name: "group policy beats cost", older: fakePD("older", true, true), olderCost: "100", newerCost: "-100", custom: true, want: "older"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			older := tc.older
			older.SetCreationTimestamp(metav1.NewTime(time.Unix(100, 0)))
			newer := fakePD("newer", true, true)
			newer.SetCreationTimestamp(metav1.NewTime(time.Unix(200, 0)))
			cli := client.NewFakeClient(podWithDeletionCost(newer, tc.newerCost))
			if !tc.missingPod {
				require.NoError(t, cli.Create(context.Background(), podWithDeletionCost(older, tc.olderCost)))
			}
			b := New[runtime.PDTuple]().WithClient(cli)
			if tc.custom {
				b.WithScaleInPreferPolicy(PreferPolicyFunc[*runtime.PD](func([]*runtime.PD) []*runtime.PD {
					return []*runtime.PD{older}
				}))
			}
			act := b.Build().(*executor).act.(*actor[runtime.PDTuple, *v1alpha1.PD, *runtime.PD])
			instances := []*runtime.PD{older, newer}
			chosen, err := act.chooseToScaleIn(context.Background(), instances)
			require.NoError(t, err)
			assert.Equal(t, tc.want, chosen)
			// Update and cancel-offline selectors do not use Pod deletion cost.
			assert.Equal(t, "older", act.updateSelector.Choose(instances))
			assert.Equal(t, "older", act.cancelOfflineSelector.Choose(instances))
		})
	}
}

func TestBuilderPodDeletionCostScaleIn(t *testing.T) {
	for _, outdated := range []bool{false, true} {
		t.Run(fmt.Sprintf("outdated=%t", outdated), func(t *testing.T) {
			older, newer := fakePD("older", true, true), fakePD("newer", true, true)
			older.CreationTimestamp = metav1.NewTime(time.Unix(100, 0))
			newer.CreationTimestamp = metav1.NewTime(time.Unix(200, 0))
			cli := client.NewFakeClient(runtime.ToPD(older), runtime.ToPD(newer),
				podWithDeletionCost(older, "-10"), podWithDeletionCost(newer, "0"))
			rev := "test"
			if outdated {
				rev = "next"
			}
			act := New[runtime.PDTuple]().WithInstances(older, newer).WithRevision(rev).
				WithClient(cli).Build().(*executor).act.(*actor[runtime.PDTuple, *v1alpha1.PD, *runtime.PD])
			var err error
			if outdated {
				_, err = act.scaleInOutdated(context.Background(), "", false)
			} else {
				_, err = act.ScaleInUpdate(context.Background())
			}
			require.NoError(t, err)
			require.True(t, errors.IsNotFound(cli.Get(context.Background(), client.ObjectKeyFromObject(older), &v1alpha1.PD{})))
			require.NoError(t, cli.Get(context.Background(), client.ObjectKeyFromObject(newer), &v1alpha1.PD{}))
		})
	}
}

func TestPodDeletionCostBelowExcessCapacity(t *testing.T) {
	small, large := fakePD("small", true, true), fakePD("large", true, true)
	meta.SetStatusCondition(&large.Status.Conditions, metav1.Condition{
		Type: v1alpha1.CondVolumeCapacityExceedsRequest, Status: metav1.ConditionTrue,
		Reason: v1alpha1.ReasonCapacityExceedsRequest, ObservedGeneration: large.Generation,
	})
	cli := client.NewFakeClient(podWithDeletionCost(small, "-100"), podWithDeletionCost(large, "100"))
	act := New[runtime.PDTuple]().WithClient(cli).Build().(*executor).act.(*actor[runtime.PDTuple, *v1alpha1.PD, *runtime.PD])
	chosen, err := act.chooseToScaleIn(context.Background(), []*runtime.PD{small, large})
	require.NoError(t, err)
	assert.Equal(t, "large", chosen)
}

func TestPodDeletionCostOnlyReadsPod(t *testing.T) {
	older := fakePD("older", true, true)
	newer := fakePD("newer", true, true)
	older.Annotations = map[string]string{corev1.PodDeletionCost: "-100"}
	newer.Annotations = map[string]string{corev1.PodDeletionCost: "100"}
	olderPod := podWithDeletionCost(older, "10")
	newerPod := podWithDeletionCost(newer, "-10")
	p := newPodDeletionCostPolicy[*runtime.PD]()
	instances := []*runtime.PD{older, newer}
	require.NoError(t, p.load(context.Background(), client.NewFakeClient(olderPod, newerPod), instances))
	assert.Equal(t, []*runtime.PD{newer}, p.Prefer(instances))

	// Without Pod annotations, even valid CR annotations do not break a tie.
	olderPod.Annotations = nil
	newerPod.Annotations = nil
	p = newPodDeletionCostPolicy[*runtime.PD]()
	require.NoError(t, p.load(context.Background(), client.NewFakeClient(olderPod, newerPod), instances))
	assert.Equal(t, instances, p.Prefer(instances))
}

func TestPodDeletionCostReadErrors(t *testing.T) {
	in := fakePD("instance", true, true)
	in.UID = types.UID("instance-uid")
	for _, tc := range []struct {
		name   string
		modify func(client.FakeClient, *corev1.Pod)
	}{
		{name: "list error", modify: func(c client.FakeClient, _ *corev1.Pod) {
			c.WithError("list", "pods", fmt.Errorf("pod read failed"))
		}},
		{name: "wrong owner", modify: func(c client.FakeClient, pod *corev1.Pod) {
			pod.OwnerReferences[0].UID = types.UID("other-uid")
			require.NoError(t, c.Update(context.Background(), pod))
		}},
		{name: "multiple pods", modify: func(c client.FakeClient, pod *corev1.Pod) {
			other := pod.DeepCopy()
			other.Name += "-duplicate"
			require.NoError(t, c.Create(context.Background(), other))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pod := podWithDeletionCost(in, "-10")
			cli := client.NewFakeClient(pod)
			tc.modify(cli, pod)
			act := New[runtime.PDTuple]().WithClient(cli).Build().(*executor).act.(*actor[runtime.PDTuple, *v1alpha1.PD, *runtime.PD])
			chosen, err := act.chooseToScaleIn(context.Background(), []*runtime.PD{in, fakePD("other", true, true)})
			require.Error(t, err)
			assert.Empty(t, chosen)
		})
	}
}

func TestPodDeletionCostSnapshot(t *testing.T) {
	in := fakePD("instance", true, true)
	pod := podWithDeletionCost(in, "-10")
	cli := client.NewFakeClient(pod)
	p := newPodDeletionCostPolicy[*runtime.PD]()
	instances := []*runtime.PD{in}
	require.NoError(t, p.load(context.Background(), cli, instances))
	pod.Annotations[corev1.PodDeletionCost] = "20"
	require.NoError(t, cli.Update(context.Background(), pod))
	require.NoError(t, p.load(context.Background(), cli, instances))
	key := types.NamespacedName{Namespace: in.Namespace, Name: in.Name}
	assert.EqualValues(t, -10, p.costs[key])
	// A new updater execution observes changes made on the Pod.
	p = newPodDeletionCostPolicy[*runtime.PD]()
	require.NoError(t, p.load(context.Background(), cli, instances))
	assert.EqualValues(t, 20, p.costs[key])
}
