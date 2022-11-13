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
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/testutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

type helper struct {
	testutils.Helper
}

func newHelper(t *testing.T) *helper {
	h := testutils.NewHelper(t)
	return &helper{*h}
}

func (h *helper) createRestore(restore *v1alpha1.Restore) {
	h.T.Helper()
	g := NewGomegaWithT(h.T)
	var err error

	_, err = h.Deps.Clientset.PingcapV1alpha1().Restores(restore.Namespace).Create(context.TODO(), restore, metav1.CreateOptions{})
	g.Expect(err).Should(BeNil())
	g.Eventually(func() error {
		_, err := h.Deps.RestoreLister.Restores(restore.Namespace).Get(restore.Name)
		return err
	}, time.Second*10).Should(BeNil())
}

func (h *helper) hasCondition(ns string, name string, tp v1alpha1.RestoreConditionType, reasonSub string) {
	h.T.Helper()
	g := NewGomegaWithT(h.T)
	get, err := h.Deps.Clientset.PingcapV1alpha1().Restores(ns).Get(context.TODO(), name, metav1.GetOptions{})
	g.Expect(err).Should(BeNil())
	for _, c := range get.Status.Conditions {
		if c.Type == tp {
			if reasonSub == "" || strings.Contains(c.Reason, reasonSub) {
				return
			}
			h.T.Fatalf("%s do not match reason %s", reasonSub, c.Reason)
		}
	}
	h.T.Fatalf("%s/%s do not has condition type: %s, cur conds: %v", ns, name, tp, get.Status.Conditions)
}

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
		Env: []corev1.EnvVar{
			{
				Name:  "env_name",
				Value: "env_value",
			},
			// existing env name will be overwritten for backup
			{
				Name:  "S3_PROVIDER",
				Value: "fake_provider",
			},
		},
	},
}

func genValidBRRestores() []*v1alpha1.Restore {
	var rs []*v1alpha1.Restore
	for i, sp := range testutils.GenValidStorageProviders() {
		r := &v1alpha1.Restore{
			Spec: v1alpha1.RestoreSpec{
				To: &v1alpha1.TiDBAccessConfig{
					Host:       "localhost",
					SecretName: fmt.Sprintf("backup_secret_%d", i),
				},
				StorageSize:     "1G",
				StorageProvider: sp,
				Type:            v1alpha1.BackupTypeDB,
				BR: &v1alpha1.BRConfig{
					ClusterNamespace: "ns",
					Cluster:          fmt.Sprintf("tidb_%d", i),
					DB:               "dbName",
				},
				Env: []corev1.EnvVar{
					{
						Name:  fmt.Sprintf("env_name_%d", i),
						Value: fmt.Sprintf("env_value_%d", i),
					},
					// existing env name will be overwritten for restore
					{
						Name:  "BR_LOG_TO_TERM",
						Value: "value",
					},
					// existing env name will be overwritten for cleaner
					{
						Name:  "S3_PROVIDER",
						Value: "value",
					},
				},
			},
		}
		r.Namespace = "ns"
		r.Name = fmt.Sprintf("restore_name_%d", i)
		rs = append(rs, r)
	}

	return rs
}

func TestInvalid(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps
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

func TestLightningRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps
	var err error

	// create restore
	restore := validDumpRestore.DeepCopy()
	restore.Namespace = "ns"
	restore.Name = "name"
	helper.createRestore(restore)
	helper.CreateSecret(restore)

	m := NewRestoreManager(deps)
	err = m.Sync(restore)
	g.Expect(err).Should(BeNil())
	helper.hasCondition(restore.Namespace, restore.Name, v1alpha1.RestoreScheduled, "")
	job, err := helper.Deps.KubeClientset.BatchV1().Jobs(restore.Namespace).Get(context.TODO(), restore.GetRestoreJobName(), metav1.GetOptions{})
	g.Expect(err).Should(BeNil())

	// check pod env are set correctly
	env1 := corev1.EnvVar{
		Name:  "env_name",
		Value: "env_value",
	}
	env2Yes := corev1.EnvVar{
		Name:  "S3_PROVIDER",
		Value: "fake_provider",
	}
	env2No := corev1.EnvVar{
		Name:  "S3_PROVIDER",
		Value: "",
	}
	g.Expect(job.Spec.Template.Spec.Containers[0].Env).To(gomega.ContainElement(env1))
	g.Expect(job.Spec.Template.Spec.Containers[0].Env).To(gomega.ContainElement(env2Yes))
	g.Expect(job.Spec.Template.Spec.Containers[0].Env).NotTo(gomega.ContainElement(env2No))
}

func TestBRRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps
	var err error

	for i, restore := range genValidBRRestores() {
		helper.createRestore(restore)
		helper.CreateSecret(restore)
		helper.CreateTC(restore.Spec.BR.ClusterNamespace, restore.Spec.BR.Cluster)

		m := NewRestoreManager(deps)
		err = m.Sync(restore)
		g.Expect(err).Should(BeNil())
		helper.hasCondition(restore.Namespace, restore.Name, v1alpha1.RestoreScheduled, "")
		job, err := helper.Deps.KubeClientset.BatchV1().Jobs(restore.Namespace).Get(context.TODO(), restore.GetRestoreJobName(), metav1.GetOptions{})
		g.Expect(err).Should(BeNil())

		// check pod env are set correctly
		env1 := corev1.EnvVar{
			Name:  fmt.Sprintf("env_name_%d", i),
			Value: fmt.Sprintf("env_value_%d", i),
		}
		env2Yes := corev1.EnvVar{
			Name:  "BR_LOG_TO_TERM",
			Value: "value",
		}
		env2No := corev1.EnvVar{
			Name:  "BR_LOG_TO_TERM",
			Value: string(rune(1)),
		}
		g.Expect(job.Spec.Template.Spec.Containers[0].Env).To(gomega.ContainElement(env1))
		g.Expect(job.Spec.Template.Spec.Containers[0].Env).To(gomega.ContainElement(env2Yes))
		g.Expect(job.Spec.Template.Spec.Containers[0].Env).NotTo(gomega.ContainElement(env2No))
	}
}

func TestBRRestoreByEBS(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps

	cases := []struct {
		name    string
		restore *v1alpha1.Restore
	}{
		{
			name: "restore-volume",
			restore: &v1alpha1.Restore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-1",
					Namespace: "ns-1",
				},
				Spec: v1alpha1.RestoreSpec{
					Type: v1alpha1.BackupTypeFull,
					Mode: v1alpha1.RestoreModeVolumeSnapshot,
					BR: &v1alpha1.BRConfig{
						ClusterNamespace: "ns-1",
						Cluster:          "cluster-1",
					},
					StorageProvider: v1alpha1.StorageProvider{
						S3: &v1alpha1.S3StorageProvider{
							Bucket:   "s3",
							Prefix:   "prefix-",
							Endpoint: "s3://localhost:80",
						},
					},
				},
				Status: v1alpha1.RestoreStatus{},
			},
		},
		{
			name: "restore-volume-complete",
			restore: &v1alpha1.Restore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-2",
					Namespace: "ns-2",
				},
				Spec: v1alpha1.RestoreSpec{
					Type: v1alpha1.BackupTypeFull,
					Mode: v1alpha1.RestoreModeVolumeSnapshot,
					BR: &v1alpha1.BRConfig{
						ClusterNamespace: "ns-2",
						Cluster:          "cluster-2",
					},
					StorageProvider: v1alpha1.StorageProvider{
						S3: &v1alpha1.S3StorageProvider{
							Bucket:   "s3",
							Prefix:   "prefix-",
							Endpoint: "s3://localhost:80",
						},
					},
				},
				Status: v1alpha1.RestoreStatus{
					Phase: v1alpha1.RestoreRunning,
					Conditions: []v1alpha1.RestoreCondition{
						{
							Type:   v1alpha1.RestoreVolumeComplete,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
		{
			name: "restore-data",
			restore: &v1alpha1.Restore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-3",
					Namespace: "ns-3",
				},
				Spec: v1alpha1.RestoreSpec{
					Type: v1alpha1.BackupTypeFull,
					Mode: v1alpha1.RestoreModeVolumeSnapshot,
					BR: &v1alpha1.BRConfig{
						ClusterNamespace: "ns-3",
						Cluster:          "cluster-3",
					},
					StorageProvider: v1alpha1.StorageProvider{
						S3: &v1alpha1.S3StorageProvider{
							Bucket:   "s3",
							Prefix:   "prefix-",
							Endpoint: "s3://localhost:80",
						},
					},
				},
				Status: v1alpha1.RestoreStatus{
					Phase: v1alpha1.RestoreRunning,
					Conditions: []v1alpha1.RestoreCondition{
						{
							Type:   v1alpha1.RestoreVolumeComplete,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
		{
			name: "restore-data-complete",
			restore: &v1alpha1.Restore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-4",
					Namespace: "ns-4",
				},
				Spec: v1alpha1.RestoreSpec{
					Type: v1alpha1.BackupTypeFull,
					Mode: v1alpha1.RestoreModeVolumeSnapshot,
					BR: &v1alpha1.BRConfig{
						ClusterNamespace: "ns-4",
						Cluster:          "cluster-4",
					},
					StorageProvider: v1alpha1.StorageProvider{
						S3: &v1alpha1.S3StorageProvider{
							Bucket:   "s3",
							Prefix:   "prefix-",
							Endpoint: "s3://localhost:80",
						},
					},
				},
				Status: v1alpha1.RestoreStatus{
					Phase: v1alpha1.RestoreRunning,
					Conditions: []v1alpha1.RestoreCondition{
						{
							Type:   v1alpha1.RestoreVolumeComplete,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   v1alpha1.RestoreDataComplete,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			helper.CreateTC(tt.restore.Spec.BR.ClusterNamespace, tt.restore.Spec.BR.Cluster)
			helper.CreateRestore(tt.restore)
			tt.restore.Annotations = map[string]string{
				label.AnnBackupCloudSnapKey: testutils.ConstructRestoreMetaStr(),
			}
			m := NewRestoreManager(deps)
			err := m.Sync(tt.restore)
			g.Expect(err).Should(BeNil())
		})
	}
}
