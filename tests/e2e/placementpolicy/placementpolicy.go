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

package placementpolicy

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/onsi/ginkgo/v2"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/action"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/desc"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("PlacementPolicy", label.TiKV, label.P1, func() {
	f := framework.New()
	f.Setup()

	ginkgo.It("syncs keyspace placement rules to normal and exclusive TiKV groups", func(ctx context.Context) {
		o := desc.DefaultOptions()
		pdg := action.MustCreatePD(ctx, f, o,
			data.WithReplicas[scope.PDGroup](1),
			data.GroupPatchFunc[*v1alpha1.PDGroup](func(pdg *v1alpha1.PDGroup) {
				pdg.Spec.Template.Spec.Config = `[replication]
enable-placement-rules = true
`
			}),
		)

		normal := action.MustCreateTiKV(ctx, f, o, data.WithName[scope.TiKVGroup]("kvg"))
		exclusive := action.MustCreateTiKV(ctx, f, o,
			data.WithName[scope.TiKVGroup]("kvg-exclusive"),
			data.GroupPatchFunc[*v1alpha1.TiKVGroup](func(kvg *v1alpha1.TiKVGroup) {
				kvg.Spec.Template.Spec.Placement = &v1alpha1.TiKVStorePlacement{
					Exclusive: ptr.To(true),
				}
			}),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, normal)
		f.WaitForTiKVGroupReady(ctx, exclusive)

		policy := action.MustCreatePlacementPolicy(ctx, f,
			data.WithPlacementPolicyName("keyspace-1"),
			data.WithPlacementPolicyTiKVGroups(normal.Name, exclusive.Name),
			data.WithPlacementPolicyKeyspaceRule("voters", 3, "1"),
		)
		ginkgo.By("Waiting for placement policy synced")
		f.Must(waiter.WaitForPlacementPolicySynced(ctx, f.Client, policy, waiter.LongTaskTimeout))

		pdc := newPDClient(ctx, f, pdg)
		ginkgo.By("Checking TiKVGroup placement store labels in PD")
		f.Must(waiter.WaitForStoreLabelValue(ctx, f.Client, pdc, normal, waiter.LongTaskTimeout))
		f.Must(waiter.WaitForStoreLabelValue(ctx, f.Client, pdc, exclusive, waiter.LongTaskTimeout))

		ginkgo.By("Checking placement rules in PD")
		expectedRules := expectedKeyspacePlacementRules(policy.Name, "voters", "1")
		err := waiter.WaitForPlacementRuleExists(ctx, pdc, policy, expectedPlacementRuleBundle(expectedRules...), waiter.LongTaskTimeout)
		f.Must(err)
	})

	ginkgo.It("moves keyspace regions after placement policy update", func(ctx context.Context) {
		o := desc.DefaultOptions()
		pdg := action.MustCreatePD(ctx, f, o,
			data.WithReplicas[scope.PDGroup](1),
			data.GroupPatchFunc[*v1alpha1.PDGroup](func(pdg *v1alpha1.PDGroup) {
				pdg.Spec.Template.Spec.Config = `[replication]
enable-placement-rules = true
max-replicas = 1
`
			}),
		)

		source := action.MustCreateTiKV(ctx, f, o, data.WithName[scope.TiKVGroup]("kvg-source"))
		target := action.MustCreateTiKV(ctx, f, o, data.WithName[scope.TiKVGroup]("kvg-target"))

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, source)
		f.WaitForTiKVGroupReady(ctx, target)

		pdc := newPDClient(ctx, f, pdg)
		keyspaceID := createKeyspace(ctx, f, pdc, "placement-policy-move")
		ginkgo.By("Checking TiKVGroup placement store labels in PD")
		f.Must(waiter.WaitForStoreLabelValue(ctx, f.Client, pdc, source, waiter.LongTaskTimeout))
		f.Must(waiter.WaitForStoreLabelValue(ctx, f.Client, pdc, target, waiter.LongTaskTimeout))

		policy := action.MustCreatePlacementPolicy(ctx, f,
			data.WithPlacementPolicyName("keyspace-move"),
			data.WithPlacementPolicyTiKVGroups(source.Name),
			data.WithPlacementPolicyKeyspaceRule("voters", 1, keyspaceID),
		)
		ginkgo.By("Waiting for placement policy synced")
		f.Must(waiter.WaitForPlacementPolicySynced(ctx, f.Client, policy, waiter.LongTaskTimeout))

		ginkgo.By("Checking placement rules in PD")
		expectedRules := expectedKeyspacePlacementRules(policy.Name, "voters", keyspaceID)
		err := waiter.WaitForPlacementRuleExists(ctx, pdc, policy, expectedPlacementRuleBundle(expectedRules...), waiter.LongTaskTimeout)
		f.Must(err)
		ginkgo.By("Checking keyspace regions scheduling result")
		f.Must(waiter.WaitForKeyspaceRegionsScheduled(ctx, f.Client, pdc, keyspaceID, source, 1, waiter.LongTaskTimeout))

		ginkgo.By("Updating placement policy to target TiKVGroup")
		var latest v1alpha1.PlacementPolicy
		f.Must(f.Client.Get(ctx, k8stypes.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}, &latest))
		latest.Spec.GroupRefs = []v1alpha1.PlacementPolicyGroupRef{
			{
				Group: v1alpha1.GroupName,
				Kind:  "TiKVGroup",
				Name:  target.Name,
			},
		}
		f.Must(f.Client.Update(ctx, &latest))
		ginkgo.By("Waiting for placement policy synced")
		f.Must(waiter.WaitForPlacementPolicySynced(ctx, f.Client, &latest, waiter.LongTaskTimeout))

		ginkgo.By("Checking placement rules in PD")
		expectedRules = expectedKeyspacePlacementRules(latest.Name, "voters", keyspaceID)
		err = waiter.WaitForPlacementRuleExists(ctx, pdc, &latest, expectedPlacementRuleBundle(expectedRules...), waiter.LongTaskTimeout)
		f.Must(err)
		ginkgo.By("Checking keyspace regions scheduling result")
		f.Must(waiter.WaitForKeyspaceRegionsScheduled(ctx, f.Client, pdc, keyspaceID, target, 1, waiter.LongTaskTimeout))
	})

	ginkgo.It("keeps exclusive TiKV groups empty until targeted by placement policy", func(ctx context.Context) {
		o := desc.DefaultOptions()
		pdg := action.MustCreatePD(ctx, f, o,
			data.WithReplicas[scope.PDGroup](1),
			data.GroupPatchFunc[*v1alpha1.PDGroup](func(pdg *v1alpha1.PDGroup) {
				pdg.Spec.Template.Spec.Config = `[replication]
enable-placement-rules = true
max-replicas = 1
`
			}),
		)

		normal := action.MustCreateTiKV(ctx, f, o, data.WithName[scope.TiKVGroup]("kvg-normal"))
		exclusive := action.MustCreateTiKV(ctx, f, o,
			data.WithName[scope.TiKVGroup]("kvg-exclusive"),
			data.GroupPatchFunc[*v1alpha1.TiKVGroup](func(kvg *v1alpha1.TiKVGroup) {
				kvg.Spec.Template.Spec.Placement = &v1alpha1.TiKVStorePlacement{
					Exclusive: ptr.To(true),
				}
			}),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, normal)
		f.WaitForTiKVGroupReady(ctx, exclusive)

		pdc := newPDClient(ctx, f, pdg)
		ginkgo.By("Checking TiKVGroup placement store labels in PD")
		f.Must(waiter.WaitForStoreLabelValue(ctx, f.Client, pdc, normal, waiter.LongTaskTimeout))
		f.Must(waiter.WaitForStoreLabelValue(ctx, f.Client, pdc, exclusive, waiter.LongTaskTimeout))

		keyspaceID := createKeyspace(ctx, f, pdc, "placement-policy-exclusive")
		ginkgo.By("Checking exclusive TiKVGroup store has no regions")
		f.Must(waiter.WaitForStoreRegionCount(ctx, f.Client, pdc, exclusive, 0, 30*time.Second))

		policy := action.MustCreatePlacementPolicy(ctx, f,
			data.WithPlacementPolicyName("keyspace-exclusive"),
			data.WithPlacementPolicyTiKVGroups(exclusive.Name),
			data.WithPlacementPolicyKeyspaceRule("voters", 1, keyspaceID),
		)
		ginkgo.By("Waiting for placement policy synced")
		f.Must(waiter.WaitForPlacementPolicySynced(ctx, f.Client, policy, waiter.LongTaskTimeout))

		ginkgo.By("Checking placement rules in PD")
		expectedRules := expectedKeyspacePlacementRules(policy.Name, "voters", keyspaceID)
		err := waiter.WaitForPlacementRuleExists(ctx, pdc, policy, expectedPlacementRuleBundle(expectedRules...), waiter.LongTaskTimeout)
		f.Must(err)
		ginkgo.By("Checking keyspace regions scheduling result")
		f.Must(waiter.WaitForKeyspaceRegionsScheduled(ctx, f.Client, pdc, keyspaceID, exclusive, 1, waiter.LongTaskTimeout))
	})
})

