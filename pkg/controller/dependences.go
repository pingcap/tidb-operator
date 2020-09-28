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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/dmapi"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	corev1 "k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
	eventv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	extensionslister "k8s.io/client-go/listers/extensions/v1beta1"
	storagelister "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
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

type Controls struct {
	JobControl         JobControlInterface
	ConfigMapControl   ConfigMapControlInterface
	StatefulSetControl StatefulSetControlInterface
	ServiceControl     ServiceControlInterface
	PVCControl         PVCControlInterface
	GeneralPVCControl  GeneralPVCControlInterface
	PVControl          PVControlInterface
	PodControl         PodControlInterface
	TypedControl       TypedControlInterface
	PDControl          pdapi.PDControlInterface
	DMMasterControl    dmapi.MasterControlInterface
	TiDBClusterControl TidbClusterControlInterface
	DMClusterControl   DMClusterControlInterface
	CDCControl         TiCDCControlInterface
	TiDBControl        TiDBControlInterface
	BackupControl      BackupControlInterface
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

	// Listers
	ServiceLister               corelisterv1.ServiceLister
	EndpointLister              corelisterv1.EndpointsLister
	PVCLister                   corelisterv1.PersistentVolumeClaimLister
	PVLister                    corelisterv1.PersistentVolumeLister
	PodLister                   corelisterv1.PodLister
	NodeLister                  corelisterv1.NodeLister
	SecretLister                corelisterv1.SecretLister
	StatefulSetLister           appslisters.StatefulSetLister
	DeploymentLister            appslisters.DeploymentLister
	JobLister                   batchlisters.JobLister
	IngressLister               extensionslister.IngressLister
	StorageClassLister          storagelister.StorageClassLister
	TiDBClusterLister           listers.TidbClusterLister
	TiDBClusterAutoScalerLister listers.TidbClusterAutoScalerLister
	DMClusterLister             listers.DMClusterLister
	BackupLister                listers.BackupLister
	RestoreLister               listers.RestoreLister
	BackupScheduleLister        listers.BackupScheduleLister
	TiDBInitializerLister       listers.TidbInitializerLister
	TiDBMonitorLister           listers.TidbMonitorLister

	// Controls
	Controls
}

func newRealControls(
	clientset versioned.Interface,
	kubeClientset kubernetes.Interface,
	genericCli client.Client,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder) Controls {
	// Shared variables to construct `Dependencies` and some of its fields
	var (
		pdControl         = pdapi.NewDefaultPDControl(kubeClientset)
		masterControl     = dmapi.NewDefaultMasterControl(kubeClientset)
		tidbClusterLister = informerFactory.Pingcap().V1alpha1().TidbClusters().Lister()
		dmClusterLister   = informerFactory.Pingcap().V1alpha1().DMClusters().Lister()
		statefulSetLister = kubeInformerFactory.Apps().V1().StatefulSets().Lister()
		serviceLister     = kubeInformerFactory.Core().V1().Services().Lister()
		pvcLister         = kubeInformerFactory.Core().V1().PersistentVolumeClaims().Lister()
		pvLister          = kubeInformerFactory.Core().V1().PersistentVolumes().Lister()
		podLister         = kubeInformerFactory.Core().V1().Pods().Lister()
	)
	return Controls{
		JobControl:         NewRealJobControl(kubeClientset, recorder),
		ConfigMapControl:   NewRealConfigMapControl(kubeClientset, recorder),
		StatefulSetControl: NewRealStatefuSetControl(kubeClientset, statefulSetLister, recorder),
		ServiceControl:     NewRealServiceControl(kubeClientset, serviceLister, recorder),
		PVControl:          NewRealPVControl(kubeClientset, pvcLister, pvLister, recorder),
		PVCControl:         NewRealPVCControl(kubeClientset, recorder, pvcLister),
		GeneralPVCControl:  NewRealGeneralPVCControl(kubeClientset, recorder),
		PodControl:         NewRealPodControl(kubeClientset, pdControl, podLister, recorder),
		TypedControl:       NewTypedControl(NewRealGenericControl(genericCli, recorder)),
		PDControl:          pdControl,
		DMMasterControl:    masterControl,
		TiDBClusterControl: NewRealTidbClusterControl(clientset, tidbClusterLister, recorder),
		DMClusterControl:   NewRealDMClusterControl(clientset, dmClusterLister, recorder),
		CDCControl:         NewDefaultTiCDCControl(kubeClientset),
		TiDBControl:        NewDefaultTiDBControl(kubeClientset),
		BackupControl:      NewRealBackupControl(clientset, recorder),
	}
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

	return &Dependencies{
		CLIConfig:           cliCfg,
		InformerFactory:     informerFactory,
		Clientset:           clientset,
		KubeClientset:       kubeClientset,
		GenericClient:       genericCli,
		KubeInformerFactory: kubeInformerFactory,
		Recorder:            recorder,

		// Listers
		ServiceLister:               kubeInformerFactory.Core().V1().Services().Lister(),
		EndpointLister:              kubeInformerFactory.Core().V1().Endpoints().Lister(),
		PVCLister:                   kubeInformerFactory.Core().V1().PersistentVolumeClaims().Lister(),
		PVLister:                    kubeInformerFactory.Core().V1().PersistentVolumes().Lister(),
		PodLister:                   kubeInformerFactory.Core().V1().Pods().Lister(),
		NodeLister:                  kubeInformerFactory.Core().V1().Nodes().Lister(),
		SecretLister:                kubeInformerFactory.Core().V1().Secrets().Lister(),
		StatefulSetLister:           kubeInformerFactory.Apps().V1().StatefulSets().Lister(),
		DeploymentLister:            kubeInformerFactory.Apps().V1().Deployments().Lister(),
		StorageClassLister:          kubeInformerFactory.Storage().V1().StorageClasses().Lister(),
		JobLister:                   kubeInformerFactory.Batch().V1().Jobs().Lister(),
		IngressLister:               kubeInformerFactory.Extensions().V1beta1().Ingresses().Lister(),
		TiDBClusterLister:           informerFactory.Pingcap().V1alpha1().TidbClusters().Lister(),
		TiDBClusterAutoScalerLister: informerFactory.Pingcap().V1alpha1().TidbClusterAutoScalers().Lister(),
		DMClusterLister:             informerFactory.Pingcap().V1alpha1().DMClusters().Lister(),
		BackupLister:                informerFactory.Pingcap().V1alpha1().Backups().Lister(),
		RestoreLister:               informerFactory.Pingcap().V1alpha1().Restores().Lister(),
		BackupScheduleLister:        informerFactory.Pingcap().V1alpha1().BackupSchedules().Lister(),
		TiDBInitializerLister:       informerFactory.Pingcap().V1alpha1().TidbInitializers().Lister(),
		TiDBMonitorLister:           informerFactory.Pingcap().V1alpha1().TidbMonitors().Lister(),

		// Controls
		Controls: newRealControls(clientset, kubeClientset, genericCli, informerFactory, kubeInformerFactory, recorder),
	}
}

