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
	"context"
	"fmt"
	"os/exec"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/test/e2e/framework/log"
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
	TargetCluster   *SourceTidbClusterConfig
	IsAdditional    bool
}

func (t *BackupTarget) GetDrainerConfig(source *SourceTidbClusterConfig, ts string) *DrainerConfig {
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

func (oa *OperatorActions) DeployDrainer(info *DrainerConfig, source *SourceTidbClusterConfig) error {
	log.Logf("begin to deploy drainer [%s] namespace[%s], source cluster [%s]", info.DrainerName,
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

	cmd := fmt.Sprintf("helm install %s %s --namespace %s --set-string %s -f %s",
		info.DrainerName,
		oa.drainerChartPath(source.OperatorTag),
		source.Namespace,
		info.DrainerHelmString(override, source),
		valuesPath)
	log.Logf(cmd)

	if res, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to deploy drainer [%s/%s], %v, %s",
			source.Namespace, info.DrainerName, err, string(res))
	}

	return nil
}

func (oa *OperatorActions) CheckDrainer(info *DrainerConfig, source *SourceTidbClusterConfig) error {
	log.Logf("checking drainer [%s/%s]", info.DrainerName, source.Namespace)

	ns := source.Namespace
	stsName := fmt.Sprintf("%s-%s-drainer", source.ClusterName, info.DrainerName)
	fn := func() (bool, error) {
		sts, err := oa.kubeCli.AppsV1().StatefulSets(source.Namespace).Get(context.TODO(), stsName, v1.GetOptions{})
		if err != nil {
			log.Logf("ERROR: failed to get drainer StatefulSet %s ,%v", sts, err)
			return false, nil
		}
		if *sts.Spec.Replicas != DrainerReplicas {
			log.Logf("StatefulSet: %s/%s .spec.Replicas(%d) != %d",
				ns, sts.Name, *sts.Spec.Replicas, DrainerReplicas)
			return false, nil
		}
		if sts.Status.ReadyReplicas != DrainerReplicas {
			log.Logf("StatefulSet: %s/%s .state.ReadyReplicas(%d) != %d",
				ns, sts.Name, sts.Status.ReadyReplicas, DrainerReplicas)
			return false, nil
		}
		for i := 0; i < int(*sts.Spec.Replicas); i++ {
			podName := fmt.Sprintf("%s-%d", stsName, i)
			if !oa.drainerHealth(source.ClusterName, source.Namespace, podName, info.TLSCluster) {
				log.Logf("%s is not health yet", podName)
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
