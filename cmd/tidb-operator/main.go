// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlcli "sigs.k8s.io/controller-runtime/pkg/client"
	runtimeConfig "sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	brv1alpha1 "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/backup"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/restore"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/tibr"
	"github.com/pingcap/tidb-operator/pkg/controllers/cluster"
	"github.com/pingcap/tidb-operator/pkg/controllers/pd"
	"github.com/pingcap/tidb-operator/pkg/controllers/pdgroup"
	"github.com/pingcap/tidb-operator/pkg/controllers/scheduler"
	"github.com/pingcap/tidb-operator/pkg/controllers/schedulergroup"
	"github.com/pingcap/tidb-operator/pkg/controllers/ticdc"
	"github.com/pingcap/tidb-operator/pkg/controllers/ticdcgroup"
	"github.com/pingcap/tidb-operator/pkg/controllers/tidb"
	"github.com/pingcap/tidb-operator/pkg/controllers/tidbgroup"
	"github.com/pingcap/tidb-operator/pkg/controllers/tiflash"
	"github.com/pingcap/tidb-operator/pkg/controllers/tiflashgroup"
	"github.com/pingcap/tidb-operator/pkg/controllers/tikv"
	"github.com/pingcap/tidb-operator/pkg/controllers/tikvgroup"
	"github.com/pingcap/tidb-operator/pkg/controllers/tiproxy"
	"github.com/pingcap/tidb-operator/pkg/controllers/tiproxygroup"
	"github.com/pingcap/tidb-operator/pkg/controllers/tso"
	"github.com/pingcap/tidb-operator/pkg/controllers/tsogroup"
	"github.com/pingcap/tidb-operator/pkg/metrics"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	tsom "github.com/pingcap/tidb-operator/pkg/timanager/tso"
	"github.com/pingcap/tidb-operator/pkg/utils/informertest"
	"github.com/pingcap/tidb-operator/pkg/utils/kubefeat"
	"github.com/pingcap/tidb-operator/pkg/version"
	"github.com/pingcap/tidb-operator/pkg/volumes"
)

var setupLog = ctrl.Log.WithName("setup").WithValues(version.Get().KeysAndValues()...)

type brConfig struct {
	backupManagerImage string
}

var brConf = &brConfig{}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var maxConcurrentReconciles int
	var watchDelayDuration time.Duration
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&brConf.backupManagerImage, "backup-manager-image", "", "The image of backup-manager.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	//nolint:mnd // easy to understand
	flag.IntVar(&maxConcurrentReconciles, "max-concurrent-reconciles", 4, "Max concurrent reconciles")
	flag.DurationVar(&watchDelayDuration, "watch-delay-duration", time.Duration(0),
		"A duration for e2e test, it's useful to avoid bugs because of "+
			"kube clients's read-after-write inconsistency(always read from cache).")

	opts := zap.Options{
		Development:     false,
		StacktraceLevel: zapcore.PanicLevel, // stacktrace on panic only
		// use console encoder now for development, switch to json if needed later
		Encoder: zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			TimeKey:        "T",
			LevelKey:       "L",
			NameKey:        "N",
			CallerKey:      "C",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "M",
			StacktraceKey:  "S",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}),
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	utilruntime.PanicHandlers = append(utilruntime.PanicHandlers, func(_ context.Context, _ any) {
		metrics.ControllerPanic.WithLabelValues().Inc()
	})

	kubeconfig := ctrl.GetConfigOrDie()
	// Set user agent to tidb-operator to make sure all field managers are tidb-operator
	// if it's not set explicitly, it may be binary name
	kubeconfig.UserAgent = client.DefaultFieldManager
	kubefeat.MustInitFeatureGates(kubeconfig)

	cacheOpt := cache.Options{
		// Disable label selector for our own CRs
		// These CRs don't need to be filtered
		ByObject: BuildCacheByObject(),
		DefaultLabelSelector: labels.SelectorFromSet(labels.Set{
			v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		}),
	}

	if watchDelayDuration != 0 {
		cacheOpt.NewInformer = informertest.NewDelayedInformerFunc(watchDelayDuration)
	}

	mgr, err := ctrl.NewManager(kubeconfig, ctrl.Options{
		Scheme:                 scheme.Scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		Controller: runtimeConfig.Controller{
			MaxConcurrentReconciles: maxConcurrentReconciles,
		},
		Cache: cacheOpt,
		NewClient: func(cfg *rest.Config, opts ctrlcli.Options) (ctrlcli.Client, error) {
			return client.New(cfg, opts)
		},
		LeaderElection:   enableLeaderElection,
		LeaderElectionID: "c6a50700.pingcap.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to new manager")
		os.Exit(1)
	}

	if err := setup(context.Background(), mgr); err != nil {
		setupLog.Error(err, "failed to setup")
		os.Exit(1)
	}
}

