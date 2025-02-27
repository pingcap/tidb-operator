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
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/clean"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	backupMgr "github.com/pingcap/tidb-operator/pkg/controllers/br/manager/backup"
	"github.com/pingcap/tidb-operator/pkg/scheme"
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
