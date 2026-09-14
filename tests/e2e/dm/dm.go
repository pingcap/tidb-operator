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

package dm

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

const changedConfig = `log-level = 'warn'`

var _ = ginkgo.Describe("DM", label.DM, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("Basic Lifecycle", label.P0, func() {
		ginkgo.It("applies PVC overlays to built-in and additional volumes", label.KindBasic, func(ctx context.Context) {
			const (
				labelKey      = "test.pingcap.com/volume"
				annotationKey = "test.pingcap.com/volume"
				extraVolume   = "extra"
			)
			newOverlay := func(names ...string) *v1alpha1.Overlay {
				overlay := &v1alpha1.Overlay{}
				for _, name := range names {
					overlay.PersistentVolumeClaims = append(overlay.PersistentVolumeClaims, v1alpha1.NamedPersistentVolumeClaimOverlay{
						Name: name,
						PersistentVolumeClaim: v1alpha1.PersistentVolumeClaimOverlay{
							ObjectMeta: v1alpha1.ObjectMeta{
								Labels:      map[string]string{labelKey: name},
								Annotations: map[string]string{annotationKey: name},
							},
						},
					})
				}
				return overlay
			}
			extra := v1alpha1.Volume{
				Name:    extraVolume,
				Storage: resource.MustParse("1Gi"),
				Mounts:  []v1alpha1.VolumeMount{{MountPath: "/extra"}},
			}

			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dmg := f.MustCreateDM(ctx, data.GroupPatchFunc[*v1alpha1.DMGroup](func(obj *v1alpha1.DMGroup) {
				obj.Spec.Template.Spec.Volumes = []v1alpha1.Volume{extra}
				obj.Spec.Template.Spec.Overlay = newOverlay(obj.Spec.Template.Spec.DataVolume.Name, extraVolume)
			}))
			dwg := f.MustCreateDMWorker(ctx, data.GroupPatchFunc[*v1alpha1.DMWorkerGroup](func(obj *v1alpha1.DMWorkerGroup) {
				obj.Spec.Template.Spec.Volumes = []v1alpha1.Volume{extra}
				obj.Spec.Template.Spec.Overlay = newOverlay(obj.Spec.Template.Spec.RelayVolume.Name, extraVolume)
			}))

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForDMGroupReady(ctx, dmg)
			f.WaitForDMWorkerGroupReady(ctx, dwg)

			// Check persisted PVCs so both CRD admission and controller propagation are covered.
			ginkgo.By("Checking labels and annotations on built-in and additional PVCs")
			for _, group := range []struct {
				name      string
				component string
				builtIn   string
			}{
				{dmg.Name, v1alpha1.LabelValComponentDMMaster, dmg.Spec.Template.Spec.DataVolume.Name},
				{dwg.Name, v1alpha1.LabelValComponentDMWorker, dwg.Spec.Template.Spec.RelayVolume.Name},
			} {
				var pvcs corev1.PersistentVolumeClaimList
				f.Must(f.Client.List(ctx, &pvcs, client.InNamespace(f.Namespace.Name), client.MatchingLabels{
					v1alpha1.LabelKeyGroup:     group.name,
					v1alpha1.LabelKeyComponent: group.component,
				}))
				var names []string
				for _, pvc := range pvcs.Items {
					name := pvc.Labels[v1alpha1.LabelKeyVolumeName]
					names = append(names, name)
					gomega.Expect(pvc.Labels).To(gomega.HaveKeyWithValue(labelKey, name), pvc.Name)
					gomega.Expect(pvc.Annotations).To(gomega.HaveKeyWithValue(annotationKey, name), pvc.Name)
					gomega.Expect(pvc.Status.Phase).To(gomega.Equal(corev1.ClaimBound), pvc.Name)
				}
				gomega.Expect(names).To(gomega.ConsistOf(group.builtIn, extraVolume), group.name)
			}
		})

		ginkgo.It("deploys and reaches Ready state", label.KindBasic, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx)
			dmg := f.MustCreateDM(ctx)
			dwg := f.MustCreateDMWorker(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForDMGroupReady(ctx, dmg)
			f.WaitForDMWorkerGroupReady(ctx, dwg)
		})

		ginkgo.It("deletes groups and owned resources", label.Delete, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dmg := f.MustCreateDM(ctx)
			dwg := f.MustCreateDMWorker(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForDMGroupReady(ctx, dmg)
			f.WaitForDMWorkerGroupReady(ctx, dwg)

			ginkgo.By("Delete DMWorkerGroup")
			f.Must(f.Client.Delete(ctx, dwg))
			f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, dwg, waiter.LongTaskTimeout))
			f.Must(waiter.WaitForPodsDeleted[scope.DMWorkerGroup](ctx, f.Client, dwg, waiter.LongTaskTimeout))
			f.Must(waiter.WaitForListDeleted(ctx, f.Client, &corev1.ConfigMapList{}, waiter.LongTaskTimeout, client.InNamespace(dwg.Namespace), client.MatchingLabels{
				v1alpha1.LabelKeyCluster:   dwg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     dwg.Name,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentDMWorker,
			}))

			ginkgo.By("Delete DMGroup")
			f.Must(f.Client.Delete(ctx, dmg))
			f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, dmg, waiter.LongTaskTimeout))
			f.Must(waiter.WaitForPodsDeleted[scope.DMGroup](ctx, f.Client, dmg, waiter.LongTaskTimeout))
			f.Must(waiter.WaitForListDeleted(ctx, f.Client, &corev1.ConfigMapList{}, waiter.LongTaskTimeout, client.InNamespace(dmg.Namespace), client.MatchingLabels{
				v1alpha1.LabelKeyCluster:   dmg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     dmg.Name,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentDMMaster,
			}))
		})
	})

	ginkgo.Context("Scale and Update", label.P0, func() {
		ginkgo.It("supports scaling DMWorkerGroup out and in", label.Scale, label.DMWorker, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dmg := f.MustCreateDM(ctx)
			dwg := f.MustCreateDMWorker(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForDMGroupReady(ctx, dmg)
			f.WaitForDMWorkerGroupReady(ctx, dwg)

			ginkgo.By("Scale DMWorkerGroup to 2")
			patch := client.MergeFrom(dwg.DeepCopy())
			dwg.Spec.Replicas = ptr.To[int32](2)
			f.Must(f.Client.Patch(ctx, dwg, patch))
			f.WaitForDMWorkerGroupReady(ctx, dwg)

			ginkgo.By("Scale DMWorkerGroup back to 1")
			patch = client.MergeFrom(dwg.DeepCopy())
			dwg.Spec.Replicas = ptr.To[int32](1)
			f.Must(f.Client.Patch(ctx, dwg, patch))
			f.WaitForDMWorkerGroupReady(ctx, dwg)
		})

		ginkgo.It("supports rolling update of DMGroup", label.Update, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dmg := f.MustCreateDM(ctx, data.WithReplicas[scope.DMGroup](3))
			dwg := f.MustCreateDMWorker(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForDMGroupReady(ctx, dmg)
			f.WaitForDMWorkerGroupReady(ctx, dwg)

			nctx, cancel := context.WithCancel(ctx)
			done := framework.AsyncWaitPodsRollingUpdateOnce[scope.DMGroup](nctx, f, dmg, 3)
			defer func() { <-done }()
			defer cancel()

			changeTime, err := waiter.MaxPodsCreateTimestamp[scope.DMGroup](ctx, f.Client, dmg)
			f.Must(err)

			ginkgo.By("Change config of the DMGroup")
			patch := client.MergeFrom(dmg.DeepCopy())
			dmg.Spec.Template.Spec.Config = changedConfig
			f.Must(f.Client.Patch(ctx, dmg, patch))

			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromDMGroup(dmg), *changeTime, waiter.LongTaskTimeout))
			f.WaitForDMGroupReady(ctx, dmg)
		})
	})
})
