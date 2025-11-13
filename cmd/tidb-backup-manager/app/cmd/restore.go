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
	// registry mysql drive

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/cmd/tidb-backup-manager/app/restore"
	"github.com/pingcap/tidb-operator/v2/cmd/tidb-backup-manager/app/util"
	restoreMgr "github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/restore"
	"github.com/pingcap/tidb-operator/v2/pkg/scheme"
)

// NewRestoreCommand implements the restore command
func NewRestoreCommand() *cobra.Command {
	ro := restore.Options{}

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore specific tidb cluster.",
		Run: func(cmd *cobra.Command, args []string) {
			util.ValidCmdFlags(cmd.CommandPath(), cmd.LocalFlags())
			cmdutil.CheckErr(runRestore(ro, kubecfg))
		},
	}

	cmd.Flags().StringVar(&ro.Namespace, "namespace", "", "Restore CR's namespace")
	cmd.Flags().StringVar(&ro.ResourceName, "restoreName", "", "Restore CRD object name")
	cmd.Flags().StringVar(&ro.TiKVVersion, "tikvVersion", util.DefaultVersion, "TiKV version")
	cmd.Flags().BoolVar(&ro.TLSClient, "client-tls", false, "Whether client tls is enabled")
	cmd.Flags().BoolVar(&ro.TLSCluster, "cluster-tls", false, "Whether cluster tls is enabled")
	cmd.Flags().BoolVar(&ro.SkipClientCA, "skipClientCA", false, "Whether to skip tidb server's certificates validation")
	cmd.Flags().StringVar(&ro.Mode, "mode", string(v1alpha1.RestoreModeSnapshot), "restore mode, which is pitr or snapshot(default)")
	cmd.Flags().StringVar(&ro.PitrRestoredTs, "pitrRestoredTs", "0", "The pitr restored ts")
	cmd.Flags().BoolVar(&ro.Prepare, "prepare", false, "Whether to prepare for restore")
	cmd.Flags().StringVar(&ro.TargetAZ, "target-az", "", "For volume-snapshot restore, which az the volume snapshots restore to")
	cmd.Flags().BoolVar(&ro.UseFSR, "use-fsr", false, "EBS snapshot restore use FSR for TiKV data volumes or not")
	cmd.Flags().StringVar(&ro.PDAddress, "pd-addr", "", "PD Address, for example: db-pd.tidb1234:2379")
	return cmd
}

//nolint:gocritic
func runRestore(restoreOpts restore.Options, kubecfg string) error {
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

	newcli, err := newClient(context.Background(), kubeconfig, scheme.Scheme, []client.Object{&v1alpha1.Restore{}})
	if err != nil {
		return err
	}

	recorder := record.NewBroadcaster().NewRecorder(scheme.Scheme, corev1.EventSource{Component: "restore"})
	statusUpdater := restoreMgr.NewRealRestoreConditionUpdater(newcli, recorder)
	klog.Infof("start to process restore %s", restoreOpts.String())
	rm := restore.NewManager(newcli, statusUpdater, restoreOpts)
	return rm.ProcessRestore()
}