func newFakeControl(kubeClientset kubernetes.Interface, informerFactory informers.SharedInformerFactory, kubeInformerFactory kubeinformers.SharedInformerFactory) Controls {
	// Shared variables to construct `Dependencies` and some of its fields
	return Controls{
		ConfigMapControl:   NewFakeConfigMapControl(kubeInformerFactory.Core().V1().ConfigMaps()),
		StatefulSetControl: NewFakeStatefulSetControl(kubeInformerFactory.Apps().V1().StatefulSets()),
		ServiceControl:     NewFakeServiceControl(kubeInformerFactory.Core().V1().Services(), kubeInformerFactory.Core().V1().Endpoints()),
		PVControl:          NewFakePVControl(kubeInformerFactory.Core().V1().PersistentVolumes(), kubeInformerFactory.Core().V1().PersistentVolumeClaims()),
		PVCControl:         NewFakePVCControl(kubeInformerFactory.Core().V1().PersistentVolumeClaims()),
		GeneralPVCControl:  NewFakeGeneralPVCControl(kubeInformerFactory.Core().V1().PersistentVolumeClaims()),
		PodControl:         NewFakePodControl(kubeInformerFactory.Core().V1().Pods()),
		TypedControl:       NewTypedControl(NewFakeGenericControl()),
		PDControl:          pdapi.NewDefaultPDControl(kubeClientset),
		DMMasterControl:    dmapi.NewDefaultMasterControl(kubeClientset),
		TiDBClusterControl: NewFakeTidbClusterControl(informerFactory.Pingcap().V1alpha1().TidbClusters()),
		CDCControl:         NewDefaultTiCDCControl(kubeClientset),
		TiDBControl:        NewDefaultTiDBControl(kubeClientset),
		BackupControl:      NewFakeBackupControl(informerFactory.Pingcap().V1alpha1().Backups()),
	}
}

// NewFakeDependencies returns a fake dependencies for testing
func NewFakeDependencies() *Dependencies {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	genCli := controllerfake.NewFakeClientWithScheme(scheme.Scheme)
	cliCfg := DefaultCLIConfig()
	d := NewDependencies("", cliCfg, cli, kubeCli, genCli)
	informerFactory := informers.NewSharedInformerFactory(cli, 0)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
	d.Controls = newFakeControl(kubeCli, informerFactory, kubeInformerFactory)
	return d
}
