// Copyright 2019 PingCAP, Inc.
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

package controller

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/label"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

// JobControlInterface manages Jobs used in backup、restore and clean
type JobControlInterface interface {
	CreateJob(object runtime.Object, job *batchv1.Job) error
	DeleteJob(object runtime.Object, job *batchv1.Job) error
}

type realJobControl struct {
	kubeCli  kubernetes.Interface
	recorder record.EventRecorder
}

// NewRealJobControl creates a new JobControlInterface
func NewRealJobControl(
	kubeCli kubernetes.Interface,
	recorder record.EventRecorder,
) JobControlInterface {
	return &realJobControl{
		kubeCli:  kubeCli,
		recorder: recorder,
	}
}

func (rjc *realJobControl) CreateJob(object runtime.Object, job *batchv1.Job) error {
	ns := job.GetNamespace()
	jobName := job.GetName()
	instanceName := job.GetLabels()[label.InstanceLabelKey]
	kind := object.GetObjectKind().GroupVersionKind().Kind

	_, err := rjc.kubeCli.BatchV1().Jobs(ns).Create(job)
	if err != nil {
		glog.Errorf("failed to create Job: [%s/%s], %s: %s, %v", ns, jobName, kind, instanceName, err)
	} else {
		glog.V(4).Infof("create Job: [%s/%s] successfully, %s: %s", ns, jobName, kind, instanceName)
	}
	rjc.recordJobEvent("create", object, job, err)
	return err
}

func (rjc *realJobControl) DeleteJob(object runtime.Object, job *batchv1.Job) error {
	ns := job.GetNamespace()
	jobName := job.GetName()
	instanceName := job.GetLabels()[label.InstanceLabelKey]
	kind := object.GetObjectKind().GroupVersionKind().Kind

	propForeground := metav1.DeletePropagationForeground
	opts := &metav1.DeleteOptions{
		PropagationPolicy: &propForeground,
	}
	err := rjc.kubeCli.BatchV1().Jobs(ns).Delete(jobName, opts)
	if err != nil {
		glog.Errorf("failed to delete Job: [%s/%s], %s/%s, err: %v", ns, jobName, kind, instanceName, err)
	} else {
		glog.V(4).Infof("delete job: [%s/%s] successfully, %s/%s", ns, jobName, kind, instanceName)
	}
	rjc.recordJobEvent("delete", object, job, err)
	return err
}

func (rjc *realJobControl) recordJobEvent(verb string, obj runtime.Object, job *batchv1.Job, err error) {
	jobName := job.GetName()
	ns := job.GetNamespace()
	instanceName := job.GetLabels()[label.InstanceLabelKey]
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Job %s/%s for %s/%s successful",
			strings.ToLower(verb), ns, jobName, kind, instanceName)
		rjc.recorder.Event(obj, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Job %s/%s for %s/%s failed error: %s",
			strings.ToLower(verb), ns, jobName, kind, instanceName, err)
		rjc.recorder.Event(obj, corev1.EventTypeWarning, reason, msg)
	}
}

var _ JobControlInterface = &realJobControl{}
