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
	"cmp"
	"context"
	"slices"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

type ReconcileContext struct {
	context.Context
	Key types.NamespacedName

	PDClient pdm.PDClient
	// this means whether pd is available
	IsAvailable bool
	// This is single truth whether pd is initialized
	Initialized bool
	Healthy     bool
	MemberID    string
	IsLeader    bool

	PD      *v1alpha1.PD
	PDGroup *v1alpha1.PDGroup
	Peers   []*v1alpha1.PD
	Cluster *v1alpha1.Cluster
	Pod     *corev1.Pod

	// ConfigHash stores the hash of **user-specified** config (i.e.`.Spec.Config`),
	// which will be used to determine whether the config has changed.
	// This ensures that our config overlay logic will not restart the tidb cluster unexpectedly.
	ConfigHash string

	// Pod cannot be updated when call DELETE API, so we have to set this field to indicate
	// the underlay pod has been deleting
	PodIsTerminating bool
}

func (ctx *ReconcileContext) PDKey() types.NamespacedName {
	return ctx.Key
}

func (ctx *ReconcileContext) SetPD(pd *v1alpha1.PD) {
	ctx.PD = pd
}

func (ctx *ReconcileContext) GetPD() *v1alpha1.PD {
	return ctx.PD
}

func (ctx *ReconcileContext) ClusterKey() types.NamespacedName {
	return types.NamespacedName{
		Namespace: ctx.PD.Namespace,
		Name:      ctx.PD.Spec.Cluster.Name,
	}
}

func (ctx *ReconcileContext) GetCluster() *v1alpha1.Cluster {
	return ctx.Cluster
}

func (ctx *ReconcileContext) SetCluster(c *v1alpha1.Cluster) {
	ctx.Cluster = c
}

// Pod always uses same namespace and name of PD
func (ctx *ReconcileContext) PodKey() types.NamespacedName {
	return ctx.Key
}

func (ctx *ReconcileContext) GetPod() *corev1.Pod {
	return ctx.Pod
}

func (ctx *ReconcileContext) SetPod(pod *corev1.Pod) {
	ctx.Pod = pod
	if !pod.DeletionTimestamp.IsZero() {
		ctx.PodIsTerminating = true
	}
}

func TaskContextInfoFromPD(ctx *ReconcileContext, cm pdm.PDClientManager) task.Task {
	return task.NameTaskFunc("ContextInfoFromPD", func() task.Result {
		ck := ctx.ClusterKey()
		pc, ok := cm.Get(pdm.PrimaryKey(ck.Namespace, ck.Name))
		if !ok {
			return task.Wait().With("pd client has not been registered yet")
		}

		ctx.PDClient = pc

		if !pc.HasSynced() {
			return task.Complete().With("context without member info is completed, cache of pd info is not synced")
		}

		ctx.Initialized = true

		m, err := pc.Members().Get(ctx.PD.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				return task.Complete().With("context without member info is completed, pd is not initialized")
			}
			return task.Fail().With("cannot get member: %w", err)
		}

		ctx.MemberID = m.ID
		ctx.IsLeader = m.IsLeader

		// set available and trust health info only when member info is valid
		if !m.Invalid {
			ctx.IsAvailable = true
			ctx.Healthy = m.Health
		}

		return task.Complete().With("pd is ready")
	})
}

func TaskContextPeers(ctx *ReconcileContext, c client.Client) task.Task {
	return task.NameTaskFunc("ContextPeers", func() task.Result {
		// TODO: don't get pdg in pd task, move MountClusterClientSecret opt to pd spec
		if len(ctx.PD.OwnerReferences) == 0 {
			return task.Fail().With("pd instance has no owner, this should not happen")
		}
		var pdg v1alpha1.PDGroup
		if err := c.Get(ctx, client.ObjectKey{
			Name:      ctx.PD.OwnerReferences[0].Name, // only one owner now
			Namespace: ctx.PD.Namespace,
		}, &pdg); err != nil {
			return task.Fail().With("cannot find pd group %s: %v", ctx.PD.OwnerReferences[0].Name, err)
		}
		ctx.PDGroup = &pdg

		var pdl v1alpha1.PDList
		if err := c.List(ctx, &pdl, client.InNamespace(ctx.PD.Namespace), client.MatchingLabels{
			v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
			v1alpha1.LabelKeyCluster:   ctx.Cluster.Name,
			v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
		}); err != nil {
			return task.Fail().With("cannot list pd peers: %w", err)
		}

		peers := []*v1alpha1.PD{}
		for i := range pdl.Items {
			peers = append(peers, &pdl.Items[i])
		}
		slices.SortFunc(peers, func(a, b *v1alpha1.PD) int {
			return cmp.Compare(a.Name, b.Name)
		})
		ctx.Peers = peers
		return task.Complete().With("peers is set")
	})
}

func CondPDClientIsNotRegisterred(ctx *ReconcileContext) task.Condition {
	return task.CondFunc(func() bool {
		// TODO: do not use HasSynced twice, it may return different results
		return ctx.PDClient == nil || !ctx.PDClient.HasSynced()
	})
}
