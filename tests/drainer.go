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
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"text/template"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/slack"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/test/e2e/framework/log"
)

type DbType string

const (
	DrainerReplicas int32 = 1

	DbTypeMySQL DbType = "mysql"
	DbTypeFile  DbType = "file"
	DbTypeTiDB  DbType = "tidb"
	// The following downstream types are not supported in e2e & stability tests yet
	// DbTypeKafka DbType = "kafka"
	// DbTypeFlash DbType = "flash"
)

var sqlDrainerConfigTemp string = drainerConfigCommon + `
  db-type = "{{ .DbType }}"
  [syncer.to]
  host = "{{ .Host }}"
  user = "{{ .User }}"
  password = "{{ .Password }}"
  port = {{ .Port }}
`

var fileDrainerConfigTemp string = drainerConfigCommon + `
  db-type = "file"
  [syncer.to]
  dir = "/data/pb"
`

var drainerConfigCommon string = `
initialCommitTs: "{{ .InitialCommitTs }}"
config: |
  detect-interval = 10
  compressor = ""
  [syncer]
  worker-count = 16
  disable-dispatch = false
  ignore-schemas = "INFORMATION_SCHEMA,PERFORMANCE_SCHEMA,mysql"
  safe-mode = false
  txn-batch = 20
`

type DrainerSourceConfig struct {
	Namespace      string
	ClusterName    string
	ClusterVersion string
	OperatorTag    string
}

type DrainerConfig struct {
	DrainerName       string
	InitialCommitTs   string
	OperatorTag       string
	SourceClusterName string
	Namespace         string

	DbType   DbType
	Host     string
	User     string
	Password string
	// use string type in case of empty port (db-type=file)
	Port       string
	TLSCluster bool
}

func (oa *OperatorActions) DeployDrainer(info *DrainerConfig, source *DrainerSourceConfig) error {
	log.Logf("begin to deploy drainer [%s] namespace[%s], source cluster [%s]", info.DrainerName, source.Namespace, source.ClusterName)

	valuesPath, err := info.BuildSubValues(oa.drainerChartPath(source.OperatorTag))
	if err != nil {
		return err
	}

	override := map[string]string{
		"port":         strconv.FormatInt(int64(v1alpha1.DefaultDrainerPort), 10),
		"pdClientPort": strconv.FormatInt(int64(v1alpha1.DefaultPDClientPort), 10),
	}
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

func (oa *OperatorActions) CheckDrainer(info *DrainerConfig, source *DrainerSourceConfig) error {
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

func (d *DrainerConfig) BuildSubValues(dir string) (string, error) {
	values := GetDrainerSubValuesOrDie(d)
	path := fmt.Sprintf("%s/%s.yaml", dir, d.DrainerName)
	if err := ioutil.WriteFile(path, []byte(values), 0644); err != nil {
		return "", err
	}
	log.Logf("Values of drainer %s:\n %s", d.DrainerName, values)
	return path, nil
}

func (d *DrainerConfig) DrainerHelmString(m map[string]string, source *DrainerSourceConfig) string {

	set := map[string]string{
		"clusterName":    source.ClusterName,
		"clusterVersion": source.ClusterVersion,
	}

	for k, v := range m {
		set[k] = v
	}

	arr := make([]string, 0, len(set))
	for k, v := range set {
		arr = append(arr, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(arr, ",")
}

func GetDrainerSubValuesOrDie(info *DrainerConfig) string {
	if info == nil {
		slack.NotifyAndPanic(fmt.Errorf("Cannot get drainer sub values, the drainer config is nil"))
	}
	buff := new(bytes.Buffer)
	switch info.DbType {
	case DbTypeFile:
		temp, err := template.New("file-drainer").Parse(fileDrainerConfigTemp)
		if err != nil {
			slack.NotifyAndPanic(err)
		}
		if err := temp.Execute(buff, &info); err != nil {
			slack.NotifyAndPanic(err)
		}
	case DbTypeTiDB:
		fallthrough
	case DbTypeMySQL:
		temp, err := template.New("sql-drainer").Parse(sqlDrainerConfigTemp)
		if err != nil {
			slack.NotifyAndPanic(err)
		}
		if err := temp.Execute(buff, &info); err != nil {
			slack.NotifyAndPanic(err)
		}
	default:
		slack.NotifyAndPanic(fmt.Errorf("db-type %s has not been suppored yet", info.DbType))
	}
	return buff.String()
}
