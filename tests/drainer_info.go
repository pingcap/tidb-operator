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
	"fmt"
	"io/ioutil"
	"strings"

	"k8s.io/klog"
)

type DbType string

const (
	DbTypeMySQL DbType = "mysql"
	DbTypeFile  DbType = "file"
	DbTypeTiDB  DbType = "tidb"
	// The following downstream types are not supported in e2e & stability tests yet
	// DbTypeKafka DbType = "kafka"
	// DbTypeFlash DbType = "flash"
)

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
	Port string
}

func (d *DrainerConfig) DrainerHelmString(m map[string]string, source *TidbClusterConfig) string {

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

func (d *DrainerConfig) BuildSubValues(dir string) (string, error) {

	values := GetDrainerSubValuesOrDie(d)
	path := fmt.Sprintf("%s/%s.yaml", dir, d.DrainerName)
	if err := ioutil.WriteFile(path, []byte(values), 0644); err != nil {
		return "", err
	}
	klog.Infof("Values of drainer %s:\n %s", d.DrainerName, values)
	return path, nil
}
