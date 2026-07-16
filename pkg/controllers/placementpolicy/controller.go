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
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/keyspace"
)

const (
	pdPlacementRuleGroupIndex = 99
	pdDefaultRuleGroupID      = "pd"
	pdDefaultRuleID           = "default"

	reasonSynced     = "Synced"
	reasonSyncFailed = "SyncFailed"

	// PD excludes stores with $-prefixed labels from rules that do not reference
	// that label key. Referencing $k/exclusive with notIn on an intentionally
	// nonexistent value makes exclusive stores eligible without filtering out
	// normal stores or stores labeled $k/exclusive=true.
	placementExclusiveConstraintNonexistentValue = "__null__"
)

type Reconciler struct {
	Logger          logr.Logger
	Client          client.Client
	PDClientManager pdm.PDClientManager
}

func Setup(mgr manager.Manager, c client.Client, pdcm pdm.PDClientManager) error {
	r := &Reconciler{
		Logger:          mgr.GetLogger().WithName("PlacementPolicy"),
		Client:          c,
		PDClientManager: pdcm,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.PlacementPolicy{}).
		Watches(&v1alpha1.Cluster{}, r.ClusterEventHandler()).
		Watches(&v1alpha1.TiKVGroup{}, r.TiKVGroupEventHandler()).
		WithOptions(controller.Options{RateLimiter: k8s.RateLimiter}).
		Complete(r)
}

func (r *Reconciler) ClusterEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return handler.TypedFuncs[client.Object, reconcile.Request]{
		CreateFunc: func(ctx context.Context, event event.TypedCreateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			cluster := event.Object.(*v1alpha1.Cluster)
			r.enqueuePoliciesForCluster(ctx, queue, cluster.Namespace, cluster.Name)
		},
		UpdateFunc: func(ctx context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			oldCluster := event.ObjectOld.(*v1alpha1.Cluster)
			cluster := event.ObjectNew.(*v1alpha1.Cluster)
			if oldCluster.Spec.Paused == cluster.Spec.Paused &&
				reflect.DeepEqual(oldCluster.Spec.SuspendAction, cluster.Spec.SuspendAction) {
				return
			}
			r.enqueuePoliciesForCluster(ctx, queue, cluster.Namespace, cluster.Name)
		},
		DeleteFunc: func(ctx context.Context, event event.TypedDeleteEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			cluster := event.Object.(*v1alpha1.Cluster)
			r.enqueuePoliciesForCluster(ctx, queue, cluster.Namespace, cluster.Name)
		},
	}
}

func (r *Reconciler) TiKVGroupEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return handler.TypedFuncs[client.Object, reconcile.Request]{
		CreateFunc: func(ctx context.Context, event event.TypedCreateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			kvg := event.Object.(*v1alpha1.TiKVGroup)
			r.enqueuePoliciesForTiKVGroup(ctx, queue, kvg)
		},
		UpdateFunc: func(ctx context.Context, event event.TypedUpdateEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			oldKVG := event.ObjectOld.(*v1alpha1.TiKVGroup)
			kvg := event.ObjectNew.(*v1alpha1.TiKVGroup)
			if coreutil.TiKVStorePlacementExclusive(oldKVG.Spec.Template.Spec.Placement) ==
				coreutil.TiKVStorePlacementExclusive(kvg.Spec.Template.Spec.Placement) {
				return
			}

			r.enqueuePoliciesForTiKVGroup(ctx, queue, kvg)
		},
		DeleteFunc: func(ctx context.Context, event event.TypedDeleteEvent[client.Object],
			queue workqueue.TypedRateLimitingInterface[reconcile.Request],
		) {
			kvg := event.Object.(*v1alpha1.TiKVGroup)
			r.enqueuePoliciesForTiKVGroup(ctx, queue, kvg)
		},
	}
}

func (r *Reconciler) enqueuePoliciesForCluster(
	ctx context.Context,
	queue workqueue.TypedRateLimitingInterface[reconcile.Request],
	namespace, clusterName string,
) {
	var list v1alpha1.PlacementPolicyList
	if err := r.Client.List(ctx, &list,
		client.InNamespace(namespace),
		client.MatchingFields{"spec.cluster.name": clusterName},
	); err != nil {
		if !apierrors.IsNotFound(err) {
			r.Logger.Error(err, "cannot list placement policies", "ns", namespace, "cluster", clusterName)
		}
		return
	}

	for i := range list.Items {
		policy := &list.Items[i]
		queue.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      policy.Name,
				Namespace: policy.Namespace,
			},
		})
	}
}

