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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

// WaitForAllAPIServicesAvaiable waits for all apiservices to be available
func WaitForAllAPIServicesAvaiable(aggrclient aggregatorclient.Interface) error {
	klog.Infof("Wait for all apiesrvices are available")
	isStatusAvaiable := func(status apiregistrationv1.APIServiceStatus) bool {
		for _, condition := range status.Conditions {
			if condition.Type == apiregistrationv1.Available {
				return condition.Status == apiregistrationv1.ConditionTrue
			}
		}
		return false
	}
	return wait.PollImmediate(5*time.Second, 3*time.Minute, func() (bool, error) {
		apiServiceList, err := aggrclient.ApiregistrationV1().APIServices().List(metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		for _, apiService := range apiServiceList.Items {
			if !isStatusAvaiable(apiService.Status) {
				klog.Infof("APIService %q is not available yet", apiService.Name)
				return false, nil
			}
		}
		for _, apiService := range apiServiceList.Items {
			klog.Infof("APIService %q is available", apiService.Name)
		}
		return true, nil
	})
}
