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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1alpha1br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestAssembleT2Cronjob(t *testing.T) {
	// setup
	rtx := &ReconcileContext{
		namespacedName: types.NamespacedName{
			Name:      "test-tibrgc",
			Namespace: "default",
		},
		ctx: context.Background(),
		cli: nil, // We don't need a real client for assembleT2Cronjob tests
	}
	rtx.cluster = &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: v1alpha1.ClusterSpec{
			TLSCluster: &v1alpha1.TLSCluster{
				Enabled: true,
			},
		},
		Status: v1alpha1.ClusterStatus{
			PD: "test-pd-address",
		},
	}
	tibrgcYAML := `apiVersion: br.pingcap.com/v1alpha1
kind: TiBRGC
metadata:
  name: test-tibrgc
  namespace: default
spec:
  cluster: 
    name: test-cluster
  gcStrategy:
    type: "tiered-storage"
    tieredStrategies:
    - name: "to-t2-storage"
      timeThresholdDays: 7
      schedule: "10 2 * * *"
      resources:
        cpu: 100m
        memory: 128Mi
      options:
      - --extra-option
      - dummy-value
  image: custom-image:latest
  overlay:
    pods:
    - name: to-t2-storage
      overlay:
        pod:
          spec:
            containers:
              - name: to-t2-storage
                env:
                - name: DFS_PREFIX
                  value: cse
                - name: DFS_S3_BUCKET
                  value: bucket-xxx
                - name: DFS_S3_REGION
                  value: us-west-2
                - name: DFS_S3_KEY_ID
                  valueFrom:
                    secretKeyRef:
                      name: s3-credentials
                      key: aws_access_key_id
                - name: DFS_S3_SECRET_KEY
                  valueFrom:
                    secretKeyRef:
                      name: s3-credentials
                      key: aws_secret_access_key`

	rtx.tibrgc = &v1alpha1br.TiBRGC{}
	err := yaml.Unmarshal([]byte(tibrgcYAML), rtx.tibrgc)
	require.NoError(t, err)

	// test
	cronjob := assembleT2Cronjob(rtx)

	// assert
	// Convert to YAML for comparison
	actualYAML, err := yaml.Marshal(cronjob)
	require.NoError(t, err)

	expectedYAML := `
metadata:
  name: test-tibrgc-to-t2-storage
  namespace: default
  labels:
    pingcap.com/cluster: test-cluster
    pingcap.com/managed-by: tidb-operator
    pingcap.com/component: tibrgc
    pingcap.com/instance: test-tibrgc
    pingcap.com/name: to-t2-storage
  ownerReferences:
  - apiVersion: br.pingcap.com/v1alpha1
    kind: TiBRGC
    name: test-tibrgc
    uid: ""
    controller: true
    blockOwnerDeletion: true
  creationTimestamp: null
spec:
  concurrencyPolicy: Forbid
  failedJobsHistoryLimit: 1
  successfulJobsHistoryLimit: 3
  suspend: false
  schedule: "10 2 * * *"
  jobTemplate:
    metadata:
      creationTimestamp: null
    spec:
      backoffLimit: 16
      template:
        metadata:
          creationTimestamp: null
          labels:
            pingcap.com/cluster: test-cluster
            pingcap.com/managed-by: tidb-operator
            pingcap.com/component: tibrgc
            pingcap.com/instance: test-tibrgc
            pingcap.com/name: to-t2-storage
        spec:
          restartPolicy: OnFailure
          containers:
          - name: to-t2-storage
            image: custom-image:latest
            resources:
              requests:
                cpu: 100m
                memory: 128Mi
              limits:
                cpu: 100m
                memory: 128Mi
            command:
            - /cse-ctl
            - dfs-gc
            - --pd
            - test-pd-address
            - --start-time-safe-interval
            - "7d"
            - --extra-option
            - dummy-value
            - --cacert
            - /var/lib/tibrgc-tls/ca.crt
            - --cert
            - /var/lib/tibrgc-tls/tls.crt
            - --key
            - /var/lib/tibrgc-tls/tls.key
            volumeMounts:
            - name: tibrgc-tls
              mountPath: /var/lib/tibrgc-tls
            env:
                - name: DFS_PREFIX
                  value: cse
                - name: DFS_S3_BUCKET
                  value: bucket-xxx
                - name: DFS_S3_REGION
                  value: us-west-2
                - name: DFS_S3_KEY_ID
                  valueFrom:
                    secretKeyRef:
                      name: s3-credentials
                      key: aws_access_key_id
                - name: DFS_S3_SECRET_KEY
                  valueFrom:
                    secretKeyRef:
                      name: s3-credentials
                      key: aws_secret_access_key
          volumes:
          - name: tibrgc-tls
            secret:
              secretName: test-tibrgc-tibrgc-cluster-secret
              defaultMode: 420
status: {}`
	fmt.Println(string(actualYAML))
	assert.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestAssembleT3Cronjob(t *testing.T) {
	// setup
	rtx := &ReconcileContext{
		namespacedName: types.NamespacedName{
			Name:      "test-tibrgc",
			Namespace: "default",
		},
		ctx: context.Background(),
		cli: nil, // We don't need a real client for assembleT3Cronjob tests
	}
	rtx.cluster = &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: v1alpha1.ClusterSpec{
			TLSCluster: &v1alpha1.TLSCluster{
				Enabled: true,
			},
		},
		Status: v1alpha1.ClusterStatus{
			PD: "test-pd-address",
		},
	}
	tibrgcYAML := `apiVersion: br.pingcap.com/v1alpha1
kind: TiBRGC
metadata:
  name: test-tibrgc
  namespace: default
spec:
  cluster: 
    name: test-cluster
  gcStrategy:
    type: "tiered-storage"
    tieredStrategies:
    - name: "to-t3-storage"
      timeThresholdDays: 30
      schedule: "30 3 * * *"
      resources:
        cpu: 200m
        memory: 256Mi
      options:
      - --archive-option
      - archive-value
      volumes:
      - name: data
        storageClassName: fast-ssd
        storage: 100Gi
        mounts:
        - mountPath: /data/archive
  image: custom-tikv:latest
  overlay:
    pods:
    - name: to-t3-storage
      overlay:
        pod:
          spec:
            containers:
              - name: to-t3-storage
                env:
                - name: ARCHIVE_PREFIX
                  value: archive
                - name: ARCHIVE_S3_BUCKET
                  value: archive-bucket
                - name: ARCHIVE_S3_REGION
                  value: us-east-1
                - name: ARCHIVE_S3_KEY_ID
                  valueFrom:
                    secretKeyRef:
                      name: archive-credentials
                      key: aws_access_key_id
                - name: ARCHIVE_S3_SECRET_KEY
                  valueFrom:
                    secretKeyRef:
                      name: archive-credentials
                      key: aws_secret_access_key`

	rtx.tibrgc = &v1alpha1br.TiBRGC{}
	err := yaml.Unmarshal([]byte(tibrgcYAML), rtx.tibrgc)
	require.NoError(t, err)

	// test
	cronjob := assembleT3Cronjob(rtx)

	// assert
	// Convert to YAML for comparison
	actualYAML, err := yaml.Marshal(cronjob)
	require.NoError(t, err)

	expectedYAML := `
metadata:
  name: test-tibrgc-to-t3-storage
  namespace: default
  labels:
    pingcap.com/cluster: test-cluster
    pingcap.com/managed-by: tidb-operator
    pingcap.com/component: tibrgc
    pingcap.com/instance: test-tibrgc
    pingcap.com/name: to-t3-storage
  ownerReferences:
  - apiVersion: br.pingcap.com/v1alpha1
    kind: TiBRGC
    name: test-tibrgc
    uid: ""
    controller: true
    blockOwnerDeletion: true
  creationTimestamp: null
spec:
  concurrencyPolicy: Forbid
  failedJobsHistoryLimit: 1
  successfulJobsHistoryLimit: 3
  suspend: false
  schedule: "30 3 * * *"
  jobTemplate:
    metadata:
      creationTimestamp: null
    spec:
      backoffLimit: 32
      template:
        metadata:
          creationTimestamp: null
          labels:
            pingcap.com/cluster: test-cluster
            pingcap.com/managed-by: tidb-operator
            pingcap.com/component: tibrgc
            pingcap.com/instance: test-tibrgc
            pingcap.com/name: to-t3-storage
        spec:
          restartPolicy: OnFailure
          containers:
          - name: to-t3-storage
            image: custom-tikv:latest
            resources:
              requests:
                cpu: 200m
                memory: 256Mi
              limits:
                cpu: 200m
                memory: 256Mi
            command:
            - /cse-ctl
            - archive
            - --pd
            - test-pd-address
            - --start-archive-duration
            - "30d"
            - --data-dir
            - /data/archive
            - --archive-option
            - archive-value
            - --cacert
            - /var/lib/tibrgc-tls/ca.crt
            - --cert
            - /var/lib/tibrgc-tls/tls.crt
            - --key
            - /var/lib/tibrgc-tls/tls.key
            volumeMounts:
            - name: tibrgc-tls
              mountPath: /var/lib/tibrgc-tls
            - name: data
              mountPath: /data/archive
            env:
                - name: ARCHIVE_PREFIX
                  value: archive
                - name: ARCHIVE_S3_BUCKET
                  value: archive-bucket
                - name: ARCHIVE_S3_REGION
                  value: us-east-1
                - name: ARCHIVE_S3_KEY_ID
                  valueFrom:
                    secretKeyRef:
                      name: archive-credentials
                      key: aws_access_key_id
                - name: ARCHIVE_S3_SECRET_KEY
                  valueFrom:
                    secretKeyRef:
                      name: archive-credentials
                      key: aws_secret_access_key
          volumes:
          - name: tibrgc-tls
            secret:
              secretName: test-tibrgc-tibrgc-cluster-secret
              defaultMode: 420
          - name: data
            ephemeral:
              volumeClaimTemplate:
                metadata:
                  creationTimestamp: null
                spec:
                  storageClassName: fast-ssd
                  accessModes:
                  - ReadWriteOnce
                  resources:
                    requests:
                      storage: 100Gi
status: {}`
	fmt.Println(string(actualYAML))
	assert.YAMLEq(t, expectedYAML, string(actualYAML))
}

