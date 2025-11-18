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

package testutils

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	brv1alpha1 "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	corev1alpha1 "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/constants"
)

// Helper is for testing backup related code only
type Helper struct {
	T   *testing.T
	Cli client.Client
}

// NewHelper returns a helper encapsulation
func NewHelper(t *testing.T, cli client.Client) *Helper {
	return &Helper{
		T:   t,
		Cli: cli,
	}
}

// Close closes the stop channel
func (h *Helper) Close() {
}

// JobExists checks whether a k8s Job exists
func (h *Helper) JobExists(restore *brv1alpha1.Restore) {
	h.T.Helper()
	g := NewGomegaWithT(h.T)
	job := &batchv1.Job{}
	err := h.Cli.Get(context.TODO(), types.NamespacedName{Namespace: restore.Namespace, Name: restore.GetRestoreJobName()}, job)
	g.Expect(err).Should(BeNil())
}

func (h *Helper) createSecret(namespace, secretName string) {
	g := NewGomegaWithT(h.T)
	s := &corev1.Secret{}
	s.Data = map[string][]byte{
		constants.TidbPasswordKey:   []byte("dummy"),
		constants.GcsCredentialsKey: []byte("dummy"),
		constants.S3AccessKey:       []byte("dummy"),
		constants.S3SecretKey:       []byte("dummy"),
	}
	s.Namespace = namespace
	s.Name = secretName
	err := h.Cli.Create(context.TODO(), s)
	g.Expect(err).Should(BeNil())
}

// CreateSecret creates secrets based on backup/restore spec
func (h *Helper) CreateSecret(obj interface{}) {
	h.T.Helper()
	g := NewGomegaWithT(h.T)
	if obj1, ok := obj.(*brv1alpha1.Backup); ok {
		if obj1.Spec.StorageProvider.S3 != nil && obj1.Spec.StorageProvider.S3.SecretName != "" {
			h.createSecret(obj1.Namespace, obj1.Spec.StorageProvider.S3.SecretName)
			g.Eventually(func() error {
				secret := &corev1.Secret{}
				err := h.Cli.Get(context.TODO(), types.NamespacedName{Namespace: obj1.Namespace, Name: obj1.Spec.StorageProvider.S3.SecretName}, secret)
				return err
			}, time.Second*10).Should(BeNil())
		} else if obj1.Spec.StorageProvider.Gcs != nil && obj1.Spec.StorageProvider.Gcs.SecretName != "" {
			h.createSecret(obj1.Namespace, obj1.Spec.StorageProvider.Gcs.SecretName)
			g.Eventually(func() error {
				secret := &corev1.Secret{}
				err := h.Cli.Get(context.TODO(), types.NamespacedName{Namespace: obj1.Namespace, Name: obj1.Spec.StorageProvider.Gcs.SecretName}, secret)
				return err
			}, time.Second*10).Should(BeNil())
		}
	} else if obj2, ok := obj.(*brv1alpha1.Restore); ok {
		if obj2.Spec.StorageProvider.S3 != nil && obj2.Spec.StorageProvider.S3.SecretName != "" {
			h.createSecret(obj2.Namespace, obj2.Spec.StorageProvider.S3.SecretName)
			g.Eventually(func() error {
				secret := &corev1.Secret{}
				err := h.Cli.Get(context.TODO(), types.NamespacedName{Namespace: obj2.Namespace, Name: obj2.Spec.StorageProvider.S3.SecretName}, secret)
				return err
			}, time.Second*10).Should(BeNil())
		} else if obj2.Spec.StorageProvider.Gcs != nil && obj2.Spec.StorageProvider.Gcs.SecretName != "" {
			h.createSecret(obj2.Namespace, obj2.Spec.StorageProvider.Gcs.SecretName)
			g.Eventually(func() error {
				secret := &corev1.Secret{}
				err := h.Cli.Get(context.TODO(), types.NamespacedName{Namespace: obj2.Namespace, Name: obj2.Spec.StorageProvider.Gcs.SecretName}, secret)
				return err
			}, time.Second*10).Should(BeNil())
		}
	}
}

