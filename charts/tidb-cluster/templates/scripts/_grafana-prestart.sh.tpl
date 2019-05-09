#!/bin/bash

#check tidb cluster version
tidbversion={{ .Values.tidb.image }}
tikvversion={{ .Values.tikv.image }}
pdversion={{ .Values.pd.image }}
v3=v3

#check binlog switch
enablePump={{ .Values.binlog.pump.create | default false }}

#replace dashboard name using cluster name
clusterName={{ template "cluster.name" . }}
clusterName=${clusterName:-"TiDB-Cluster"}

#TiDB dashboard copy and replace name
if [[ $tidbversion =~ $v3 ]]
then
    cp /tmp/dashboard-v3/tidbV3.json /grafana-dashboard-definitions/tidb/
else
    cp /tmp/dashboard-v2/tidbV2.json /grafana-dashboard-definitions/tidb/
fi
sed -i 's/TIDB-Cluster-TiDB/'$clusterName'-TiDB/g'  /grafana-dashboard-definitions/tidb/tidbV*.json

#Overview dashboard copy and replace name
if [[ $tidbversion =~ $v3 ]]
then
    cp /tmp/dashboard-v3/overviewV3.json /grafana-dashboard-definitions/tidb/
else
    cp /tmp/dashboard-v2/overviewV2.json /grafana-dashboard-definitions/tidb/
fi
sed -i 's/TIDB-Cluster-Overview/'$clusterName'-Overview/g'  /grafana-dashboard-definitions/tidb/overviewV*.json

#PD dashboard copy and replace name
if [[ $pdversion =~ $v3 ]]
then
    cp /tmp/dashboard-v3/pdV3.json /grafana-dashboard-definitions/tidb/
else
    cp /tmp/dashboard-v2/pdV2.json /grafana-dashboard-definitions/tidb/
fi
sed -i 's/TIDB-Cluster-PD/'$clusterName'-PD/g'  /grafana-dashboard-definitions/tidb/pdV*.json

#TIKV dashboard copy and replace name
if [[ $tikvversion =~ $v3 ]]
then
    cp /tmp/dashboard-v3/tikvV3.json /grafana-dashboard-definitions/tidb/
    cp /tmp/dashboard-v3/extra/tikvSummaryV3.json /grafana-dashboard-definitions/tidb/
    cp /tmp/dashboard-v3/extra/tikvTSV3.json /grafana-dashboard-definitions/tidb/
    sed -i 's/TIDB-Cluster-TiKV-Summary/'$clusterName'-TiKV-Summary/g'  /grafana-dashboard-definitions/tidb/tikvSummaryV3.json
    sed -i 's/TIDB-Cluster-TiKV-Trouble-Shooting/'$clusterName'-TiKV-Trouble-Shooting/g'  /grafana-dashboard-definitions/tidb/tikvTSV3.json
else
    cp /tmp/dashboard-v2/tikvV2.json /grafana-dashboard-definitions/tidb/
fi
sed -i 's/TIDB-Cluster-TiKV/'$clusterName'-TiKV/g'  /grafana-dashboard-definitions/tidb/tikvV*.json


if [[ $tikvversion =~ $v3 ]] && $enablePump
then
    if $enablePump
    then
        cp /tmp/dashboard-v3/binlogV3.json /grafana-dashboard-definitions/tidb/
    fi
else
    if $enablePump
    then
        cp /tmp/dashboard-v3/binlogV2.json /grafana-dashboard-definitions/tidb/
    fi
fi