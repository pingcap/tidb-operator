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

	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
	"github.com/pingcap/tidb-operator/v2/manifests"
	"github.com/pingcap/tidb-operator/v2/pkg/adoption"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/backup"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/restore"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/tibr"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/tibrgc"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/cluster"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/pdgroup"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/scheduler"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/schedulergroup"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/scheduling"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/schedulinggroup"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/ticdc"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/ticdcgroup"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tidb"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tidbgroup"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tiflash"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tiflashgroup"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tikv"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tikvgroup"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tikvworker"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tikvworkergroup"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tiproxy"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tiproxygroup"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tso"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/tsogroup"
	"github.com/pingcap/tidb-operator/v2/pkg/crd"
	"github.com/pingcap/tidb-operator/v2/pkg/image"
	"github.com/pingcap/tidb-operator/v2/pkg/metrics"
	"github.com/pingcap/tidb-operator/v2/pkg/scheme"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	fm "github.com/pingcap/tidb-operator/v2/pkg/timanager/tiflash"
	tsom "github.com/pingcap/tidb-operator/v2/pkg/timanager/tso"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/informertest"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/kubefeat"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
	"github.com/pingcap/tidb-operator/v2/pkg/version"
	"github.com/pingcap/tidb-operator/v2/pkg/volumes"
)

var setupLog = ctrl.Log.WithName("setup")

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
	var allowEmptyOldVersion bool

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
	flag.Var(&image.PrestopChecker, "default-prestop-checker-image", "Default image of prestop checker.")
	flag.Var(&image.ResourceSyncer, "default-resource-syncer-image", "Default image of resource syncer.")
	flag.BoolVar(&allowEmptyOldVersion, "allow-empty-old-version", false,
		"allow crd version is empty, if false, cannot update crd which has no version annotation")

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

	ctx := logr.NewContext(context.Background(), setupLog)

	setupLog.Info("start tidb operator", version.Get().KeysAndValues()...)

	utilruntime.PanicHandlers = append(utilruntime.PanicHandlers, func(_ context.Context, _ any) {
		metrics.ControllerPanic.WithLabelValues().Inc()
	})

	kubeconfig := ctrl.GetConfigOrDie()
	// Set user agent to tidb-operator to make sure all field managers are tidb-operator
	// if it's not set explicitly, it may be binary name
	kubeconfig.UserAgent = client.DefaultFieldManager
	kubefeat.MustInitFeatureGates(kubeconfig)

	if err := applyCRDs(ctx, kubeconfig, allowEmptyOldVersion); err != nil {
		setupLog.Error(err, "failed to apply crds")
		os.Exit(1)
	}

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

	if err := setup(ctx, mgr); err != nil {
		setupLog.Error(err, "failed to setup")
		os.Exit(1)
	}
}

func applyCRDs(ctx context.Context, kubeconfig *rest.Config, allowEmptyOldVersion bool) error {
	c, err := client.New(kubeconfig, ctrlcli.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		return fmt.Errorf("cannot new client for applying crd: %w", err)
	}

	info := version.Get()
	cfg := crd.Config{
		Client:               c,
		Version:              info.GitVersion,
		AllowEmptyOldVersion: allowEmptyOldVersion,
		IsDirty:              info.IsDirty(),
		CRDDir:               manifests.CRD,
	}

	if err := crd.ApplyCRDs(ctx, &cfg); err != nil {
		return err
	}

	return nil
}

