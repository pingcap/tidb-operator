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
package ops

import (
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const DefaultPoll = 10 * time.Second
const DefaultTimeout = 10 * time.Minute

type PollFn func(conditionFunc wait.ConditionFunc) error

func PollImmediate(ds ...time.Duration) PollFn {
	poll, timeout := DefaultPoll, DefaultTimeout
	if len(ds) > 0 {
		poll = ds[0]
	}
	if len(ds) > 1 {
		timeout = ds[1]
	}
	return func(f wait.ConditionFunc) error { return wait.PollImmediate(poll, timeout, f) }
}

func Poll(ds ...time.Duration) PollFn {
	poll, timeout := DefaultPoll, DefaultTimeout
	if len(ds) > 0 {
		poll = ds[0]
	}
	if len(ds) > 1 {
		timeout = ds[1]
	}
	return func(f wait.ConditionFunc) error { return wait.Poll(poll, timeout, f) }
}

var defaultPollFn = PollImmediate()

func pollOrDefault(optionalPoll []PollFn) PollFn {
	if len(optionalPoll) > 0 {
		return optionalPoll[0]
	}
	return defaultPollFn
}

type ClientOps struct {
	client.Client
}

func (cli *ClientOps) WaitForPod(ns string, name string, cond func(po *corev1.Pod, err error) (bool, error), optionalPoll ...PollFn) error {
	return pollOrDefault(optionalPoll)(func() (done bool, err error) {
		return cond(cli.CoreV1().Pods(ns).Get(name, metav1.GetOptions{}))
	})
}

func (cli *ClientOps) WaitForStatefulSet(ns string, name string, cond func(ss *appsv1.StatefulSet, err error) (bool, error), optionalPoll ...PollFn) error {
	return pollOrDefault(optionalPoll)(func() (done bool, err error) {
		return cond(cli.AppsV1().StatefulSets(ns).Get(name, metav1.GetOptions{}))
	})
}

func (cli *ClientOps) WaitForTiDBCluster(ns string, name string, cond func(tc *v1alpha1.TidbCluster, err error) (bool, error), optionalPoll ...PollFn) error {
	return pollOrDefault(optionalPoll)(func() (done bool, err error) {
		return cond(cli.PingcapV1alpha1().TidbClusters(ns).Get(name, metav1.GetOptions{}))
	})
}