func (r *Reconciler) enqueuePoliciesForTiKVGroup(
	ctx context.Context,
	queue workqueue.TypedRateLimitingInterface[reconcile.Request],
	kvg *v1alpha1.TiKVGroup,
) {
	var list v1alpha1.PlacementPolicyList
	if err := r.Client.List(ctx, &list,
		client.InNamespace(kvg.Namespace),
		client.MatchingFields{"spec.cluster.name": kvg.Spec.Cluster.Name},
	); err != nil {
		if !apierrors.IsNotFound(err) {
			r.Logger.Error(err, "cannot list placement policies", "ns", kvg.Namespace, "cluster", kvg.Spec.Cluster.Name)
		}
		return
	}

	for i := range list.Items {
		policy := &list.Items[i]
		if !policyReferencesTiKVGroup(policy, kvg.Name) {
			continue
		}
		queue.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      policy.Name,
				Namespace: policy.Namespace,
			},
		})
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logr.FromContextOrDiscard(ctx)
	logger.Info("start reconcile")
	startTime := time.Now()
	defer func() {
		logger.Info("end reconcile", "duration", time.Since(startTime))
	}()

	policy, err := apicall.Get[v1alpha1.PlacementPolicy](ctx, r.Client, req.Namespace, req.Name)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !policy.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.reconcileDelete(ctx, policy)
	}

	if err := apicall.EnsureFinalizer(ctx, r.Client, policy); err != nil {
		return ctrl.Result{}, err
	}

	cluster, err := r.policyCluster(ctx, policy)
	if err != nil {
		return ctrl.Result{}, r.updateStatus(ctx, policy, err)
	}
	if clusterPlacementPolicySyncDisabled(cluster) {
		logger.Info("skip placement policy sync as cluster is paused or suspending",
			"paused", coreutil.ShouldPauseReconcile(cluster),
			"suspendCompute", coreutil.ShouldSuspendCompute(cluster))
		return ctrl.Result{}, nil
	}

	pdc, ok := r.PDClientManager.Get(timanager.PrimaryKey(policy.Namespace, policy.Spec.Cluster.Name))
	if !ok {
		err := fmt.Errorf("pd client is not ready")
		return ctrl.Result{}, r.updateStatus(ctx, policy, err)
	}

	expected, err := r.buildRules(ctx, policy)
	if err != nil {
		return ctrl.Result{}, r.updateStatus(ctx, policy, fmt.Errorf("failed to build placement rules: %w", err))
	}

	underlay := pdc.Underlay()
	rulePrefix := coreutil.PlacementPolicyRuleIDPrefix(policy.Name)
	if err := syncPolicyRules(ctx, underlay, rulePrefix, expected); err != nil {
		return ctrl.Result{}, r.updateStatus(ctx, policy, err)
	}

	return ctrl.Result{}, r.updateStatus(ctx, policy, nil)
}

func (r *Reconciler) reconcileDelete(ctx context.Context, policy *v1alpha1.PlacementPolicy) error {
	cluster, deletingOrGone, err := r.clusterForPolicyDeletion(ctx, policy)
	if err != nil {
		return err
	}
	if deletingOrGone {
		return apicall.RemoveFinalizer(ctx, r.Client, policy)
	}
	if clusterPlacementPolicySyncDisabled(cluster) {
		return nil
	}

	pdc, ok := r.PDClientManager.Get(timanager.PrimaryKey(policy.Namespace, policy.Spec.Cluster.Name))
	if ok {
		if err := deletePolicyRules(ctx, pdc.Underlay(), policy); err != nil {
			return fmt.Errorf("delete placement policy rules failed: %w", err)
		}
		return apicall.RemoveFinalizer(ctx, r.Client, policy)
	}

	return fmt.Errorf("pd client is not ready")
}