func TestGetStrategyResources(t *testing.T) {
	t.Run("should return resources when CPU and Memory are set", func(t *testing.T) {
		// setup
		cpu := resource.MustParse("100m")
		memory := resource.MustParse("128Mi")
		strategy := &v1alpha1br.TieredStorageStrategy{
			Resources: v1alpha1.ResourceRequirements{
				CPU:    &cpu,
				Memory: &memory,
			},
		}

		// test
		resources := getStrategyResources(strategy)

		// assert
		assert.Equal(t, cpu, resources.Requests[corev1.ResourceCPU])
		assert.Equal(t, memory, resources.Requests[corev1.ResourceMemory])
		assert.Equal(t, cpu, resources.Limits[corev1.ResourceCPU])
		assert.Equal(t, memory, resources.Limits[corev1.ResourceMemory])
	})

	t.Run("should return empty resources when CPU and Memory are nil", func(t *testing.T) {
		// setup
		strategy := &v1alpha1br.TieredStorageStrategy{
			Resources: v1alpha1.ResourceRequirements{},
		}

		// test
		resources := getStrategyResources(strategy)

		// assert
		assert.Nil(t, resources.Requests)
		assert.Nil(t, resources.Limits)
	})

	t.Run("should return resources when only CPU is set", func(t *testing.T) {
		// setup
		cpu := resource.MustParse("200m")
		strategy := &v1alpha1br.TieredStorageStrategy{
			Resources: v1alpha1.ResourceRequirements{
				CPU: &cpu,
			},
		}

		// test
		resources := getStrategyResources(strategy)

		// assert
		assert.Equal(t, cpu, resources.Requests[corev1.ResourceCPU])
		assert.Zero(t, resources.Requests[corev1.ResourceMemory])
		assert.Equal(t, cpu, resources.Limits[corev1.ResourceCPU])
		assert.Zero(t, resources.Limits[corev1.ResourceMemory])
	})

	t.Run("should return resources when only Memory is set", func(t *testing.T) {
		// setup
		memory := resource.MustParse("512Mi")
		strategy := &v1alpha1br.TieredStorageStrategy{
			Resources: v1alpha1.ResourceRequirements{
				Memory: &memory,
			},
		}

		// test
		resources := getStrategyResources(strategy)

		// assert
		assert.Zero(t, resources.Requests[corev1.ResourceCPU])
		assert.Equal(t, memory, resources.Requests[corev1.ResourceMemory])
		assert.Zero(t, resources.Limits[corev1.ResourceCPU])
		assert.Equal(t, memory, resources.Limits[corev1.ResourceMemory])
	})
}

