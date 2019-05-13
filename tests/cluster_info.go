package tests

import (
	"fmt"
	"strconv"
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

func (tc *TidbClusterConfig) DSN(dbName string) string {
	return fmt.Sprintf("root:%s@tcp(%s-tidb.%s:4000)/%s", tc.Password, tc.ClusterName, tc.Namespace, dbName)
}