func (r *Reconciler) policyCluster(ctx context.Context, policy *v1alpha1.PlacementPolicy) (*v1alpha1.Cluster, error) {
	cluster, err := apicall.Get[v1alpha1.Cluster](ctx, r.Client, policy.Namespace, policy.Spec.Cluster.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("referenced cluster %q does not exist", policy.Spec.Cluster.Name)
		}
		return nil, err
	}

	return cluster, nil
}

func (r *Reconciler) clusterForPolicyDeletion(ctx context.Context, policy *v1alpha1.PlacementPolicy) (*v1alpha1.Cluster, bool, error) {
	cluster, err := apicall.Get[v1alpha1.Cluster](ctx, r.Client, policy.Namespace, policy.Spec.Cluster.Name)
	if apierrors.IsNotFound(err) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}

	return cluster, !cluster.DeletionTimestamp.IsZero(), nil
}

func clusterPlacementPolicySyncDisabled(cluster *v1alpha1.Cluster) bool {
	return coreutil.ShouldPauseReconcile(cluster) || coreutil.ShouldSuspendCompute(cluster)
}

func deletePolicyRules(ctx context.Context, pdc pdapi.PDClient, policy *v1alpha1.PlacementPolicy) error {
	rulePrefix := coreutil.PlacementPolicyRuleIDPrefix(policy.Name)
	return pdc.DeletePlacementRuleGroupRulesByIDPrefix(ctx, coreutil.PlacementPolicyGroupID(), rulePrefix)
}

func (r *Reconciler) buildRules(ctx context.Context, policy *v1alpha1.PlacementPolicy) ([]pdapi.PlacementRule, error) {
	targets, err := r.resolveTargets(ctx, policy)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, nil
	}

	rules := []pdapi.PlacementRule{}
	for _, rule := range policy.Spec.Rules {
		if rule.Selector.Keyspace == nil {
			return nil, fmt.Errorf("rule %q selector.keyspace is required", rule.Name)
		}

		for _, keyspaceID := range rule.Selector.Keyspace.IDs {
			keyRanges, err := keyspace.MakeRegionBound(keyspaceID)
			if err != nil {
				return nil, err
			}
			groupID := coreutil.PlacementPolicyGroupID()
			for _, keyRange := range keyRanges {
				rules = append(rules, pdapi.PlacementRule{
					GroupID:     groupID,
					ID:          coreutil.PlacementPolicyRuleID(policy.Name, rule.Name, keyspaceID, keyRange.Type),
					StartKeyHex: keyRange.StartKeyHex,
					EndKeyHex:   keyRange.EndKeyHex,
					Role:        rule.Role,
					Count:       rule.Count,
					LabelConstraints: []pdapi.PlacementLabelConstraint{
						{
							Key:    v1alpha1.PlacementTiKVGroupLabelKey,
							Op:     "in",
							Values: targets,
						},
						{
							Key:    v1alpha1.PlacementTiKVGroupExclusiveLabelKey,
							Op:     "notIn",
							Values: []string{placementExclusiveConstraintNonexistentValue},
						},
					},
				})
			}
		}
	}

	slices.SortFunc(rules, comparePlacementRule)

	return rules, nil
}

func (r *Reconciler) resolveTargets(ctx context.Context, policy *v1alpha1.PlacementPolicy) ([]string, error) {
	targets := []string{}
	for _, ref := range policy.Spec.GroupRefs {
		if ref.Group != v1alpha1.GroupName {
			return nil, fmt.Errorf("unsupported groupRef group %s", ref.Group)
		}
		if ref.Kind != "TiKVGroup" {
			return nil, fmt.Errorf("unsupported groupRef kind %s", ref.Kind)
		}

		kvg, err := apicall.Get[v1alpha1.TiKVGroup](ctx, r.Client, policy.Namespace, ref.Name)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("referenced TiKVGroup %q does not exist", ref.Name)
			}

			return nil, err
		}
		if kvg.Spec.Cluster.Name != policy.Spec.Cluster.Name {
			return nil, fmt.Errorf(
				"referenced TiKVGroup %q belongs to cluster %q, not %q",
				ref.Name,
				kvg.Spec.Cluster.Name,
				policy.Spec.Cluster.Name,
			)
		}

		targets = append(
			targets,
			coreutil.PlacementTiKVGroupLabelValue(policy.Namespace, ref.Name),
		)
	}
	slices.Sort(targets)

	return targets, nil
}