func createKeyspace(ctx context.Context, f *framework.Framework, pdc pdapi.PDClient, name string) string {
	ginkgo.By(fmt.Sprintf("Creating keyspace %s in PD", name))
	meta, err := waiter.WaitForKeyspaceCreated(ctx, pdc, name, waiter.LongTaskTimeout)
	f.Must(err)
	return strconv.FormatUint(uint64(meta.ID), 10)
}

func newPDClient(ctx context.Context, f *framework.Framework, pdg *v1alpha1.PDGroup) pdapi.PDClient {
	forwardCtx, cancel := context.WithCancel(ctx)
	ginkgo.DeferCleanup(cancel)
	ports := framework.PortForwardGroup[scope.PDGroup](forwardCtx, f, pdg, []string{fmt.Sprintf(":%d", v1alpha1.DefaultPDPortClient)})
	return pdapi.NewPDClient(fmt.Sprintf("http://127.0.0.1:%d", ports[0].Local), 30*time.Second, nil)
}

func expectedPlacementRuleBundle(rules ...pdapi.PlacementRule) *pdapi.PlacementRuleGroupBundle {
	groupID := ""
	if len(rules) != 0 {
		groupID = rules[0].GroupID
	}
	return &pdapi.PlacementRuleGroupBundle{
		ID:    groupID,
		Rules: rules,
	}
}

func expectedKeyspacePlacementRules(policyName, ruleName, keyspaceID string) []pdapi.PlacementRule {
	groupID := coreutil.PlacementPolicyGroupID()
	return []pdapi.PlacementRule{
		{GroupID: groupID, ID: coreutil.PlacementPolicyRuleID(policyName, ruleName, keyspaceID, "txn")},
	}
}
