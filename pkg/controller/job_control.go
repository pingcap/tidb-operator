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
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/tools/record"
)

// JobControlInterface manages Jobs used in backup、restore and clean
type JobControlInterface interface {
	CreateJob(job *batchv1.Job) error
	DeleteJob(namespace, name string) error
	GetJob(namespace, name string) (*batchv1.Job, error)
}

type realJobControl struct {
	kubeCli   kubernetes.Interface
	jobLister batchlisters.JobLister
	recorder  record.EventRecorder
}

// NewRealJobControl creates a new JobControlInterface
func NewRealJobControl(
	kubeCli kubernetes.Interface,
	jobLister batchlisters.JobLister,
	recorder record.EventRecorder,
) JobControlInterface {
	return &realJobControl{
		kubeCli:   kubeCli,
		jobLister: jobLister,
		recorder:  recorder,
	}
}

func (rjc *realJobControl) CreateJob(job *batchv1.Job) error {
	// TODO: Implement this method
	return nil
}

func (rjc *realJobControl) DeleteJob(namespace, name string) error {
	// TODO: Implement this method
	return nil
}

func (rjc *realJobControl) GetJob(namespace, name string) (*batchv1.Job, error) {
	// TODO: Implement this method
	return nil, nil
}

var _ JobControlInterface = &realJobControl{}
