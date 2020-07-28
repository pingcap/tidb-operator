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

package tests

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/pingcap/tidb-operator/pkg/tkctl/util"
	sql_util "github.com/pingcap/tidb-operator/tests/pkg/util"
	"github.com/pingcap/tidb-operator/tests/slack"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
)

const (
	DrainerReplicas int32 = 1
	// TODO: better way to do incremental restore from pb files
	RunReparoCommandTemplate = `kubectl exec -n={{ .Namespace }} {{ .PodName }} -- sh -c \
"while [ \$(grep -r 'commitTS' /data/savepoint| awk '{print (\$3)}') -lt {{ .StopTSO }} ]; do echo 'wait end tso reached' && sleep 60; done; \
printf '{{ .ReparoConfig }}' > reparo.toml && \
./reparo -config reparo.toml > /data/reparo.log" `
)

type BackupTarget struct {
	IncrementalType DbType
	TargetCluster   *TidbClusterConfig
	IsAdditional    bool
}

func (t *BackupTarget) GetDrainerConfig(source *TidbClusterConfig, ts string) *DrainerConfig {
	drainerConfig := &DrainerConfig{
		DrainerName:       fmt.Sprintf("%s-%s-drainer", source.ClusterName, t.IncrementalType),
		InitialCommitTs:   ts,
		OperatorTag:       source.OperatorTag,
		SourceClusterName: source.ClusterName,
		Namespace:         source.Namespace,
		DbType:            t.IncrementalType,
	}
	if t.IncrementalType == DbTypeMySQL || t.IncrementalType == DbTypeTiDB {
		drainerConfig.Host = fmt.Sprintf("%s.%s.svc.cluster.local",
			t.TargetCluster.ClusterName, t.TargetCluster.Namespace)
		drainerConfig.Port = "4000"
	}
	return drainerConfig
}

