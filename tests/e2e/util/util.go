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

package util

import (
	"context"
	"time"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	apiutilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	aggregatorclientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
)

// WaitForAPIServicesAvaiable waits for apiservices to be available
func WaitForAPIServicesAvaiable(client aggregatorclientset.Interface, selector labels.Selector) error {
	isAvaiable := func(status apiregistrationv1.APIServiceStatus) bool {
		if status.Conditions == nil {
			return false
		}
		for _, condition := range status.Conditions {
			if condition.Type == apiregistrationv1.Available {
				return condition.Status == apiregistrationv1.ConditionTrue
			}
		}
		return false
	}
	return wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
		apiServiceList, err := client.ApiregistrationV1().APIServices().List(context.TODO(), metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			return false, err
		}
		for _, apiService := range apiServiceList.Items {
			if !isAvaiable(apiService.Status) {
				framework.Logf("APIService %q is not available yet", apiService.Name)
				return false, nil
			}
		}
		for _, apiService := range apiServiceList.Items {
			framework.Logf("APIService %q is available", apiService.Name)
		}
		return true, nil
	})
}

// WaitForCRDsEstablished waits for all CRDs to be established
func WaitForCRDsEstablished(client apiextensionsclientset.Interface, selector labels.Selector) error {
	isEstalbished := func(status apiextensionsv1beta1.CustomResourceDefinitionStatus) bool {
		if status.Conditions == nil {
			return false
		}
		for _, condition := range status.Conditions {
			if condition.Type == apiextensionsv1beta1.Established {
				return condition.Status == apiextensionsv1beta1.ConditionTrue
			}
		}
		return false
	}
	return wait.PollImmediate(5*time.Second, 3*time.Minute, func() (bool, error) {
		crdList, err := client.ApiextensionsV1beta1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			return false, err
		}
		for _, crd := range crdList.Items {
			if !isEstalbished(crd.Status) {
				framework.Logf("CRD %q is not established yet", crd.Name)
				return false, nil
			}
		}
		for _, crd := range crdList.Items {
			framework.Logf("CRD %q is established", crd.Name)
		}
		return true, nil
	})
}

// WaitForCRDNotFound waits for CRD to be not found in apiserver
func WaitForCRDNotFound(client apiextensionsclientset.Interface, name string) error {
	return wait.PollImmediate(time.Second, 1*time.Minute, func() (bool, error) {
		_, err := client.ApiextensionsV1beta1().CustomResourceDefinitions().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			if IsRetryableAPIError(err) {
				return false, nil
			}
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err // fatal errors
		}
		return false, nil
	})
}

// PollImmediateWithReason calls 'wait.PollImmediate' and return last reason
func PollImmediateWithReason(interval, timeout time.Duration, cond func() (bool, error, string /*reason*/)) (error, string) {
	lastReason := ""
	err := wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		done, err, reason := cond()
		lastReason = reason
		return done, err
	})
	return err, lastReason
}

// PollWithReason calls 'wait.Poll' and return last reason
func PollWithReason(interval, timeout time.Duration, cond func() (bool, error, string /*reason*/)) (error, string) {
	lastReason := ""
	err := wait.Poll(interval, timeout, func() (done bool, err error) {
		done, err, reason := cond()
		lastReason = reason
		return done, err
	})
	return err, lastReason
}

func IntPtr(i int) *int {
	return &i
}

func IsRetryableAPIError(err error) bool {
	// These errors may indicate a transient error that we can retry in tests.
	if apierrors.IsInternalError(err) || apierrors.IsTimeout(err) || apierrors.IsServerTimeout(err) ||
		apierrors.IsTooManyRequests(err) || apiutilnet.IsProbableEOF(err) || apiutilnet.IsConnectionReset(err) {
		return true
	}
	// If the error sends the Retry-After header, we respect it as an explicit confirmation we should retry.
	if _, shouldRetry := apierrors.SuggestsClientDelay(err); shouldRetry {
		return true
	}
	return false
}
