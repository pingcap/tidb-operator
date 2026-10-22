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

package common_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/resourcemanager"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/router"
	"github.com/pingcap/tidb-operator/v2/pkg/metrics"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
)

func TestDeletingInstanceObservesMetricsBeforeFinalizerFailure(t *testing.T) {
	for _, component := range []string{v1alpha1.LabelValComponentRouter, v1alpha1.LabelValComponentResourceManager} {
		for _, failure := range []string{"list", "delete", "update"} {
			t.Run(component+"/"+failure, func(t *testing.T) {
				ctx := context.Background()
				key := types.NamespacedName{Namespace: "delete-metrics", Name: component + "-" + failure}
				var obj client.Object
				var instance runtime.Instance
				groupKind := "RouterGroup"
				if component == v1alpha1.LabelValComponentRouter {
					routerObj := &v1alpha1.Router{}
					obj, instance = routerObj, runtime.FromRouter(routerObj)
				} else {
					rmObj := &v1alpha1.ResourceManager{}
					obj, instance = rmObj, runtime.FromResourceManager(rmObj)
					groupKind = "ResourceManagerGroup"
				}
				obj.SetNamespace(key.Namespace)
				obj.SetName(key.Name)
				obj.SetUID(types.UID(key.Name))
				now := metav1.Now()
				obj.SetDeletionTimestamp(&now)
				obj.SetFinalizers([]string{metav1alpha1.Finalizer})
				obj.SetLabels(map[string]string{
					v1alpha1.LabelKeyCluster: "c", v1alpha1.LabelKeyComponent: component, v1alpha1.LabelKeyGroup: "g",
				})
				controller := true
				obj.SetOwnerReferences([]metav1.OwnerReference{{
					APIVersion: "core.pingcap.com/v1alpha1", Kind: groupKind, Name: "g", Controller: &controller,
				}})
				instance.SetCluster("c")
				// Observation of existing failures must not require reading the Cluster.
				instance.SetConditions([]metav1.Condition{
					{Type: v1alpha1.CondReady, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonPodNotReady},
					{Type: v1alpha1.CondSynced, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonPodNotDeleted},
				})
				metrics.ClearInstanceConditionMetrics(obj) // Model an empty registry after restart.
				t.Cleanup(func() { metrics.ClearInstanceConditionMetrics(obj) })
				objs := []client.Object{obj}
				resource := "pods"
				if failure == "delete" {
					objs = append(objs, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
						Namespace: key.Namespace, Name: "pod", Labels: map[string]string{
							v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
							v1alpha1.LabelKeyCluster:   "c", v1alpha1.LabelKeyComponent: component,
							v1alpha1.LabelKeyInstance: key.Name,
						},
					}})
				}
				if failure == "update" {
					resource = "*"
				}
				c := client.NewFakeClient(objs...)
				c.WithError(failure, resource, errors.NewInternalError(fmt.Errorf("injected finalizer failure")))
				var err error
				if component == v1alpha1.LabelValComponentRouter {
					reconciler := &router.Reconciler{Client: c, Tracker: tracker.New().Tracker(component)}
					_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key})
				} else {
					reconciler := &resourcemanager.Reconciler{Client: c, Tracker: tracker.New().Tracker(component)}
					_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key})
				}
				require.ErrorContains(t, err, "injected finalizer failure")
				values := deletingInstanceMetricValues(t, key)
				for _, condition := range []string{v1alpha1.CondReady, v1alpha1.CondSynced} {
					value, present := values[condition]
					require.True(t, present, "%s series must exist even when finalizer cleanup fails", condition)
					assert.InDelta(t, float64(1), value, 1e-9)
				}
				stored := obj.DeepCopyObject().(client.Object)
				require.NoError(t, c.Get(ctx, key, stored))
				assert.Contains(t, stored.GetFinalizers(), metav1alpha1.Finalizer)
			})
		}
	}
}

func deletingInstanceMetricValues(t *testing.T, key types.NamespacedName) map[string]float64 {
	t.Helper()
	ch := make(chan prometheus.Metric, 64)
	metrics.AbnormalInstance.Collect(ch)
	close(ch)
	values := map[string]float64{}
	for sample := range ch {
		metric := &dto.Metric{}
		require.NoError(t, sample.Write(metric))
		labels := map[string]string{}
		for _, label := range metric.Label {
			labels[label.GetName()] = label.GetValue()
		}
		if labels["namespace"] == key.Namespace && labels["instance"] == key.Name {
			values[labels["condition"]] = metric.GetGauge().GetValue()
		}
	}
	return values
}
