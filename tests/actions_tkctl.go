package tests

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"text/template"

	"github.com/pingcap/tidb-operator/tests/slack"
	"k8s.io/kubernetes/test/e2e/framework/job"
)

var waitCheckArgsTemplate = `
use {{ .clustername }}
list --namespace={{- .namespace }}
info -t {{ .clustername }}
get all
`

func Int32Ptr(i int32) *int32 { return &i }

func (oa *operatorActions) waitOnceJobComplete(index int, args []string, info *TidbClusterConfig) error {
	name := fmt.Sprintf("tkctl-check%d", index)
	batchjob := batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: info.Namespace,
		},
		Spec: batchv1.JobSpec{
			Parallelism: Int32Ptr(1),
			Completions: Int32Ptr(1),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name:    name,
							Image:   info.TiDBImage,
							Command: []string{"tkctl"},
							Args:    args,
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
	job.CreateJob(oa.kubeCli, info.Namespace, &batchjob)
	err := job.WaitForJobFinish(oa.kubeCli, info.Namespace, name)
	if err != nil {
		return err
	}
	jobstat, err := job.GetJob(oa.kubeCli, info.Namespace, name)
	if err != nil {
		return err
	}
	for _, c := range jobstat.Status.Conditions {
		if c.Status == corev1.ConditionTrue && c.Type == batchv1.JobFailed {
			return fmt.Errorf("%s", c.Reason)
		}
	}
	return nil
}

func RunPlan(reader io.Reader, info *TidbClusterConfig, fn func(index int, args []string, info *TidbClusterConfig) error) error {
	var (
		count     int
		bufreader = bufio.NewReader(reader)
	)
	for {
		line, err := bufreader.ReadString('\n')
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		err = fn(count, strings.Fields(line), info)
		if err != nil {
			return err
		}
		count++
	}
}

func (oa *operatorActions) CheckTkctlOrDie(ctx context.Context, info *TidbClusterConfig) error {
	var (
		tmpmaps = map[string]interface{}{}
		buf     = new(bytes.Buffer)
	)
	tmpmaps["clustername"] = info.ClusterName
	tmpmaps["namespace"] = info.Namespace

	tmp, err := template.New("").Parse(waitCheckArgsTemplate)
	if err != nil {
		return err
	}
	err = tmp.Execute(buf, tmpmaps)
	if err != nil {
		return err
	}

	if err := RunPlan(buf, info, oa.waitOnceJobComplete); err != nil {
		slack.NotifyAndPanic(err)
	}
	return nil
}

