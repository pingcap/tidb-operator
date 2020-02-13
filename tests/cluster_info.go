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
	"os"
	"strconv"

	"k8s.io/klog"
)

func (tc *TidbClusterConfig) set(name string, value string) (string, bool) {
	// NOTE: not thread-safe, maybe make info struct immutable
	if tc.Args == nil {
		tc.Args = make(map[string]string)
	}
	origVal, ok := tc.Args[name]
	tc.Args[name] = value
	return origVal, ok
}

func (tc *TidbClusterConfig) ScalePD(replicas uint) *TidbClusterConfig {
	tc.set("pd.replicas", strconv.Itoa(int(replicas)))
	return tc
}

func (tc *TidbClusterConfig) ScaleTiKV(replicas uint) *TidbClusterConfig {
	tc.set("tikv.replicas", strconv.Itoa(int(replicas)))
	return tc
}

func (tc *TidbClusterConfig) ScaleTiDB(replicas uint) *TidbClusterConfig {
	tc.set("tidb.replicas", strconv.Itoa(int(replicas)))
	return tc
}

func (tc *TidbClusterConfig) RunInHost(flag bool) *TidbClusterConfig {
	val := "false"
	if flag {
		val = "true"
	}
	tc.set("pd.hostNetwork", val)
	tc.set("tikv.hostNetwork", val)
	tc.set("tidb.hostNetwork", val)
	return tc
}

func (tc *TidbClusterConfig) UpgradePD(image string) *TidbClusterConfig {
	tc.PDImage = image
	return tc
}

func (tc *TidbClusterConfig) UpgradeTiKV(image string) *TidbClusterConfig {
	tc.TiKVImage = image
	return tc
}

func (tc *TidbClusterConfig) UpgradeTiDB(image string) *TidbClusterConfig {
	tc.TiDBImage = image
	return tc
}

func (tc *TidbClusterConfig) UpgradeAll(tag string) *TidbClusterConfig {
	tc.ClusterVersion = tag
	return tc.
		UpgradePD("pingcap/pd:" + tag).
		UpgradeTiKV("pingcap/tikv:" + tag).
		UpgradeTiDB("pingcap/tidb:" + tag)
}

// FIXME: update of PD configuration do not work now #487
func (tc *TidbClusterConfig) UpdatePdMaxReplicas(maxReplicas int) *TidbClusterConfig {
	tc.PDMaxReplicas = maxReplicas
	return tc
}

func (tc *TidbClusterConfig) UpdateTiKVGrpcConcurrency(concurrency int) *TidbClusterConfig {
	tc.TiKVGrpcConcurrency = concurrency
	return tc
}

func (tc *TidbClusterConfig) UpdateTiDBTokenLimit(tokenLimit int) *TidbClusterConfig {
	tc.TiDBTokenLimit = tokenLimit
	return tc
}

func (tc *TidbClusterConfig) UpdatePDLogLevel(logLevel string) *TidbClusterConfig {
	tc.PDLogLevel = logLevel
	return tc
}

func (tc *TidbClusterConfig) DSN(dbName string) string {
	return fmt.Sprintf("root:%s@tcp(%s-tidb.%s:4000)/%s", tc.Password, tc.ClusterName, tc.Namespace, dbName)
}

func (tc *TidbClusterConfig) BuildSubValues(path string) (string, error) {
	pdLogLevel := tc.PDLogLevel
	if pdLogLevel == "" {
		pdLogLevel = "info"
	}
	pdMaxReplicas := tc.PDMaxReplicas
	if pdMaxReplicas == 0 {
		pdMaxReplicas = 3
	}
	tikvGrpcConcurrency := tc.TiKVGrpcConcurrency
	if tikvGrpcConcurrency == 0 {
		tikvGrpcConcurrency = 4
	}
	tidbTokenLimit := tc.TiDBTokenLimit
	if tidbTokenLimit == 0 {
		tidbTokenLimit = 1000
	}
	pdConfig := []string{
		"[log]",
		fmt.Sprintf(`level = "%s"`, pdLogLevel),
		"[replication]",
		fmt.Sprintf("max-replicas = %d", pdMaxReplicas),
		fmt.Sprintf(`location-labels = ["%s"]`, tc.TopologyKey),
	}
	tikvConfig := []string{
		"[log]",
		`level = "info"`,
		"[server]",
		fmt.Sprintf("grpc-concurrency = %d", tikvGrpcConcurrency),
	}
	tidbConfig := []string{
		fmt.Sprintf("token-limit = %d", tidbTokenLimit),
		"[log]",
		`level = "info"`,
	}
	subValues := GetSubValuesOrDie(tc.ClusterName, tc.Namespace, tc.TopologyKey, pdConfig, tikvConfig, tidbConfig, tc.pumpConfig, tc.drainerConfig)
	subVaulesPath := fmt.Sprintf("%s/%s.yaml", path, tc.ClusterName)
	_, err := os.Stat(subVaulesPath)
	if err != nil {
		if os.IsNotExist(err) {
			_, err = os.Create(subVaulesPath)
			if err != nil {
				return "", err
			}
		}
	}

	svFile, err := os.OpenFile(subVaulesPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return "", err
	}
	defer svFile.Close()
	_, err = svFile.WriteString(subValues)
	if err != nil {
		return "", err
	}
	klog.V(4).Infof("subValues:\n %s", subValues)
	return subVaulesPath, nil
}
