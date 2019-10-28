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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	DefaultPollInterval = 10 * time.Second
	DefaultPollTimeout  = 10 * time.Minute
)

type PollFn func(time.Duration, time.Duration, wait.ConditionFunc) error

type ClientOps struct {
	client.Client

	PollInterval *time.Duration
	PollTimeout  *time.Duration
}

func (cli *ClientOps) pollArgs(cond wait.ConditionFunc) (time.Duration, time.Duration, wait.ConditionFunc) {
	interval := DefaultPollInterval
	if cli.PollInterval != nil {
		interval = *cli.PollInterval
	}
	timeout := DefaultPollTimeout
	if cli.PollTimeout != nil {
		timeout = *cli.PollTimeout
	}
	return interval, timeout, cond
}

func (cli *ClientOps) SetPoll(interval time.Duration, timeout time.Duration) {
	cli.PollInterval = &interval
	cli.PollTimeout = &timeout
}

func (cli *ClientOps) Poll(cond wait.ConditionFunc) error {
	return wait.Poll(cli.pollArgs(cond))
}

func (cli *ClientOps) PollImmediate(cond wait.ConditionFunc) error {
	return wait.PollImmediate(cli.pollArgs(cond))
}

func (cli *ClientOps) PollPod(ns string, name string, cond func(po *corev1.Pod, err error) (bool, error)) error {
	return cli.Poll(func() (done bool, err error) {
		return cond(cli.CoreV1().Pods(ns).Get(name, metav1.GetOptions{}))
	})
}

func (cli *ClientOps) PollStatefulSet(ns string, name string, cond func(ss *appsv1.StatefulSet, err error) (bool, error)) error {
	return cli.Poll(func() (done bool, err error) {
		return cond(cli.AppsV1().StatefulSets(ns).Get(name, metav1.GetOptions{}))
	})
}

func (cli *ClientOps) PollTiDBCluster(ns string, name string, cond func(tc *v1alpha1.TidbCluster, err error) (bool, error)) error {
	return cli.Poll(func() (done bool, err error) {
		return cond(cli.PingcapV1alpha1().TidbClusters(ns).Get(name, metav1.GetOptions{}))
	})
}
