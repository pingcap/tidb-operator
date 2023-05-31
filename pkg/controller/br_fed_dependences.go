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

package controller

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
	eventv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	fedversioned "github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	fedfake "github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/client/federation/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/federation/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/federation/informers/externalversions"
	listers "github.com/pingcap/tidb-operator/pkg/client/federation/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/scheme"
)

const (
	FakeDataPlaneName1 = "dataplane-1"
	FakeDataPlaneName2 = "dataplane-2"
	FakeDataPlaneName3 = "dataplane-3"
)

type BrFedControls struct {
	FedVolumeBackupControl FedVolumeBackupControlInterface
}

// BrFedDependencies is used to store all shared dependent resources to avoid
// pass parameters everywhere.
type BrFedDependencies struct {
	// CLIConfig represents all parameters read from command line
	CLIConfig *BrFedCLIConfig
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
	BrFedControls

	// FedClientset is the clientset for the federation clusters
	FedClientset map[string]fedversioned.Interface
}

// NewBrFedDependencies is used to construct the dependencies
func NewBrFedDependencies(cliCfg *BrFedCLIConfig, clientset versioned.Interface, kubeClientset kubernetes.Interface,
	genericCli client.Client, fedClientset map[string]fedversioned.Interface) *BrFedDependencies {
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

	deps := newBrFedDependencies(cliCfg, clientset, kubeClientset, genericCli, informerFactory, kubeInformerFactory, labelFilterKubeInformerFactory, recorder, fedClientset)
	deps.BrFedControls = newRealBrFedControls(cliCfg, clientset, kubeClientset, genericCli, informerFactory, kubeInformerFactory, recorder)
	return deps
}

func NewFakeBrFedDependencies() *BrFedDependencies {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	genCli := controllerfake.NewFakeClientWithScheme(scheme.Scheme)
	cliCfg := DefaultBrFedCLIConfig()
	informerFactory := informers.NewSharedInformerFactory(cli, 0)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
	labelFilterKubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
	recorder := record.NewFakeRecorder(100)

	kubeCli.Fake.Resources = append(kubeCli.Fake.Resources, &metav1.APIResourceList{
		GroupVersion: "networking.k8s.io/v1",
		APIResources: []metav1.APIResource{
			{
				Name: "ingresses",
			},
		},
	})
	fedClientset := make(map[string]fedversioned.Interface, 3)
	fedClientset[FakeDataPlaneName1] = fedfake.NewSimpleClientset()
	fedClientset[FakeDataPlaneName2] = fedfake.NewSimpleClientset()
	fedClientset[FakeDataPlaneName3] = fedfake.NewSimpleClientset()

	deps := newBrFedDependencies(cliCfg, cli, kubeCli, genCli, informerFactory, kubeInformerFactory, labelFilterKubeInformerFactory, recorder, fedClientset)
	deps.BrFedControls = newFakeBrFedControls(informerFactory)
	return deps
}

func newBrFedDependencies(
	cliCfg *BrFedCLIConfig,
	clientset versioned.Interface,
	kubeClientset kubernetes.Interface,
	genericCli client.Client,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	labelFilterKubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder,
	fedClientset map[string]fedversioned.Interface) *BrFedDependencies {
	return &BrFedDependencies{
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

		FedClientset: fedClientset,
	}
}

func newRealBrFedControls(
	cliCfg *BrFedCLIConfig,
	clientset versioned.Interface,
	kubeClientset kubernetes.Interface,
	genericCli client.Client,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder) BrFedControls {
	return BrFedControls{
		FedVolumeBackupControl: NewRealFedVolumeBackupControl(clientset, recorder),
	}
}

func newFakeBrFedControls(informerFactory informers.SharedInformerFactory) BrFedControls {
	return BrFedControls{
		FedVolumeBackupControl: NewFakeFedVolumeBackupControl(informerFactory.Federation().V1alpha1().VolumeBackups()),
	}
}
