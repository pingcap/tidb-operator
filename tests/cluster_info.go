package tests

import (
	"fmt"
	"strconv"
)

func (tc *TidbClusterInfo) set(name string, value string) (string, bool) {
	// NOTE: not thread-safe, maybe make info struct immutable
	if tc.Args == nil {
		tc.Args = make(map[string]string)
	}
	origVal, ok := tc.Args[name]
	tc.Args[name] = value
	return origVal, ok
}

func (tc *TidbClusterInfo) ScalePD(replicas uint) *TidbClusterInfo {
	tc.set("pd.replicas", strconv.Itoa(int(replicas)))
	return tc
}

func (tc *TidbClusterInfo) ScaleTiKV(replicas uint) *TidbClusterInfo {
	tc.set("tikv.replicas", strconv.Itoa(int(replicas)))
	return tc
}

func (tc *TidbClusterInfo) ScaleTiDB(replicas uint) *TidbClusterInfo {
	tc.set("tidb.replicas", strconv.Itoa(int(replicas)))
	return tc
}

func (tc *TidbClusterInfo) UpgradePD(image string) *TidbClusterInfo {
	tc.PDImage = image
	return tc
}

func (tc *TidbClusterInfo) UpgradeTiKV(image string) *TidbClusterInfo {
	tc.TiKVImage = image
	return tc
}

func (tc *TidbClusterInfo) UpgradeTiDB(image string) *TidbClusterInfo {
	tc.TiDBImage = image
	return tc
}

func (tc *TidbClusterInfo) UpgradeAll(tag string) *TidbClusterInfo {
	return tc.
		UpgradePD("pingcap/pd:" + tag).
		UpgradeTiKV("pingcap/tikv:" + tag).
		UpgradeTiDB("pingcap/tidb:" + tag)
}

func (tc *TidbClusterInfo) DSN(dbName string) string {
	return fmt.Sprintf("root:%s@tcp(%s-tidb.%s:4000)/%s", tc.Password, tc.ClusterName, tc.Namespace, dbName)
}
