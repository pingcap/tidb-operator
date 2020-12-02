// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package testutils

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Helper is for testing backup related code only
type Helper struct {
	T    *testing.T
	Deps *controller.Dependencies
	stop chan struct{}
}

// NewHelper returns a helper encapsulation
func NewHelper(t *testing.T) *Helper {
	deps := controller.NewSimpleClientDependencies()
	stop := make(chan struct{})
	deps.InformerFactory.Start(stop)
	deps.KubeInformerFactory.Start(stop)
	deps.InformerFactory.WaitForCacheSync(stop)
	deps.KubeInformerFactory.WaitForCacheSync(stop)

	return &Helper{
		stop: stop,
		T:    t,
		Deps: deps,
	}
}

// Close closes the stop channel
func (h *Helper) Close() {
	close(h.stop)
}

// JobExists checks whether a k8s Job exists
func (h *Helper) JobExists(restore *v1alpha1.Restore) {
	h.T.Helper()
	g := NewGomegaWithT(h.T)
	_, err := h.Deps.KubeClientset.BatchV1().Jobs(restore.Namespace).Get(restore.GetRestoreJobName(), metav1.GetOptions{})
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
	_, err := h.Deps.KubeClientset.CoreV1().Secrets(s.Namespace).Create(s)
	g.Expect(err).Should(BeNil())
}

// CreateSecret creates secrets based on backup/restore spec
func (h *Helper) CreateSecret(obj interface{}) {
	h.T.Helper()
	if obj1, ok := obj.(*v1alpha1.Backup); ok {
		h.createSecret(obj1.Namespace, obj1.Spec.From.SecretName)
		if obj1.Spec.StorageProvider.S3 != nil && obj1.Spec.StorageProvider.S3.SecretName != "" {
			h.createSecret(obj1.Namespace, obj1.Spec.StorageProvider.S3.SecretName)
		} else if obj1.Spec.StorageProvider.Gcs != nil && obj1.Spec.StorageProvider.Gcs.SecretName != "" {
			h.createSecret(obj1.Namespace, obj1.Spec.StorageProvider.Gcs.SecretName)
		}
	} else if obj2, ok := obj.(*v1alpha1.Restore); ok {
		h.createSecret(obj2.Namespace, obj2.Spec.To.SecretName)
		if obj2.Spec.StorageProvider.S3 != nil && obj2.Spec.StorageProvider.S3.SecretName != "" {
			h.createSecret(obj2.Namespace, obj2.Spec.StorageProvider.S3.SecretName)
		} else if obj2.Spec.StorageProvider.Gcs != nil && obj2.Spec.StorageProvider.Gcs.SecretName != "" {
			h.createSecret(obj2.Namespace, obj2.Spec.StorageProvider.Gcs.SecretName)
		}
	}
}

// CreateTC creates a TidbCluster with name `clusterName` in ns `namespace`
func (h *Helper) CreateTC(namespace, clusterName string) {
	h.T.Helper()
	g := NewGomegaWithT(h.T)
	var err error

	tc := &v1alpha1.TidbCluster{
		Spec: v1alpha1.TidbClusterSpec{
			TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
			TiKV: &v1alpha1.TiKVSpec{
				BaseImage: "pingcap/tikv",
			},
			TiDB: &v1alpha1.TiDBSpec{
				TLSClient: &v1alpha1.TiDBTLSClient{Enabled: true},
			},
		},
	}
	tc.Namespace = namespace
	tc.Name = clusterName
	_, err = h.Deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
	g.Expect(err).Should(BeNil())
	// make sure can read tc from lister
	g.Eventually(func() error {
		_, err := h.Deps.TiDBClusterLister.TidbClusters(tc.Namespace).Get(tc.Name)
		return err
	}, time.Second*10).Should(BeNil())
	g.Expect(err).Should(BeNil())
}
