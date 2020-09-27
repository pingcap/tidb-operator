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
	"flag"
	"time"

	storagelister "k8s.io/client-go/listers/storage/v1"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	informeralphav1 "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	corev1 "k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	kubeinformersv1 "k8s.io/client-go/informers/storage/v1"
	"k8s.io/client-go/kubernetes"
	eventv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CLIConfig is used save all configuration read from command line parameters
type CLIConfig struct {
	PrintVersion bool
	// The number of workers that are allowed to sync concurrently.
	// Larger number = more responsive management, but more CPU
	// (and network) load
	Workers int
	// Controls whether operator should manage kubernetes cluster
	// wide TiDB clusters
	ClusterScoped         bool
	AutoFailover          bool
	PDFailoverPeriod      time.Duration
	TiKVFailoverPeriod    time.Duration
	TiDBFailoverPeriod    time.Duration
	TiFlashFailoverPeriod time.Duration
	MasterFailoverPeriod  time.Duration
	WorkerFailoverPeriod  time.Duration
	LeaseDuration         time.Duration
	RenewDuration         time.Duration
	RetryPeriod           time.Duration
	WaitDuration          time.Duration
	// ResyncDuration is the resync time of informer
	ResyncDuration time.Duration
	// Defines whether tidb operator run in test mode, test mode is
	// only open when test
	TestMode               bool
	TiDBBackupManagerImage string
	TiDBDiscoveryImage     string
	// PodWebhookEnabled is the key to indicate whether pod admission
	// webhook is set up.
	PodWebhookEnabled bool
}

// DefaultCLIConfig returns the default command line configuration
func DefaultCLIConfig() *CLIConfig {
	return &CLIConfig{
		Workers:                5,
		ClusterScoped:          true,
		AutoFailover:           true,
		PDFailoverPeriod:       5 * time.Minute,
		TiKVFailoverPeriod:     5 * time.Minute,
		TiDBFailoverPeriod:     5 * time.Minute,
		TiFlashFailoverPeriod:  5 * time.Minute,
		MasterFailoverPeriod:   5 * time.Minute,
		WorkerFailoverPeriod:   5 * time.Minute,
		LeaseDuration:          15 * time.Second,
		RenewDuration:          5 * time.Second,
		RetryPeriod:            5 * time.Second,
		WaitDuration:           5 * time.Second,
		ResyncDuration:         5 * time.Minute,
		TiDBBackupManagerImage: "pingcap/tidb-backup-manager:latest",
		TiDBDiscoveryImage:     "pingcap/tidb-operator:latest",
	}
}

// AddFlag adds a flag for setting global feature gates to the specified FlagSet.
func (c *CLIConfig) AddFlag(_ *flag.FlagSet) {
	flag.BoolVar(&c.PrintVersion, "V", false, "Show version and quit")
	flag.BoolVar(&c.PrintVersion, "version", false, "Show version and quit")
	flag.IntVar(&c.Workers, "workers", c.Workers, "The number of workers that are allowed to sync concurrently. Larger number = more responsive management, but more CPU (and network) load")
	flag.BoolVar(&c.ClusterScoped, "cluster-scoped", c.ClusterScoped, "Whether tidb-operator should manage kubernetes cluster wide TiDB Clusters")
	flag.BoolVar(&c.AutoFailover, "auto-failover", c.AutoFailover, "Auto failover")
	flag.DurationVar(&c.PDFailoverPeriod, "pd-failover-period", c.PDFailoverPeriod, "PD failover period default(5m)")
	flag.DurationVar(&c.TiKVFailoverPeriod, "tikv-failover-period", c.TiKVFailoverPeriod, "TiKV failover period default(5m)")
	flag.DurationVar(&c.TiFlashFailoverPeriod, "tiflash-failover-period", c.TiFlashFailoverPeriod, "TiFlash failover period default(5m)")
	flag.DurationVar(&c.TiDBFailoverPeriod, "tidb-failover-period", c.TiDBFailoverPeriod, "TiDB failover period")
	flag.DurationVar(&c.MasterFailoverPeriod, "dm-master-failover-period", c.MasterFailoverPeriod, "dm-master failover period")
	flag.DurationVar(&c.WorkerFailoverPeriod, "dm-worker-failover-period", c.WorkerFailoverPeriod, "dm-worker failover period")
	flag.DurationVar(&c.ResyncDuration, "resync-duration", c.ResyncDuration, "Resync time of informer")
	flag.BoolVar(&c.TestMode, "test-mode", false, "whether tidb-operator run in test mode")
	flag.StringVar(&c.TiDBBackupManagerImage, "tidb-backup-manager-image", c.TiDBBackupManagerImage, "The image of backup manager tool")
	// TODO: actually we just want to use the same image with tidb-controller-manager, but DownwardAPI cannot get image ID, see if there is any better solution
	flag.StringVar(&c.TiDBDiscoveryImage, "tidb-discovery-image", c.TiDBDiscoveryImage, "The image of the tidb discovery service")
	flag.BoolVar(&c.PodWebhookEnabled, "pod-webhook-enabled", false, "Whether Pod admission webhook is enabled")
}

