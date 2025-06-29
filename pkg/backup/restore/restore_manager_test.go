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
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/backup/testutils"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
	"k8s.io/utils/ptr"
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

func (h *helper) createRestoreWarmupJobFailed(restore *v1alpha1.Restore) {
	g := NewGomegaWithT(h.T)
	deps := h.Deps
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "warm-up",
			Namespace: restore.Namespace,
			Labels:    label.NewRestore().RestoreWarmUpJob().Restore(restore.Name),
		},
		Status: batchv1.JobStatus{
			CompletionTime: &metav1.Time{},
			Conditions: []batchv1.JobCondition{
				{
					Type:   batchv1.JobFailed,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	_, err := deps.KubeClientset.BatchV1().Jobs(restore.Namespace).Create(context.TODO(), job, metav1.CreateOptions{})
	g.Expect(err).Should(BeNil())

	g.Eventually(func() error {
		_, err := deps.JobLister.Jobs(job.GetNamespace()).Get(job.GetName())
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

func genValidPiTRRestores() []*v1alpha1.Restore {
	var rs []*v1alpha1.Restore
	for i, sp := range testutils.GenValidStorageProviders() {
		r := &v1alpha1.Restore{
			Spec: v1alpha1.RestoreSpec{
				To: &v1alpha1.TiDBAccessConfig{
					Host:       "localhost",
					SecretName: fmt.Sprintf("pitr_secret_%d", i),
				},
				StorageSize:       "1G",
				StorageProvider:   sp,
				Type:              v1alpha1.BackupTypeFull,
				Mode:              v1alpha1.RestoreModePiTR,
				PitrRestoredTs:    "443123456789",
				LogRestoreStartTs: "443123456780",
				BR: &v1alpha1.BRConfig{
					ClusterNamespace: "ns",
					Cluster:          fmt.Sprintf("tidb_%d", i),
					DB:               "dbName",
				},
				Env: []corev1.EnvVar{
					{
						Name:  fmt.Sprintf("pitr_env_name_%d", i),
						Value: fmt.Sprintf("pitr_env_value_%d", i),
					},
					// existing env name will be overwritten for pitr restore
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
		r.Name = fmt.Sprintf("pitr_restore_name_%d", i)
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
		helper.CreateTC(restore.Spec.BR.ClusterNamespace, restore.Spec.BR.Cluster, false, false)

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

		tc2, err := deps.TiDBClusterLister.TidbClusters(restore.Spec.BR.ClusterNamespace).Get(restore.Spec.BR.Cluster)
		g.Expect(err).Should(BeNil())
		g.Expect(tc2.Status.TiKV.PiTRStatus.State).To(Equal(v1alpha1.PiTRStateInactive))
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
						Local: &v1alpha1.LocalStorageProvider{
							//	Prefix: "prefix",
							Volume: corev1.Volume{
								Name: "nfs",
								VolumeSource: corev1.VolumeSource{
									NFS: &corev1.NFSVolumeSource{
										Server:   "fake-server",
										Path:     "/tmp",
										ReadOnly: true,
									},
								},
							},
							VolumeMount: corev1.VolumeMount{
								Name:      "nfs",
								MountPath: "/tmp",
							},
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
						Local: &v1alpha1.LocalStorageProvider{
							//	Prefix: "prefix1",
							Volume: corev1.Volume{
								Name: "nfs",
								VolumeSource: corev1.VolumeSource{
									NFS: &corev1.NFSVolumeSource{
										Server:   "fake-server",
										Path:     "/tmp",
										ReadOnly: true,
									},
								},
							},
							VolumeMount: corev1.VolumeMount{
								Name:      "nfs",
								MountPath: "/tmp",
							},
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
						Local: &v1alpha1.LocalStorageProvider{
							//	Prefix: "prefix",
							Volume: corev1.Volume{
								Name: "nfs",
								VolumeSource: corev1.VolumeSource{
									NFS: &corev1.NFSVolumeSource{
										Server:   "fake-server",
										Path:     "/tmp",
										ReadOnly: true,
									},
								},
							},
							VolumeMount: corev1.VolumeMount{
								Name:      "nfs",
								MountPath: "/tmp",
							},
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
						Local: &v1alpha1.LocalStorageProvider{
							//	Prefix: "prefix",
							Volume: corev1.Volume{
								Name: "nfs",
								VolumeSource: corev1.VolumeSource{
									NFS: &corev1.NFSVolumeSource{
										Server:   "fake-server",
										Path:     "/tmp",
										ReadOnly: true,
									},
								},
							},
							VolumeMount: corev1.VolumeMount{
								Name:      "nfs",
								MountPath: "/tmp",
							},
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
	//generate the restore meta in local nfs
	err := os.WriteFile("/tmp/restoremeta", []byte(testutils.ConstructRestoreMetaStr()), 0644) //nolint:gosec
	g.Expect(err).To(Succeed())

	//generate the backup meta in local nfs, tiflash check need backupmeta to validation
	err = os.WriteFile("/tmp/backupmeta", []byte(testutils.ConstructRestoreMetaStr()), 0644) //nolint:gosec
	g.Expect(err).To(Succeed())
	defer func() {
		err = os.Remove("/tmp/restoremeta")
		g.Expect(err).To(Succeed())

		err = os.Remove("/tmp/backupmeta")
		g.Expect(err).To(Succeed())
	}()

	os.Setenv(constants.AWSRegionEnv, "us-west-1")

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {

			helper.CreateTC(tt.restore.Spec.BR.ClusterNamespace, tt.restore.Spec.BR.Cluster, true, true)
			helper.CreateRestore(tt.restore)
			m := NewRestoreManager(deps)
			err := m.Sync(tt.restore)
			g.Expect(err).Should(BeNil())
		})
	}
}

func TestInvalidReplicasBRRestoreByEBS(t *testing.T) {
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
						Local: &v1alpha1.LocalStorageProvider{
							//	Prefix: "prefix",
							Volume: corev1.Volume{
								Name: "nfs",
								VolumeSource: corev1.VolumeSource{
									NFS: &corev1.NFSVolumeSource{
										Server:   "fake-server",
										Path:     "/tmp",
										ReadOnly: true,
									},
								},
							},
							VolumeMount: corev1.VolumeMount{
								Name:      "nfs",
								MountPath: "/tmp",
							},
						},
					},
				},
				Status: v1alpha1.RestoreStatus{},
			},
		},
	}

	// Verify invalid tc with mismatch tikv replicas
	//generate the restore meta in local nfs, with only 2 tikv replicas
	err := os.WriteFile("/tmp/restoremeta", []byte(testutils.ConstructRestore2TiKVMetaStr()), 0644) //nolint:gosec
	g.Expect(err).To(Succeed())

	//generate the backup meta in local nfs, tiflash check need backupmeta to validation
	err = os.WriteFile("/tmp/backupmeta", []byte(testutils.ConstructRestore2TiKVMetaStr()), 0644) //nolint:gosec
	g.Expect(err).To(Succeed())
	defer func() {
		err = os.Remove("/tmp/restoremeta")
		g.Expect(err).To(Succeed())

		err = os.Remove("/tmp/backupmeta")
		g.Expect(err).To(Succeed())
	}()

	t.Run(cases[0].name, func(t *testing.T) {
		helper.CreateTC(cases[0].restore.Spec.BR.ClusterNamespace, cases[0].restore.Spec.BR.Cluster, true, true)
		helper.CreateRestore(cases[0].restore)
		m := NewRestoreManager(deps)
		err := m.Sync(cases[0].restore)
		g.Expect(err).Should(MatchError("tikv replica mismatch"))
	})
}

func TestInvalidModeBRRestoreByEBS(t *testing.T) {
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
						Local: &v1alpha1.LocalStorageProvider{
							//	Prefix: "prefix",
							Volume: corev1.Volume{
								Name: "nfs",
								VolumeSource: corev1.VolumeSource{
									NFS: &corev1.NFSVolumeSource{
										Server:   "fake-server",
										Path:     "/tmp",
										ReadOnly: true,
									},
								},
							},
							VolumeMount: corev1.VolumeMount{
								Name:      "nfs",
								MountPath: "/tmp",
							},
						},
					},
				},
				Status: v1alpha1.RestoreStatus{},
			},
		},
	}

	// Verify invalid tc with mismatch tikv replicas
	//generate the restore meta in local nfs, with only 2 tikv replicas
	err := os.WriteFile("/tmp/restoremeta", []byte(testutils.ConstructRestoreMetaStr()), 0644) //nolint:gosec
	g.Expect(err).To(Succeed())

	//generate the backup meta in local nfs, tiflash check need backupmeta to validation
	err = os.WriteFile("/tmp/backupmeta", []byte(testutils.ConstructRestoreMetaStr()), 0644) //nolint:gosec
	g.Expect(err).To(Succeed())
	defer func() {
		err = os.Remove("/tmp/restoremeta")
		g.Expect(err).To(Succeed())

		err = os.Remove("/tmp/backupmeta")
		g.Expect(err).To(Succeed())
	}()

	t.Run(cases[0].name, func(t *testing.T) {
		helper.CreateTC(cases[0].restore.Spec.BR.ClusterNamespace, cases[0].restore.Spec.BR.Cluster, true, false)
		helper.CreateRestore(cases[0].restore)
		m := NewRestoreManager(deps)
		err := m.Sync(cases[0].restore)
		g.Expect(err).Should(MatchError("recovery mode is off"))
	})
}

func TestVolumeNumMismatchBRRestoreByEBS(t *testing.T) {
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
						Local: &v1alpha1.LocalStorageProvider{
							//	Prefix: "prefix",
							Volume: corev1.Volume{
								Name: "nfs",
								VolumeSource: corev1.VolumeSource{
									NFS: &corev1.NFSVolumeSource{
										Server:   "fake-server",
										Path:     "/tmp",
										ReadOnly: true,
									},
								},
							},
							VolumeMount: corev1.VolumeMount{
								Name:      "nfs",
								MountPath: "/tmp",
							},
						},
					},
				},
				Status: v1alpha1.RestoreStatus{},
			},
		},
	}

	// Verify invalid tc with mismatch tikv replicas
	//generate the restore meta in local nfs, with 3 volumes for each tikv
	err := os.WriteFile("/tmp/restoremeta", []byte(testutils.ConstructRestoreTiKVVolumesMetaWithStr()), 0644) //nolint:gosec
	g.Expect(err).To(Succeed())

	//generate the backup meta in local nfs, tiflash check need backupmeta to validation
	err = os.WriteFile("/tmp/backupmeta", []byte(testutils.ConstructRestoreTiKVVolumesMetaWithStr()), 0644) //nolint:gosec
	g.Expect(err).To(Succeed())
	defer func() {
		err = os.Remove("/tmp/restoremeta")
		g.Expect(err).To(Succeed())

		err = os.Remove("/tmp/backupmeta")
		g.Expect(err).To(Succeed())
	}()

	t.Run(cases[0].name, func(t *testing.T) {
		helper.CreateTC(cases[0].restore.Spec.BR.ClusterNamespace, cases[0].restore.Spec.BR.Cluster, true, true)
		helper.CreateRestore(cases[0].restore)
		m := NewRestoreManager(deps)
		err := m.Sync(cases[0].restore)
		g.Expect(err).Should(MatchError("additional volumes mismatched"))
	})
}