func TestGetT2CronjobCommand(t *testing.T) {
	t.Run("should return basic command without TLS and options", func(t *testing.T) {
		// setup
		rtx := &ReconcileContext{
			cluster: &v1alpha1.Cluster{
				Status: v1alpha1.ClusterStatus{
					PD: "test-pd-address:2379",
				},
			},
		}
		strategy := &v1alpha1br.TieredStorageStrategy{
			TimeThresholdDays: 7,
		}

		// test
		command := getT2CronjobCommand(rtx, strategy)

		// assert
		expected := []string{
			"/cse-ctl",
			"dfs-gc",
			"--pd",
			"test-pd-address:2379",
			"--start-time-safe-interval",
			"7d",
		}
		assert.Equal(t, expected, command)
	})

	t.Run("should return command with TLS enabled", func(t *testing.T) {
		// setup
		rtx := &ReconcileContext{
			cluster: &v1alpha1.Cluster{
				Spec: v1alpha1.ClusterSpec{
					TLSCluster: &v1alpha1.TLSCluster{
						Enabled: true,
					},
				},
				Status: v1alpha1.ClusterStatus{
					PD: "test-pd-address:2379",
				},
			},
		}
		strategy := &v1alpha1br.TieredStorageStrategy{
			TimeThresholdDays: 14,
		}

		// test
		command := getT2CronjobCommand(rtx, strategy)

		// assert
		expected := []string{
			"/cse-ctl",
			"dfs-gc",
			"--pd",
			"test-pd-address:2379",
			"--start-time-safe-interval",
			"14d",
			"--cacert",
			"/var/lib/tibrgc-tls/ca.crt",
			"--cert",
			"/var/lib/tibrgc-tls/tls.crt",
			"--key",
			"/var/lib/tibrgc-tls/tls.key",
		}
		assert.Equal(t, expected, command)
	})

	t.Run("should return command with custom options", func(t *testing.T) {
		// setup
		rtx := &ReconcileContext{
			cluster: &v1alpha1.Cluster{
				Status: v1alpha1.ClusterStatus{
					PD: "test-pd-address:2379",
				},
			},
		}
		strategy := &v1alpha1br.TieredStorageStrategy{
			TimeThresholdDays: 30,
			Options: []string{
				"--extra-option",
				"dummy-value",
				"--another-flag",
			},
		}

		// test
		command := getT2CronjobCommand(rtx, strategy)

		// assert
		expected := []string{
			"/cse-ctl",
			"dfs-gc",
			"--pd",
			"test-pd-address:2379",
			"--start-time-safe-interval",
			"30d",
			"--extra-option",
			"dummy-value",
			"--another-flag",
		}
		assert.Equal(t, expected, command)
	})

	t.Run("should return command with TLS and custom options", func(t *testing.T) {
		// setup
		rtx := &ReconcileContext{
			cluster: &v1alpha1.Cluster{
				Spec: v1alpha1.ClusterSpec{
					TLSCluster: &v1alpha1.TLSCluster{
						Enabled: true,
					},
				},
				Status: v1alpha1.ClusterStatus{
					PD: "test-pd-address:2379",
				},
			},
		}
		strategy := &v1alpha1br.TieredStorageStrategy{
			TimeThresholdDays: 1,
			Options: []string{
				"--debug",
				"--log-level",
				"info",
			},
		}

		// test
		command := getT2CronjobCommand(rtx, strategy)

		// assert
		expected := []string{
			"/cse-ctl",
			"dfs-gc",
			"--pd",
			"test-pd-address:2379",
			"--start-time-safe-interval",
			"1d",
			"--debug",
			"--log-level",
			"info",
			"--cacert",
			"/var/lib/tibrgc-tls/ca.crt",
			"--cert",
			"/var/lib/tibrgc-tls/tls.crt",
			"--key",
			"/var/lib/tibrgc-tls/tls.key",
		}
		assert.Equal(t, expected, command)
	})

	t.Run("should handle nil options", func(t *testing.T) {
		// setup
		rtx := &ReconcileContext{
			cluster: &v1alpha1.Cluster{
				Status: v1alpha1.ClusterStatus{
					PD: "test-pd-address:2379",
				},
			},
		}
		strategy := &v1alpha1br.TieredStorageStrategy{
			TimeThresholdDays: 7,
			Options:           nil,
		}

		// test
		command := getT2CronjobCommand(rtx, strategy)

		// assert
		expected := []string{
			"/cse-ctl",
			"dfs-gc",
			"--pd",
			"test-pd-address:2379",
			"--start-time-safe-interval",
			"7d",
		}
		assert.Equal(t, expected, command)
	})
}

