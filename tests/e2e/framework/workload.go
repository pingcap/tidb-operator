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
	"net/http"
	"strconv"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/tests/e2e/framework/workload"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

const (
	workloadJobName = "testing-workload-job"
)

type Workload struct {
	f *Framework

	jobs []*batchv1.Job
}

func (f *Framework) SetupWorkload() *Workload {
	w := &Workload{
		f: f,
	}
	w.DeferPrintLogs()

	return w
}

func (w *Workload) MustPing(ctx context.Context, host string, opts ...workload.Option) {
	o := workload.DefaultOptions()
	for _, opt := range opts {
		opt.With(o)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: workloadJobName,
			Namespace:    w.f.Namespace.Name,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "ping",
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
								"--port", strconv.Itoa(o.Port),
								"--user", o.User,
								"--password", o.Password,
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

	job = workload.ConfigJobWithTLS(job, o)

	ginkgo.By("Creating ping job")
	w.f.Must(w.f.Client.Create(ctx, job))
	w.jobs = append(w.jobs, job)

	w.f.Must(waiter.WaitForJobComplete(ctx, w.f.Client, job, waiter.ShortTaskTimeout))
}

func (w *Workload) MustImportData(ctx context.Context, host string, opts ...workload.Option) {
	o := workload.DefaultOptions()
	for _, opt := range opts {
		opt.With(o)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: workloadJobName,
			Namespace:    w.f.Namespace.Name,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "import",
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
								"--port", strconv.Itoa(o.Port),
								"--user", o.User,
								"--password", o.Password,
								"--max-connections", "30",
								"--split-region-count", fmt.Sprintf("%d", o.RegionCount),
								"--tiflash-replicas", strconv.Itoa(o.TiFlashReplicas),
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			BackoffLimit: ptr.To[int32](0),
		},
	}

	job = workload.ConfigJobWithTLS(job, o)

	ginkgo.By("Creating import job")
	w.f.Must(w.f.Client.Create(ctx, job))
	w.jobs = append(w.jobs, job)

	w.f.Must(waiter.WaitForJobComplete(ctx, w.f.Client, job, waiter.ShortTaskTimeout))
}

func (w *Workload) MustRunWorkload(ctx context.Context, host string, opts ...workload.Option) (done chan struct{}) {
	o := workload.DefaultOptions()
	for _, opt := range opts {
		opt.With(o)
	}

	args := []string{
		"--action", "workload",
		"--host", host,
		"--port", strconv.Itoa(o.Port),
		"--user", o.User,
		"--password", o.Password,
		// an arbitrary timeout
		// NOTE: maybe changed to use a http api to stop
		"--max-connections", "30",
		"--max-execution-time", strconv.Itoa(o.MaxExecutionTime),
		"--tiflash-replicas", strconv.Itoa(o.TiFlashReplicas),
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: workloadJobName,
			Namespace:    w.f.Namespace.Name,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "workload",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "testing-workload",
							Image:           "pingcap/testing-workload:latest",
							Args:            args,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			BackoffLimit: ptr.To[int32](0),
		},
	}

	job = workload.ConfigJobWithTLS(job, o)

	ginkgo.By("Creating workload job")
	w.f.Must(w.f.Client.Create(ctx, job))
	w.f.Must(waiter.WaitForJobRunning(ctx, w.f.Client, job, waiter.ShortTaskTimeout))

	w.jobs = append(w.jobs, job)

	nctx := w.StopJob(ctx, job)

	done = make(chan struct{})
	go func() {
		defer close(done)
		defer ginkgo.GinkgoRecover()
		w.f.Must(waiter.WaitForJobComplete(nctx, w.f.Client, job, waiter.LongTaskTimeout))
	}()

	return done
}

