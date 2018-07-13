// Copyright 2018 PingCAP, Inc.
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

	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
)

// ServiceControlInterface manages Services used in TidbCluster
type ServiceControlInterface interface {
	CreateService(*v1.TidbCluster, *corev1.Service) error
	UpdateService(*v1.TidbCluster, *corev1.Service) error
	DeleteService(*v1.TidbCluster, *corev1.Service) error
}

type realServiceControl struct {
	kubeCli   kubernetes.Interface
	svcLister corelisters.ServiceLister
	recorder  record.EventRecorder
}

// NewRealServiceControl creates a new ServiceControlInterface
func NewRealServiceControl(kubeCli kubernetes.Interface, svcLister corelisters.ServiceLister, recorder record.EventRecorder) ServiceControlInterface {
	return &realServiceControl{
		kubeCli,
		svcLister,
		recorder,
	}
}

func (sc *realServiceControl) CreateService(tc *v1.TidbCluster, svc *corev1.Service) error {
	_, err := sc.kubeCli.CoreV1().Services(tc.Namespace).Create(svc)
	if apierrors.IsAlreadyExists(err) {
		return err
	}
	sc.recordServiceEvent("create", tc, svc, err)
	return err
}

func (sc *realServiceControl) UpdateService(tc *v1.TidbCluster, svc *corev1.Service) error {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		_, updateErr := sc.kubeCli.CoreV1().Services(tc.Namespace).Update(svc)
		if updateErr == nil {
			return nil
		}
		if updated, err := sc.svcLister.Services(tc.Namespace).Get(svc.Name); err != nil {
			svc = updated.DeepCopy()
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated Secret %s/%s from lister: %v", tc.Namespace, svc.Name, err))
		}
		return updateErr
	})
	sc.recordServiceEvent("update", tc, svc, err)
	return err
}

func (sc *realServiceControl) DeleteService(tc *v1.TidbCluster, svc *corev1.Service) error {
	err := sc.kubeCli.CoreV1().Services(tc.Namespace).Delete(svc.Name, nil)
	sc.recordServiceEvent("delete", tc, svc, err)
	return err
}

func (sc *realServiceControl) recordServiceEvent(verb string, tc *v1.TidbCluster, svc *corev1.Service, err error) {
	tcName := tc.Name
	svcName := svc.Name
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in TidbCluster %s failed error: %s",
			strings.ToLower(verb), svcName, tcName, err)
		sc.recorder.Event(tc, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Service %s in TidbCluster %s successful",
			strings.ToLower(verb), svcName, tcName)
		sc.recorder.Event(tc, corev1.EventTypeWarning, reason, msg)
	}
}

var _ ServiceControlInterface = &realServiceControl{}
