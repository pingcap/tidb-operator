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

package restore

import (
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/controller"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

var validDumpRestore = &v1alpha1.Restore{
	Spec: v1alpha1.RestoreSpec{
		To: &v1alpha1.TiDBAccessConfig{
			Host:                "localhost",
			SecretName:          "secretName",
			TLSClientSecretName: pointer.StringPtr("secretName"),
		},
		StorageSize: "1G",
		StorageProvider: v1alpha1.StorageProvider{
			S3: &v1alpha1.S3StorageProvider{
				Bucket:   "bname",
				Endpoint: "s3://pingcap/",
			},
		},
	},
}

var validBRRestore = &v1alpha1.Restore{
	Spec: v1alpha1.RestoreSpec{
		To: &v1alpha1.TiDBAccessConfig{
			Host:       "localhost",
			SecretName: "secretName",
		},
		BR: &v1alpha1.BRConfig{
			ClusterNamespace: "ns",
			Cluster:          "tcName",
		},
		Type: v1alpha1.BackupTypeFull,
		StorageProvider: v1alpha1.StorageProvider{
			S3: &v1alpha1.S3StorageProvider{
				Bucket:   "bname",
				Endpoint: "s3://pingcap/",
			},
		},
	},
}

func TestInvalid(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.close()
	deps := helper.deps
	var err error

	restore := &v1alpha1.Restore{}
	restore.Namespace = "ns"
	restore.Name = "restore"
	helper.createRestore(restore)

	m := NewRestoreManager(deps)
	err = m.Sync(restore)
	g.Expect(err).ShouldNot(BeNil())
	helper.hasCondition(restore.Namespace, restore.Name, v1alpha1.RestoreInvalid, "InvalidSpec")
}

func TestDumplingRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.close()
	deps := helper.deps
	var err error

	// create restore
	restore := validDumpRestore.DeepCopy()
	restore.Namespace = "ns"
	restore.Name = "name"
	helper.createRestore(restore)
	helper.createSecret(restore)

	m := NewRestoreManager(deps)
	err = m.Sync(restore)
	g.Expect(err).Should(BeNil())
	helper.hasCondition(restore.Namespace, restore.Name, v1alpha1.RestoreScheduled, "")
	helper.jobCreated(restore)
}

func TestBRRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.close()
	deps := helper.deps
	var err error

	// create restore
	restore := validBRRestore.DeepCopy()
	restore.Namespace = "ns"
	restore.Name = "name"
	helper.createRestore(restore)
	helper.createSecret(restore)
	helper.createTC(restore)

	m := NewRestoreManager(deps)
	err = m.Sync(restore)
	g.Expect(err).Should(BeNil())
	helper.hasCondition(restore.Namespace, restore.Name, v1alpha1.RestoreScheduled, "")
	helper.jobCreated(restore)
}

type helper struct {
	t    *testing.T
	deps *controller.Dependencies
	stop chan struct{}
}

func newHelper(t *testing.T) *helper {
	deps := controller.NewSimpleClientDependencies()
	stop := make(chan struct{})
	deps.InformerFactory.Start(stop)
	deps.KubeInformerFactory.Start(stop)
	deps.InformerFactory.WaitForCacheSync(stop)
	deps.KubeInformerFactory.WaitForCacheSync(stop)

	return &helper{
		stop: stop,
		t:    t,
		deps: deps,
	}
}

func (h *helper) close() {
	close(h.stop)
}

func (h *helper) hasCondition(ns string, name string, tp v1alpha1.RestoreConditionType, reasonSub string) {
	h.t.Helper()
	g := NewGomegaWithT(h.t)
	get, err := h.deps.Clientset.PingcapV1alpha1().Restores(ns).Get(name, metav1.GetOptions{})
	g.Expect(err).Should(BeNil())
	for _, c := range get.Status.Conditions {
		if c.Type == tp {
			if reasonSub == "" || strings.Contains(c.Reason, reasonSub) {
				return
			}
			h.t.Fatalf("%s do not match reason %s", reasonSub, c.Reason)
		}
	}
	h.t.Fatalf("%s/%s do not has condition type: %s, cur conds: %v", ns, name, tp, get.Status.Conditions)
}

func (h *helper) jobCreated(restore *v1alpha1.Restore) {
	h.t.Helper()
	g := NewGomegaWithT(h.t)
	_, err := h.deps.KubeClientset.BatchV1().Jobs(restore.Namespace).Get(restore.GetRestoreJobName(), metav1.GetOptions{})
	g.Expect(err).Should(BeNil())
}

func (h *helper) createSecret(restore *v1alpha1.Restore) {
	h.t.Helper()
	g := NewGomegaWithT(h.t)
	s := &v1.Secret{}
	s.Data = map[string][]byte{
		constants.TidbPasswordKey:   []byte("dummy"),
		constants.GcsCredentialsKey: []byte("dummy"),
		constants.S3AccessKey:       []byte("dummy"),
		constants.S3SecretKey:       []byte("dummy"),
	}
	s.Namespace = restore.Namespace
	s.Name = restore.Spec.To.SecretName
	_, err := h.deps.KubeClientset.CoreV1().Secrets(s.Namespace).Create(s)
	g.Expect(err).Should(BeNil())
}

func (h *helper) createRestore(restore *v1alpha1.Restore) {
	h.t.Helper()
	g := NewGomegaWithT(h.t)
	var err error

	_, err = h.deps.Clientset.PingcapV1alpha1().Restores(restore.Namespace).Create(restore)
	g.Expect(err).Should(BeNil())
	g.Eventually(func() error {
		_, err := h.deps.RestoreLister.Restores(restore.Namespace).Get(restore.Name)
		return err
	}, time.Second*10).Should(BeNil())
}

func (h *helper) createTC(restore *v1alpha1.Restore) {
	h.t.Helper()
	g := NewGomegaWithT(h.t)
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
	tc.Namespace = restore.Spec.BR.ClusterNamespace
	tc.Name = restore.Spec.BR.Cluster
	_, err = h.deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
	g.Expect(err).Should(BeNil())
	// make sure can read tc from lister
	g.Eventually(func() error {
		_, err := h.deps.TiDBClusterLister.TidbClusters(tc.Namespace).Get(tc.Name)
		return err
	}, time.Second*10).Should(BeNil())
	g.Expect(err).Should(BeNil())
}
