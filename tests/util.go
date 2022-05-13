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

package tests

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	PodPollInterval = 2 * time.Second
	// PodTimeout is how long to wait for the pod to be started or
	// terminated.
	PodTimeout = 5 * time.Minute
)

// GetSecretListerWithCacheSynced return SecretLister with cache synced
func GetSecretListerWithCacheSynced(c kubernetes.Interface, resync time.Duration) corelisterv1.SecretLister {
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(c, resync)
	secretInformer := kubeInformerFactory.Core().V1().Secrets()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go kubeInformerFactory.Start(ctx.Done())

	// waiting for the shared informer's store has synced.
	cache.WaitForCacheSync(ctx.Done(), secretInformer.Informer().HasSynced)

	return secretInformer.Lister()
}

func waitForPodNotFoundInNamespace(c kubernetes.Interface, podName, ns string, timeout time.Duration) error {
	return wait.PollImmediate(PodPollInterval, timeout, func() (bool, error) {
		_, err := c.CoreV1().Pods(ns).Get(context.TODO(), podName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil // done
		}
		if err != nil {
			return true, err // stop wait with error
		}
		return false, nil
	})
}
