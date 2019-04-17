groups:
- name: tidb-alert-rules
  rules:
  - alert: PD_cluster_offline_tikv_nums
    expr: sum ( pd_cluster_status{type="store_down_count"} ) > 0
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: emergency
      expr:  sum ( pd_cluster_status{type="store_down_count"} ) > 0
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}   values:{{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: PD_cluster_offline_tikv_nums

  - alert: PD_etcd_write_disk_latency
    expr: histogram_quantile(0.99, sum(rate(etcd_disk_wal_fsync_duration_seconds_bucket[1m])) by (instance,job,le) ) > 1
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr:  histogram_quantile(0.99, sum(rate(etcd_disk_wal_fsync_duration_seconds_bucket[1m])) by (instance,job,le) ) > 1
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}   values:{{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: PD_etcd_write_disk_latency

  - alert: PD_miss_peer_region_count
    expr: sum( pd_regions_status{type="miss_peer_region_count"} )  > 100
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr:  sum( pd_regions_status{type="miss_peer_region_count"} )  > 100
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}   values:{{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: PD_miss_peer_region_count

  - alert: PD_cluster_lost_connect_tikv_nums
    expr: sum ( pd_cluster_status{type="store_disconnected_count"} )   > 0
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  sum ( pd_cluster_status{type="store_disconnected_count"} )   > 0
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}   values:{{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: PD_cluster_lost_connect_tikv_nums

  - alert: PD_cluster_low_space
    expr: sum ( pd_cluster_status{type="store_low_space_count"} )  > 0
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  sum ( pd_cluster_status{type="store_low_space_count"} )  > 0
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}   values:{{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: PD_cluster_low_space

  - alert: PD_etcd_network_peer_latency
    expr: histogram_quantile(0.99, sum(rate(etcd_network_peer_round_trip_time_seconds_bucket[1m])) by (To,instance,job,le) ) > 1
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  histogram_quantile(0.99, sum(rate(etcd_network_peer_round_trip_time_seconds_bucket[1m])) by (To,instance,job,le) ) > 1
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}   values:{{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: PD_etcd_network_peer_latency

  - alert: PD_tidb_handle_requests_duration
    expr: histogram_quantile(0.99, sum(rate(pd_client_request_handle_requests_duration_seconds_bucket{type="tso"}[1m])) by (instance,job,le) ) > 0.1
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  histogram_quantile(0.99, sum(rate(pd_client_request_handle_requests_duration_seconds_bucket{type="tso"}[1m])) by (instance,job,le) ) > 0.1
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}   values:{{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: PD_tidb_handle_requests_duration

  - alert: PD_down_peer_region_nums
    expr: sum ( pd_regions_status{type="down_peer_region_count"} ) > 0
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  sum ( pd_regions_status{type="down_peer_region_count"} ) > 0
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}   values:{{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: PD_down_peer_region_nums

  - alert: PD_incorrect_namespace_region_count
    expr: sum ( pd_regions_status{type="incorrect_namespace_region_count"} ) > 100
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr: sum ( pd_regions_status{type="incorrect_namespace_region_count"} ) > 0
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}   values:{{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: PD_incorrect_namespace_region_count

  - alert: PD_pending_peer_region_count
    expr: sum( pd_regions_status{type="pending_peer_region_count"} )  > 100
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  sum( pd_regions_status{type="pending_peer_region_count"} )  > 100
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}   values:{{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: PD_pending_peer_region_count

  - alert: PD_leader_change
    expr: count( changes(pd_server_tso{type="save"}[10m]) > 0 )   >= 2
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  count( changes(pd_server_tso{type="save"}[10m]) > 0 )   >= 2
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}   values:{{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: PD_leader_change

  - alert: TiKV_space_used_more_than_80%
    expr: sum(pd_cluster_status{type="storage_size"}) / sum(pd_cluster_status{type="storage_capacity"}) * 100  > 80
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  sum(pd_cluster_status{type="storage_size"}) / sum(pd_cluster_status{type="storage_capacity"}) * 100  > 80
    annotations:
      description: 'alert: type: {{ .Values.metaType }} instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV_space_used_more_than_80%
  - alert: TiDB_schema_error
    expr: increase(tidb_session_schema_lease_error_total{type="outdated"}[15m]) > 0
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: emergency
      expr:  increase(tidb_session_schema_lease_error_total{type="outdated"}[15m]) > 0
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }} values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiDB schema error

  - alert: TiDB_tikvclient_region_err_total
    expr: increase( tidb_tikvclient_region_err_total[10m] )  > 6000
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: emergency
      expr:  increase( tidb_tikvclient_region_err_total[10m] )  > 6000
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }} values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiDB tikvclient_backoff_count error

  - alert: TiDB_domain_load_schema_total
    expr: increase( tidb_domain_load_schema_total{type="failed"}[10m] )  > 10
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: emergency
      expr:  increase( tidb_domain_load_schema_total{type="failed"}[10m] )  > 10
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }} values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiDB domain_load_schema_total error

  - alert: TiDB_monitor_keep_alive
    expr: increase(tidb_monitor_keep_alive_total[10m]) < 100
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: emergency
      expr:  increase(tidb_monitor_keep_alive_total[10m]) < 100
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }} values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiDB monitor_keep_alive error

  - alert: TiDB_server_panic_total
    expr: increase(tidb_server_panic_total[10m]) > 0
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr:  increase(tidb_server_panic_total[10m]) > 0
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }} values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiDB server panic total

  - alert: TiDB_memery_abnormal
    expr: go_memstats_heap_inuse_bytes{job="tidb"} > 1e+10
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr: go_memstats_heap_inuse_bytes{job="tidb"} > 1e+10
    annotations:
      description: 'alert: values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiDB mem heap is over 1GiB

  - alert: TiDB_query_duration
    expr: histogram_quantile(0.99, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) BY (le, instance)) > 1
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  histogram_quantile(0.99, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) BY (le, instance)) > 1
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }} values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiDB query duration 99th percentile is above 1s

  - alert: TiDB_server_event_error
    expr: increase(tidb_server_server_event{type=~"server_start|server_hang"}[15m])  > 0
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  increase(tidb_server_server_event{type=~"server_start|server_hang"}[15m])  > 0
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }} values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiDB server event error

  - alert: TiDB_tikvclient_backoff_count
    expr: increase( tidb_tikvclient_backoff_count[10m] )  > 10
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  increase( tidb_tikvclient_backoff_count[10m] )  > 10
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }} values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiDB tikvclient_backoff_count error

  - alert: TiDB_monitor_time_jump_back_error
    expr: increase(tidb_monitor_time_jump_back_total[10m])  > 0
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  increase(tidb_monitor_time_jump_back_total[10m])  > 0
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }} values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiDB monitor time_jump_back error

  - alert: TiDB_ddl_waiting_jobs
    expr: sum(tidb_ddl_waiting_jobs) > 5
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  sum(tidb_ddl_waiting_jobs) > 5
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }} values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiDB ddl waiting_jobs too much

  - alert: TiKV_memory_used_too_fast
    expr: (node_memory_MemAvailable offset 5m) - node_memory_MemAvailable > 5*1024*1024*1024
    for: 5m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: emergency
      expr: (node_memory_MemAvailable offset 5m) - node_memory_MemAvailable > 5*1024*1024*1024
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV memory used too fast

  - alert: TiKV_GC_can_not_work
    expr: sum(increase(tidb_tikvclient_gc_action_result{type="success"}[6h])) < 1
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: emergency
      expr: sum(increase(tidb_tikvclient_gc_action_result{type="success"}[6h])) < 1
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV GC can not work

  - alert: TiKV_server_report_failure_msg_total
    expr:  sum(rate(tikv_server_report_failure_msg_total{type="unreachable"}[10m])) BY (store_id) > 10
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr:  sum(rate(tikv_server_report_failure_msg_total{type="unreachable"}[10m])) BY (store_id) > 10
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV server_report_failure_msg_total error

  - alert: TiKV_channel_full_total
    expr: sum(rate(tikv_channel_full_total[10m])) BY (type, instance) > 0
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr:  sum(rate(tikv_channel_full_total[10m])) BY (type, instance) > 0
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV channel full

  - alert: TiKV_write_stall
    expr: delta( tikv_engine_write_stall[10m])  > 0
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr:  delta( tikv_engine_write_stall[10m])  > 0
    annotations:
      description: 'alert: type: {{ .Values.metaType }} instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV write stall

  - alert: TiKV_raft_log_lag
    expr: histogram_quantile(0.99, sum(rate(tikv_raftstore_log_lag_bucket[1m])) by (le, instance, job))  > 5000
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr:  histogram_quantile(0.99, sum(rate(tikv_raftstore_log_lag_bucket[1m])) by (le, instance, job))  > 5000
    annotations:
      description: 'alert: instance {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV raftstore log lag more than 5000

  - alert: TiKV_async_request_snapshot_duration_seconds
    expr: histogram_quantile(0.99, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{type="snapshot"}[1m])) by (le, instance, job,type)) > 1
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr:  histogram_quantile(0.99, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{type="snapshot"}[1m])) by (le, instance, job,type)) > 1
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV async request snapshot duration seconds more than 1s

  - alert: TiKV_async_request_write_duration_seconds
    expr: histogram_quantile(0.99, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{type="write"}[1m])) by (le, instance, job,type)) > 1
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr:  histogram_quantile(0.99, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{type="write"}[1m])) by (le, instance, job,type)) > 1
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV async request write duration seconds more than 1s

  - alert: TiKV_coprocessor_request_wait_seconds
    expr: histogram_quantile(0.9999, sum(rate(tikv_coprocessor_request_wait_seconds_bucket[1m])) by (le, instance, job,req)) > 10
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr:  histogram_quantile(0.9999, sum(rate(tikv_coprocessor_request_wait_seconds_bucket[1m])) by (le, instance, job,req)) > 10
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV coprocessor request wait seconds more than 10s

  - alert: TiKV_raftstore_thread_cpu_seconds_total
    expr: sum(rate(tikv_thread_cpu_seconds_total{name=~"raftstore_.*"}[1m])) by (job, name)  > 0.8
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr: sum(rate(tikv_thread_cpu_seconds_total{name=~"raftstore_.*"}[1m])) by (job, name)  > 0.8
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV raftstore thread CPU seconds is high

  - alert: TiKV_raft_append_log_duration_secs
    expr: histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket[1m])) by (le, instance, job)) > 1
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr: histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket[1m])) by (le, instance, job)) > 1
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV_raft_append_log_duration_secs

  - alert: TiKV_raft_apply_log_duration_secs
    expr: histogram_quantile(0.99, sum(rate(tikv_raftstore_apply_log_duration_seconds_bucket[1m])) by (le, instance, job)) > 1
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr: histogram_quantile(0.99, sum(rate(tikv_raftstore_apply_log_duration_seconds_bucket[1m])) by (le, instance, job)) > 1
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV_raft_apply_log_duration_secs

  - alert: TiKV_scheduler_latch_wait_duration_seconds
    expr: histogram_quantile(0.99, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket[1m])) by (le, instance, job,type))  > 1
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr:  histogram_quantile(0.99, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket[1m])) by (le, instance, job,type))  > 1
    annotations:
      description: 'alert: instance:
        {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV scheduler latch wait duration seconds more than 1s

  - alert: TiKV_thread_apply_worker_cpu_seconds
    expr: sum(rate(tikv_thread_cpu_seconds_total{name="apply_worker"}[1m])) by (job) > 0.9
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr:  sum(rate(tikv_thread_cpu_seconds_total{name="apply_worker"}[1m])) by (job) > 0.9
    annotations:
      description: 'alert: type: {{ .Values.metaType }} instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV thread apply worker cpu seconds is high

  - alert: TiDB_tikvclient_gc_action_fail
    expr: sum(increase(tidb_tikvclient_gc_action_result{type="fail"}[1m])) > 10
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: critical
      expr: sum(increase(tidb_tikvclient_gc_action_result{type="fail"}[1m])) > 10
    annotations:
      description: 'alert: type: {{ .Values.metaType }} instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiDB_tikvclient_gc_action_fail

  - alert: TiKV_leader_drops
    expr: delta(tikv_pd_heartbeat_tick_total{type="leader"}[30s]) < -10
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr: delta(tikv_pd_heartbeat_tick_total{type="leader"}[30s]) < -10
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values:{{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV leader drops

  - alert: TiKV_raft_process_ready_duration_secs
    expr: histogram_quantile(0.999, sum(rate(tikv_raftstore_raft_process_duration_secs_bucket{type='ready'}[1m])) by (le, instance, job,type)) > 2
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr: histogram_quantile(0.999, sum(rate(tikv_raftstore_raft_process_duration_secs_bucket{type='ready'}[1m])) by (le, instance, job,type)) > 2
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values:{{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV_raft_process_ready_duration_secs

  - alert: TiKV_raft_process_tick_duration_secs
    expr: histogram_quantile(0.999, sum(rate(tikv_raftstore_raft_process_duration_secs_bucket{type='tick'}[1m])) by (le, instance, job,type)) > 2
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr: histogram_quantile(0.999, sum(rate(tikv_raftstore_raft_process_duration_secs_bucket{type='tick'}[1m])) by (le, instance, job,type)) > 2
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values:{{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV_raft_process_tick_duration_secs

  - alert: TiKV_scheduler_context_total
    expr: abs(delta( tikv_scheduler_contex_total[5m])) > 1000
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  abs(delta( tikv_scheduler_contex_total[5m])) > 1000
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV scheduler context total

  - alert: TiKV_scheduler_command_duration_seconds
    expr: histogram_quantile(0.99, sum(rate(tikv_scheduler_command_duration_seconds_bucket[1m])) by (le, instance, job,type)  / 1000)  > 1
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  histogram_quantile(0.99, sum(rate(tikv_scheduler_command_duration_seconds_bucket[1m])) by (le, instance, job,type)  / 1000)  > 1
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV scheduler command duration seconds more than 1s

  - alert: TiKV_thread_storage_scheduler_cpu_seconds
    expr: sum(rate(tikv_thread_cpu_seconds_total{name=~"storage_schedul.*"}[1m])) by (job) > 0.8
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  sum(rate(tikv_thread_cpu_seconds_total{name=~"storage_schedul.*"}[1m])) by (job) > 0.8
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV storage scheduler cpu seconds more than 80%

  - alert: TiKV_coprocessor_outdated_request_wait_seconds
    expr: delta( tikv_coprocessor_outdated_request_wait_seconds_count[10m] )  > 0
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  delta( tikv_coprocessor_outdated_request_wait_seconds_count[10m] )  > 0
    annotations:
      description: 'alert: instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV coprocessor outdated request wait seconds

  - alert: TiKV_coprocessor_request_error
    expr: increase(tikv_coprocessor_request_error[10m]) > 100
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  increase(tikv_coprocessor_request_error[10m]) > 100
    annotations:
      description: 'alert: type: {{ .Values.metaType }} instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV coprocessor_request_error

  - alert: TiKV_coprocessor_pending_request
    expr: delta( tikv_coprocessor_pending_request[10m]) > 5000
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  delta( tikv_coprocessor_pending_request[10m]) > 5000
    annotations:
      description: 'alert: type: {{ .Values.metaType }} instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV pending {{ .Values.metaType }} request is high

  - alert: TiKV_batch_request_snapshot_nums
    expr: sum(rate(tikv_thread_cpu_seconds_total{name=~"cop_.*"}[1m])) by (job) / ( count(tikv_thread_cpu_seconds_total{name=~"cop_.*"}) *  0.9 ) / count(count(tikv_thread_cpu_seconds_total) by (instance)) > 0
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  sum(rate(tikv_thread_cpu_seconds_total{name=~"cop_.*"}[1m])) by (job) / ( count(tikv_thread_cpu_seconds_total{name=~"cop_.*"}) *  0.9 ) / count(count(tikv_thread_cpu_seconds_total) by (instance)) > 0
    annotations:
      description: 'alert: type: {{ .Values.metaType }} instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV batch request snapshot nums is high

  - alert: TiKV_pending_task
    expr: sum(tikv_worker_pending_task_total) BY (job,instance,name)  > 1000
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  sum(tikv_worker_pending_task_total) BY (job,instance,name)  > 1000
    annotations:
      description: 'alert: type: {{ .Values.metaType }} instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV pending task too much

  - alert: TiKV_low_space_and_add_region
    expr: count( (sum(tikv_store_size_bytes{type="available"}) by (job) / sum(tikv_store_size_bytes{type="capacity"}) by (job) < 0.2) and (sum(tikv_raftstore_snapshot_traffic_total{type="applying"}) by (job) > 0 ) ) > 0
    for: 1m
    labels:
      env: '{{ template "cluster.name" . }}'
      level: warning
      expr:  count( (sum(tikv_store_size_bytes{type="available"}) by (job) / sum(tikv_store_size_bytes{type="capacity"}) by (job) < 0.2) and (sum(tikv_raftstore_snapshot_traffic_total{type="applying"}) by (job) > 0 ) ) > 0
    annotations:
      description: 'alert: type: {{ .Values.metaType }} instance: {{ .Values.metaInstance }}  values: {{ .Values.metaValue }}'
      value: '{{ .Values.metaValue }}'
      summary: TiKV low_space and add_region