func setup(ctx context.Context, mgr ctrl.Manager) error {
	// client of manager should be newed by options.NewClient
	// which actually returns tidb-operator/pkg/client.Client
	c := mgr.GetClient().(client.Client)

	if err := addIndexer(ctx, mgr); err != nil {
		return fmt.Errorf("unable to add indexer: %w", err)
	}

	logger := mgr.GetLogger()

	logger.Info("setup client manager")
	pdcm := pdm.NewPDClientManager(mgr.GetLogger(), c)
	tsocm := tsom.NewTSOClientManager(mgr.GetLogger(), c)

	logger.Info("setup volume modifier")
	vm := volumes.NewModifierFactory(mgr.GetLogger().WithName("VolumeModifier"), c)

	setupLog.Info("setup controllers")
	if err := setupControllers(mgr, c, pdcm, tsocm, vm); err != nil {
		setupLog.Error(err, "unable to setup controllers")
		os.Exit(1)
	}
	if err := setupBRControllers(mgr, c, pdcm); err != nil {
		setupLog.Error(err, "unable to setup BR controllers")
		os.Exit(1)
	}

	logger.Info("start client manager")
	pdcm.Start(ctx)
	tsocm.Start(ctx)

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}

	logger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("start manager failed: %w", err)
	}

	return nil
}

