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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/pkg/utils/tracker"
)

const (
	testOldRevision = "old-rev-123"
	testNewRevision = "new-rev-456"
	testNamespace   = "test-ns"
	testCluster     = "test-cluster"
	testKVGroup     = "test-kvgroup"
)

func TestTaskUpdater_ScaleDown(t *testing.T) {
	cases := []struct {
		desc            string
		initialReplicas int
		desiredReplicas int
		existingTiKVs   []*v1alpha1.TiKV
		annotations     map[string]map[string]string // map[tikvName]map[annotationKey]annotationValue
		expectedOffline []string                     // names of TiKVs that should be marked offline
		expectedStatus  task.Status
	}{
		{
			desc:            "scale in from 3 to 1, no annotations",
			initialReplicas: 3,
			desiredReplicas: 1,
			existingTiKVs: []*v1alpha1.TiKV{
				createTestTiKV("tikv-0", false, ""),
				createTestTiKV("tikv-1", false, ""),
				createTestTiKV("tikv-2", false, ""),
			},
			expectedOffline: []string{"tikv-0", "tikv-1"}, // First 2 selected by default
			expectedStatus:  task.SWait,
		},
		{
			desc:            "scale in from 3 to 1, with deletion annotation",
			initialReplicas: 3,
			desiredReplicas: 1,
			existingTiKVs: []*v1alpha1.TiKV{
				createTestTiKV("tikv-0", false, ""),
				createTestTiKV("tikv-1", false, ""),
				createTestTiKV("tikv-2", false, ""),
			},
			annotations: map[string]map[string]string{
				"tikv-2": {v1alpha1.ScaleInDeleteAnnotation: "true"},
			},
			expectedOffline: []string{"tikv-2", "tikv-0"}, // Annotated instance preferred
			expectedStatus:  task.SWait,
		},
		{
			desc:            "scale in from 3 to 2, one already offline",
			initialReplicas: 3,
			desiredReplicas: 2,
			existingTiKVs: []*v1alpha1.TiKV{
				createTestTiKV("tikv-0", false, ""),
				createTestTiKV("tikv-1", true, v1alpha1.OfflineReasonActive),
				createTestTiKV("tikv-2", false, ""),
			},
			expectedOffline: nil, // No new instances should be marked offline
			expectedStatus:  task.SWait,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			// Setup test environment
			kvg := createTestTiKVGroup(tc.desiredReplicas)
			cluster := createTestCluster()

			// Apply annotations if specified
			for tikvName, annotations := range tc.annotations {
				for _, tikv := range tc.existingTiKVs {
					if tikv.Name == tikvName {
						if tikv.Annotations == nil {
							tikv.Annotations = make(map[string]string)
						}
						for k, v := range annotations {
							tikv.Annotations[k] = v
						}
					}
				}
			}

			objs := []client.Object{kvg, cluster}
			for _, tikv := range tc.existingTiKVs {
				objs = append(objs, tikv)
			}

			cli := client.NewFakeClient(objs...)
			state := createTestReconcileContext(kvg, cluster, tc.existingTiKVs)
			testTracker := tracker.New[*v1alpha1.TiKVGroup, *v1alpha1.TiKV]()

			for _, tikv := range tc.existingTiKVs {
				require.NoError(t, cli.Apply(context.Background(), tikv))
			}

			// Execute the two-step updater task
			taskFunc := TaskUpdater(state, cli, testTracker)
			result, _ := task.RunTask(context.Background(), taskFunc)

			assert.Equal(t, tc.expectedStatus, result.Status(), "unexpected task status")

			// Verify that the expected instances are marked offline
			for _, expectedOfflineName := range tc.expectedOffline {
				var foundTiKV v1alpha1.TiKV
				err := cli.Get(context.Background(), client.ObjectKey{
					Namespace: testNamespace,
					Name:      expectedOfflineName,
				}, &foundTiKV)
				require.NoError(t, err)
				assert.True(t, foundTiKV.Spec.Offline, "TiKV %s should be marked offline", expectedOfflineName)
			}

			// Verify that other instances are not marked offline
			for _, tikv := range tc.existingTiKVs {
				shouldBeOffline := false
				for _, name := range tc.expectedOffline {
					if tikv.Name == name {
						shouldBeOffline = true
						break
					}
				}
				if !shouldBeOffline && !tikv.Spec.Offline {
					var foundTiKV v1alpha1.TiKV
					err := cli.Get(context.Background(), client.ObjectKey{
						Namespace: testNamespace,
						Name:      tikv.Name,
					}, &foundTiKV)
					require.NoError(t, err)
					assert.False(t, foundTiKV.Spec.Offline, "TiKV %s should not be marked offline", tikv.Name)
				}
			}
		})
	}
}

