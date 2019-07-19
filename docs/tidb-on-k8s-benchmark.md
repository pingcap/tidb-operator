# TiDB on K8s benchmark

## 服务器信息

| 类别 | 名称 |
| ---- | ---- |
|OS | Linux (RHEL 7.4.1708) |
|CPU | 48 vCPUs, Intel(R) Xeon(R) Gold 5118 CPU @ 2.30GHz |
|RAM | 370G |
|DISK | NVMe SSD 1.6T|
|NIC | 10GbE SFP+|
## 版本信息
| 组件 | 版本 |
|----|----|
|TiDB-operator | v1.0.0-beta.3|
|TiDB-cluster | v3.0.0-rc.1|
## 集群拓扑
| 机器 IP | 部署实例 |
|----|----|
|x.x.x.2 | 3\*sysbench 1\*pd|
|x.x.x.3 | 1\*tikv |
|x.x.x.4 | 1\*tidb 1\*pd 1\*tikv|
|x.x.x.5 | 1\*tikv |
|x.x.x.6 | 1\*tidb 1\*pd 1\*monitor|
## 压测命令
| 类别 | 名称 |
| ---- | ----|
| 导入数据 | sysbench --config-file=config oltp_point_select --tables=32 --table-size=10000000 prepare |
| Point select | sysbench --config-file=config oltp_point_select --tables=32 --table-size=10000000 run |
| Update index | sysbench --config-file=config oltp_update_index --tables=32 --table-size=10000000 run |
| Read-only | sysbench --config-file=config oltp_read_only --tables=32 --table-size=10000000 run |
## 压测结果
|类型|Thread|TPS|QPS|avg.latency(ms)|.95.latency(ms)|max.latency(ms)|
|----|:----:|:----:|:----:|:----:|:----:|:----:|
|Point select | 3*16 | 91346.93	| 91346.93 | 0.52 | 0.70 | 17.65|
|Update index | 3*16 | 9708.47 | 9708.47 | 4.94 | 8.95| 4600|
|Read-only | 3*16 | 2862.34 | 45797.58 | 16.76 | 16.76 |45.87|
