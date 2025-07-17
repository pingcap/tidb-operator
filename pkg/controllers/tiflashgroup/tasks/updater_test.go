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
	testTiFlashOldRevision = "old-rev-123"
	testTiFlashNewRevision = "new-rev-456"
	testTiFlashNamespace   = "test-ns"
	testTiFlashCluster     = "test-cluster"
	testTiFlashGroup       = "test-tiflashgroup"
)

func TestTaskUpdater_ScaleIn(t *testing.T) {
	cases := []struct {
		desc            string
		initialReplicas int
		desiredReplicas int
		existingTiFlash []*v1alpha1.TiFlash
		annotations     map[string]map[string]string // map[tiflashName]map[annotationKey]annotationValue
		expectedOffline []string                     // names of TiFlash that should be marked offline
		expectedStatus  task.Status
	}{
		{
			desc:            "scale in from 3 to 1, no annotations",
			initialReplicas: 3,
			desiredReplicas: 1,
			existingTiFlash: []*v1alpha1.TiFlash{
				createTestTiFlash("tiflash-0", false, ""),
				createTestTiFlash("tiflash-1", false, ""),
				createTestTiFlash("tiflash-2", false, ""),
			},
			expectedOffline: []string{"tiflash-0", "tiflash-1"}, // First 2 selected by default
			expectedStatus:  task.SWait,
		},
		{
			desc:            "scale in from 3 to 1, with deletion annotation",
			initialReplicas: 3,
			desiredReplicas: 1,
			existingTiFlash: []*v1alpha1.TiFlash{
				createTestTiFlash("tiflash-0", false, ""),
				createTestTiFlash("tiflash-1", false, ""),
				createTestTiFlash("tiflash-2", false, ""),
			},
			annotations: map[string]map[string]string{
				"tiflash-2": {v1alpha1.ScaleInDeleteAnnotation: "true"},
			},
			expectedOffline: []string{"tiflash-2", "tiflash-0"}, // Annotated instance preferred
			expectedStatus:  task.SWait,
		},
		{
			desc:            "scale in from 3 to 2, one already offline",
			initialReplicas: 3,
			desiredReplicas: 2,
			existingTiFlash: []*v1alpha1.TiFlash{
				createTestTiFlash("tiflash-0", false, ""),
				createTestTiFlash("tiflash-1", true, v1alpha1.OfflineReasonActive),
				createTestTiFlash("tiflash-2", false, ""),
			},
			expectedOffline: nil, // No new instances should be marked offline
			expectedStatus:  task.SWait,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			// Setup test environment
			tfg := createTestTiFlashGroup(tc.desiredReplicas)
			cluster := createTestCluster()

			// Apply annotations if specified
			for tiflashName, annotations := range tc.annotations {
				for _, tiflash := range tc.existingTiFlash {
					if tiflash.Name == tiflashName {
						if tiflash.Annotations == nil {
							tiflash.Annotations = make(map[string]string)
						}
						for k, v := range annotations {
							tiflash.Annotations[k] = v
						}
					}
				}
			}

			objs := []client.Object{tfg, cluster}
			for _, tiflash := range tc.existingTiFlash {
				objs = append(objs, tiflash)
			}

			cli := client.NewFakeClient(objs...)
			state := createTestReconcileContext(tfg, cluster, tc.existingTiFlash)
			tracker := tracker.New[*v1alpha1.TiFlashGroup, *v1alpha1.TiFlash]()

			for _, tiflash := range tc.existingTiFlash {
				require.NoError(t, cli.Apply(context.Background(), tiflash))
			}

			// Execute the two-step updater task
			taskFunc := TaskUpdater(state, cli, tracker)
			result, _ := task.RunTask(context.Background(), taskFunc)

			assert.Equal(t, tc.expectedStatus, result.Status(), "unexpected task status")

			// Verify that the expected instances are marked offline
			for _, expectedOfflineName := range tc.expectedOffline {
				var foundTiFlash v1alpha1.TiFlash
				err := cli.Get(context.Background(), client.ObjectKey{
					Namespace: testTiFlashNamespace,
					Name:      expectedOfflineName,
				}, &foundTiFlash)
				require.NoError(t, err)
				assert.True(t, foundTiFlash.Spec.Offline, "TiFlash %s should be marked offline", expectedOfflineName)
			}

			// Verify that other instances are not marked offline
			for _, tiflash := range tc.existingTiFlash {
				shouldBeOffline := false
				for _, name := range tc.expectedOffline {
					if tiflash.Name == name {
						shouldBeOffline = true
						break
					}
				}
				if !shouldBeOffline && !tiflash.Spec.Offline {
					var foundTiFlash v1alpha1.TiFlash
					err := cli.Get(context.Background(), client.ObjectKey{
						Namespace: testTiFlashNamespace,
						Name:      tiflash.Name,
					}, &foundTiFlash)
					require.NoError(t, err)
					assert.False(t, foundTiFlash.Spec.Offline, "TiFlash %s should not be marked offline", tiflash.Name)
				}
			}
		})
	}
}

// Helper functions for creating test objects

func createTestTiFlash(name string, offline bool, offlineReason string) *v1alpha1.TiFlash {
	tiflash := &v1alpha1.TiFlash{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testTiFlashNamespace,
		},
		Spec: v1alpha1.TiFlashSpec{
			Offline: offline,
		},
		Status: v1alpha1.TiFlashStatus{},
	}

	// Add offline condition if specified
	if offlineReason != "" {
		status := metav1.ConditionTrue
		if offlineReason == v1alpha1.OfflineReasonCompleted {
			status = metav1.ConditionFalse // Completed means offline process is done
		}
		condition := v1alpha1.NewOfflineCondition(offlineReason, "test condition", status)
		v1alpha1.SetOfflineCondition(&tiflash.Status.Conditions, condition)
	}

	return tiflash
}

func createTestTiFlashGroup(replicas int) *v1alpha1.TiFlashGroup {
	return &v1alpha1.TiFlashGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testTiFlashGroup,
			Namespace: testTiFlashNamespace,
		},
		Spec: v1alpha1.TiFlashGroupSpec{
			Cluster:  v1alpha1.ClusterReference{Name: testTiFlashCluster},
			Replicas: ptr.To(int32(replicas)),
			Template: v1alpha1.TiFlashTemplate{
				Spec: v1alpha1.TiFlashTemplateSpec{
					Version: "v7.5.0",
					Volumes: []v1alpha1.Volume{
						{Name: "data", Storage: resource.MustParse("10Gi"), Mounts: []v1alpha1.VolumeMount{{Type: "data"}}},
					},
				},
			},
		},
		Status: v1alpha1.TiFlashGroupStatus{},
	}
}

func createTestCluster() *v1alpha1.Cluster {
	return &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testTiFlashCluster,
			Namespace: testTiFlashNamespace,
		},
		Spec: v1alpha1.ClusterSpec{},
	}
}

func createTestReconcileContext(tfg *v1alpha1.TiFlashGroup, cluster *v1alpha1.Cluster, tiflashs []*v1alpha1.TiFlash) *ReconcileContext {
	return &ReconcileContext{
		State: &state{
			fg:             tfg,
			cluster:        cluster,
			fs:             tiflashs,
			updateRevision: testTiFlashNewRevision,
		},
	}
}
