#!/bin/sh

#decompress dashboard files
gzip -dc /tmp/dashboard-gz/tidb.json.gz > /grafana-dashboard-definitions/tidb/tidb.json
gzip -dc /tmp/dashboard-gz/pd.json.gz > /grafana-dashboard-definitions/tidb/pd.json
gzip -dc /tmp/dashboard-gz/tikv.json.gz > /grafana-dashboard-definitions/tidb/tikv.json
gzip -dc /tmp/dashboard-gz/overview.json.gz > /grafana-dashboard-definitions/tidb/overview.json

#replace dashboard name using cluster name
clusterName={{ template "cluster.name" . }}
clusterName=${clusterName:-"TiDB-Cluster"}

sed -i 's/TIDB-Cluster-TiDB/'$clusterName'-TiDB/g'  /grafana-dashboard-definitions/tidb/tidb.json
sed -i 's/TIDB-Cluster-PD/'$clusterName'-PD/g'  /grafana-dashboard-definitions/tidb/pd.json
sed -i 's/TIDB-Cluster-TiKV/'$clusterName'-TiKV/g'  /grafana-dashboard-definitions/tidb/tikv.json
sed -i 's/TIDB-Cluster-Overview/'$clusterName'-Overview/g'  /grafana-dashboard-definitions/tidb/overview.json
