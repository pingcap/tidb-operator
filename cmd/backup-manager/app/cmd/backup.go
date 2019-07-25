// Copyright 2019 PingCAP, Inc.
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

package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/backup"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

// backupOpts contains the input arguments to the backup command
type backupOpts struct {
	namespace   string
	tcName      string
	tidbSvc     string
	password    string
	user        string
	storageType string
	backup      string
}

func (bo *backupOpts) String() string {
	return fmt.Sprintf("%s/%s", bo.namespace, bo.tcName)
}

// NewBackupCommand implements the backup command
func NewBackupCommand(kubecfg string) *cobra.Command {
	bo := &backupOpts{}

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup specific tidb cluster.",
		Run: func(cmd *cobra.Command, args []string) {
			kubeCli, cli, err := util.NewKubeAndCRCli(kubecfg)
			cmdutil.CheckErr(err)
			options := []informers.SharedInformerOption{
				informers.WithNamespace(bo.namespace),
			}
			informerFactory := informers.NewSharedInformerFactoryWithOptions(cli, constants.ResyncDuration, options...)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go informerFactory.Start(ctx.Done())
			recorder := util.NewEventRecorder(kubeCli, "backup")
			backupInformer := informerFactory.Pingcap().V1alpha1().Backups()
			statusUpdater := controller.NewRealBackupConditionUpdater(cli, backupInformer.Lister(), recorder)
			cmdutil.CheckErr(bo.Run(kubeCli, statusUpdater))
		},
	}

	cmd.Flags().StringVarP(&bo.namespace, "namespace", "n", "", "Tidb cluster's namespace")
	cmd.Flags().StringVarP(&bo.tcName, "tidbcluster", "t", "", "Tidb cluster name")
	cmd.Flags().StringVarP(&bo.tidbSvc, "tidbservice", "s", "", "Tidb cluster access service address")
	cmd.Flags().StringVarP(&bo.password, "password", "p", "", "Password to use when connecting to tidb cluster")
	cmd.Flags().StringVarP(&bo.user, "user", "u", "", "User for login tidb cluster")
	cmd.Flags().StringVarP(&bo.storageType, "storageType", "S", "", "Backend storage type")
	cmd.Flags().StringVarP(&bo.backup, "backup", "k", "", "Backup CRD object name")
	return cmd
}

func (bo *backupOpts) Run(kubeCli kubernetes.Interface, statusUpdater controller.BackupConditionUpdaterInterface) error {
	db, err := util.OpenDB(bo.getDSN(constants.TidbMetaDB))
	if err != nil {
		return err
	}
	defer db.Close()

	oldTikvGCTime, err := bo.getTikvGCLifeTime(db)
	if err != nil {
		return err
	}
	glog.Infof("cluster %s %s is %s", bo, constants.TikvGCVariable, oldTikvGCTime)

	err = bo.setTikvGClifeTime(db, constants.TikvGCLifeTime)
	if err != nil {
		return err
	}
	glog.Infof("increase cluster %s %s to %s success", bo, constants.TikvGCVariable, constants.TikvGCLifeTime)

	backupFullPath, err := bo.dumpTidbClusterData()
	if err != nil {
		return err
	}
	glog.Infof("dump cluster %s data success", bo)

	err = bo.setTikvGClifeTime(db, oldTikvGCTime)
	if err != nil {
		return err
	}
	glog.Infof("reset cluster %s %s to %s success", bo, constants.TikvGCVariable, oldTikvGCTime)

	// TODO: Concurrent get file size and upload backup data to speed up processing time
	archiveBackupPath := backupFullPath + constants.DefaultArchiveExtention
	err = backup.ArchiveBackupData(backupFullPath, archiveBackupPath)
	if err != nil {
		return err
	}
	glog.Infof("archive cluster %s backup data %s success", bo, archiveBackupPath)

	// TODO: Maybe use rclone size command to get the archived backup file size is more efficiently than du
	_, err := backup.GetBackupSize(archiveBackupPath)
	if err != nil {
		return err
	}
	glog.Infof("get cluster %s archived backup file %s failed, err: %v", bo, archiveBackupPath, err)

	commitTs, err := backup.GetCommitTsFromMetadata(backupFullPath)
	if err != nil {
		return err
	}
	glog.Infof("get cluster %s commitTs %s success", bo, commitTs)

	err = bo.backupDataToRemote()
	if err != nil {
		return err
	}
	glog.Infof("backup cluster %s data to %s success", bo, bo.storageType)
	return nil
}

