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

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/cmd/tidb-backup-manager/app/backup"
	"github.com/pingcap/tidb-operator/cmd/tidb-backup-manager/app/util"
	backupMgr "github.com/pingcap/tidb-operator/pkg/controllers/br/manager/backup"
	"github.com/pingcap/tidb-operator/pkg/scheme"
)

const (
	defaultCacheSyncWaitTimeout = 30 * time.Second
)

// NewBackupCommand implements the backup command
func NewBackupCommand() *cobra.Command {
	bo := backup.Options{}

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup specific tidb cluster.",
		Run: func(cmd *cobra.Command, args []string) {
			util.ValidCmdFlags(cmd.CommandPath(), cmd.LocalFlags())
			cmdutil.CheckErr(runBackup(bo, kubecfg))
		},
	}

	cmd.Flags().StringVar(&bo.Namespace, "namespace", "", "Backup CR's namespace")
	cmd.Flags().StringVar(&bo.ResourceName, "backupName", "", "Backup CRD object name")
	cmd.Flags().StringVar(&bo.TiKVVersion, "tikvVersion", util.DefaultVersion, "TiKV version")
	cmd.Flags().BoolVar(&bo.TLSClient, "client-tls", false, "Whether client tls is enabled")
	cmd.Flags().BoolVar(&bo.TLSCluster, "cluster-tls", false, "Whether cluster tls is enabled")
	cmd.Flags().BoolVar(&bo.SkipClientCA, "skipClientCA", false, "Whether to skip tidb server's certificates validation")
	cmd.Flags().StringVar(&bo.Mode, "mode", string(v1alpha1.BackupModeSnapshot), "backup mode, which is log or snapshot(default)")
	cmd.Flags().StringVar(&bo.SubCommand, "subcommand", string(v1alpha1.LogStartCommand), "the log backup subcommand")
	cmd.Flags().StringVar(&bo.CommitTS, "commit-ts", "0", "the log backup start ts")
	cmd.Flags().StringVar(&bo.TruncateUntil, "truncate-until", "0", "the log backup truncate until")
	cmd.Flags().BoolVar(&bo.Initialize, "initialize", false, "Whether execute initialize process for volume backup")
	cmd.Flags().StringVar(&bo.PDAddress, "pd-addr", "", "PD Address, for example: db-pd.tidb1234:2379")
	return cmd
}

//nolint:gocritic
func runBackup(backupOpts backup.Options, kubecfg string) error {
	var (
		kubeconfig *rest.Config
		err        error
	)
	if kubecfg != "" {
		kubeconfig, err = clientcmd.BuildConfigFromFlags("", kubecfg)
	} else {
		kubeconfig, err = rest.InClusterConfig()
	}
	if err != nil {
		return err
	}

	newcli, err := newClient(context.Background(), kubeconfig, scheme.Scheme, []client.Object{&v1alpha1.Backup{}})
	if err != nil {
		return err
	}

	recorder := record.NewBroadcaster().NewRecorder(scheme.Scheme, corev1.EventSource{Component: "backup"})
	statusUpdater := backupMgr.NewRealBackupConditionUpdater(newcli, recorder)
	klog.Infof("start to process backup %s", backupOpts.String())
	bm := backup.NewManager(newcli, statusUpdater, backupOpts)
	return bm.ProcessBackup()
}

func BuildCacheByObject() map[client.Object]cache.ByObject {
	byObj := map[client.Object]cache.ByObject{
		&v1alpha1.Backup{}: {
			Label: labels.Everything(),
		},
	}

	return byObj
}

func newClient(ctx context.Context, cfg *rest.Config, s *runtime.Scheme, disabledFor []client.Object) (client.Client, error) {
	hc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}

	reader, err := cache.New(cfg, cache.Options{
		Scheme:     s,
		HTTPClient: hc,
	})
	if err != nil {
		return nil, err
	}

	// it will not return any errors
	//nolint: errcheck
	go reader.Start(ctx)

	nctx, cancel := context.WithTimeout(ctx, defaultCacheSyncWaitTimeout)
	defer cancel()
	if !reader.WaitForCacheSync(nctx) {
		return nil, fmt.Errorf("cache is not synced: %w", nctx.Err())
	}

	c, err := client.New(cfg, client.Options{
		Scheme:     s,
		HTTPClient: hc,
		Cache: &client.CacheOptions{
			Reader:     reader,
			DisableFor: disabledFor,
		},
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}
