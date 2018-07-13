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

// SecretControlInterface manages Secrets used in TidbCluster
type SecretControlInterface interface {
	CreateSecret(*v1.TidbCluster, *corev1.Secret) error
	UpdateSecret(*v1.TidbCluster, *corev1.Secret) error
	DeleteSecret(*v1.TidbCluster, *corev1.Secret) error
}

// realSecretControl implements SecretControlInterface
type realSecretControl struct {
	kubeCli      kubernetes.Interface
	secretLister corelisters.SecretLister
	recorder     record.EventRecorder
}

// NewRealSecretControl creates a new SecretControlInterface
func NewRealSecretControl(kubeCli kubernetes.Interface, secretLister corelisters.SecretLister, recorder record.EventRecorder) SecretControlInterface {
	return &realSecretControl{
		kubeCli,
		secretLister,
		recorder,
	}
}

func (sc *realSecretControl) CreateSecret(tc *v1.TidbCluster, secret *corev1.Secret) error {
	_, err := sc.kubeCli.CoreV1().Secrets(tc.Namespace).Create(secret)
	if apierrors.IsAlreadyExists(err) {
		return err
	}
	sc.recordSecretEvent("create", tc, secret, err)
	return err
}

func (sc *realSecretControl) UpdateSecret(tc *v1.TidbCluster, secret *corev1.Secret) error {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		_, updateErr := sc.kubeCli.CoreV1().Secrets(tc.Namespace).Update(secret)
		if updateErr == nil {
			return nil
		}
		if updated, err := sc.secretLister.Secrets(tc.Namespace).Get(secret.Name); err != nil {
			secret = updated.DeepCopy()
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated Secret %s/%s from lister: %v", tc.Namespace, secret.Name, err))
		}
		return updateErr
	})
	sc.recordSecretEvent("update", tc, secret, err)
	return err
}

func (sc *realSecretControl) DeleteSecret(tc *v1.TidbCluster, secret *corev1.Secret) error {
	err := sc.kubeCli.CoreV1().Secrets(tc.Namespace).Delete(secret.Name, nil)
	sc.recordSecretEvent("delete", tc, secret, err)
	return err
}

func (sc *realSecretControl) recordSecretEvent(verb string, tc *v1.TidbCluster, secret *corev1.Secret, err error) {
	tcName := tc.Name
	secretName := secret.Name
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Secret %s in TidbCluster %s failed error: %s",
			strings.ToLower(verb), secretName, tcName, err)
		sc.recorder.Event(tc, corev1.EventTypeNormal, reason, msg)

	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s Secret %s in TidbCluster %s successful",
			strings.ToLower(verb), secretName, tcName)
		sc.recorder.Event(tc, corev1.EventTypeWarning, reason, msg)
	}
}

var _ SecretControlInterface = &realSecretControl{}
