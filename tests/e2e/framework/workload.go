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

package framework

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

const (
	workloadJobName = "testing-workload-job"
)

type Workload struct {
	f *Framework

	job *batchv1.Job
}

func (f *Framework) SetupWorkload() *Workload {
	w := &Workload{
		f: f,
	}
	w.DeferPrintLogs()

	return w
}

func (w *Workload) MustPing(ctx context.Context, host, user, password, tlsSecretName string) {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workloadJobName,
			Namespace: w.f.Namespace.Name,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": workloadJobName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "testing-workload",
							Image: "pingcap/testing-workload:latest",
							Args: []string{
								"--action", "ping",
								"--host", host,
								"--user", user,
								"--password", password,
								"--duration", "8",
								"--max-connections", "30",
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			BackoffLimit: ptr.To[int32](0),
		},
	}

	if tlsSecretName != "" {
		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "tls-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: tlsSecretName,
				},
			},
		})
		job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "tls-certs",
			MountPath: "/var/lib/tidb-tls",
			ReadOnly:  true,
		})
		job.Spec.Template.Spec.Containers[0].Args = append(job.Spec.Template.Spec.Containers[0].Args,
			"--enable-tls",
			"--tls-mount-path", "/var/lib/tidb-tls",
		)
	}

	ginkgo.By("Creating ping job")
	w.f.Must(w.f.Client.Create(ctx, job))
	w.job = job

	w.f.Must(waiter.WaitForJobComplete(ctx, w.f.Client, job, waiter.ShortTaskTimeout))
}

func (w *Workload) MustImportData(ctx context.Context, host, user, password, tlsSecretName string, regionCount int) {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workloadJobName,
			Namespace: w.f.Namespace.Name,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": workloadJobName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "testing-workload",
							Image: "pingcap/testing-workload:latest",
							Args: []string{
								"--action", "import",
								"--host", host,
								"--user", user,
								"--password", password,
								"--duration", "8",
								"--max-connections", "30",
								"--split-region-count", fmt.Sprintf("%d", regionCount),
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			BackoffLimit: ptr.To[int32](0),
		},
	}

	if tlsSecretName != "" {
		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "tls-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: tlsSecretName,
				},
			},
		})
		job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "tls-certs",
			MountPath: "/var/lib/tidb-tls",
			ReadOnly:  true,
		})
		job.Spec.Template.Spec.Containers[0].Args = append(job.Spec.Template.Spec.Containers[0].Args,
			"--enable-tls",
			"--tls-mount-path", "/var/lib/tidb-tls",
		)
	}

	ginkgo.By("Creating import job")
	w.f.Must(w.f.Client.Create(ctx, job))
	w.job = job

	w.f.Must(waiter.WaitForJobComplete(ctx, w.f.Client, job, waiter.ShortTaskTimeout))
}

func (w *Workload) DeferPrintLogs() {
	ginkgo.JustAfterEach(func(ctx context.Context) {
		if ginkgo.CurrentSpecReport().Failed() && w.job != nil {
			podList := corev1.PodList{}
			ginkgo.By("Try to get the workload pod")
			w.f.Must(w.f.Client.List(ctx, &podList, client.InNamespace(w.f.Namespace.Name), client.MatchingLabels{
				"app": w.job.Name,
			}))
			gomega.Expect(len(podList.Items)).To(gomega.Equal(1))

			pod := &podList.Items[0]
			logs, err := logPod(ctx, w.f.podLogClient, pod, false)
			gomega.Expect(err).To(gomega.Succeed())
			defer logs.Close()

			buf := bytes.Buffer{}
			_, err = io.Copy(&buf, logs)
			gomega.Expect(err).To(gomega.Succeed())

			// TODO(liubo02): add color for logs
			ginkgo.AddReportEntry("WorkloadLogs", buf.String())
		}
	})
}