func TestTaskUpdater_ScaleUpCancellation(t *testing.T) {
	cases := []struct {
		desc            string
		initialReplicas int
		desiredReplicas int
		existingTiKVs   []*v1alpha1.TiKV
		expectedCancel  []string // names of TiKVs that should have offline canceled
		expectedStatus  task.Status
	}{
		{
			desc:            "full cancellation: scale from 1 to 3, cancel 2 offline instances",
			initialReplicas: 1,
			desiredReplicas: 3,
			existingTiKVs: []*v1alpha1.TiKV{
				createTestTiKV("tikv-0", false, ""),
				createTestTiKV("tikv-1", true, v1alpha1.OfflineReasonActive),
				createTestTiKV("tikv-2", true, v1alpha1.OfflineReasonActive),
			},
			expectedCancel: []string{"tikv-1", "tikv-2"},
			expectedStatus: task.SWait,
		},
		{
			desc:            "partial cancellation: scale from 1 to 2, cancel 1 of 2 offline instances",
			initialReplicas: 1,
			desiredReplicas: 2,
			existingTiKVs: []*v1alpha1.TiKV{
				createTestTiKV("tikv-0", false, ""),
				createTestTiKV("tikv-1", true, v1alpha1.OfflineReasonActive),
				createTestTiKV("tikv-2", true, v1alpha1.OfflineReasonActive),
			},
			expectedCancel: []string{"tikv-1"}, // Only cancel 1 of the 2 offline instances
			expectedStatus: task.SWait,
		},
		{
			desc:            "pure scale out: no offline instances to cancel",
			initialReplicas: 2,
			desiredReplicas: 3,
			existingTiKVs: []*v1alpha1.TiKV{
				createTestTiKV("tikv-0", false, ""),
				createTestTiKV("tikv-1", false, ""),
			},
			expectedCancel: nil,
			expectedStatus: task.SWait, // Will trigger scale out logic which requires waiting
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			// Setup test environment
			kvg := createTestTiKVGroup(tc.desiredReplicas)
			cluster := createTestCluster()

			objs := []client.Object{kvg, cluster}
			for _, tikv := range tc.existingTiKVs {
				objs = append(objs, tikv)
			}

			cli := client.NewFakeClient(objs...)
			state := createTestReconcileContext(kvg, cluster, tc.existingTiKVs)
			testTracker := tracker.New[*v1alpha1.TiKVGroup, *v1alpha1.TiKV]()

			for _, tikv := range tc.existingTiKVs {
				require.NoError(t, cli.Apply(context.Background(), tikv))
			}

			// Execute the two-step updater task
			taskFunc := TaskUpdater(state, cli, testTracker)
			result, _ := task.RunTask(context.Background(), taskFunc)

			assert.Equal(t, tc.expectedStatus, result.Status(), "unexpected task status")

			// Verify that the expected instances have offline canceled
			for _, expectedCancelName := range tc.expectedCancel {
				var foundTiKV v1alpha1.TiKV
				err := cli.Get(context.Background(), client.ObjectKey{
					Namespace: testNamespace,
					Name:      expectedCancelName,
				}, &foundTiKV)
				require.NoError(t, err)
				assert.False(t, foundTiKV.Spec.Offline, "TiKV %s should have offline canceled", expectedCancelName)
			}
		})
	}
}