// Dependencies is used to store all shared dependent resources to avoid
// pass parameters everywhere.
type Dependencies struct {
	// CLIConfig represents all parameters read from command line
	CLIConfig *CLIConfig
	// Operator client interface
	Clientset versioned.Interface
	// Kubernetes client interface
	KubeClientset       kubernetes.Interface
	GenericClient       client.Client
	InformerFactory     informers.SharedInformerFactory
	KubeInformerFactory kubeinformers.SharedInformerFactory
	Recorder            record.EventRecorder

	// Informers
	PVCInformer          coreinformers.PersistentVolumeClaimInformer
	StatefulSetInformer  appsinformers.StatefulSetInformer
	StorageClassInformer kubeinformersv1.StorageClassInformer
	TiDBClusterInformer  informeralphav1.TidbClusterInformer

	// Listers
	ServiceLister               corelisterv1.ServiceLister
	EndpointLister              corelisterv1.EndpointsLister
	PVCLister                   corelisterv1.PersistentVolumeClaimLister
	PVLister                    corelisterv1.PersistentVolumeLister
	PodLister                   corelisterv1.PodLister
	NodeLister                  corelisterv1.NodeLister
	SecretLister                corelisterv1.SecretLister
	StatefulSetLister           appslisters.StatefulSetLister
	StorageClassLister          storagelister.StorageClassLister
	TiDBClusterLister           listers.TidbClusterLister
	TiDBClusterAutoScalerLister listers.TidbClusterAutoScalerLister

	// Controls
	TiDBClusterControl TidbClusterControlInterface
	PDControl          pdapi.PDControlInterface
	CDCControl         TiCDCControlInterface
	TiDBControl        TiDBControlInterface
	ConfigMapControl   ConfigMapControlInterface
	StatefulSetControl StatefulSetControlInterface
	ServiceControl     ServiceControlInterface
	PVCControl         PVCControlInterface
	PVControl          PVControlInterface
	PodControl         PodControlInterface
	TypedControl       TypedControlInterface
}

// NewDependencies is used to construct the dependencies
func NewDependencies(ns string, cliCfg *CLIConfig, clientset versioned.Interface, kubeClientset kubernetes.Interface, genericCli client.Client) *Dependencies {
	var (
		options     []informers.SharedInformerOption
		kubeoptions []kubeinformers.SharedInformerOption
	)
	if !cliCfg.ClusterScoped {
		options = append(options, informers.WithNamespace(ns))
		kubeoptions = append(kubeoptions, kubeinformers.WithNamespace(ns))
	}

	// Initialize the informaer factories
	informerFactory := informers.NewSharedInformerFactoryWithOptions(clientset, cliCfg.ResyncDuration, options...)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(kubeClientset, cliCfg.ResyncDuration, kubeoptions...)

	// Initialize the event recorder
	eventBroadcaster := record.NewBroadcasterWithCorrelatorOptions(record.CorrelatorOptions{QPS: 1})
	eventBroadcaster.StartLogging(klog.V(2).Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeClientset.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(v1alpha1.Scheme, corev1.EventSource{Component: "tidb-controller-manager"})

	// Shared variables to construct `Dependencies` and some of its fields
	var (
		pdControl         = pdapi.NewDefaultPDControl(kubeClientset)
		tidbClusterLister = informerFactory.Pingcap().V1alpha1().TidbClusters().Lister()
		statefulSetLister = kubeInformerFactory.Apps().V1().StatefulSets().Lister()
		serviceLister     = kubeInformerFactory.Core().V1().Services().Lister()
		pvcLister         = kubeInformerFactory.Core().V1().PersistentVolumeClaims().Lister()
		pvLister          = kubeInformerFactory.Core().V1().PersistentVolumes().Lister()
		podLister         = kubeInformerFactory.Core().V1().Pods().Lister()
	)

	return &Dependencies{
		CLIConfig:           cliCfg,
		InformerFactory:     informerFactory,
		Clientset:           clientset,
		KubeClientset:       kubeClientset,
		GenericClient:       genericCli,
		KubeInformerFactory: kubeInformerFactory,
		Recorder:            recorder,

		// Informers
		PVCInformer:          kubeInformerFactory.Core().V1().PersistentVolumeClaims(),
		StatefulSetInformer:  kubeInformerFactory.Apps().V1().StatefulSets(),
		StorageClassInformer: kubeInformerFactory.Storage().V1().StorageClasses(),
		TiDBClusterInformer:  informerFactory.Pingcap().V1alpha1().TidbClusters(),

		// Listers
		ServiceLister:               serviceLister,
		EndpointLister:              kubeInformerFactory.Core().V1().Endpoints().Lister(),
		PVCLister:                   pvcLister,
		PVLister:                    pvLister,
		PodLister:                   podLister,
		NodeLister:                  kubeInformerFactory.Core().V1().Nodes().Lister(),
		SecretLister:                kubeInformerFactory.Core().V1().Secrets().Lister(),
		StatefulSetLister:           statefulSetLister,
		StorageClassLister:  kubeInformerFactory.Storage().V1().StorageClasses().Lister(),
		TiDBClusterLister:           tidbClusterLister,
		TiDBClusterAutoScalerLister: informerFactory.Pingcap().V1alpha1().TidbClusterAutoScalers().Lister(),

		// Controls
		TiDBClusterControl: NewRealTidbClusterControl(clientset, tidbClusterLister, recorder),
		PDControl:          pdControl,
		CDCControl:         NewDefaultTiCDCControl(kubeClientset),
		TiDBControl:        NewDefaultTiDBControl(kubeClientset),
		ConfigMapControl:   NewRealConfigMapControl(kubeClientset, recorder),
		StatefulSetControl: NewRealStatefuSetControl(kubeClientset, statefulSetLister, recorder),
		ServiceControl:     NewRealServiceControl(kubeClientset, serviceLister, recorder),
		PVControl:          NewRealPVControl(kubeClientset, pvcLister, pvLister, recorder),
		PVCControl:         NewRealPVCControl(kubeClientset, recorder, pvcLister),
		PodControl:         NewRealPodControl(kubeClientset, pdControl, podLister, recorder),
		TypedControl:       NewTypedControl(NewRealGenericControl(genericCli, recorder)),
	}
}