func (w *Workload) MustRunPDRegionAccess(ctx context.Context, pdEndpoints string, opts ...workload.Option) (done chan struct{}) {
	o := workload.DefaultOptions()
	for _, opt := range opts {
		opt.With(o)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: workloadJobName,
			Namespace:    w.f.Namespace.Name,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "pd-region",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "testing-workload",
							Image: "pingcap/testing-workload:latest",
							Args: []string{
								"--action", "pd-region",
								"--pd-endpoints", pdEndpoints,
								// an arbitrary timeout
								// NOTE: maybe changed to use a http api to stop
								"--max-connections", "30",
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			BackoffLimit: ptr.To[int32](0),
		},
	}

	ginkgo.By("Creating PD region access job")
	w.f.Must(w.f.Client.Create(ctx, job))
	w.f.Must(waiter.WaitForJobRunning(ctx, w.f.Client, job, waiter.ShortTaskTimeout))

	w.jobs = append(w.jobs, job)

	nctx := w.StopJob(ctx, job)

	done = make(chan struct{})
	go func() {
		defer close(done)
		defer ginkgo.GinkgoRecover()
		w.f.Must(waiter.WaitForJobComplete(nctx, w.f.Client, job, waiter.LongTaskTimeout))
	}()

	return done
}

func (w *Workload) DeferPrintLogs() {
	ginkgo.JustAfterEach(func(ctx context.Context) {
		if ginkgo.CurrentSpecReport().Failed() {
			for _, job := range w.jobs {
				podList := corev1.PodList{}
				ginkgo.By("Try to get the workload pod: " + job.Name)

				s, err := metav1.LabelSelectorAsSelector(job.Spec.Selector)
				w.f.Must(err)

				w.f.Must(w.f.Client.List(ctx, &podList, client.InNamespace(w.f.Namespace.Name), client.MatchingLabelsSelector{
					Selector: s,
				}))

				gomega.Expect(len(podList.Items)).To(gomega.Equal(1))

				pod := &podList.Items[0]
				logs, err := logPod(ctx, w.f.podClient, pod, false)
				gomega.Expect(err).To(gomega.Succeed())
				defer logs.Close()

				buf := bytes.Buffer{}
				_, err = io.Copy(&buf, logs)
				gomega.Expect(err).To(gomega.Succeed())

				// TODO(liubo02): add color for logs
				ginkgo.AddReportEntry("WorkloadLogs:"+job.Name, buf.String())

			}
		}
	})
}

func (w *Workload) stopJob(ctx context.Context, job *batchv1.Job) error {
	<-ctx.Done()

	ginkgo.By("Stop workload job")
	nctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	podList := corev1.PodList{}

	s, err := metav1.LabelSelectorAsSelector(job.Spec.Selector)
	if err != nil {
		return err
	}

	if err := w.f.Client.List(nctx, &podList, client.InNamespace(w.f.Namespace.Name), client.MatchingLabelsSelector{
		Selector: s,
	}); err != nil {
		return err
	}

	if len(podList.Items) != 1 {
		return fmt.Errorf("not only one pod of the job")
	}

	pod := &podList.Items[0]

	ports := w.f.PortForwardPod(nctx, pod, []string{":8080"})
	port := ports[0].Local

	req, err := http.NewRequestWithContext(nctx, http.MethodPost, fmt.Sprintf("http://localhost:%d/cancel", port), nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("not ok")
	}

	ginkgo.By("Stop job successfully")
	return nil
}

// StopJob stops the job if ctx is cancelled
// If stop failed, the returned ctx will be cancelled
func (w *Workload) StopJob(ctx context.Context, job *batchv1.Job) context.Context {
	nctx, cancel := context.WithCancel(context.Background())
	jc := job.DeepCopy()
	go func() {
		defer ginkgo.GinkgoRecover()
		defer cancel()
		if err := w.stopJob(ctx, jc); err == nil {
			// wait until the job is done
			time.Sleep(time.Second * 30)
		} else {
			ginkgo.Fail("cannot stop job " + err.Error())
		}
	}()

	return nctx
}