func TestFailWarmupBRRestoreByEBS(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps

	errorCases := []struct {
		name    string
		restore *v1alpha1.Restore
	}{
		{
			name: "restore-volume-warmup-check-sync",
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
					Warmup:         v1alpha1.RestoreWarmupModeSync,
					WarmupStrategy: v1alpha1.RestoreWarmupStrategyCheckOnly,
					StorageProvider: v1alpha1.StorageProvider{
						Local: &v1alpha1.LocalStorageProvider{
							//	Prefix: "prefix",
							Volume: corev1.Volume{
								Name: "nfs",
								VolumeSource: corev1.VolumeSource{
									NFS: &corev1.NFSVolumeSource{
										Server:   "fake-server",
										Path:     "/tmp",
										ReadOnly: true,
									},
								},
							},
							VolumeMount: corev1.VolumeMount{
								Name:      "nfs",
								MountPath: "/tmp",
							},
						},
					},
				},
				Status: v1alpha1.RestoreStatus{
					Conditions: []v1alpha1.RestoreCondition{
						{
							Type:   v1alpha1.RestoreVolumeComplete,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   v1alpha1.RestoreWarmUpStarted,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
		{
			name: "restore-volume-warmup-check-async",
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
					Warmup:         v1alpha1.RestoreWarmupModeASync,
					WarmupStrategy: v1alpha1.RestoreWarmupStrategyCheckOnly,
					StorageProvider: v1alpha1.StorageProvider{
						Local: &v1alpha1.LocalStorageProvider{
							//	Prefix: "prefix",
							Volume: corev1.Volume{
								Name: "nfs",
								VolumeSource: corev1.VolumeSource{
									NFS: &corev1.NFSVolumeSource{
										Server:   "fake-server",
										Path:     "/tmp",
										ReadOnly: true,
									},
								},
							},
							VolumeMount: corev1.VolumeMount{
								Name:      "nfs",
								MountPath: "/tmp",
							},
						},
					},
				},
				Status: v1alpha1.RestoreStatus{
					Conditions: []v1alpha1.RestoreCondition{
						{
							Type:   v1alpha1.RestoreVolumeComplete,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   v1alpha1.RestoreWarmUpStarted,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   v1alpha1.RestoreTiKVComplete,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
	}

	successCases := []struct {
		name    string
		restore *v1alpha1.Restore
	}{
		{
			name: "restore-volume-warmup-no-check-sync",
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
					Warmup: v1alpha1.RestoreWarmupModeSync,
					StorageProvider: v1alpha1.StorageProvider{
						Local: &v1alpha1.LocalStorageProvider{
							//	Prefix: "prefix",
							Volume: corev1.Volume{
								Name: "nfs",
								VolumeSource: corev1.VolumeSource{
									NFS: &corev1.NFSVolumeSource{
										Server:   "fake-server",
										Path:     "/tmp",
										ReadOnly: true,
									},
								},
							},
							VolumeMount: corev1.VolumeMount{
								Name:      "nfs",
								MountPath: "/tmp",
							},
						},
					},
				},
				Status: v1alpha1.RestoreStatus{
					Conditions: []v1alpha1.RestoreCondition{
						{
							Type:   v1alpha1.RestoreVolumeComplete,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   v1alpha1.RestoreWarmUpStarted,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
		{
			name: "restore-volume-warmup-no-check-async",
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
					Warmup: v1alpha1.RestoreWarmupModeASync,
					StorageProvider: v1alpha1.StorageProvider{
						Local: &v1alpha1.LocalStorageProvider{
							//	Prefix: "prefix",
							Volume: corev1.Volume{
								Name: "nfs",
								VolumeSource: corev1.VolumeSource{
									NFS: &corev1.NFSVolumeSource{
										Server:   "fake-server",
										Path:     "/tmp",
										ReadOnly: true,
									},
								},
							},
							VolumeMount: corev1.VolumeMount{
								Name:      "nfs",
								MountPath: "/tmp",
							},
						},
					},
				},
				Status: v1alpha1.RestoreStatus{
					Conditions: []v1alpha1.RestoreCondition{
						{
							Type:   v1alpha1.RestoreVolumeComplete,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   v1alpha1.RestoreWarmUpStarted,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   v1alpha1.RestoreTiKVComplete,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
	}

	err := os.WriteFile("/tmp/restoremeta", []byte(testutils.ConstructRestoreMetaStr()), 0644) //nolint:gosec
	g.Expect(err).To(Succeed())

	err = os.WriteFile("/tmp/backupmeta", []byte(testutils.ConstructRestoreMetaStr()), 0644) //nolint:gosec
	g.Expect(err).To(Succeed())
	defer func() {
		err = os.Remove("/tmp/restoremeta")
		g.Expect(err).To(Succeed())

		err = os.Remove("/tmp/backupmeta")
		g.Expect(err).To(Succeed())
	}()

	for _, tt := range errorCases {
		t.Run(tt.name, func(t *testing.T) {
			helper.CreateTC(tt.restore.Spec.BR.ClusterNamespace, tt.restore.Spec.BR.Cluster, true, true)
			helper.CreateRestore(tt.restore)
			helper.createRestoreWarmupJobFailed(tt.restore)
			m := NewRestoreManager(deps)
			err := m.Sync(tt.restore)
			g.Expect(err).Should(MatchError(fmt.Sprintf("warmup job %s/warm-up failed", tt.restore.Namespace)))
		})
	}

	for _, tt := range successCases {
		t.Run(tt.name, func(t *testing.T) {
			helper.CreateTC(tt.restore.Spec.BR.ClusterNamespace, tt.restore.Spec.BR.Cluster, true, true)
			helper.CreateRestore(tt.restore)
			helper.createRestoreWarmupJobFailed(tt.restore)
			m := NewRestoreManager(deps)
			err := m.Sync(tt.restore)
			g.Expect(err).Should(BeNil())
		})
	}
}

func TestGenerateWarmUpArgs(t *testing.T) {
	mountPoints := []corev1.VolumeMount{
		{
			Name:      "data",
			MountPath: constants.TiKVDataVolumeMountPath,
		},
		{
			Name:      "logs",
			MountPath: "/logs",
		},
	}
	testCases := []struct {
		name     string
		strategy v1alpha1.RestoreWarmupStrategy
		expected []string
		errMsg   string
	}{
		{
			name:     "all by block",
			strategy: v1alpha1.RestoreWarmupStrategyFio,
			expected: []string{"--block", constants.TiKVDataVolumeMountPath, "--block", "/logs"},
		},
		{
			name:     "data by fsr other by block",
			strategy: v1alpha1.RestoreWarmupStrategyFsr,
			expected: []string{"--block", "/logs"},
		},
		{
			name:     "data by fs other by block",
			strategy: v1alpha1.RestoreWarmupStrategyHybrid,
			expected: []string{"--fs", constants.TiKVDataVolumeMountPath, "--block", "/logs"},
		},
		{
			name:     "check-wal-only",
			strategy: v1alpha1.RestoreWarmupStrategyCheckOnly,
			expected: []string{"--exit-on-corruption", "--block", "/logs"},
		},
		{
			name:     "unknown strategy",
			strategy: "unknown",
			errMsg:   `unknown warmup strategy "unknown"`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args, err := generateWarmUpArgs(tc.strategy, mountPoints)
			if tc.errMsg != "" {
				require.EqualError(t, err, tc.errMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, args)
		})
	}
}

func TestPiTRRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps

	for i, restore := range genValidPiTRRestores() {
		helper.createRestore(restore)
		helper.CreateSecret(restore)
		helper.CreateTC(restore.Spec.BR.ClusterNamespace, restore.Spec.BR.Cluster, false, false)

		// Initialize PiTRStatus for the TiKV to avoid nil pointer dereference
		tc, err := deps.TiDBClusterLister.TidbClusters(restore.Spec.BR.ClusterNamespace).Get(restore.Spec.BR.Cluster)
		g.Expect(err).Should(BeNil())
		tc.Status.TiKV.PiTRStatus = v1alpha1.PiTRStatus{}
		_, err = deps.TiDBClusterControl.Update(tc)
		g.Expect(err).Should(BeNil())

		// Create TiKV StatefulSet and ConfigMap for PiTR testing
		helper.createTiKVStatefulSetAndConfigMap(restore.Spec.BR.ClusterNamespace, restore.Spec.BR.Cluster)

		m := NewRestoreManager(deps)

		// First sync should return RequeueError as it transitions to WaitingForConfig state
		err = m.Sync(restore)
		g.Expect(err).Should(MatchError(ContainSubstring("config reset, waiting for configmap updated")))

		// Verify the state is now WaitingForConfig
		tc2, err := deps.TiDBClusterLister.TidbClusters(restore.Spec.BR.ClusterNamespace).Get(restore.Spec.BR.Cluster)
		g.Expect(err).Should(BeNil())
		g.Expect(tc2.Status.TiKV.PiTRStatus.State).To(Equal(v1alpha1.PiTRStateWaitingForConfig))
		g.Expect(tc2.Spec.TiKV.Config.Get("gc.ratio-threshold").MustFloat()).To(Equal(-1.0))

		// Simulate the configmap being updated by the operator, set state to Running
		helper.updateConfigMapRatioThreshold(restore.Spec.BR.ClusterNamespace, restore.Spec.BR.Cluster, -1.0)
		err = m.Sync(restore)
		g.Expect(err).Should(BeNil())
		g.Expect(tc2.Status.TiKV.PiTRStatus.State).To(Equal(v1alpha1.PiTRStateRunning))
		// Second sync should succeed now
		helper.hasCondition(restore.Namespace, restore.Name, v1alpha1.RestoreScheduled, "")
		job, err := helper.Deps.KubeClientset.BatchV1().Jobs(restore.Namespace).Get(context.TODO(), restore.GetRestoreJobName(), metav1.GetOptions{})
		g.Expect(err).Should(BeNil())

		tc3, err := deps.TiDBClusterLister.TidbClusters(restore.Spec.BR.ClusterNamespace).Get(restore.Spec.BR.Cluster)
		g.Expect(err).Should(BeNil())
		st := tc3.Status.TiKV.PiTRStatus
		g.Expect(st.State).To(Equal(v1alpha1.PiTRStateRunning))

		// check pod env are set correctly for PiTR restore
		env1 := corev1.EnvVar{
			Name:  fmt.Sprintf("pitr_env_name_%d", i),
			Value: fmt.Sprintf("pitr_env_value_%d", i),
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

		// check PiTR specific args are set correctly
		args := job.Spec.Template.Spec.Containers[0].Args
		g.Expect(args).To(gomega.ContainElement("--mode=pitr"))
		g.Expect(args).To(gomega.ContainElement("--pitrRestoredTs=443123456789"))

		// Complete the restore
		err = m.UpdateCondition(restore, &v1alpha1.RestoreCondition{
			Type:   v1alpha1.RestoreComplete,
			Status: corev1.ConditionTrue,
		})
		g.Expect(err).Should(BeNil())

		for range 3 {
			rs, err := helper.Deps.RestoreLister.Restores(restore.Namespace).List(labels.Everything())
			g.Expect(err).Should(BeNil())
			for _, r := range rs {
				if !v1alpha1.IsRestoreComplete(r) {
					continue
				}
			}
			time.Sleep(100 * time.Microsecond)
		}
		err = m.Sync(restore)
		// Wait for reconile...
		g.Expect(err).Should(MatchError(ContainSubstring("config reset, waiting for configmap updated")))
		// Verify the state is now WaitingForConfig again
		tc2, err = deps.TiDBClusterLister.TidbClusters(restore.Spec.BR.ClusterNamespace).Get(restore.Spec.BR.Cluster)
		g.Expect(err).Should(BeNil())
		g.Expect(tc2.Status.TiKV.PiTRStatus.State).To(Equal(v1alpha1.PiTRStateWaitingForConfig))

		// Verify the original configmap is stored
		helper.updateConfigMapRatioThreshold(restore.Spec.BR.ClusterNamespace, restore.Spec.BR.Cluster, math.NaN())
		err = m.Sync(restore)
		g.Expect(err).Should(BeNil())
		tc2, err = deps.TiDBClusterLister.TidbClusters(restore.Spec.BR.ClusterNamespace).Get(restore.Spec.BR.Cluster)
		g.Expect(err).Should(BeNil())
		g.Expect(tc2.Status.TiKV.PiTRStatus.State).To(Equal(v1alpha1.PiTRStateInactive))
	}
}

func TestPiTRHelperFunctions(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.Close()
	deps := helper.Deps

	// Create a test TiDB cluster with PiTR status
	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tc",
			Namespace: "ns",
		},
		Spec: v1alpha1.TidbClusterSpec{
			TiKV: &v1alpha1.TiKVSpec{
				Config: &v1alpha1.TiKVConfigWraper{},
			},
		},
		Status: v1alpha1.TidbClusterStatus{
			TiKV: v1alpha1.TiKVStatus{
				PiTRStatus: v1alpha1.PiTRStatus{},
			},
		},
	}

	// Initialize TiKV config if nil
	tc.Spec.TiKV.Config = v1alpha1.NewTiKVConfig()
	tc.Spec.TiKV.Config.Set("gc.ratio-threshold", 1.2)
	helper.CreateTC("ns", "test-tc", false, false)

	// Create TiKV StatefulSet and ConfigMap for PiTR testing
	helper.createTiKVStatefulSetAndConfigMap("ns", "test-tc")

	m := PiTRManager{deps}

	// Test configMapOfTiKV - should successfully retrieve and parse TiKV config map
	tikvConfig, err := m.configMapOfTiKV(tc)
	g.Expect(err).Should(BeNil())
	g.Expect(tikvConfig).ShouldNot(BeNil())
	g.Expect(tikvConfig.Get("gc.ratio-threshold").MustFloat()).Should(Equal(1.2))

	// Test pitrEnable - first call should transition to WaitingForConfig state
	err = m.Enable(tc)
	g.Expect(err).Should(MatchError(ContainSubstring("config reset, waiting for configmap updated")))
	g.Expect(tc.Status.TiKV.PiTRStatus.State).Should(Equal(v1alpha1.PiTRStateWaitingForConfig))
	g.Expect(tc.Status.TiKV.PiTRStatus.OriginConfigMap).ShouldNot(BeNil())
	g.Expect(*tc.Status.TiKV.PiTRStatus.OriginConfigMap.GCRatioThreshold).Should(Equal(1.2))
	// Verify that GC threshold is set to -1.0 for PiTR
	g.Expect(tc.Spec.TiKV.Config.Get("gc.ratio-threshold").MustFloat()).Should(Equal(-1.0))

	// Test configMapOfTiKV after Enable - config should reflect the PiTR setting
	// Update the ConfigMap to simulate the new config being applied
	// Update the ConfigMap to set ratio-threshold to -1.0 for PiTR
	helper.updateConfigMapRatioThreshold("ns", "test-tc", -1.0)

	err = m.Enable(tc)
	g.Expect(err).Should(BeNil())
	g.Expect(tc.Status.TiKV.PiTRStatus.State).Should(Equal(v1alpha1.PiTRStateRunning))

	// Test pitrDisable - should set state to WaitingForConfig and return requeue error
	err = m.disable(tc)
	g.Expect(err).Should(MatchError(ContainSubstring("config reset, waiting for configmap updated")))
	g.Expect(tc.Status.TiKV.PiTRStatus.State).Should(Equal(v1alpha1.PiTRStateWaitingForConfig))
	g.Expect(tc.Status.TiKV.PiTRStatus.OriginConfigMap).Should(BeNil())

	// Test disable with state WaitingForConfig and original configmap applied
	// First restore the original config in the configmap
	helper.updateConfigMapRatioThreshold("ns", "test-tc", 1.2)

	// Test disable when in WaitingForConfig state - should complete disable successfully
	err = m.disable(tc)
	g.Expect(err).Should(BeNil())
	g.Expect(tc.Status.TiKV.PiTRStatus.State).Should(Equal(v1alpha1.PiTRStateInactive))

	// Test MaybeDisable with no restores
	tc.Status.TiKV.PiTRStatus.State = v1alpha1.PiTRStateRunning
	tc.Status.TiKV.PiTRStatus.OriginConfigMap = &v1alpha1.PiTROverriddenConfig{
		GCRatioThreshold: ptr.To(1.2),
	}
	err = m.MaybeDisable(tc)
	g.Expect(err).Should(MatchError(ContainSubstring("config reset, waiting for configmap updated"))) // Should try to disable

	// Test MaybeDisable with running restores
	runningRestore := &v1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-restore",
			Namespace: "ns",
		},
		Spec: v1alpha1.RestoreSpec{
			BR: &v1alpha1.BRConfig{
				Cluster:          "test-tc",
				ClusterNamespace: "ns",
			},
		},
		Status: v1alpha1.RestoreStatus{
			Conditions: []v1alpha1.RestoreCondition{
				{
					Type:   v1alpha1.RestoreRunning,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	helper.CreateRestore(runningRestore)

	err = m.MaybeDisable(tc)
	g.Expect(err).Should(BeNil()) // Should not disable with running restores

	// Test pitrTasksAreDone with no restores
	done := pitrTasksAreDone([]*v1alpha1.Restore{})
	g.Expect(done).Should(BeTrue())

	// Test pitrTasksAreDone with completed restore
	completedRestore := &v1alpha1.Restore{
		Status: v1alpha1.RestoreStatus{
			Conditions: []v1alpha1.RestoreCondition{
				{
					Type:   v1alpha1.RestoreComplete,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	done = pitrTasksAreDone([]*v1alpha1.Restore{completedRestore})
	g.Expect(done).Should(BeTrue())

	// Test pitrTasksAreDone with running restore
	done = pitrTasksAreDone([]*v1alpha1.Restore{runningRestore})
	g.Expect(done).Should(BeFalse())

	// Test pitrTaskIsDone with recently failed restore
	recentFailedRestore := &v1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "recent-failed-restore",
			Namespace: "ns",
		},
		Status: v1alpha1.RestoreStatus{
			Conditions: []v1alpha1.RestoreCondition{
				{
					Type:               v1alpha1.RestoreFailed,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
				},
			},
		},
	}
	done = pitrTaskIsDone(recentFailedRestore)
	g.Expect(done).Should(BeFalse())
}

// createTiKVStatefulSetAndConfigMap creates TiKV StatefulSet and ConfigMap for PiTR testing
func (h *helper) createTiKVStatefulSetAndConfigMap(namespace, clusterName string) {
	h.T.Helper()
	g := NewGomegaWithT(h.T)
	deps := h.Deps

	// Create TiKV StatefulSet
	tikvSts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-tikv", clusterName),
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "tidb-cluster",
				"app.kubernetes.io/managed-by": "tidb-operator",
				"app.kubernetes.io/instance":   clusterName,
				"app.kubernetes.io/component":  "tikv",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-tikv-00000", clusterName),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := deps.KubeClientset.AppsV1().StatefulSets(namespace).Create(context.TODO(), tikvSts, metav1.CreateOptions{})
	g.Expect(err).Should(BeNil())

	// Wait for StatefulSet to be available in lister
	g.Eventually(func() error {
		_, err := deps.StatefulSetLister.StatefulSets(namespace).Get(fmt.Sprintf("%s-tikv", clusterName))
		return err
	}, time.Second*10).Should(BeNil())

	// Create TiKV ConfigMap
	tikvCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-tikv-00000", clusterName),
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "tidb-cluster",
				"app.kubernetes.io/managed-by": "tidb-operator",
				"app.kubernetes.io/instance":   clusterName,
				"app.kubernetes.io/component":  "tikv",
			},
		},
		Data: map[string]string{
			"config-file": `[gc]
ratio-threshold = 1.2

[server]
grpc-keepalive-timeout = "30s"
`,
			"startup-script": "#!/bin/bash\necho 'starting tikv'",
		},
	}
	_, err = deps.KubeClientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), tikvCM, metav1.CreateOptions{})
	g.Expect(err).Should(BeNil())

	// Wait for ConfigMap to be available in lister
	g.Eventually(func() error {
		_, err := deps.ConfigMapLister.ConfigMaps(namespace).Get(fmt.Sprintf("%s-tikv-00000", clusterName))
		return err
	}, time.Second*10).Should(BeNil())
}

// updateConfigMapRatioThreshold updates the gc.ratio-threshold in TiKV ConfigMap
//
// If value is NaN, will set this to empty.
func (h *helper) updateConfigMapRatioThreshold(namespace, clusterName string, value float64) {
	h.T.Helper()
	g := NewGomegaWithT(h.T)
	deps := h.Deps

	gcCfg := ""
	if !math.IsNaN(value) {
		gcCfg = fmt.Sprintf("ratio-threshold = %g", value)
	}
	g.Eventually(func() error {
		tikvCM, err := deps.KubeClientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), fmt.Sprintf("%s-tikv-00000", clusterName), metav1.GetOptions{})
		if err != nil {
			return err
		}
		updatedTiKVCM := tikvCM.DeepCopy()
		updatedTiKVCM.Data["config-file"] = fmt.Sprintf(`[gc]
%s

[server]
grpc-keepalive-timeout = "30s"
`, gcCfg)
		_, err = deps.KubeClientset.CoreV1().ConfigMaps(namespace).Update(context.TODO(), updatedTiKVCM, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		// Verify the config was properly updated in the configmap
		cm, err := deps.ConfigMapLister.ConfigMaps(namespace).Get(fmt.Sprintf("%s-tikv-00000", clusterName))
		if err != nil {
			return err
		}
		if math.IsNaN(value) {
			if strings.Contains(cm.Data["config-file"], "ratio-threshold") {
				return fmt.Errorf("configmap should not contain ratio-threshold when value is NaN")
			}
		} else if !strings.Contains(cm.Data["config-file"], fmt.Sprintf("ratio-threshold = %g", value)) {
			return fmt.Errorf("configmap not updated yet")
		}
		return nil
	}, time.Second*10).Should(BeNil())
}