func (oa *operatorActions) BackupAndRestoreToMultipleClusters(source *TidbClusterConfig, targets []BackupTarget) error {
	err := oa.DeployAndCheckPump(source)
	if err != nil {
		return err
	}

	err = oa.BeginInsertDataTo(source)
	if err != nil {
		return err
	}
	klog.Infof("waiting 30 seconds to insert into more records")
	time.Sleep(30 * time.Second)

	// we should stop insert data before backup
	// Restoring via reparo is slow, so we stop inserting data as early as possible to reduce the size of incremental data
	oa.StopInsertDataTo(source)

	err = oa.DeployAdHocBackup(source)
	if err != nil {
		klog.Errorf("cluster:[%s] deploy happen error: %v", source.ClusterName, err)
		return err
	}

	ts, err := oa.CheckAdHocBackup(source)
	if err != nil {
		klog.Errorf("cluster:[%s] deploy happen error: %v", source.ClusterName, err)
		return err
	}

	prepareIncremental := func(source *TidbClusterConfig, target BackupTarget) error {
		err = oa.CheckTidbClusterStatus(target.TargetCluster)
		if err != nil {
			klog.Errorf("cluster:[%s] deploy faild error: %v", target.TargetCluster.ClusterName, err)
			return err
		}

		err = oa.Restore(source, target.TargetCluster)
		if err != nil {
			klog.Errorf("from cluster:[%s] to cluster [%s] restore happen error: %v",
				source.ClusterName, target.TargetCluster.ClusterName, err)
			return err
		}
		err = oa.CheckRestore(source, target.TargetCluster)
		if err != nil {
			klog.Errorf("from cluster:[%s] to cluster [%s] restore failed error: %v",
				source.ClusterName, target.TargetCluster.ClusterName, err)
			return err
		}

		if target.IsAdditional {
			// Deploy an additional drainer release
			drainerConfig := target.GetDrainerConfig(source, ts)
			if err := oa.DeployDrainer(drainerConfig, source); err != nil {
				return err
			}
			if err := oa.CheckDrainer(drainerConfig, source); err != nil {
				return err
			}
		} else {
			// Enable drainer of the source TiDB cluster release
			if err := oa.DeployAndCheckIncrementalBackup(source, target.TargetCluster, ts); err != nil {
				return err
			}
		}
		return nil
	}

	checkIncremental := func(source *TidbClusterConfig, target BackupTarget, stopWriteTS int64) error {
		if target.IncrementalType == DbTypeFile {
			var eg errgroup.Group
			// Run reparo restoring and check concurrently to show restoring progress
			eg.Go(func() error {
				return oa.RestoreIncrementalFiles(target.GetDrainerConfig(source, ts), target.TargetCluster, stopWriteTS)
			})
			eg.Go(func() error {
				return oa.CheckDataConsistency(source, target.TargetCluster, 60*time.Minute)
			})
			if err := eg.Wait(); err != nil {
				return err
			}
		} else {
			if err := oa.CheckDataConsistency(source, target.TargetCluster, 30*time.Minute); err != nil {
				return err
			}
		}

		return nil
	}

	var eg errgroup.Group
	for _, target := range targets {
		target := target
		eg.Go(func() error {
			return prepareIncremental(source, target)
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	oa.BeginInsertDataToOrDie(source)
	if err != nil {
		return err
	}
	klog.Infof("waiting 30 seconds to insert into more records")
	time.Sleep(30 * time.Second)

	klog.Infof("cluster[%s] stop insert data", source.ClusterName)
	oa.StopInsertDataTo(source)

	klog.Infof("wait on-going inserts to be drained for 60 seconds")
	time.Sleep(60 * time.Second)

	dsn, cancel, err := oa.getTiDBDSN(source.Namespace, source.ClusterName, "test", source.Password)
	if err != nil {
		return err
	}
	defer cancel()
	stopWriteTS, err := sql_util.ShowMasterCommitTS(dsn)
	if err != nil {
		return err
	}

	for _, target := range targets {
		target := target
		eg.Go(func() error {
			return checkIncremental(source, target, stopWriteTS)
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	err = oa.BeginInsertDataTo(source)
	if err != nil {
		return err
	}
	klog.Infof("waiting 30 seconds to insert into more records")
	time.Sleep(30 * time.Second)

	klog.Infof("cluster[%s] stop insert data", source.ClusterName)
	oa.StopInsertDataTo(source)

	err = oa.DeployScheduledBackup(source)
	if err != nil {
		klog.Errorf("cluster:[%s] scheduler happen error: %v", source.ClusterName, err)
		return err
	}

	return oa.CheckScheduledBackup(source)
}

func (oa *operatorActions) BackupAndRestoreToMultipleClustersOrDie(source *TidbClusterConfig, targets []BackupTarget) {
	if err := oa.BackupAndRestoreToMultipleClusters(source, targets); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) BackupRestore(from, to *TidbClusterConfig) error {

	return oa.BackupAndRestoreToMultipleClusters(from, []BackupTarget{
		{
			TargetCluster:   to,
			IncrementalType: DbTypeTiDB,
			IsAdditional:    false,
		},
	})
}

func (oa *operatorActions) BackupRestoreOrDie(from, to *TidbClusterConfig) {
	if err := oa.BackupRestore(from, to); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) DeployAndCheckPump(tc *TidbClusterConfig) error {
	if err := oa.DeployIncrementalBackup(tc, nil, false, ""); err != nil {
		return err
	}

	return oa.CheckIncrementalBackup(tc, false)
}

func (oa *operatorActions) DeployAndCheckIncrementalBackup(from, to *TidbClusterConfig, ts string) error {
	if err := oa.DeployIncrementalBackup(from, to, true, ts); err != nil {
		return err
	}

	return oa.CheckIncrementalBackup(from, true)
}

func (oa *operatorActions) CheckDataConsistency(from, to *TidbClusterConfig, timeout time.Duration) error {
	fn := func() (bool, error) {
		b, err := oa.DataIsTheSameAs(to, from)
		if err != nil {
			klog.Error(err)
			return false, nil
		}
		if b {
			return true, nil
		}
		return false, nil
	}

	return wait.Poll(DefaultPollInterval, timeout, fn)
}

func (oa *operatorActions) DeployDrainer(info *DrainerConfig, source *TidbClusterConfig) error {
	oa.EmitEvent(source, "DeployDrainer")
	klog.Infof("begin to deploy drainer [%s] namespace[%s], source cluster [%s]", info.DrainerName,
		source.Namespace, source.ClusterName)

	valuesPath, err := info.BuildSubValues(oa.drainerChartPath(source.OperatorTag))
	if err != nil {
		return err
	}

	override := map[string]string{}
	if len(oa.cfg.AdditionalDrainerVersion) > 0 {
		override["clusterVersion"] = oa.cfg.AdditionalDrainerVersion
	}
	if info.TLSCluster {
		override["clusterVersion"] = source.ClusterVersion
		override["tlsCluster.enabled"] = "true"
	}

	cmd := fmt.Sprintf("helm install %s  --name %s --namespace %s --set-string %s -f %s",
		oa.drainerChartPath(source.OperatorTag), info.DrainerName, source.Namespace, info.DrainerHelmString(override, source), valuesPath)
	klog.Info(cmd)

	if res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to deploy drainer [%s/%s], %v, %s",
			source.Namespace, info.DrainerName, err, string(res))
	}

	return nil
}

func (oa *operatorActions) DeployDrainerOrDie(info *DrainerConfig, source *TidbClusterConfig) {
	if err := oa.DeployDrainer(info, source); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) CheckDrainer(info *DrainerConfig, source *TidbClusterConfig) error {
	klog.Infof("checking drainer [%s/%s]", info.DrainerName, source.Namespace)

	ns := source.Namespace
	stsName := fmt.Sprintf("%s-%s-drainer", source.ClusterName, info.DrainerName)
	fn := func() (bool, error) {
		sts, err := oa.kubeCli.AppsV1().StatefulSets(source.Namespace).Get(stsName, v1.GetOptions{})
		if err != nil {
			klog.Errorf("failed to get drainer StatefulSet %s ,%v", sts, err)
			return false, nil
		}
		if *sts.Spec.Replicas != DrainerReplicas {
			klog.Infof("StatefulSet: %s/%s .spec.Replicas(%d) != %d",
				ns, sts.Name, *sts.Spec.Replicas, DrainerReplicas)
			return false, nil
		}
		if sts.Status.ReadyReplicas != DrainerReplicas {
			klog.Infof("StatefulSet: %s/%s .state.ReadyReplicas(%d) != %d",
				ns, sts.Name, sts.Status.ReadyReplicas, DrainerReplicas)
			return false, nil
		}
		for i := 0; i < int(*sts.Spec.Replicas); i++ {
			podName := fmt.Sprintf("%s-%d", stsName, i)
			if !oa.drainerHealth(source.ClusterName, source.Namespace, podName, info.TLSCluster) {
				klog.Infof("%s is not health yet", podName)
				return false, nil
			}
		}
		return true, nil
	}

	err := wait.Poll(DefaultPollInterval, DefaultPollTimeout, fn)
	if err != nil {
		return fmt.Errorf("failed to install drainer [%s/%s], %v", source.Namespace, info.DrainerName, err)
	}

	return nil
}

func (oa *operatorActions) RestoreIncrementalFiles(from *DrainerConfig, to *TidbClusterConfig, stopTSO int64) error {
	klog.Infof("restoring incremental data from drainer [%s/%s] to TiDB cluster [%s/%s]",
		from.Namespace, from.DrainerName, to.Namespace, to.ClusterName)

	// TODO: better incremental files restore solution
	reparoConfig := strings.Join([]string{
		`data-dir = \"/data/pb\"`,
		`log-level = \"info\"`,
		`dest-type = \"mysql\"`,
		`safe-mode = true`,
		fmt.Sprintf(`stop-tso = %d`, stopTSO),
		`[dest-db]`,
		fmt.Sprintf(`host = \"%s\"`, util.GetTidbServiceName(to.ClusterName)),
		"port = 4000",
		`user = \"root\"`,
		fmt.Sprintf(`password = \"%s\"`, to.Password),
	}, "\n")

	temp, err := template.New("reparo-command").Parse(RunReparoCommandTemplate)
	if err != nil {
		return err
	}
	buff := new(bytes.Buffer)
	if err := temp.Execute(buff, &struct {
		Namespace    string
		ReparoConfig string
		PodName      string
		StopTSO      int64
	}{
		Namespace:    from.Namespace,
		ReparoConfig: reparoConfig,
		PodName:      fmt.Sprintf("%s-%s-drainer-0", from.SourceClusterName, from.DrainerName),
		StopTSO:      stopTSO,
	}); err != nil {
		return err
	}

	cmd := buff.String()
	klog.Infof("Restore incremental data, command: \n%s", cmd)

	if res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restore incremental files from dainer [%s/%s] to TiDB cluster [%s/%s], %v, %s",
			from.Namespace, from.DrainerName, to.Namespace, to.ClusterName, err, res)
	}
	return nil
}