func addIndexer(ctx context.Context, mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.PDGroup{}, "spec.cluster.name",
		func(obj client.Object) []string {
			pdGroup := obj.(*v1alpha1.PDGroup)
			return []string{pdGroup.Spec.Cluster.Name}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.TiKVGroup{}, "spec.cluster.name",
		func(obj client.Object) []string {
			tikvGroup := obj.(*v1alpha1.TiKVGroup)
			return []string{tikvGroup.Spec.Cluster.Name}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.TiDBGroup{}, "spec.cluster.name",
		func(obj client.Object) []string {
			tidbGroup := obj.(*v1alpha1.TiDBGroup)
			return []string{tidbGroup.Spec.Cluster.Name}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.TiFlashGroup{}, "spec.cluster.name",
		func(obj client.Object) []string {
			tiflashGroup := obj.(*v1alpha1.TiFlashGroup)
			return []string{tiflashGroup.Spec.Cluster.Name}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.TiCDCGroup{}, "spec.cluster.name",
		func(obj client.Object) []string {
			ticdcGroup := obj.(*v1alpha1.TiCDCGroup)
			return []string{ticdcGroup.Spec.Cluster.Name}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.TSOGroup{}, "spec.cluster.name",
		func(obj client.Object) []string {
			tg := obj.(*v1alpha1.TSOGroup)
			return []string{tg.Spec.Cluster.Name}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.SchedulerGroup{}, "spec.cluster.name",
		func(obj client.Object) []string {
			sg := obj.(*v1alpha1.SchedulerGroup)
			return []string{sg.Spec.Cluster.Name}
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.TiProxyGroup{}, "spec.cluster.name",
		func(obj client.Object) []string {
			pg := obj.(*v1alpha1.TiProxyGroup)
			return []string{pg.Spec.Cluster.Name}
		}); err != nil {
		return err
	}

	return nil
}

type controllerSetup struct {
	name      string
	setupFunc func() error
}

func setupControllers(
	mgr ctrl.Manager,
	c client.Client,
	pdcm pdm.PDClientManager,
	tsocm tsom.TSOClientManager,
	vm volumes.ModifierFactory,
) error {
	setups := []controllerSetup{
		{
			name: "Cluster",
			setupFunc: func() error {
				return cluster.Setup(mgr, c, pdcm)
			},
		},
		{
			name: "PDGroup",
			setupFunc: func() error {
				return pdgroup.Setup(mgr, c, pdcm)
			},
		},
		{
			name: "PD",
			setupFunc: func() error {
				return pd.Setup(mgr, c, pdcm, vm)
			},
		},
		{
			name: "TiDBGroup",
			setupFunc: func() error {
				return tidbgroup.Setup(mgr, c)
			},
		},
		{
			name: "TiDB",
			setupFunc: func() error {
				return tidb.Setup(mgr, c, pdcm, vm)
			},
		},
		{
			name: "TiKVGroup",
			setupFunc: func() error {
				return tikvgroup.Setup(mgr, c)
			},
		},
		{
			name: "TiKV",
			setupFunc: func() error {
				return tikv.Setup(mgr, c, pdcm, vm)
			},
		},
		{
			name: "TiFlashGroup",
			setupFunc: func() error {
				return tiflashgroup.Setup(mgr, c)
			},
		},
		{
			name: "TiFlash",
			setupFunc: func() error {
				return tiflash.Setup(mgr, c, pdcm, vm)
			},
		},
		{
			name: "TiCDCGroup",
			setupFunc: func() error {
				return ticdcgroup.Setup(mgr, c)
			},
		},
		{
			name: "TiCDC",
			setupFunc: func() error {
				return ticdc.Setup(mgr, c, vm)
			},
		},
		{
			name: "TSOGroup",
			setupFunc: func() error {
				return tsogroup.Setup(mgr, c, tsocm)
			},
		},
		{
			name: "TSO",
			setupFunc: func() error {
				return tso.Setup(mgr, c, pdcm, tsocm, vm)
			},
		},
		{
			name: "SchedulerGroup",
			setupFunc: func() error {
				return schedulergroup.Setup(mgr, c)
			},
		},
		{
			name: "Scheduler",
			setupFunc: func() error {
				return scheduler.Setup(mgr, c, pdcm, vm)
			},
		},
		{
			name: "TiProxyGroup",
			setupFunc: func() error {
				return tiproxygroup.Setup(mgr, c)
			},
		},
		{
			name: "TiProxy",
			setupFunc: func() error {
				return tiproxy.Setup(mgr, c, pdcm, vm)
			},
		},
		{
			name: "TiBR",
			setupFunc: func() error {
				return tibr.Setup(mgr, c)
			},
		},
	}

	for _, s := range setups {
		if err := s.setupFunc(); err != nil {
			return fmt.Errorf("unable to create controller %s: %w", s.name, err)
		}
	}
	return nil
}

func setupBRControllers(mgr ctrl.Manager, c client.Client, pdcm pdm.PDClientManager) error {
	if err := backup.Setup(mgr, c, pdcm, backup.Config{
		BackupManagerImage: brConf.backupManagerImage,
	}); err != nil {
		return fmt.Errorf("unable to create controller Backup: %w", err)
	}
	if err := restore.Setup(mgr, c, pdcm, restore.Config{
		BackupManagerImage: brConf.backupManagerImage,
	}); err != nil {
		return fmt.Errorf("unable to create controller Restore: %w", err)
	}
	return nil
}

func BuildCacheByObject() map[client.Object]cache.ByObject {
	managedByOperator := labels.SelectorFromSet(labels.Set{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
	})

	byObj := map[client.Object]cache.ByObject{
		&v1alpha1.Cluster{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.PDGroup{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.PD{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.TiKVGroup{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.TiKV{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.TiDBGroup{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.TiDB{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.TiFlashGroup{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.TiFlash{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.TiProxyGroup{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.TiProxy{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.TiCDCGroup{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.TiCDC{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.TSOGroup{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.TSO{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.SchedulerGroup{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.Scheduler{}: {
			Label: labels.Everything(),
		},
		&corev1.Secret{}: {
			// TLS secrets managed by cert-manager or user
			Label: labels.Everything(),
		},
		&corev1.Node{}: {
			// need to sync some labels from nodes to TiKV, TiDB, ...
			Label: labels.Everything(),
		},
		&corev1.PersistentVolume{}: {
			Label: labels.Everything(),
		},
		&storagev1.StorageClass{}: {
			Label: labels.Everything(),
		},

		// BR objects start //
		&brv1alpha1.Backup{}: {
			Label: labels.Everything(),
		},
		&brv1alpha1.Restore{}: {
			Label: labels.Everything(),
		},
		// BR objects end //

		// TiBR objects start
		&brv1alpha1.TiBR{}: {
			Label: labels.Everything(),
		},
		&appsv1.StatefulSet{}: {
			Label: managedByOperator,
		},
		&corev1.ConfigMap{}: {
			Label: managedByOperator,
		},
		&corev1.Service{}: {
			Label: managedByOperator,
		},
		&corev1.PersistentVolumeClaim{}: {
			Label: managedByOperator,
		},
		// TiBR objects end
	}
	if kubefeat.Stage(kubefeat.VolumeAttributesClass).Enabled(kubefeat.BETA) {
		byObj[&storagev1beta1.VolumeAttributesClass{}] = cache.ByObject{
			Label: labels.Everything(),
		}
	}

	return byObj
}
