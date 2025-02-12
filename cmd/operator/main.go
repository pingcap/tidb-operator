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

	"go.uber.org/zap/zapcore"
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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/cluster"
	"github.com/pingcap/tidb-operator/pkg/controllers/pd"
	"github.com/pingcap/tidb-operator/pkg/controllers/pdgroup"
	"github.com/pingcap/tidb-operator/pkg/controllers/ticdcgroup"
	"github.com/pingcap/tidb-operator/pkg/controllers/tidb"
	"github.com/pingcap/tidb-operator/pkg/controllers/tidbgroup"
	"github.com/pingcap/tidb-operator/pkg/controllers/tiflash"
	"github.com/pingcap/tidb-operator/pkg/controllers/tiflashgroup"
	"github.com/pingcap/tidb-operator/pkg/controllers/tikv"
	"github.com/pingcap/tidb-operator/pkg/controllers/tikvgroup"
	"github.com/pingcap/tidb-operator/pkg/metrics"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/kubefeat"
	"github.com/pingcap/tidb-operator/pkg/version"
	"github.com/pingcap/tidb-operator/pkg/volumes"
)

var setupLog = ctrl.Log.WithName("setup").WithValues(version.Get().KeysAndValues()...)

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var maxConcurrentReconciles int
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	//nolint:mnd // easy to understand
	flag.IntVar(&maxConcurrentReconciles, "max-concurrent-reconciles", 4, "Max concurrent reconciles")
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
	kubefeat.MustInitFeatureGates(kubeconfig)

	mgr, err := ctrl.NewManager(kubeconfig, ctrl.Options{
		Scheme:                 scheme.Scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		Controller: runtimeConfig.Controller{
			MaxConcurrentReconciles: maxConcurrentReconciles,
		},
		Cache: cache.Options{
			// Disable label selector for our own CRs
			// These CRs don't need to be filtered
			ByObject: BuildCacheByObject(),
			DefaultLabelSelector: labels.SelectorFromSet(labels.Set{
				v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
			}),
		},
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

	logger.Info("setup pd client manager")
	pdcm := pdm.NewPDClientManager(mgr.GetLogger(), c)

	logger.Info("setup volume modifier")
	vm, err := volumes.NewModifier(ctx, mgr.GetLogger().WithName("VolumeModifier"), c)
	if err != nil {
		return fmt.Errorf("failed to create volume modifier: %w", err)
	}

	setupLog.Info("setup controllers")
	if err := setupControllers(mgr, c, pdcm, vm); err != nil {
		setupLog.Error(err, "unable to setup controllers")
		os.Exit(1)
	}

	logger.Info("start pd client manager")
	pdcm.Start(ctx)

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
	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.PDGroup{}, "spec.cluster.name", func(obj client.Object) []string {
		pdGroup := obj.(*v1alpha1.PDGroup)
		return []string{pdGroup.Spec.Cluster.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.TiKVGroup{}, "spec.cluster.name", func(obj client.Object) []string {
		tikvGroup := obj.(*v1alpha1.TiKVGroup)
		return []string{tikvGroup.Spec.Cluster.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.TiDBGroup{}, "spec.cluster.name", func(obj client.Object) []string {
		tidbGroup := obj.(*v1alpha1.TiDBGroup)
		return []string{tidbGroup.Spec.Cluster.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.TiFlashGroup{}, "spec.cluster.name", func(obj client.Object) []string {
		tiflashGroup := obj.(*v1alpha1.TiFlashGroup)
		return []string{tiflashGroup.Spec.Cluster.Name}
	}); err != nil {
		return err
	}

	return nil
}

func setupControllers(mgr ctrl.Manager, c client.Client, pdcm pdm.PDClientManager, vm volumes.Modifier) error {
	if err := cluster.Setup(mgr, c, pdcm); err != nil {
		return fmt.Errorf("unable to create controller Cluster: %w", err)
	}
	if err := pdgroup.Setup(mgr, c, pdcm); err != nil {
		return fmt.Errorf("unable to create controller PDGroup: %w", err)
	}
	if err := pd.Setup(mgr, c, pdcm, vm); err != nil {
		return fmt.Errorf("unable to create controller PD: %w", err)
	}
	if err := tidbgroup.Setup(mgr, c); err != nil {
		return fmt.Errorf("unable to create controller TiDBGroup: %w", err)
	}
	if err := tidb.Setup(mgr, c, pdcm, vm); err != nil {
		return fmt.Errorf("unable to create controller TiDB: %w", err)
	}
	if err := tikvgroup.Setup(mgr, c); err != nil {
		return fmt.Errorf("unable to create controller TiKVGroup: %w", err)
	}
	if err := tikv.Setup(mgr, c, pdcm, vm); err != nil {
		return fmt.Errorf("unable to create controller TiKV: %w", err)
	}
	if err := tiflashgroup.Setup(mgr, c); err != nil {
		return fmt.Errorf("unable to create controller TiFlashGroup: %w", err)
	}
	if err := tiflash.Setup(mgr, c, pdcm, vm); err != nil {
		return fmt.Errorf("unable to create controller TiFlash: %w", err)
	}
	if err := ticdcgroup.Setup(mgr, c); err != nil {
		return fmt.Errorf("unable to create controller TiCDCGroup: %w", err)
	}
	return nil
}

func BuildCacheByObject() map[client.Object]cache.ByObject {
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
	}
	if kubefeat.FeatureGates.Stage(kubefeat.VolumeAttributesClass).Enabled(kubefeat.BETA) {
		byObj[&storagev1beta1.VolumeAttributesClass{}] = cache.ByObject{
			Label: labels.Everything(),
		}
	}

	return byObj
}
