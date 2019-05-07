package tests

import (
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/tests/slack"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (oa *operatorActions) BackupRestore(from, to *TidbClusterConfig) error {
	oa.StopInsertDataTo(from)

	err := oa.DeployAdHocBackup(from)
	if err != nil {
		glog.Errorf("cluster:[%s] deploy happen error: %v", from.ClusterName, err)
		return err
	}

	err = oa.CheckAdHocBackup(from)
	if err != nil {
		glog.Errorf("cluster:[%s] deploy happen error: %v", from.ClusterName, err)
		return err
	}

	err = oa.CheckTidbClusterStatus(to)
	if err != nil {
		glog.Errorf("cluster:[%s] deploy faild error: %v", to.ClusterName, err)
		return err
	}

	err = oa.Restore(from, to)
	if err != nil {
		glog.Errorf("from cluster:[%s] to cluster [%s] restore happen error: %v",
			from.ClusterName, to.ClusterName, err)
		return err
	}

	err = oa.CheckRestore(from, to)
	if err != nil {
		glog.Errorf("from cluster:[%s] to cluster [%s] restore failed error: %v",
			from.ClusterName, to.ClusterName, err)
		return err
	}

	go oa.BeginInsertDataToOrDie(from)
	err = oa.DeployScheduledBackup(from)
	if err != nil {
		glog.Errorf("cluster:[%s] scheduler happen error: %v", from.ClusterName, err)
		return err
	}

	err = oa.CheckScheduledBackup(from)
	if err != nil {
		glog.Errorf("cluster:[%s] scheduler failed error: %v", from.ClusterName, err)
		return err
	}

	err = oa.DeployIncrementalBackup(from, to)
	if err != nil {
		return err
	}

	err = oa.CheckIncrementalBackup(from)
	if err != nil {
		return err
	}

	glog.Infof("waiting 1 minutes for binlog to work")
	time.Sleep(1 * time.Minute)

	glog.Infof("cluster[%s] begin insert data", from.ClusterName)
	go oa.BeginInsertDataTo(from)

	time.Sleep(5 * time.Minute)

	glog.Infof("cluster[%s] stop insert data", from.ClusterName)
	oa.StopInsertDataTo(from)

	fn := func() (bool, error) {
		b, err := to.DataIsTheSameAs(from)
		if err != nil {
			glog.Error(err)
			return false, nil
		}
		if b {
			return true, nil
		}
		return false, nil
	}

	return wait.Poll(DefaultPollInterval, DefaultPollTimeout, fn)
}

func (oa *operatorActions) BackupRestoreOrDie(from, to *TidbClusterConfig) {
	if err := oa.BackupRestore(from, to); err != nil {
		slack.NotifyAndPanic(err)
	}
}