func TestTaskUpdater_OfflineCompletion(t *testing.T) {
	cases := []struct {
		desc            string
		initialReplicas int
		desiredReplicas int
		existingTiKVs   []*v1alpha1.TiKV
		expectDeleted   []string // names of TiKVs that should be deleted
		expectedStatus  task.Status
	}{
		{
			desc:            "delete instances after offline completion",
			initialReplicas: 2,
			desiredReplicas: 2,
			existingTiKVs: []*v1alpha1.TiKV{
				createTestTiKV("tikv-0", false, ""),
				createTestTiKV("tikv-1", false, ""),
				createTestTiKV("tikv-2", true, v1alpha1.OfflineReasonCompleted),
			},
			expectDeleted:  []string{"tikv-2"},
			expectedStatus: task.SWait,
		},
		{
			desc:            "no offline completed instances",
			initialReplicas: 2,
			desiredReplicas: 2,
			existingTiKVs: []*v1alpha1.TiKV{
				createTestTiKV("tikv-0", false, ""),
				createTestTiKV("tikv-1", true, v1alpha1.OfflineReasonActive),
			},
			expectDeleted:  nil,
			expectedStatus: task.SWait, // Normal updates may require waiting
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			// Setup test environment
			kvg := createTestTiKVGroup(tc.desiredReplicas)
			cluster := createTestCluster()

			objs := []client.Object{kvg, cluster}
			for _, tikv := range tc.existingTiKVs {
				objs = append(objs, tikv)
			}

			cli := client.NewFakeClient(objs...)
			state := createTestReconcileContext(kvg, cluster, tc.existingTiKVs)
			testTracker := tracker.New[*v1alpha1.TiKVGroup, *v1alpha1.TiKV]()

			for _, tikv := range tc.existingTiKVs {
				require.NoError(t, cli.Apply(context.Background(), tikv))
			}

			// Execute the two-step updater task
			taskFunc := TaskUpdater(state, cli, testTracker)
			result, _ := task.RunTask(context.Background(), taskFunc)

			assert.Equal(t, tc.expectedStatus, result.Status(), "unexpected task status")

			// Verify that the expected instances are deleted (not found)
			for _, expectedDeletedName := range tc.expectDeleted {
				var foundTiKV v1alpha1.TiKV
				err := cli.Get(context.Background(), client.ObjectKey{
					Namespace: testNamespace,
					Name:      expectedDeletedName,
				}, &foundTiKV)
				// Should have deletion timestamp set
				assert.True(t, err != nil || !foundTiKV.DeletionTimestamp.IsZero(),
					"TiKV %s should be deleted or have deletion timestamp", expectedDeletedName)
			}
		})
	}
}

// Helper functions for creating test objects

func createTestTiKV(name string, offline bool, offlineReason string) *v1alpha1.TiKV {
	tikv := &v1alpha1.TiKV{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Spec: v1alpha1.TiKVSpec{
			Offline: offline,
		},
		Status: v1alpha1.TiKVStatus{},
	}

	// Add offline condition if specified
	if offlineReason != "" {
		status := metav1.ConditionTrue
		if offlineReason == v1alpha1.OfflineReasonCompleted {
			status = metav1.ConditionFalse // Completed means offline process is done
		}
		condition := v1alpha1.NewOfflineCondition(offlineReason, "test condition", status)
		v1alpha1.SetOfflineCondition(&tikv.Status.Conditions, condition)
	}

	return tikv
}

func createTestTiKVGroup(replicas int) *v1alpha1.TiKVGroup {
	return &v1alpha1.TiKVGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testKVGroup,
			Namespace: testNamespace,
		},
		Spec: v1alpha1.TiKVGroupSpec{
			Cluster:  v1alpha1.ClusterReference{Name: testCluster},
			Replicas: ptr.To(int32(replicas)),
			Template: v1alpha1.TiKVTemplate{
				Spec: v1alpha1.TiKVTemplateSpec{
					Version: "v7.5.0",
					Volumes: []v1alpha1.Volume{
						{Name: "data", Storage: resource.MustParse("10Gi"), Mounts: []v1alpha1.VolumeMount{{Type: "data"}}},
					},
				},
			},
		},
		Status: v1alpha1.TiKVGroupStatus{},
	}
}

func createTestCluster() *v1alpha1.Cluster {
	return &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testCluster,
			Namespace: testNamespace,
		},
		Spec: v1alpha1.ClusterSpec{},
	}
}

func createTestReconcileContext(kvg *v1alpha1.TiKVGroup, cluster *v1alpha1.Cluster, tikvs []*v1alpha1.TiKV) *ReconcileContext {
	return &ReconcileContext{
		State: &state{
			kvg:            kvg,
			cluster:        cluster,
			kvs:            tikvs,
			updateRevision: testNewRevision,
		},
	}
}