func TestGetT3CronjobVolumeMounts(t *testing.T) {
	t.Run("should return empty slice when TLS disabled and no strategy volumes", func(t *testing.T) {
		// setup
		tlsEnabled := false
		var strategy *v1alpha1br.TieredStorageStrategy = nil

		// test
		volumeMounts := getT3CronjobVolumeMounts(tlsEnabled, strategy)

		// assert
		assert.Empty(t, volumeMounts)
	})

	t.Run("should return only TLS volume mount when TLS enabled and no strategy volumes", func(t *testing.T) {
		// setup
		tlsEnabled := true
		var strategy *v1alpha1br.TieredStorageStrategy = nil

		// test
		volumeMounts := getT3CronjobVolumeMounts(tlsEnabled, strategy)

		// assert
		assert.Len(t, volumeMounts, 1)
		assert.Equal(t, "tibrgc-tls", volumeMounts[0].Name)
		assert.Equal(t, "/var/lib/tibrgc-tls", volumeMounts[0].MountPath)
	})

	t.Run("should return strategy volume mounts when TLS disabled and strategy has volumes", func(t *testing.T) {
		// setup
		tlsEnabled := false
		strategy := &v1alpha1br.TieredStorageStrategy{
			Volumes: []v1alpha1.Volume{
				{
					Name: "data",
					Mounts: []v1alpha1.VolumeMount{
						{
							MountPath: "/data",
							SubPath:   "archive",
						},
					},
				},
			},
		}

		// test
		volumeMounts := getT3CronjobVolumeMounts(tlsEnabled, strategy)

		// assert
		assert.Len(t, volumeMounts, 1)
		assert.Equal(t, "data", volumeMounts[0].Name)
		assert.Equal(t, "/data", volumeMounts[0].MountPath)
		assert.Equal(t, "archive", volumeMounts[0].SubPath)
	})

	t.Run("should return TLS and strategy volume mounts when TLS enabled and strategy has volumes", func(t *testing.T) {
		// setup
		tlsEnabled := true
		strategy := &v1alpha1br.TieredStorageStrategy{
			Volumes: []v1alpha1.Volume{
				{
					Name: "data",
					Mounts: []v1alpha1.VolumeMount{
						{
							MountPath: "/data",
							SubPath:   "archive",
						},
					},
				},
			},
		}

		// test
		volumeMounts := getT3CronjobVolumeMounts(tlsEnabled, strategy)

		// assert
		assert.Len(t, volumeMounts, 2)

		// TLS volume should be first
		assert.Equal(t, "tibrgc-tls", volumeMounts[0].Name)
		assert.Equal(t, "/var/lib/tibrgc-tls", volumeMounts[0].MountPath)

		// Strategy volume should be second
		assert.Equal(t, "data", volumeMounts[1].Name)
		assert.Equal(t, "/data", volumeMounts[1].MountPath)
		assert.Equal(t, "archive", volumeMounts[1].SubPath)
	})

	t.Run("should handle multiple volumes in strategy", func(t *testing.T) {
		// setup
		tlsEnabled := true
		strategy := &v1alpha1br.TieredStorageStrategy{
			Volumes: []v1alpha1.Volume{
				{
					Name: "data",
					Mounts: []v1alpha1.VolumeMount{
						{
							MountPath: "/data",
							SubPath:   "archive",
						},
					},
				},
				{
					Name: "logs",
					Mounts: []v1alpha1.VolumeMount{
						{
							MountPath: "/var/log",
							SubPath:   "",
						},
					},
				},
			},
		}

		// test
		volumeMounts := getT3CronjobVolumeMounts(tlsEnabled, strategy)

		// assert
		assert.Len(t, volumeMounts, 3)

		// TLS volume should be first
		assert.Equal(t, "tibrgc-tls", volumeMounts[0].Name)
		assert.Equal(t, "/var/lib/tibrgc-tls", volumeMounts[0].MountPath)

		// Data volume should be second
		assert.Equal(t, "data", volumeMounts[1].Name)
		assert.Equal(t, "/data", volumeMounts[1].MountPath)
		assert.Equal(t, "archive", volumeMounts[1].SubPath)

		// Logs volume should be third
		assert.Equal(t, "logs", volumeMounts[2].Name)
		assert.Equal(t, "/var/log", volumeMounts[2].MountPath)
		assert.Empty(t, volumeMounts[2].SubPath)
	})
}