// CreateTC creates a Cluster with name `clusterName` in ns `namespace`
func (h *Helper) CreateTC(namespace, clusterName string, acrossK8s, recoverMode bool) {
	h.T.Helper()
	g := NewGomegaWithT(h.T)
	var err error

	tc := &corev1alpha1.Cluster{
		Spec: corev1alpha1.ClusterSpec{
			SuspendAction:        &corev1alpha1.SuspendAction{},
			TLSCluster:           &corev1alpha1.TLSCluster{},
			BootstrapSQL:         &corev1.LocalObjectReference{},
			UpgradePolicy:        "",
			Paused:               false,
			RevisionHistoryLimit: new(int32),
		},
		Status: corev1alpha1.ClusterStatus{},
	}
	tc.Namespace = namespace
	tc.Name = clusterName
	err = h.Cli.Create(context.TODO(), tc)
	g.Expect(err).Should(BeNil())
	// make sure can read tc from lister
	g.Eventually(func() error {
		tc := &corev1alpha1.Cluster{}
		err := h.Cli.Get(context.TODO(), types.NamespacedName{Namespace: tc.Namespace, Name: tc.Name}, tc)
		return err
	}, time.Second*10).Should(BeNil())
	g.Expect(err).Should(BeNil())
}

// // CreateTCWithNoTiKV creates a TidbCluster with name `clusterName` in ns `namespace` with no TiKV nodes
// func (h *Helper) CreateTCWithNoTiKV(namespace, clusterName string, acrossK8s, recoverMode bool) {
// 	h.T.Helper()
// 	g := NewGomegaWithT(h.T)
// 	var err error

// 	tc := &v1alpha1.TidbCluster{
// 		Spec: v1alpha1.TidbClusterSpec{
// 			AcrossK8s:    acrossK8s,
// 			RecoveryMode: recoverMode,
// 			TLSCluster:   &v1alpha1.TLSCluster{Enabled: true},
// 			TiKV: &v1alpha1.TiKVSpec{
// 				BaseImage: "pingcap/tikv",
// 				Replicas:  3,
// 				StorageVolumes: []v1alpha1.StorageVolume{
// 					{MountPath: "/var/lib/raft", Name: "raft", StorageSize: "50Gi"},
// 					{MountPath: "/var/lib/wal", Name: "wal", StorageSize: "50Gi"},
// 				},
// 			},
// 			TiDB: &v1alpha1.TiDBSpec{
// 				TLSClient: &v1alpha1.TiDBTLSClient{Enabled: true},
// 			},
// 			PD: &v1alpha1.PDSpec{
// 				Replicas: 1,
// 			},
// 		},
// 		Status: v1alpha1.TidbClusterStatus{
// 			PD: v1alpha1.PDStatus{
// 				Members: map[string]v1alpha1.PDMember{
// 					"pd-0": {Name: "pd-0", Health: true},
// 				},
// 			},
// 		},
// 	}
// 	tc.Namespace = namespace
// 	tc.Name = clusterName
// 	_, err = h.Deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(context.TODO(), tc, metav1.CreateOptions{})
// 	g.Expect(err).Should(BeNil())
// 	// make sure can read tc from lister
// 	g.Eventually(func() error {
// 		_, err := h.Deps.TiDBClusterLister.TidbClusters(tc.Namespace).Get(tc.Name)
// 		return err
// 	}, time.Second*10).Should(BeNil())
// 	g.Expect(err).Should(BeNil())
// }

func (h *Helper) CreateRestore(restore *brv1alpha1.Restore) {
	h.T.Helper()
	g := NewGomegaWithT(h.T)
	err := h.Cli.Create(context.TODO(), restore)
	g.Expect(err).Should(BeNil())
	// make sure can read tc from lister
	g.Eventually(func() error {
		restore := &brv1alpha1.Restore{}
		err := h.Cli.Get(context.TODO(), types.NamespacedName{Namespace: restore.Namespace, Name: restore.Name}, restore)
		return err
	}, time.Second).Should(BeNil())
	g.Expect(err).Should(BeNil())
}
