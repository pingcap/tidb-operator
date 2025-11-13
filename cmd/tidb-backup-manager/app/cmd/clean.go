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

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/cmd/tidb-backup-manager/app/clean"
	"github.com/pingcap/tidb-operator/v2/cmd/tidb-backup-manager/app/util"
	backupMgr "github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/backup"
	"github.com/pingcap/tidb-operator/v2/pkg/scheme"
)

// NewCleanCommand implements the clean command
func NewCleanCommand() *cobra.Command {
	bo := clean.Options{}

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean specific tidb cluster backup.",
		Run: func(cmd *cobra.Command, args []string) {
			util.ValidCmdFlags(cmd.CommandPath(), cmd.LocalFlags())
			cmdutil.CheckErr(runClean(bo, kubecfg))
		},
	}

	cmd.Flags().StringVar(&bo.Namespace, "namespace", "", "Tidb cluster's namespace")
	cmd.Flags().StringVar(&bo.BackupName, "backupName", "", "Backup CRD object name")
	return cmd
}

func runClean(backupOpts clean.Options, kubecfg string) error {
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

	recorder := record.NewBroadcaster().NewRecorder(scheme.Scheme, corev1.EventSource{Component: "clean"})
	statusUpdater := backupMgr.NewRealBackupConditionUpdater(newcli, recorder)
	klog.Infof("start to clean backup %s", backupOpts.String())
	bm := clean.NewManager(newcli, statusUpdater, backupOpts)
	return bm.ProcessCleanBackup()
}
