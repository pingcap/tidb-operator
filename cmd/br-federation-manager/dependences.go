// Copyright 2023 PingCAP, Inc.
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

package main

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	eventv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/federation/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/federation/informers/externalversions"
	listers "github.com/pingcap/tidb-operator/pkg/client/federation/listers/pingcap/v1alpha1"
)

type Controls struct {
	// TODO(csuzhangxc): backup, backup-scheduler, restore
}

// Dependencies is used to store all shared dependent resources to avoid
// pass parameters everywhere.
type Dependencies struct {
	// CLIConfig represents all parameters read from command line
	CLIConfig *CLIConfig
	// Operator client interface
	Clientset versioned.Interface
	// Kubernetes client interface
	KubeClientset                  kubernetes.Interface
	GenericClient                  client.Client
	InformerFactory                informers.SharedInformerFactory
	KubeInformerFactory            kubeinformers.SharedInformerFactory
	LabelFilterKubeInformerFactory kubeinformers.SharedInformerFactory
	Recorder                       record.EventRecorder

	// Listers
	VolumeBackupLister         listers.VolumeBackupLister
	VolumeRestoreLister        listers.VolumeRestoreLister
	VolumeBackupScheduleLister listers.VolumeBackupScheduleLister

	// Controls
	Controls
}

// NewDependencies is used to construct the dependencies
func NewDependencies(cliCfg *CLIConfig, clientset versioned.Interface, kubeClientset kubernetes.Interface, genericCli client.Client) *Dependencies {
	tweakListOptionsFunc := func(options *metav1.ListOptions) {
		if len(options.LabelSelector) > 0 {
			options.LabelSelector += ",app.kubernetes.io/managed-by=tidb-operator"
		} else {
			options.LabelSelector = "app.kubernetes.io/managed-by=tidb-operator"
		}
	}
	tweakListOptions := kubeinformers.WithTweakListOptions(tweakListOptionsFunc)

	// Initialize the informer factories
	informerFactory := informers.NewSharedInformerFactoryWithOptions(clientset, cliCfg.ResyncDuration)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(kubeClientset, cliCfg.ResyncDuration)
	labelFilterKubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(kubeClientset, cliCfg.ResyncDuration, tweakListOptions)

	// Initialize the event recorder
	eventBroadcaster := record.NewBroadcasterWithCorrelatorOptions(record.CorrelatorOptions{QPS: 1})
	eventBroadcaster.StartLogging(klog.V(2).Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeClientset.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "br-federation-manager"})

	deps := newDependencies(cliCfg, clientset, kubeClientset, genericCli, informerFactory, kubeInformerFactory, labelFilterKubeInformerFactory, recorder)
	deps.Controls = newRealControls(cliCfg, clientset, kubeClientset, genericCli, informerFactory, kubeInformerFactory, recorder)
	return deps
}

func newDependencies(
	cliCfg *CLIConfig,
	clientset versioned.Interface,
	kubeClientset kubernetes.Interface,
	genericCli client.Client,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	labelFilterKubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder) *Dependencies {
	return &Dependencies{
		CLIConfig:                      cliCfg,
		Clientset:                      clientset,
		KubeClientset:                  kubeClientset,
		GenericClient:                  genericCli,
		InformerFactory:                informerFactory,
		KubeInformerFactory:            kubeInformerFactory,
		LabelFilterKubeInformerFactory: labelFilterKubeInformerFactory,
		Recorder:                       recorder,

		// Listers
		VolumeBackupLister:         informerFactory.Federation().V1alpha1().VolumeBackups().Lister(),
		VolumeRestoreLister:        informerFactory.Federation().V1alpha1().VolumeRestores().Lister(),
		VolumeBackupScheduleLister: informerFactory.Federation().V1alpha1().VolumeBackupSchedules().Lister(),
	}
}

func newRealControls(
	cliCfg *CLIConfig,
	clientset versioned.Interface,
	kubeClientset kubernetes.Interface,
	genericCli client.Client,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder) Controls {
	return Controls{}
}