func syncPolicyRules(
	ctx context.Context,
	pdc pdapi.PDClient,
	rulePrefix string,
	rules []pdapi.PlacementRule,
) error {
	if len(rules) > 0 {
		locationLabels, err := defaultRuleLocationLabels(ctx, pdc)
		if err != nil {
			return err
		}
		setPlacementRulesLocationLabels(rules, locationLabels)
	}

	if err := pdc.SetPlacementRuleGroup(ctx, placementRuleGroup()); err != nil {
		return err
	}
	return pdc.SetPlacementRuleGroupRulesByIDPrefix(ctx, coreutil.PlacementPolicyGroupID(), rulePrefix, rules)
}

func defaultRuleLocationLabels(ctx context.Context, pdc pdapi.PDClient) ([]string, error) {
	bundle, err := pdc.GetPlacementRuleBundle(ctx, pdDefaultRuleGroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default placement rule bundle: %w", err)
	}
	if bundle == nil {
		return nil, fmt.Errorf("default placement rule bundle %q is nil", pdDefaultRuleGroupID)
	}

	for i := range bundle.Rules {
		if bundle.Rules[i].ID == pdDefaultRuleID {
			return bundle.Rules[i].LocationLabels, nil
		}
	}

	// not found, return empty location labels
	return nil, nil
}

func setPlacementRulesLocationLabels(rules []pdapi.PlacementRule, locationLabels []string) {
	if len(locationLabels) == 0 {
		locationLabels = nil
	}

	for i := range rules {
		rules[i].LocationLabels = locationLabels
	}
}

func placementRuleGroup() *pdapi.PlacementRuleGroup {
	return &pdapi.PlacementRuleGroup{
		ID:       coreutil.PlacementPolicyGroupID(),
		Index:    pdPlacementRuleGroupIndex,
		Override: true,
	}
}

func comparePlacementRule(a, b pdapi.PlacementRule) int { //nolint:gocritic // slices.SortFunc requires value parameters.
	for _, c := range []int{
		strings.Compare(a.GroupID, b.GroupID),
		strings.Compare(a.ID, b.ID),
		strings.Compare(a.StartKeyHex, b.StartKeyHex),
		strings.Compare(a.EndKeyHex, b.EndKeyHex),
		strings.Compare(a.Role, b.Role),
	} {
		if c != 0 {
			return c
		}
	}
	if a.Count < b.Count {
		return -1
	}
	if a.Count > b.Count {
		return 1
	}
	if c := slices.Compare(a.LocationLabels, b.LocationLabels); c != 0 {
		return c
	}

	return slices.CompareFunc(a.LabelConstraints, b.LabelConstraints, comparePlacementLabelConstraint)
}

func comparePlacementLabelConstraint(a, b pdapi.PlacementLabelConstraint) int {
	for _, c := range []int{
		strings.Compare(a.Key, b.Key),
		strings.Compare(a.Op, b.Op),
		slices.Compare(a.Values, b.Values),
	} {
		if c != 0 {
			return c
		}
	}

	return 0
}

func (r *Reconciler) updateStatus(ctx context.Context, policy *v1alpha1.PlacementPolicy, err error) error {
	status := metav1.ConditionTrue
	reason := reasonSynced
	message := "Placement policy synced"
	if err != nil {
		status = metav1.ConditionFalse
		reason = reasonSyncFailed
		message = err.Error()
	}

	next := policy.DeepCopy()
	next.Status.ObservedGeneration = next.Generation
	apimeta.SetStatusCondition(&next.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.CondSynced,
		Status:             status,
		ObservedGeneration: next.Generation,
		Reason:             reason,
		Message:            message,
	})
	if reflect.DeepEqual(policy.Status, next.Status) {
		return err
	}
	if updateErr := r.Client.Status().Update(ctx, next); updateErr != nil {
		return errors.Join(err, fmt.Errorf("failed to update placement policy status: %w", updateErr))
	}
	return err
}

func policyReferencesTiKVGroup(policy *v1alpha1.PlacementPolicy, name string) bool {
	for _, ref := range policy.Spec.GroupRefs {
		if ref.Group == v1alpha1.GroupName &&
			ref.Kind == "TiKVGroup" &&
			ref.Name == name {
			return true
		}
	}
	return false
}
