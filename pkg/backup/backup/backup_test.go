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

package backup

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

func invalidBackup() *v1alpha1.Backup {
	b := &v1alpha1.Backup{}
	b.Namespace = "ns"
	b.Name = "invalid_name"
	return b
}

func validDumplingBackup() *v1alpha1.Backup {
	b := &v1alpha1.Backup{
		Spec: v1alpha1.BackupSpec{
			From: &v1alpha1.TiDBAccessConfig{
				Host:                "localhost",
				SecretName:          "secretName",
				TLSClientSecretName: pointer.StringPtr("secretName"),
			},
			StorageSize: "1G",
			StorageProvider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					Bucket: "s3",
					Prefix: "prefix-",
				},
			},
		},
	}

	b.Namespace = "ns"
	b.Name = "dump_name"

	return b
}

func validBRBackup() *v1alpha1.Backup {
	b := &v1alpha1.Backup{
		Spec: v1alpha1.BackupSpec{
			From: &v1alpha1.TiDBAccessConfig{
				Host:       "localhost",
				SecretName: "secretName",
			},
			StorageSize: "1G",
			StorageProvider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					Bucket:   "s3",
					Prefix:   "prefix-",
					Endpoint: "s3://localhost:80",
				},
			},
			Type: v1alpha1.BackupTypeDB,
			BR: &v1alpha1.BRConfig{
				ClusterNamespace: "ns",
				Cluster:          "tidb",
				DB:               "dbName",
			},
		},
	}

	b.Namespace = "ns"
	b.Name = "backup_name"

	return b
}

func TestBackupManagerDumpling(t *testing.T) {
	g := NewGomegaWithT(t)

	helper := newHelper(t)
	defer helper.close()
	deps := helper.deps
	var err error

	bm := NewBackupManager(deps).(*backupManager)

	// create backup
	backup := validDumplingBackup()
	_, err = deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace).Create(backup)
	g.Expect(err).Should(BeNil())

	// create relate secret
	helper.createSecret(backup)

	err = bm.syncBackupJob(backup)
	g.Expect(err).Should(BeNil())
	helper.hasCondition(backup.Namespace, backup.Name, v1alpha1.BackupScheduled, "")
	_, err = deps.KubeClientset.BatchV1().Jobs(backup.Namespace).Get(backup.GetBackupJobName(), metav1.GetOptions{})
	g.Expect(err).Should(BeNil())
}

func TestBackupManagerBR(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.close()
	deps := helper.deps
	var err error

	bm := NewBackupManager(deps).(*backupManager)

	// test invalid Backup spec
	backup := invalidBackup()
	_, err = deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace).Create(backup)
	g.Expect(err).Should(BeNil())
	err = bm.syncBackupJob(backup)
	g.Expect(err).ShouldNot(BeNil())
	helper.hasCondition(backup.Namespace, backup.Name, v1alpha1.BackupInvalid, "")

	// create and use valid backup
	backup = validBRBackup()
	_, err = deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace).Create(backup)
	g.Expect(err).Should(BeNil())

	// create relate secret
	helper.createSecret(backup)

	// failed to get relate tc
	err = bm.syncBackupJob(backup)
	g.Expect(err).ShouldNot(BeNil())
	helper.hasCondition(backup.Namespace, backup.Name, v1alpha1.BackupRetryFailed, "failed to fetch tidbcluster")

	// create relate tc and try again should success and job created.
	helper.createTC(backup)
	err = bm.syncBackupJob(backup)
	g.Expect(err).Should(BeNil())
	helper.hasCondition(backup.Namespace, backup.Name, v1alpha1.BackupScheduled, "")
	_, err = deps.KubeClientset.BatchV1().Jobs(backup.Namespace).Get(backup.GetBackupJobName(), metav1.GetOptions{})
	g.Expect(err).Should(BeNil())
}

func TestClean(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.close()
	deps := helper.deps

	backup := validBRBackup()
	_, err := deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace).Create(backup)
	g.Expect(err).Should(BeNil())
	helper.createSecret(backup)
	helper.createTC(backup)

	// make the backup need to be clean
	backup.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	backup.Spec.CleanPolicy = v1alpha1.CleanPolicyTypeDelete

	statusUpdater := controller.NewRealBackupConditionUpdater(deps.Clientset, deps.BackupLister, deps.Recorder)
	bc := NewBackupCleaner(deps, statusUpdater)

	// test empty backup.Status.BackupPath
	backup.Status.BackupPath = ""
	err = bc.Clean(backup)
	g.Expect(err).Should(BeNil())
	helper.hasCondition(backup.Namespace, backup.Name, v1alpha1.BackupClean, "")

	// test clean job created
	backup.Status.BackupPath = "/path"
	err = bc.Clean(backup)
	g.Expect(err).Should(BeNil())
	helper.hasCondition(backup.Namespace, backup.Name, v1alpha1.BackupClean, "")
	_, err = deps.KubeClientset.BatchV1().Jobs(backup.Namespace).Get(backup.GetCleanJobName(), metav1.GetOptions{})
	g.Expect(err).Should(BeNil())
	// test already have a clean job running
	g.Eventually(func() error {
		_, err := deps.JobLister.Jobs(backup.Namespace).Get(backup.GetCleanJobName())
		return err
	}, time.Second*10).Should(BeNil())
	err = bc.Clean(backup)
	g.Expect(err).Should(BeNil())
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
		t:    t,
		deps: deps,
		stop: stop,
	}
}

func (h *helper) close() {
	close(h.stop)
}

func (h *helper) hasCondition(ns string, name string, tp v1alpha1.BackupConditionType, reasonSub string) {
	h.t.Helper()
	g := NewGomegaWithT(h.t)
	get, err := h.deps.Clientset.PingcapV1alpha1().Backups(ns).Get(name, metav1.GetOptions{})
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

func (h *helper) createSecret(backup *v1alpha1.Backup) {
	g := NewGomegaWithT(h.t)
	s := &v1.Secret{}
	s.Data = map[string][]byte{
		constants.TidbPasswordKey:   []byte("dummy"),
		constants.GcsCredentialsKey: []byte("dummy"),
		constants.S3AccessKey:       []byte("dummy"),
		constants.S3SecretKey:       []byte("dummy"),
	}
	s.Namespace = backup.Namespace
	s.Name = backup.Spec.From.SecretName
	_, err := h.deps.KubeClientset.CoreV1().Secrets(s.Namespace).Create(s)
	g.Expect(err).Should(BeNil())
}

func (h *helper) createTC(backup *v1alpha1.Backup) {
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
	tc.Namespace = backup.Spec.BR.ClusterNamespace
	tc.Name = backup.Spec.BR.Cluster
	_, err = h.deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
	g.Expect(err).Should(BeNil())
	// make sure can read tc from lister
	g.Eventually(func() error {
		_, err := h.deps.TiDBClusterLister.TidbClusters(tc.Namespace).Get(tc.Name)
		return err
	}, time.Second*10).Should(BeNil())
	g.Expect(err).Should(BeNil())
}