func (bo *backupOpts) getBackupFullPath() string {
	return filepath.Join(constants.BackupRootPath, bo.getBackupRelativePath())
}

func (bo *backupOpts) getBackupRelativePath() string {
	backupName := fmt.Sprintf("backup-%s", time.Now().UTC().Format(time.RFC3339))
	return fmt.Sprintf("%s_%s/%s", bo.namespace, bo.tcName, backupName)
}

func (bo *backupOpts) getDestBucket() string {
	return fmt.Sprint("%s:%s", bo.storageType, bo.getBackupRelativePath())
}

func (bo *backupOpts) getTikvGCLifeTime(db *sql.DB) (string, error) {
	var tikvGCTime string
	row := db.QueryRow("select variable_value from ? where variable_name= ?", constants.TidbMetaTable, constants.TikvGCVariable)
	err := row.Scan(&tikvGCTime)
	if err != nil {
		return tikvGCTime, fmt.Errorf("query cluster %s/%s %s failed, err: %v", bo.namespace, bo.tcName, constants.TikvGCVariable, err)
	}
	return tikvGCTime, nil
}

func (bo *backupOpts) setTikvGClifeTime(db *sql.DB, gcTime string) error {
	_, err := db.Exec("update ? set variable_value = ? where variable_name = ?", constants.TidbMetaTable, gcTime, constants.TikvGCVariable)
	if err != nil {
		return fmt.Errorf("set cluster %s/%s %s failed, err: %v", bo.namespace, bo.tcName, constants.TikvGCVariable, err)
	}
	return nil
}

func (bo *backupOpts) dumpTidbClusterData() (string, error) {
	bfPath := bo.getBackupFullPath()
	err := util.EnsureDirectoryExist(bfPath)
	if err != nil {
		return "", err
	}
	args := []string{
		fmt.Sprintf("--outputdir=%s", bfPath),
		fmt.Sprintf("--host=%s", bo.tidbSvc),
		"--port=4000",
		fmt.Sprintf("--user=%s", bo.user),
		fmt.Sprintf("--password=%s", bo.password),
		"--long-query-guard=3600",
		"--tidb-force-priority=LOW_PRIORITY",
		"--verbose=3",
	}

	dumper := exec.Command("/mydumper", args...)
	if err := dumper.Start(); err != nil {
		return bfPath, fmt.Errorf("cluster %s, start mydumper command %v falied, err: %v", bo, args, err)
	}
	if err := dumper.Wait(); err != nil {
		return bfPath, fmt.Errorf("cluster %s, execute mydumper command %v failed, err: %v", bo, args, err)
	}
	return bfPath, nil
}

func (bo *backupOpts) backupDataToRemote() error {
	destBucket := bo.getDestBucket()
	tmpDestBucket := fmt.Sprintf("%s.tmp", destBucket)
	// TODO: We may need to use exec.CommandContext to control timeouts.
	rcCopy := exec.Command("rclone", constants.RcloneConfigArg, "copyto", tmpDestBucket)
	if err := rcCopy.Start(); err != nil {
		return fmt.Errorf("cluster %s, start rclone copyto command falied, err: %v", bo, err)
	}
	if err := rcCopy.Wait(); err != nil {
		return fmt.Errorf("cluster %s, execute rclone copyto command failed, err: %v", bo, err)
	}

	glog.Info("backup was upload successfully, now move it to permanent URL")

	// the backup was a success
	// remove .tmp extension
	rcMove := exec.Command("rclone", constants.RcloneConfigArg, "moveto", tmpDestBucket, destBucket)

	if err := rcMove.Start(); err != nil {
		return fmt.Errorf("cluster %s, start rclone moveto command falied, err: %v", bo, err)
	}

	if err := rcMove.Wait(); err != nil {
		return fmt.Errorf("cluster %s, start rclone moveto command falied, err: %v", bo, err)
	}
	return nil
}

func (bo *backupOpts) getDSN(db string) string {
	return fmt.Sprintf("%s:%s@(%s:4000)/%s?/charset=utf8", bo.user, bo.password, bo.tidbSvc, db)
}
