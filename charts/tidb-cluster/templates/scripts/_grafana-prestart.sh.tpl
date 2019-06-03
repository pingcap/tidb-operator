#!/bin/bash

#check tidb cluster version
TiDBVersion={{ .Values.tidb.image }}
TiKVVersion={{ .Values.tikv.image }}
PDVersion={{ .Values.pd.image }}
PumpVersion={{ .Values.binlog.pump.image }}
V3=v3

#check binlog switch
enablePump={{ .Values.binlog.pump.create | default false }}

#replace dashboard name using cluster name
clusterName={{ template "cluster.name" . }}
clusterName=${clusterName:-"TiDB-Cluster"}

#TiDB dashboard
if [[ $TiDBVersion =~ $V3 ]]
then
    cp /tmp/dashboard-v3/tidbV3.json /grafana-dashboard-definitions/tidb/
else
    cp /tmp/dashboard-v2/tidbV2.json /grafana-dashboard-definitions/tidb/
fi
sed -i 's/TIDB-Cluster-TiDB/'$clusterName'-TiDB/g'  /grafana-dashboard-definitions/tidb/tidbV*.json

#Overview dashboard
if [[ $TiDBVersion =~ $V3 ]]
then
    cp /tmp/dashboard-v3/overviewV3.json /grafana-dashboard-definitions/tidb/
else
    cp /tmp/dashboard-v2/overviewV2.json /grafana-dashboard-definitions/tidb/
fi
sed -i 's/TIDB-Cluster-Overview/'$clusterName'-Overview/g'  /grafana-dashboard-definitions/tidb/overviewV*.json

#PD dashboard
if [[ $PDVersion =~ $V3 ]]
then
    cp /tmp/dashboard-v3/pdV3.json /grafana-dashboard-definitions/tidb/
else
    cp /tmp/dashboard-v2/pdV2.json /grafana-dashboard-definitions/tidb/
fi
sed -i 's/TIDB-Cluster-PD/'$clusterName'-PD/g'  /grafana-dashboard-definitions/tidb/pdV*.json

#TIKV dashboard
if [[ $TiKVVersion =~ $V3 ]]
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


#Binlog dashboard
if [[ $PumpVersion =~ $V3 ]]
then
    if $enablePump
    then
        cp /tmp/dashboard-v3/binlogV3.json /grafana-dashboard-definitions/tidb/
        sed -i 's/TIDB-Cluster-Binlog/'$clusterName'-Binlog/g'  /grafana-dashboard-definitions/tidb/binlogV3.json
    fi
else
    if $enablePump
    then
        cp /tmp/dashboard-v2/binlogV2.json /grafana-dashboard-definitions/tidb/
        sed -i 's/TIDB-Cluster-Binlog/'$clusterName'-Binlog/g'  /grafana-dashboard-definitions/tidb/binlogV2.json
    fi
fi