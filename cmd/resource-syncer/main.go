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
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/resourcesyncer/secret"
)

var setupLog = ctrl.Log.WithName("setup")

type Config struct {
	Namespace     string
	Labels        map[string]string
	BaseDir       string
	SecretDirName string

	ProbeAddr string
}

func (c *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Namespace, "namespace", "n", "default", "namespace of syncer")
	fs.StringVarP(&c.BaseDir, "base-dir", "d", "", "base dir of data")
	fs.StringVar(&c.SecretDirName, "secret-dir-name", "secrets", "dir name of secret data")
	fs.StringToStringVarP(&c.Labels, "labels", "l", map[string]string{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
	}, "labels of secrets")

	fs.StringVar(&c.ProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
}

func main() {
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
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	cfg := Config{}
	cfg.AddFlags(pflag.CommandLine)
	pflag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	setupLog.Info("current config", "config", &cfg)

	if err := run(&cfg); err != nil {
		setupLog.Error(err, "unable to run resource syncer")
		os.Exit(1)
	}
}

func run(cfg *Config) error {
	kubeconfig := ctrl.GetConfigOrDie()
	cacheOpt := cache.Options{
		DefaultNamespaces: map[string]cache.Config{
			cfg.Namespace: {
				LabelSelector: labels.SelectorFromSet(labels.Set(cfg.Labels)),
			},
		},
	}
	mgr, err := ctrl.NewManager(kubeconfig, ctrl.Options{
		HealthProbeBindAddress: cfg.ProbeAddr,
		Metrics: server.Options{
			// close metrics server
			BindAddress: "0",
		},
		Cache: cacheOpt,
	})
	if err != nil {
		return fmt.Errorf("unable to new manager: %w", err)
	}

	if err := secret.EnsureDirExists(afero.NewBasePathFs(afero.NewOsFs(), cfg.BaseDir), cfg.SecretDirName); err != nil {
		return fmt.Errorf("unable to ensure secret dir exists: %w", err)
	}

	if err := secret.Setup(mgr, filepath.Join(cfg.BaseDir, cfg.SecretDirName), cfg.Namespace, cfg.Labels); err != nil {
		return fmt.Errorf("unable to setup secret controller: %w", err)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}
	ctx := ctrl.SetupSignalHandler()

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("start manager failed: %w", err)
	}

	return nil
}
