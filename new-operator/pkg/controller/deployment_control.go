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
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1beta1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
)

// DeploymentControlInterface manages Deployments used in TidbCluster
type DeploymentControlInterface interface {
	CreateDeployment(*v1.TidbCluster, *apps.Deployment) error
	UpdateDeployment(*v1.TidbCluster, *apps.Deployment) error
	DeleteDeployment(*v1.TidbCluster, *apps.Deployment) error
}
type realDeploymentControl struct {
	kubeCli      kubernetes.Interface
	deployLister appslisters.DeploymentLister
	recorder     record.EventRecorder
}

// NewRealDeploymentControl creates a new DeploymentControlInterface
func NewRealDeploymentControl(kubeCli kubernetes.Interface, deployLister appslisters.DeploymentLister, recorder record.EventRecorder) DeploymentControlInterface {
	return &realDeploymentControl{kubeCli, deployLister, recorder}
}

// CreateDeployment create a Deployment in a TidbCluster.
func (dc *realDeploymentControl) CreateDeployment(tc *v1.TidbCluster, deploy *apps.Deployment) error {
	_, err := dc.kubeCli.AppsV1beta1().Deployments(tc.Namespace).Create(deploy)
	// sink already exists errors
	if apierrors.IsAlreadyExists(err) {
		return err
	}
	dc.recordDeploymentEvent("create", tc, deploy, err)
	return err
}

// UpdateDeployment update a Deployment in a TidbCluster.
func (dc *realDeploymentControl) UpdateDeployment(tc *v1.TidbCluster, deploy *apps.Deployment) error {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		_, updateErr := dc.kubeCli.AppsV1beta1().Deployments(tc.Namespace).Update(deploy)
		if updateErr == nil {
			return nil
		}

		if updated, err := dc.deployLister.Deployments(tc.Namespace).Get(deploy.Name); err != nil {
			deploy = updated.DeepCopy()
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated Deployment %s/%s from lister: %v", tc.Namespace, deploy.Name, err))
		}
		return updateErr
	})
	dc.recordDeploymentEvent("update", tc, deploy, err)
	return err
}

// DeleteDeployment delete a Deployment in a TidbCluster.
func (dc *realDeploymentControl) DeleteDeployment(tc *v1.TidbCluster, deploy *apps.Deployment) error {
	err := dc.kubeCli.AppsV1beta1().Deployments(tc.Namespace).Delete(deploy.Name, nil)
	dc.recordDeploymentEvent("delete", tc, deploy, err)
	return err
}

func (dc *realDeploymentControl) recordDeploymentEvent(verb string, tc *v1.TidbCluster, deploy *apps.Deployment, err error) {
	tcName := tc.Name
	deployName := deploy.Name
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		message := fmt.Sprintf("%s Deployment %s in TidbCluster %s successful",
			strings.ToLower(verb), deployName, tcName)
		dc.recorder.Event(tc, corev1.EventTypeNormal, reason, message)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		message := fmt.Sprintf("%s Deployment %s in TidbCluster %s failed error: %s",
			strings.ToLower(verb), deployName, tcName, err)
		dc.recorder.Event(tc, corev1.EventTypeWarning, reason, message)
	}
}

var _ DeploymentControlInterface = &realDeploymentControl{}
