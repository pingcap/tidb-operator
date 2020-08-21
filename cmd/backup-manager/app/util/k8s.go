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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	eventv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

// NewEventRecorder return the specify source's recoder
func NewEventRecorder(kubeCli kubernetes.Interface, source string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: source})
	return recorder
}

// NewCRCli create a CR cli Interface
func NewCRCli(kubeconfig string) (versioned.Interface, error) {
	cfg, err := newConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return cli, nil
}

// NewKubeCli create a kubeCli Interface
func NewKubeCli(kubeconfig string) (kubernetes.Interface, error) {
	cfg, err := newConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return kubeCli, nil
}

// NewKubeAndCRCli create both kube cli and CR cli
func NewKubeAndCRCli(kubeconfig string) (kubernetes.Interface, versioned.Interface, error) {
	crCli, err := NewCRCli(kubeconfig)
	if err != nil {
		return nil, nil, err
	}
	kubeCli, err := NewKubeCli(kubeconfig)
	if err != nil {
		return nil, nil, err
	}
	return kubeCli, crCli, nil
}

func newConfig(kubeconfig string) (cfg *rest.Config, err error) {
	if kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