func setup(ctx context.Context, mgr ctrl.Manager) error {
	// client of manager should be newed by options.NewClient
	// which actually returns tidb-operator/v2/pkg/client.Client
	c := mgr.GetClient().(client.Client)

	if err := addIndexer(ctx, mgr); err != nil {
		return fmt.Errorf("unable to add indexer: %w", err)
	}

	logger := mgr.GetLogger()

	logger.Info("setup client manager")
	pdcm := pdm.NewPDClientManager(mgr.GetLogger(), c)
	tsocm := tsom.NewTSOClientManager(mgr.GetLogger(), c)
	fcm := fm.NewTiFlashClientManager(mgr.GetLogger(), c)

	logger.Info("setup volume modifier")
	vm := volumes.NewModifierFactory(mgr.GetLogger().WithName("VolumeModifier"), c)

	am := adoption.New(ctrl.Log.WithName("adoption"))
	tf := tracker.New()
	setupLog.Info("setup controllers")
	if err := setupControllers(mgr, c, pdcm, tsocm, fcm, vm, tf, am); err != nil {
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

	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.SchedulingGroup{}, "spec.cluster.name",
		func(obj client.Object) []string {
			sg := obj.(*v1alpha1.SchedulingGroup)
			return []string{sg.Spec.Cluster.Name}
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

	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.TiKVWorkerGroup{}, "spec.cluster.name",
		func(obj client.Object) []string {
			wg := obj.(*v1alpha1.TiKVWorkerGroup)
			return []string{wg.Spec.Cluster.Name}
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
	fcm fm.TiFlashClientManager,
	vm volumes.ModifierFactory,
	tf tracker.Factory,
	am adoption.Manager,
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
				return pdgroup.Setup(mgr, c, pdcm, tf.AllocateFactory("pd"))
			},
		},
		{
			name: "PD",
			setupFunc: func() error {
				return pd.Setup(mgr, c, pdcm, vm, tf.Tracker("pd"))
			},
		},
		{
			name: "TiDBGroup",
			setupFunc: func() error {
				return tidbgroup.Setup(mgr, c, tf.AllocateFactory("tidb"), am)
			},
		},
		{
			name: "TiDB",
			setupFunc: func() error {
				return tidb.Setup(mgr, c, pdcm, vm, tf.Tracker("tidb"), am)
			},
		},
		{
			name: "TiKVGroup",
			setupFunc: func() error {
				return tikvgroup.Setup(mgr, c, tf.AllocateFactory("tikv"))
			},
		},
		{
			name: "TiKV",
			setupFunc: func() error {
				return tikv.Setup(mgr, c, pdcm, vm, tf.Tracker("tikv"))
			},
		},
		{
			name: "TiFlashGroup",
			setupFunc: func() error {
				return tiflashgroup.Setup(mgr, c, tf.AllocateFactory("tiflash"))
			},
		},
		{
			name: "TiFlash",
			setupFunc: func() error {
				return tiflash.Setup(mgr, c, pdcm, fcm, vm, tf.Tracker("tiflash"))
			},
		},
		{
			name: "TiCDCGroup",
			setupFunc: func() error {
				return ticdcgroup.Setup(mgr, c, tf.AllocateFactory("ticdc"))
			},
		},
		{
			name: "TiCDC",
			setupFunc: func() error {
				return ticdc.Setup(mgr, c, vm, tf.Tracker("ticdc"))
			},
		},
		{
			name: "TSOGroup",
			setupFunc: func() error {
				return tsogroup.Setup(mgr, c, tsocm, tf.AllocateFactory("tso"))
			},
		},
		{
			name: "TSO",
			setupFunc: func() error {
				return tso.Setup(mgr, c, pdcm, tsocm, vm, tf.Tracker("tso"))
			},
		},
		{
			name: "SchedulingGroup",
			setupFunc: func() error {
				return schedulinggroup.Setup(mgr, c, tf.AllocateFactory("scheduling"))
			},
		},
		{
			name: "Scheduling",
			setupFunc: func() error {
				return scheduling.Setup(mgr, c, pdcm, vm, tf.Tracker("scheduling"))
			},
		},
		{
			name: "SchedulerGroup",
			setupFunc: func() error {
				return schedulergroup.Setup(mgr, c, tf.AllocateFactory("scheduler"))
			},
		},
		{
			name: "Scheduler",
			setupFunc: func() error {
				return scheduler.Setup(mgr, c, pdcm, vm, tf.Tracker("scheduler"))
			},
		},
		{
			name: "TiProxyGroup",
			setupFunc: func() error {
				return tiproxygroup.Setup(mgr, c, tf.AllocateFactory("tiproxy"))
			},
		},
		{
			name: "TiProxy",
			setupFunc: func() error {
				return tiproxy.Setup(mgr, c, pdcm, vm, tf.Tracker("tiproxy"))
			},
		},
		{
			name: "TiKVWorkerGroup",
			setupFunc: func() error {
				return tikvworkergroup.Setup(mgr, c, tf.AllocateFactory("tikvworker"))
			},
		},
		{
			name: "TiKVWorker",
			setupFunc: func() error {
				return tikvworker.Setup(mgr, c, pdcm, vm, tf.Tracker("tikvworker"))
			},
		},
		{
			name: "TiBR",
			setupFunc: func() error {
				return tibr.Setup(mgr, c)
			},
		},
		{
			name: "TiBRGC",
			setupFunc: func() error {
				return tibrgc.Setup(mgr, c)
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
		&v1alpha1.SchedulingGroup{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.Scheduling{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.TiKVWorkerGroup{}: {
			Label: labels.Everything(),
		},
		&v1alpha1.TiKVWorker{}: {
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
		&rbacv1.Role{}: {
			Label: managedByOperator,
		},
		&rbacv1.RoleBinding{}: {
			Label: managedByOperator,
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

		// TiBRGC objects start
		&brv1alpha1.TiBRGC{}: {
			Label: labels.Everything(),
		},
		&batchv1.CronJob{}: {
			Label: managedByOperator,
		},
		// TiBRGC objects end
	}
	if kubefeat.Stage(kubefeat.VolumeAttributesClass).Enabled(kubefeat.BETA) {
		byObj[&storagev1beta1.VolumeAttributesClass{}] = cache.ByObject{
			Label: labels.Everything(),
		}
	}

	return byObj
}
