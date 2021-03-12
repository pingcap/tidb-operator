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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
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

	_, err = h.Deps.Clientset.PingcapV1alpha1().Restores(restore.Namespace).Create(restore)
	g.Expect(err).Should(BeNil())
	g.Eventually(func() error {
		_, err := h.Deps.RestoreLister.Restores(restore.Namespace).Get(restore.Name)
		return err
	}, time.Second*10).Should(BeNil())
}

func (h *helper) hasCondition(ns string, name string, tp v1alpha1.RestoreConditionType, reasonSub string) {
	h.T.Helper()
	g := NewGomegaWithT(h.T)
	get, err := h.Deps.Clientset.PingcapV1alpha1().Restores(ns).Get(name, metav1.GetOptions{})
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
					// existing env name will be overwritten for backup
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
		r.Name = fmt.Sprintf("backup_name_%d", i)
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
	job, err := helper.Deps.KubeClientset.BatchV1().Jobs(restore.Namespace).Get(restore.GetRestoreJobName(), metav1.GetOptions{})
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
		job, err := helper.Deps.KubeClientset.BatchV1().Jobs(restore.Namespace).Get(restore.GetRestoreJobName(), metav1.GetOptions{})
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