func TestGetT3CronjobVolumes(t *testing.T) {
	t.Run("should return empty slice when TLS disabled and no strategy volumes", func(t *testing.T) {
		// setup
		tlsEnabled := false
		tibrgc := &v1alpha1br.TiBRGC{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-tibrgc",
				Namespace: "default",
			},
			Spec: v1alpha1br.TiBRGCSpec{
				Cluster: v1alpha1.ClusterReference{
					Name: "test-cluster",
				},
				GCStrategy: v1alpha1br.TiBRGCStrategy{
					Type: v1alpha1br.TiBRGCStrategyTypeTieredStorage,
					TieredStrategies: []v1alpha1br.TieredStorageStrategy{
						{
							Name: v1alpha1br.TieredStorageStrategyNameToT3Storage,
							// No volumes defined
						},
					},
				},
			},
		}

		// test
		volumes := getT3CronjobVolumes(tlsEnabled, tibrgc)

		// assert
		assert.Empty(t, volumes)
	})

	t.Run("should return only TLS volume when TLS enabled and no strategy volumes", func(t *testing.T) {
		// setup
		tlsEnabled := true
		tibrgc := &v1alpha1br.TiBRGC{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-tibrgc",
				Namespace: "default",
			},
			Spec: v1alpha1br.TiBRGCSpec{
				Cluster: v1alpha1.ClusterReference{
					Name: "test-cluster",
				},
				GCStrategy: v1alpha1br.TiBRGCStrategy{
					Type: v1alpha1br.TiBRGCStrategyTypeTieredStorage,
					TieredStrategies: []v1alpha1br.TieredStorageStrategy{
						{
							Name: v1alpha1br.TieredStorageStrategyNameToT3Storage,
							// No volumes defined
						},
					},
				},
			},
		}

		// test
		volumes := getT3CronjobVolumes(tlsEnabled, tibrgc)

		// assert
		assert.Len(t, volumes, 1)
		assert.Equal(t, "tibrgc-tls", volumes[0].Name)
		assert.NotNil(t, volumes[0].Secret)
		assert.Equal(t, "test-tibrgc-tibrgc-cluster-secret", volumes[0].Secret.SecretName)
		assert.Equal(t, int32(420), *volumes[0].Secret.DefaultMode)
	})

	t.Run("should return strategy volumes when TLS disabled and strategy has volumes", func(t *testing.T) {
		// setup
		tlsEnabled := false
		storage := resource.MustParse("100Gi")
		tibrgc := &v1alpha1br.TiBRGC{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-tibrgc",
				Namespace: "default",
			},
			Spec: v1alpha1br.TiBRGCSpec{
				Cluster: v1alpha1.ClusterReference{
					Name: "test-cluster",
				},
				GCStrategy: v1alpha1br.TiBRGCStrategy{
					Type: v1alpha1br.TiBRGCStrategyTypeTieredStorage,
					TieredStrategies: []v1alpha1br.TieredStorageStrategy{
						{
							Name: v1alpha1br.TieredStorageStrategyNameToT3Storage,
							Volumes: []v1alpha1.Volume{
								{
									Name:    "data",
									Storage: storage,
									Mounts: []v1alpha1.VolumeMount{
										{
											MountPath: "/data",
											SubPath:   "archive",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// test
		volumes := getT3CronjobVolumes(tlsEnabled, tibrgc)

		// assert
		assert.Len(t, volumes, 1)
		assert.Equal(t, "data", volumes[0].Name)
		assert.NotNil(t, volumes[0].Ephemeral)
		assert.NotNil(t, volumes[0].Ephemeral.VolumeClaimTemplate)
		assert.Equal(t, storage, volumes[0].Ephemeral.VolumeClaimTemplate.Spec.Resources.Requests[corev1.ResourceStorage])
		assert.Equal(t, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, volumes[0].Ephemeral.VolumeClaimTemplate.Spec.AccessModes)
	})

	t.Run("should return TLS and strategy volumes when TLS enabled and strategy has volumes", func(t *testing.T) {
		// setup
		tlsEnabled := true
		storage := resource.MustParse("50Gi")
		storageClassName := "fast-ssd"
		tibrgc := &v1alpha1br.TiBRGC{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-tibrgc",
				Namespace: "default",
			},
			Spec: v1alpha1br.TiBRGCSpec{
				Cluster: v1alpha1.ClusterReference{
					Name: "test-cluster",
				},
				GCStrategy: v1alpha1br.TiBRGCStrategy{
					Type: v1alpha1br.TiBRGCStrategyTypeTieredStorage,
					TieredStrategies: []v1alpha1br.TieredStorageStrategy{
						{
							Name: v1alpha1br.TieredStorageStrategyNameToT3Storage,
							Volumes: []v1alpha1.Volume{
								{
									Name:             "data",
									Storage:          storage,
									StorageClassName: &storageClassName,
									Mounts: []v1alpha1.VolumeMount{
										{
											MountPath: "/data",
											SubPath:   "archive",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// test
		volumes := getT3CronjobVolumes(tlsEnabled, tibrgc)

		// assert
		assert.Len(t, volumes, 2)

		// TLS volume should be first
		assert.Equal(t, "tibrgc-tls", volumes[0].Name)
		assert.NotNil(t, volumes[0].Secret)
		assert.Equal(t, "test-tibrgc-tibrgc-cluster-secret", volumes[0].Secret.SecretName)

		// Strategy volume should be second
		assert.Equal(t, "data", volumes[1].Name)
		assert.NotNil(t, volumes[1].Ephemeral)
		assert.Equal(t, storage, volumes[1].Ephemeral.VolumeClaimTemplate.Spec.Resources.Requests[corev1.ResourceStorage])
		assert.Equal(t, &storageClassName, volumes[1].Ephemeral.VolumeClaimTemplate.Spec.StorageClassName)
	})

	t.Run("should handle multiple volumes in strategy", func(t *testing.T) {
		// setup
		tlsEnabled := true
		dataStorage := resource.MustParse("100Gi")
		logsStorage := resource.MustParse("10Gi")
		tibrgc := &v1alpha1br.TiBRGC{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-tibrgc",
				Namespace: "default",
			},
			Spec: v1alpha1br.TiBRGCSpec{
				Cluster: v1alpha1.ClusterReference{
					Name: "test-cluster",
				},
				GCStrategy: v1alpha1br.TiBRGCStrategy{
					Type: v1alpha1br.TiBRGCStrategyTypeTieredStorage,
					TieredStrategies: []v1alpha1br.TieredStorageStrategy{
						{
							Name: v1alpha1br.TieredStorageStrategyNameToT3Storage,
							Volumes: []v1alpha1.Volume{
								{
									Name:    "data",
									Storage: dataStorage,
									Mounts: []v1alpha1.VolumeMount{
										{
											MountPath: "/data",
											SubPath:   "archive",
										},
									},
								},
								{
									Name:    "logs",
									Storage: logsStorage,
									Mounts: []v1alpha1.VolumeMount{
										{
											MountPath: "/var/log",
											SubPath:   "",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// test
		volumes := getT3CronjobVolumes(tlsEnabled, tibrgc)

		// assert
		assert.Len(t, volumes, 3)

		// TLS volume should be first
		assert.Equal(t, "tibrgc-tls", volumes[0].Name)
		assert.NotNil(t, volumes[0].Secret)

		// Data volume should be second
		assert.Equal(t, "data", volumes[1].Name)
		assert.NotNil(t, volumes[1].Ephemeral)
		assert.Equal(t, dataStorage, volumes[1].Ephemeral.VolumeClaimTemplate.Spec.Resources.Requests[corev1.ResourceStorage])

		// Logs volume should be third
		assert.Equal(t, "logs", volumes[2].Name)
		assert.NotNil(t, volumes[2].Ephemeral)
		assert.Equal(t, logsStorage, volumes[2].Ephemeral.VolumeClaimTemplate.Spec.Resources.Requests[corev1.ResourceStorage])
	})

	t.Run("should apply PVC overlay with metadata labels", func(t *testing.T) {
		// setup
		tlsEnabled := false
		storage := resource.MustParse("100Gi")
		tibrgc := &v1alpha1br.TiBRGC{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-tibrgc",
				Namespace: "default",
			},
			Spec: v1alpha1br.TiBRGCSpec{
				Cluster: v1alpha1.ClusterReference{
					Name: "test-cluster",
				},
				GCStrategy: v1alpha1br.TiBRGCStrategy{
					Type: v1alpha1br.TiBRGCStrategyTypeTieredStorage,
					TieredStrategies: []v1alpha1br.TieredStorageStrategy{
						{
							Name: v1alpha1br.TieredStorageStrategyNameToT3Storage,
							Volumes: []v1alpha1.Volume{
								{
									Name:    "data",
									Storage: storage,
									Mounts: []v1alpha1.VolumeMount{
										{
											MountPath: "/data",
											SubPath:   "archive",
										},
									},
								},
							},
						},
					},
				},
				Overlay: v1alpha1br.TiBRGCOverlay{
					Pods: []v1alpha1br.TiBRGCPodOverlay{
						{
							Name: v1alpha1br.TieredStorageStrategyNameToT3Storage,
							Overlay: &v1alpha1.Overlay{
								PersistentVolumeClaims: []v1alpha1.NamedPersistentVolumeClaimOverlay{
									{
										Name: "data",
										PersistentVolumeClaim: v1alpha1.PersistentVolumeClaimOverlay{
											ObjectMeta: v1alpha1.ObjectMeta{
												Labels: map[string]string{
													"custom-label": "data-volume",
													"environment":  "production",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// test
		volumes := getT3CronjobVolumes(tlsEnabled, tibrgc)

		// assert
		assert.Len(t, volumes, 1)
		assert.Equal(t, "data", volumes[0].Name)
		assert.NotNil(t, volumes[0].Ephemeral)
		assert.Equal(t, storage, volumes[0].Ephemeral.VolumeClaimTemplate.Spec.Resources.Requests[corev1.ResourceStorage])

		// Check that PVC overlay was applied
		assert.Equal(t, "data-volume", volumes[0].Ephemeral.VolumeClaimTemplate.Labels["custom-label"])
		assert.Equal(t, "production", volumes[0].Ephemeral.VolumeClaimTemplate.Labels["environment"])
	})
}
