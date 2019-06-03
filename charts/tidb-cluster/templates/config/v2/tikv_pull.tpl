{{- define "tikv_dashboard_v2" -}}
{
  "__requires": [
    {
      "type": "grafana",
      "id": "grafana",
      "name": "Grafana",
      "version": "4.6.3"
    },
    {
      "type": "panel",
      "id": "graph",
      "name": "Graph",
      "version": ""
    },
    {
      "type": "datasource",
      "id": "prometheus",
      "name": "Prometheus",
      "version": "1.0.0"
    },
    {
      "type": "panel",
      "id": "singlestat",
      "name": "Singlestat",
      "version": ""
    }
  ],
  "annotations": {
    "list": [
      {
        "builtIn": 1,
        "datasource": "tidb-cluster",
        "enable": true,
        "hide": true,
        "iconColor": "rgba(0, 211, 255, 1)",
        "name": "Annotations & Alerts",
        "type": "dashboard"
      }
    ]
  },
  "editable": false,
  "gnetId": null,
  "graphTooltip": 1,
  "hideControls": false,
  "id": null,
  "links": [],
  "refresh": "1m",
  "rows": [
    {
      "collapse": false,
      "height": "300",
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 5,
          "grid": {},
          "id": 56,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": null,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(tikv_engine_size_bytes{instance=~\"$instance\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Store size",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "decbytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 5,
          "grid": {},
          "id": 1706,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": null,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(tikv_store_size_bytes{instance=~\"$instance\", type=\"available\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Available size",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "decbytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 5,
          "grid": {},
          "id": 1707,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": null,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(tikv_store_size_bytes{instance=~\"$instance\", type=\"capacity\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Capacity size",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "decbytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1708,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\"}[1m])) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "CPU",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1709,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(process_resident_memory_bytes{instance=~\"$instance\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Memory",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "bytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1710,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "rate(node_disk_io_time_ms[1m]) / 1000",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} - {{`{{`}}device}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "IO utilization",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1711,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": null,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_flow_bytes{instance=~\"$instance\", db=\"kv\", type=\"wal_file_bytes\"}[1m])) by (instance)",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-write",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_flow_bytes{instance=~\"$instance\", db=\"kv\", type=~\"bytes_read|iter_bytes_read\"}[1m])) by (instance)",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-read",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "MBps",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "Bps",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1713,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": false,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": null,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_grpc_msg_duration_seconds_count{instance=~\"$instance\", type!=\"kv_gc\"}[1m])) by (instance,type)",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} - {{`{{`}}type}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "QPS",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1712,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": null,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_grpc_msg_fail_total{instance=~\"$instance\", type!=\"kv_gc\"}[1m])) by (instance)",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-grpc-msg-fail",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "sum(delta(tikv_pd_heartbeat_message_total{instance=~\"$instance\", type=\"noop\"}[1m])) by (instance) < 1",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-pd-heartbeat",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Errps",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1715,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [
            {
              "alias": "total",
              "lines": false
            }
          ],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(tikv_pd_heartbeat_tick_total{instance=~\"$instance\", type=\"leader\"}) by (instance)",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_pd_heartbeat_tick_total",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "delta(tikv_pd_heartbeat_tick_total{instance=~\"$instance\", type=\"leader\"}[30s]) < -10",
              "format": "time_series",
              "hide": true,
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Leader",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1714,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(tikv_pd_heartbeat_tick_total{instance=~\"$instance\", type=\"region\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Region",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Cluster",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": 250,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1584,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_scheduler_too_busy_total{instance=~\"$instance\"}[1m])) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "scheduler-{{`{{`}}instance}}",
              "metric": "",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_channel_full_total{instance=~\"$instance\"}[1m])) by (instance, type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "channelfull-{{`{{`}}instance}}-{{`{{`}}type}}",
              "metric": "",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_coprocessor_request_error{instance=~\"$instance\", type='full'}[1m])) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "coprocessor-{{`{{`}}instance}}",
              "metric": "",
              "refId": "C",
              "step": 4
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", type=\"write_stall_percentile99\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "stall-{{`{{`}}instance}}",
              "refId": "D"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Server is busy",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "none",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ]
        },
        {
          "alert": {
            "conditions": [
              {
                "evaluator": {
                  "params": [
                    0
                  ],
                  "type": "gt"
                },
                "operator": {
                  "type": "and"
                },
                "query": {
                  "params": [
                    "A",
                    "10s",
                    "now"
                  ]
                },
                "reducer": {
                  "params": [],
                  "type": "max"
                },
                "type": "query"
              }
            ],
            "executionErrorState": "alerting",
            "frequency": "10s",
            "handler": 1,
            "message": "TiKV server report failures",
            "name": "server report failures alert",
            "noDataState": "ok",
            "notifications": []
          },
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 18,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_server_report_failure_msg_total{instance=~\"$instance\"}[1m])) by (type,instance,store_id)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} - {{`{{`}}type}} - to - {{`{{`}}store_id}}",
              "metric": "tikv_server_raft_store_msg_total",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [
            {
              "colorMode": "critical",
              "fill": true,
              "line": true,
              "op": "gt",
              "value": 0
            }
          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "Server report failures",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1718,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_storage_engine_async_request_total{instance=~\"$instance\", status!~\"success|all\"}[1m])) by (instance, status)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-{{`{{`}}status}}",
              "metric": "",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Raftstore error",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "none",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1719,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_scheduler_stage_total{instance=~\"$instance\", stage=~\"snapshot_err|prepare_write_err\"}[1m])) by (instance, stage)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-{{`{{`}}stage}}",
              "metric": "",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scheduler error",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "none",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1720,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_coprocessor_request_error{instance=~\"$instance\"}[1m])) by (instance, reason)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-{{`{{`}}reason}}",
              "metric": "",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Coprocessor error",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "none",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1721,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_grpc_msg_fail_total{instance=~\"$instance\"}[1m])) by (instance, type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-{{`{{`}}type}}",
              "metric": "",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "gRPC message error",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "none",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1722,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [
            {
              "alias": "total",
              "lines": false
            }
          ],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(delta(tikv_pd_heartbeat_tick_total{instance=~\"$instance\", type=\"leader\"}[1m])) by (instance)",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Leader drop",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1723,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [
            {
              "alias": "total",
              "lines": false
            }
          ],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(tikv_raftstore_leader_missing{instance=~\"$instance\"}) by (instance)",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Leader missing",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Errors",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": "300",
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 34,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [
            {
              "alias": "total",
              "lines": false
            }
          ],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(tikv_pd_heartbeat_tick_total{instance=~\"$instance\", type=\"leader\"}) by (instance)",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_pd_heartbeat_tick_total",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "delta(tikv_pd_heartbeat_tick_total{instance=~\"$instance\", type=\"leader\"}[30s]) < -10",
              "format": "time_series",
              "hide": true,
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Leader",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 37,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(tikv_pd_heartbeat_tick_total{instance=~\"$instance\", type=\"region\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Region",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 3,
          "grid": {},
          "id": 33,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(tikv_engine_size_bytes{instance=~\"$instance\"}) by (type)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "CF size",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "decbytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 5,
          "grid": {},
          "id": 1705,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "current",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 0,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": true,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(tikv_engine_size_bytes{instance=~\"$instance\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Store size",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "decbytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "alert": {
            "conditions": [
              {
                "evaluator": {
                  "params": [
                    0
                  ],
                  "type": "gt"
                },
                "operator": {
                  "type": "and"
                },
                "query": {
                  "datasourceId": 1,
                  "model": {
                    "expr": "sum(rate(tikv_channel_full_total{instance=~\"$instance\"}[1m])) by (instance, type)",
                    "intervalFactor": 2,
                    "legendFormat": "{{`{{`}}instance}} - {{`{{`}}type}}",
                    "metric": "",
                    "refId": "A",
                    "step": 10
                  },
                  "params": [
                    "A",
                    "10s",
                    "now"
                  ]
                },
                "reducer": {
                  "params": [],
                  "type": "avg"
                },
                "type": "query"
              }
            ],
            "executionErrorState": "alerting",
            "frequency": "10s",
            "handler": 1,
            "message": "TiKV channel full",
            "name": "TiKV channel full alert",
            "noDataState": "ok",
            "notifications": []
          },
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 3,
          "grid": {},
          "id": 22,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_channel_full_total{instance=~\"$instance\"}[1m])) by (instance, type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} - {{`{{`}}type}}",
              "metric": "",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [
            {
              "colorMode": "critical",
              "fill": true,
              "line": true,
              "op": "gt",
              "value": 0
            }
          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "Channel full",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1717,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_server_report_failure_msg_total{instance=~\"$instance\"}[1m])) by (type,instance,store_id)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} - {{`{{`}}type}} - to - {{`{{`}}store_id}}",
              "metric": "tikv_server_raft_store_msg_total",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Server report failures",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 57,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "connected",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_region_written_keys_sum{instance=~\"$instance\"}[1m])) by (instance) / sum(rate(tikv_region_written_keys_count{instance=~\"$instance\"}[1m])) by (instance)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_region_written_keys_bucket",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Region average written keys",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 58,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "connected",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_region_written_bytes_sum{instance=~\"$instance\"}[1m])) by (instance) / sum(rate(tikv_region_written_bytes_count{instance=~\"$instance\"}[1m])) by (instance)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_regi",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Region average written bytes",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "bytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 75,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_region_written_keys_count{instance=~\"$instance\"}[1m])) by (instance)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_region_written_keys_bucket",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Active written leaders",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ]
        },
        {
          "alert": {
            "conditions": [
              {
                "evaluator": {
                  "params": [
                    1073741824
                  ],
                  "type": "gt"
                },
                "operator": {
                  "type": "and"
                },
                "query": {
                  "params": [
                    "B",
                    "1m",
                    "now"
                  ]
                },
                "reducer": {
                  "params": [],
                  "type": "avg"
                },
                "type": "query"
              }
            ],
            "executionErrorState": "alerting",
            "frequency": "60s",
            "handler": 1,
            "name": "approximate region size alert",
            "noDataState": "no_data",
            "notifications": []
          },
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1481,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_region_size_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_raftstore_region_size_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "metric": "",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_raftstore_region_size_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_raftstore_region_size_count{instance=~\"$instance\"}[1m])) ",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "metric": "",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Approximate Region size",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 2,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "bytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Server",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": "250px",
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 31,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_apply_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": " 99%",
              "metric": "",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_raftstore_apply_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_raftstore_apply_log_duration_seconds_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_raftstore_apply_log_duration_seconds_count{instance=~\"$instance\"}[1m])) ",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Apply log duration",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 32,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_apply_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le, instance))",
              "intervalFactor": 2,
              "legendFormat": " {{`{{`}}instance}}",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Apply log duration per server",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 39,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": " 99%",
              "metric": "",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_raftstore_append_log_duration_seconds_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_raftstore_append_log_duration_seconds_count{instance=~\"$instance\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Append log duration",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 40,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le, instance))",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} ",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Append log duration per server",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Raft IO",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": "300",
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 5,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_raftstore_raft_ready_handled_total{instance=~\"$instance\"}[1m])) by (type)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "metric": "tikv_raftstore_raft_ready_handled_total",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_raftstore_raft_process_duration_secs_count{instance=~\"$instance\", type=\"ready\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "count",
              "refId": "B",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Ready handled",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 118,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_raft_process_duration_secs_bucket{instance=~\"$instance\", type='ready'}[1m])) by (le, instance))",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [
            {
              "colorMode": "critical",
              "fill": true,
              "line": true,
              "op": "gt",
              "value": 1
            }
          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "Process ready duration per server",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1165,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_raft_process_duration_secs_bucket{instance=~\"$instance\", type='tick'}[1m])) by (le, instance))",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [
            {
              "colorMode": "critical",
              "fill": true,
              "line": true,
              "op": "gt",
              "value": 1
            }
          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "Process tick duration per server",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 123,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_event_duration_bucket{instance=~\"$instance\"}[1m])) by (le, type))",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [
            {
              "colorMode": "critical",
              "fill": true,
              "line": true,
              "op": "gt",
              "value": 1
            }
          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "0.99 Duration of raft store events",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Raft process",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": 250,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 1615,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_raftstore_raft_sent_message_total{instance=~\"$instance\"}[1m])) by (instance)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Sent messages per server",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 1616,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_server_raft_message_flush_total{instance=~\"$instance\"}[1m])) by (instance)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_server_raft_message_flush_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Flush messages per server",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 106,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_server_raft_message_recv_total{instance=~\"$instance\"}[1m])) by (instance)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Receive messages per server",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 11,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_raftstore_raft_sent_message_total{instance=~\"$instance\"}[1m])) by (type)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Messages",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 25,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_raftstore_raft_sent_message_total{instance=~\"$instance\", type=\"vote\"}[1m])) by (instance)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Vote",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 1309,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_raftstore_raft_dropped_message_total{instance=~\"$instance\"}[1m])) by (type)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Raft dropped messages",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Raft message",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": 250,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 108,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_apply_proposal_bucket{instance=~\"$instance\"}[1m])) by (le, instance))",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Raft proposals per ready",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 7,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_raftstore_proposal_total{instance=~\"$instance\", type=~\"local_read|normal|read_index\"}[1m])) by (type)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "metric": "tikv_raftstore_proposal_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Raft read/write proposals",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 119,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_raftstore_proposal_total{instance=~\"$instance\", type=~\"local_read|read_index\"}[1m])) by (instance)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Raft read proposals per server",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 120,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_raftstore_proposal_total{instance=~\"$instance\", type=\"normal\"}[1m])) by (instance)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_raftstore_proposal_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Raft write proposals per server",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 41,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_request_wait_time_duration_secs_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "metric": "tikv_raftstore_request_wait_time_duration_secs_bucket",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_raftstore_request_wait_time_duration_secs_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_raftstore_request_wait_time_duration_secs_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_raftstore_request_wait_time_duration_secs_count{instance=~\"$instance\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Propose wait duration",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 42,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_raftstore_request_wait_time_duration_secs_bucket{instance=~\"$instance\"}[1m])) by (le, instance))",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Propose wait duration per server",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "fill": 1,
          "id": 1975,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": true,
            "min": true,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(rate(tikv_raftstore_propose_log_size_sum{instance=~\"$instance\"}[1m])) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Raft log speed",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "decimals": null,
              "format": "short",
              "label": "bytes/s",
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Raft propose",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": 250,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 76,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_raftstore_proposal_total{instance=~\"$instance\", type=~\"conf_change|transfer_leader\"}[1m])) by (type)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "metric": "tikv_raftstore_proposal_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Admin proposals",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 77,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_raftstore_admin_cmd_total{instance=~\"$instance\", status=\"success\", type!=\"compact\"}[1m]))  by (type)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "metric": "tikv_raftstore_admin_cmd_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Admin apply",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 70,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_raftstore_check_split_total{instance=~\"$instance\", type!=\"ignore\"}[1m])) by (type)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "metric": "tikv_raftstore_check_split_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Check split",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 71,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.9999, sum(rate(tikv_raftstore_check_split_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le, instance))",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_raftstore_check_split_duration_seconds_bucket",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "99.99% Check split duration",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Raft admin",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": 250,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "fill": 1,
          "id": 2292,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [
            {
              "alias": "/.*-total/i",
              "yaxis": 2
            }
          ],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_raftstore_local_read_reject_total{instance=~\"$instance\"}[1m])) by (instance, reason)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-reject-by-{{`{{`}}reason}}",
              "refId": "A"
            },
            {
              "expr": "sum(rate(tikv_raftstore_local_read_batch_requests_total_sum{instance=~\"$instance\"}[1m])) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-total",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Local reader requests",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "fill": 1,
          "id": 2293,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_local_read_requests_wait_duration_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "A"
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_raftstore_local_read_requests_wait_duration_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "B"
            },
            {
              "expr": "sum(rate(tikv_raftstore_local_read_requests_wait_duration_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_raftstore_local_read_requests_wait_duration_count{instance=~\"$instance\"}[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Local read requsets duration",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "fill": 1,
          "id": 2294,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_local_read_batch_requests_total_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "A"
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_raftstore_local_read_batch_requests_total_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "B"
            },
            {
              "expr": "sum(rate(tikv_raftstore_local_read_batch_requests_total_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_raftstore_local_read_batch_requests_total_count{instance=~\"$instance\"}[1m])) ",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Local read requests batch size",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Local reader",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": "300",
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 2,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_storage_command_total{instance=~\"$instance\"}[1m])) by (type)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Storage command total",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 10,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 8,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_storage_engine_async_request_total{instance=~\"$instance\", status!~\"all|success\"}[1m])) by (status)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}status}}",
              "metric": "tikv_raftstore_raft_process_duration_secs_bucket",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Storage async request error",
          "tooltip": {
            "msResolution": true,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 15,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{instance=~\"$instance\", type=\"snapshot\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{instance=~\"$instance\", type=\"snapshot\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_storage_engine_async_request_duration_seconds_sum{instance=~\"$instance\", type=\"snapshot\"}[1m])) / sum(rate(tikv_storage_engine_async_request_duration_seconds_count{instance=~\"$instance\", type=\"snapshot\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Storage async snapshot duration",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 109,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{instance=~\"$instance\", type=\"write\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{instance=~\"$instance\", type=\"write\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_storage_engine_async_request_duration_seconds_sum{instance=~\"$instance\", type=\"write\"}[1m])) / sum(rate(tikv_storage_engine_async_request_duration_seconds_count{instance=~\"$instance\", type=\"write\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Storage async write duration",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 1310,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [
            {
              "alias": "raft-95%",
              "yaxis": 2
            }
          ],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_storage_batch_commands_total_bucket{instance=~\"$instance\"}[30s])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_storage_batch_commands_total_bucket{instance=~\"$instance\"}[30s])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_storage_batch_commands_total_sum{instance=~\"$instance\"}[30s])) / sum(rate(tikv_storage_batch_commands_total_count{instance=~\"$instance\"}[30s]))",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "C",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_raftstore_batch_snapshot_commands_total_bucket{instance=~\"$instance\"}[30s])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "raft-95%",
              "refId": "D",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "storage async batch snapshot",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": "Storage Batch Size",
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": "Raftstore Batch Size",
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Storage",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": "300",
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "height": "400",
          "id": 167,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "minSpan": 12,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 12,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_scheduler_too_busy_total{instance=~\"$instance\"}[1m])) by (stage)",
              "intervalFactor": 2,
              "legendFormat": "busy",
              "refId": "A",
              "step": 20
            },
            {
              "expr": "sum(rate(tikv_scheduler_stage_total{instance=~\"$instance\"}[1m])) by (stage)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}stage}}",
              "refId": "B",
              "step": 20
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scheduler stage total",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "height": "",
          "id": 1,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "minSpan": 6,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_scheduler_commands_pri_total{instance=~\"$instance\"}[1m])) by (priority)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}priority}}",
              "metric": "",
              "refId": "A",
              "step": 40
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scheduler priority commands",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "transparent": false,
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "alert": {
            "conditions": [
              {
                "evaluator": {
                  "params": [
                    300
                  ],
                  "type": "gt"
                },
                "operator": {
                  "type": "and"
                },
                "query": {
                  "params": [
                    "A",
                    "5m",
                    "now"
                  ]
                },
                "reducer": {
                  "params": [],
                  "type": "avg"
                },
                "type": "query"
              }
            ],
            "executionErrorState": "alerting",
            "frequency": "120s",
            "handler": 1,
            "message": "TiKV scheduler context total",
            "name": "scheduler pending commands alert",
            "noDataState": "ok",
            "notifications": []
          },
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "height": "",
          "id": 193,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "minSpan": 6,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(tikv_scheduler_contex_total{instance=~\"$instance\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "",
              "refId": "A",
              "step": 40
            }
          ],
          "thresholds": [
            {
              "colorMode": "critical",
              "fill": true,
              "line": true,
              "op": "gt",
              "value": 300
            }
          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scheduler pending commands",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "transparent": false,
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Scheduler",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": "300",
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "height": "400",
          "id": 168,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "minSpan": 12,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 12,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_scheduler_too_busy_total{instance=~\"$instance\", type=\"$command\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "busy",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_scheduler_stage_total{instance=~\"$instance\", type=\"$command\"}[1m])) by (stage)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}stage}}",
              "refId": "B",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scheduler stage total",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 3,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "minSpan": null,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_scheduler_command_duration_seconds_bucket{instance=~\"$instance\", type=\"$command\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "metric": "",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_scheduler_command_duration_seconds_bucket{instance=~\"$instance\", type=\"$command\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_scheduler_command_duration_seconds_sum{instance=~\"$instance\", type=\"$command\"}[1m])) / sum(rate(tikv_scheduler_command_duration_seconds_count{instance=~\"$instance\", type=\"$command\"}[1m])) ",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "metric": "",
              "refId": "C",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scheduler command duration",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 194,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "minSpan": null,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{instance=~\"$instance\", type=\"$command\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "metric": "",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{instance=~\"$instance\", type=\"$command\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_scheduler_latch_wait_duration_seconds_sum{instance=~\"$instance\", type=\"$command\"}[1m])) / sum(rate(tikv_scheduler_latch_wait_duration_seconds_count{instance=~\"$instance\", type=\"$command\"}[1m])) ",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "metric": "",
              "refId": "C",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scheduler latch wait duration",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 195,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "minSpan": null,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_scheduler_kv_command_key_read_bucket{instance=~\"$instance\", type=\"$command\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "metric": "kv_command_key",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_scheduler_kv_command_key_read_bucket{instance=~\"$instance\", type=\"$command\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_scheduler_kv_command_key_read_sum{instance=~\"$instance\", type=\"$command\"}[1m])) / sum(rate(tikv_scheduler_kv_command_key_read_count{instance=~\"$instance\", type=\"$command\"}[1m])) ",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "metric": "",
              "refId": "C",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scheduler keys read",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 373,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "minSpan": null,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_scheduler_kv_command_key_write_bucket{instance=~\"$instance\", type=\"$command\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "metric": "kv_command_key",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_scheduler_kv_command_key_write_bucket{instance=~\"$instance\", type=\"$command\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_scheduler_kv_command_key_write_sum{instance=~\"$instance\", type=\"$command\"}[1m])) / sum(rate(tikv_scheduler_kv_command_key_write_count{instance=~\"$instance\", type=\"$command\"}[1m])) ",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "metric": "",
              "refId": "C",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scheduler keys written",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 560,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "minSpan": null,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_scheduler_kv_scan_details{instance=~\"$instance\", req=\"$command\"}[1m])) by (tag)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}tag}}",
              "metric": "",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scheduler scan details",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 675,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "minSpan": null,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_scheduler_kv_scan_details{instance=~\"$instance\", req=\"$command\", cf=\"lock\"}[1m])) by (tag)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}tag}}",
              "metric": "",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scheduler scan details [lock]",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 829,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "minSpan": null,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_scheduler_kv_scan_details{instance=~\"$instance\", req=\"$command\", cf=\"write\"}[1m])) by (tag)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}tag}}",
              "metric": "",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scheduler scan details [write]",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 830,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "minSpan": null,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_scheduler_kv_scan_details{instance=~\"$instance\", req=\"$command\", cf=\"default\"}[1m])) by (tag)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}tag}}",
              "metric": "",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scheduler scan details [default]",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": "command",
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Scheduler - $command",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": "250px",
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 5,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 16,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": null,
            "sortDesc": null,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.9999, sum(rate(tikv_coprocessor_request_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le,req))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-99.99%",
              "refId": "E"
            },
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_coprocessor_request_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le,req))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-99%",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_coprocessor_request_duration_seconds_bucket[1m])) by (le,req))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": " sum(rate(tikv_coprocessor_request_duration_seconds_sum{instance=~\"$instance\"}[1m])) by (req) / sum(rate(tikv_coprocessor_request_duration_seconds_count{instance=~\"$instance\"}[1m])) by (req)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-avg",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Request duration",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 5,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 111,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": null,
            "sortDesc": null,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.9999, sum(rate(tikv_coprocessor_request_wait_seconds_bucket{instance=~\"$instance\"}[1m])) by (le,req))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-99.99%",
              "refId": "D"
            },
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_coprocessor_request_wait_seconds_bucket{instance=~\"$instance\"}[1m])) by (le,req))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-99%",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_coprocessor_request_wait_seconds_bucket[1m])) by (le,req))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": " sum(rate(tikv_coprocessor_request_wait_seconds_sum{instance=~\"$instance\"}[1m])) by (req) / sum(rate(tikv_coprocessor_request_wait_seconds_count{instance=~\"$instance\"}[1m])) by (req)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-avg",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Wait duration",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 5,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 113,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": null,
            "sortDesc": null,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.9999, sum(rate(tikv_coprocessor_request_handle_seconds_bucket{instance=~\"$instance\"}[1m])) by (le,req))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-99.99%",
              "refId": "E"
            },
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_coprocessor_request_handle_seconds_bucket{instance=~\"$instance\"}[1m])) by (le,req))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-99%",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_coprocessor_request_handle_seconds_bucket{instance=~\"$instance\"}[1m])) by (le,req))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": " sum(rate(tikv_coprocessor_request_handle_seconds_sum{instance=~\"$instance\"}[1m])) by (req) / sum(rate(tikv_coprocessor_request_handle_seconds_count{instance=~\"$instance\"}[1m])) by (req)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-avg",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Handle duration",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 5,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 115,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": null,
            "sortDesc": null,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_coprocessor_request_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le, instance, req))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-{{`{{`}}req}}",
              "metric": "tikv_coprocessor_request_duration_seconds_bucket",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "95% Request duration by store",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 5,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 116,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": null,
            "sortDesc": null,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_coprocessor_request_wait_seconds_bucket{instance=~\"$instance\"}[1m])) by (le, instance,req))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-{{`{{`}}req}}",
              "refId": "B",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "95% Wait duration by store",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 5,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 117,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": null,
            "sortDesc": null,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_coprocessor_request_handle_seconds_bucket{instance=~\"$instance\"}[1m])) by (le, instance,req))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-{{`{{`}}req}}",
              "refId": "B",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "95% Handle duration by store",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 2,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 74,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_coprocessor_request_error{instance=~\"$instance\"}[1m])) by (reason)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}reason}}",
              "metric": "tikv_coprocessor_request_error",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Request errors",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "transparent": false,
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 551,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_coprocessor_executor_count{instance=~\"$instance\"}[1m])) by (type)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "metric": "",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "DAG executors",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 52,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.9999, avg(rate(tikv_coprocessor_scan_keys_bucket{instance=~\"$instance\"}[1m])) by (le, req))  ",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-99.99%",
              "refId": "D"
            },
            {
              "expr": "histogram_quantile(0.99, avg(rate(tikv_coprocessor_scan_keys_bucket{instance=~\"$instance\"}[1m])) by (le, req))  ",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-99%",
              "metric": "tikv_coprocessor_scan_keys_bucket",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "histogram_quantile(0.95, avg(rate(tikv_coprocessor_scan_keys_bucket{instance=~\"$instance\"}[1m])) by (le, req))  ",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-95%",
              "metric": "tikv_coprocessor_scan_keys_bucket",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "histogram_quantile(0.90, avg(rate(tikv_coprocessor_scan_keys_bucket{instance=~\"$instance\"}[1m])) by (le, req))  ",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-90%",
              "metric": "tikv_coprocessor_scan_keys_bucket",
              "refId": "C",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scan keys",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 552,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_coprocessor_scan_details{instance=~\"$instance\"}[1m])) by (tag,req)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}req}}-{{`{{`}}tag}}",
              "metric": "scan_details",
              "refId": "B",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scan details",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 122,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "repeat": null,
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_coprocessor_scan_details{instance=~\"$instance\", req=\"select\"}[1m])) by (tag,cf)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}cf}}-{{`{{`}}tag}}",
              "metric": "scan_details",
              "refId": "B",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Table Scan - Details by CF",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 554,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "repeat": "cf",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_coprocessor_scan_details{instance=~\"$instance\", req=\"index\"}[1m])) by (tag,cf)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}cf}}-{{`{{`}}tag}}",
              "metric": "scan_details",
              "refId": "B",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Index Scan - Details by CF",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "description": "RocksDB Internal Statistics for Table Scan Requests",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 2118,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [
            {
              "alias": "key_skipped",
              "yaxis": 2
            }
          ],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_coprocessor_rocksdb_perf{instance=~\"$instance\", req=\"select\",metric=\"internal_delete_skipped_count\"}[1m])) by (metric)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "delete_skipped",
              "metric": "scan_details",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_coprocessor_rocksdb_perf{instance=~\"$instance\", req=\"select\",metric=\"internal_key_skipped_count\"}[1m])) by (metric)",
              "format": "time_series",
              "instant": false,
              "intervalFactor": 2,
              "legendFormat": "key_skipped",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Table Scan - Perf Statistics",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "decimals": null,
              "format": "short",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "description": "RocksDB Internal Statistics for Index Scan Requests",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 2119,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [
            {
              "alias": "key_skipped",
              "yaxis": 2
            }
          ],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_coprocessor_rocksdb_perf{instance=~\"$instance\", req=\"index\",metric=\"internal_delete_skipped_count\"}[1m])) by (metric)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "delete_skipped",
              "metric": "scan_details",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_coprocessor_rocksdb_perf{instance=~\"$instance\", req=\"index\",metric=\"internal_key_skipped_count\"}[1m])) by (metric)",
              "format": "time_series",
              "instant": false,
              "intervalFactor": 2,
              "legendFormat": "key_skipped",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Index Scan - Perf Statistics",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "decimals": null,
              "format": "short",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Coprocessor",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": "250px",
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 26,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(1.0, sum(rate(tikv_storage_mvcc_versions_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": " max",
              "metric": "",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_storage_mvcc_versions_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "metric": "",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_storage_mvcc_versions_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": " 95%",
              "metric": "",
              "refId": "C",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_storage_mvcc_versions_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_storage_mvcc_versions_count{instance=~\"$instance\"}[1m])) ",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "metric": "",
              "refId": "D",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "MVCC versions",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 2,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 559,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(1.0, sum(rate(tikv_storage_mvcc_gc_delete_versions_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": " max",
              "metric": "",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_storage_mvcc_gc_delete_versions_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "metric": "",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_storage_mvcc_gc_delete_versions_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": " 95%",
              "metric": "",
              "refId": "C",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_storage_mvcc_gc_delete_versions_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_storage_mvcc_gc_delete_versions_count{instance=~\"$instance\"}[1m])) ",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "metric": "",
              "refId": "D",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "MVCC delete versions",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 2,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 121,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_gcworker_gc_tasks_vec{instance=~\"$instance\"}[1m])) by (task)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "total-{{`{{`}}task}}",
              "metric": "tikv_storage_command_total",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_storage_gc_skipped_counter{instance=~\"$instance\"}[1m])) by (task)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "skipped-{{`{{`}}task}}",
              "metric": "tikv_storage_gc_skipped_counter",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_gcworker_gc_task_fail_vec{instance=~\"$instance\"}[1m])) by (task)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "failed-{{`{{`}}task}}",
              "refId": "C"
            },
            {
              "expr": "sum(rate(tikv_gc_worker_too_busy{instance=~\"$instance\"}[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "refId": "D"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "GC tasks",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 2224,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(1, sum(rate(tikv_gcworker_gc_task_duration_vec_bucket{instance=~\"$instance\"}[1m])) by (le, task))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "max-{{`{{`}}task}}",
              "metric": "tikv_storage_command_total",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_gcworker_gc_task_duration_vec_bucket{instance=~\"$instance\"}[1m])) by (le, task))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99%-{{`{{`}}task}}",
              "metric": "tikv_storage_gc_skipped_counter",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_gcworker_gc_task_duration_vec_bucket{instance=~\"$instance\"}[1m])) by (le, task))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%-{{`{{`}}task}}",
              "refId": "C"
            },
            {
              "expr": "sum(rate(tikv_gcworker_gc_task_duration_vec_sum{instance=~\"$instance\"}[1m])) by (task) / sum(rate(tikv_gcworker_gc_task_duration_vec_count{instance=~\"$instance\"}[1m])) by (task)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "average-{{`{{`}}task}}",
              "refId": "D"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "GC tasks duration",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 2225,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_gcworker_gc_keys{instance=~\"$instance\", cf=\"write\"}[1m])) by (tag)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}tag}}",
              "metric": "tikv_storage_command_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "GC keys (write CF)",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 2,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 967,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tidb_tikvclient_gc_action_result_total[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "metric": "",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "TiDB GC actions result",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 2,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 2,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 966,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tidb_tikvclient_gc_worker_actions_total[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "metric": "",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "TiDB GC worker actions",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 2,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 969,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": null,
            "sortDesc": null,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(1.0, sum(rate(tidb_tikvclient_gc_seconds_bucket[1m])) by (instance, le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 40
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "TiDB GC seconds",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 2,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 968,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 2,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tidb_tikvclient_gc_failure_total[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "metric": "",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "TiDB GC failure",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 2,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "cacheTimeout": null,
          "colorBackground": false,
          "colorValue": false,
          "colors": [
            "rgba(245, 54, 54, 0.9)",
            "rgba(237, 129, 40, 0.89)",
            "rgba(50, 172, 45, 0.97)"
          ],
          "datasource": "tidb-cluster",
          "decimals": 0,
          "editable": false,
          "error": false,
          "format": "s",
          "gauge": {
            "maxValue": 100,
            "minValue": 0,
            "show": false,
            "thresholdLabels": false,
            "thresholdMarkers": true
          },
          "id": 27,
          "interval": null,
          "links": [],
          "mappingType": 1,
          "mappingTypes": [
            {
              "name": "value to text",
              "value": 1
            },
            {
              "name": "range to text",
              "value": 2
            }
          ],
          "maxDataPoints": 100,
          "nullPointMode": "null",
          "nullText": null,
          "postfix": "",
          "postfixFontSize": "50%",
          "prefix": "",
          "prefixFontSize": "50%",
          "rangeMaps": [
            {
              "from": "null",
              "text": "N/A",
              "to": "null"
            }
          ],
          "span": 3,
          "sparkline": {
            "fillColor": "rgba(31, 118, 189, 0.18)",
            "full": false,
            "lineColor": "rgb(31, 120, 193)",
            "show": false
          },
          "tableColumn": "",
          "targets": [
            {
              "expr": "max(tidb_tikvclient_gc_config{type=\"tikv_gc_life_time\"})",
              "interval": "",
              "intervalFactor": 2,
              "refId": "A",
              "step": 60
            }
          ],
          "thresholds": "",
          "title": "GC lifetime",
          "type": "singlestat",
          "valueFontSize": "80%",
          "valueMaps": [
            {
              "op": "=",
              "text": "N/A",
              "value": "null"
            }
          ],
          "valueName": "current"
        },
        {
          "cacheTimeout": null,
          "colorBackground": false,
          "colorValue": false,
          "colors": [
            "rgba(245, 54, 54, 0.9)",
            "rgba(237, 129, 40, 0.89)",
            "rgba(50, 172, 45, 0.97)"
          ],
          "datasource": "tidb-cluster",
          "decimals": 0,
          "editable": false,
          "error": false,
          "format": "s",
          "gauge": {
            "maxValue": 100,
            "minValue": 0,
            "show": false,
            "thresholdLabels": false,
            "thresholdMarkers": true
          },
          "id": 28,
          "interval": null,
          "links": [],
          "mappingType": 1,
          "mappingTypes": [
            {
              "name": "value to text",
              "value": 1
            },
            {
              "name": "range to text",
              "value": 2
            }
          ],
          "maxDataPoints": 100,
          "nullPointMode": "null",
          "nullText": null,
          "postfix": "",
          "postfixFontSize": "50%",
          "prefix": "",
          "prefixFontSize": "50%",
          "rangeMaps": [
            {
              "from": "null",
              "text": "N/A",
              "to": "null"
            }
          ],
          "span": 3,
          "sparkline": {
            "fillColor": "rgba(31, 118, 189, 0.18)",
            "full": false,
            "lineColor": "rgb(31, 120, 193)",
            "show": false
          },
          "tableColumn": "",
          "targets": [
            {
              "expr": "max(tidb_tikvclient_gc_config{type=\"tikv_gc_run_interval\"})",
              "intervalFactor": 2,
              "refId": "A",
              "step": 60
            }
          ],
          "thresholds": "",
          "title": "GC interval",
          "type": "singlestat",
          "valueFontSize": "80%",
          "valueMaps": [
            {
              "op": "=",
              "text": "N/A",
              "value": "null"
            }
          ],
          "valueName": "current"
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "GC",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": "250px",
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "datasource": "tidb-cluster",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 35,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(delta(tikv_raftstore_raft_sent_message_total{instance=~\"$instance\", type=\"snapshot\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": " ",
              "refId": "A",
              "step": 60
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Rate snapshot message",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "opm",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "datasource": "tidb-cluster",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 36,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_server_send_snapshot_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "send",
              "refId": "A",
              "step": 60
            },
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_snapshot_duration_seconds_bucket{instance=~\"$instance\", type=\"apply\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "apply",
              "refId": "B",
              "step": 60
            },
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_snapshot_duration_seconds_bucket{instance=~\"$instance\", type=\"generate\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "generate",
              "refId": "C",
              "step": 60
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "99% Handle snapshot duration",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "datasource": "tidb-cluster",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 38,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "span": 4,
          "stack": false,
          "steppedLine": true,
          "targets": [
            {
              "expr": "sum(tikv_raftstore_snapshot_traffic_total{instance=~\"$instance\"}) by (type)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "metric": "",
              "refId": "A",
              "step": 60
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Snapshot state count",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "datasource": "tidb-cluster",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 44,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.9999, sum(rate(tikv_snapshot_size_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "size",
              "metric": "tikv_snapshot_size_bucket",
              "refId": "A",
              "step": 40
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "99.99% Snapshot size",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "bytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "datasource": "tidb-cluster",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 43,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.9999, sum(rate(tikv_snapshot_kv_count_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "intervalFactor": 2,
              "legendFormat": "count",
              "metric": "tikv_snapshot_kv_count_bucket",
              "refId": "A",
              "step": 40
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "99.99% Snapshot KV count",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Snapshot",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": "300",
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 59,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 400,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_worker_handled_task_total{instance=~\"$instance\"}[1m])) by (name)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}name}}",
              "metric": "tikv_pd_heartbeat_tick_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Worker handled tasks",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 1395,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 400,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_worker_pending_task_total{instance=~\"$instance\"}[1m])) by (name)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}name}}",
              "metric": "tikv_pd_heartbeat_tick_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Worker pending tasks",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 1876,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 400,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_futurepool_handled_task_total{instance=~\"$instance\"}[1m])) by (name)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}name}}",
              "metric": "tikv_pd_heartbeat_tick_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "FuturePool handled tasks",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 1877,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 400,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_futurepool_pending_task_total{instance=~\"$instance\"}[1m])) by (name)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}name}}",
              "metric": "tikv_pd_heartbeat_tick_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "FuturePool pending tasks",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "alert": {
            "conditions": [
              {
                "evaluator": {
                  "params": [
                    2
                  ],
                  "type": "gt"
                },
                "operator": {
                  "type": "and"
                },
                "query": {
                  "params": [
                    "B",
                    "5m",
                    "now"
                  ]
                },
                "reducer": {
                  "params": [],
                  "type": "avg"
                },
                "type": "query"
              }
            ],
            "executionErrorState": "alerting",
            "frequency": "10s",
            "handler": 1,
            "message": "coprocessor pending requests",
            "name": "coprocessor pending requests alert",
            "noDataState": "ok",
            "notifications": []
          },
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 2,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 550,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_coprocessor_pending_request{instance=~\"$instance\"}[1m])) by (req, priority)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}} req }} - {{`{{`}}priority}}",
              "metric": "tikv_coprocessor_request_error",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_coprocessor_pending_request{instance=~\"$instance\"}[1m])) by (type, instance)",
              "format": "time_series",
              "hide": true,
              "intervalFactor": 2,
              "refId": "B"
            }
          ],
          "thresholds": [
            {
              "colorMode": "critical",
              "fill": true,
              "line": true,
              "op": "gt",
              "value": 2
            }
          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "coprocessor pending requests",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Task",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": 250,
      "panels": [
        {
          "alert": {
            "conditions": [
              {
                "evaluator": {
                  "params": [
                    0.8
                  ],
                  "type": "gt"
                },
                "operator": {
                  "type": "and"
                },
                "query": {
                  "datasourceId": 1,
                  "model": {
                    "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"raftstore_.*\"}[1m])) by (instance, name)",
                    "intervalFactor": 2,
                    "legendFormat": "{{`{{`}}instance}}",
                    "metric": "tikv_thread_cpu_seconds_total",
                    "refId": "A",
                    "step": 20
                  },
                  "params": [
                    "A",
                    "1m",
                    "now"
                  ]
                },
                "reducer": {
                  "params": [],
                  "type": "max"
                },
                "type": "query"
              }
            ],
            "executionErrorState": "alerting",
            "frequency": "60s",
            "handler": 1,
            "message": "TiKV raftstore thread CPU usage is high",
            "name": "TiKV raft store CPU alert",
            "noDataState": "ok",
            "notifications": []
          },
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 61,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"raftstore_.*\"}[1m])) by (instance, name)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_thread_cpu_seconds_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [
            {
              "colorMode": "critical",
              "fill": true,
              "line": true,
              "op": "gt",
              "value": 0.8
            }
          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "Raft store CPU",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 79,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"apply_worker\"}[1m])) by (instance)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_thread_cpu_seconds_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Async apply CPU",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 2291,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"local_reader\"}[1m])) by (instance, name)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_thread_cpu_seconds_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Local reader CPU",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 63,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"storage_schedul.*\"}[1m])) by (instance)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_thread_cpu_seconds_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scheduler CPU",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 64,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"sched_.*\"}[1m])) by (instance)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_thread_cpu_seconds_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scheduler worker CPU",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 1908,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"store_read.*\"}[1m])) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_thread_cpu_seconds_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Storage ReadPool CPU",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 78,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"cop_.*\"}[1m])) by (instance)",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Coprocessor CPU",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "cumulative"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 67,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"snapshot_worker\"}[1m])) by (instance)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_thread_cpu_seconds_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Snapshot worker CPU",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 68,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"split_check\"}[1m])) by (instance)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_thread_cpu_seconds_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Split check CPU",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 69,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": null,
            "sortDesc": null,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"rocksdb.*\"}[1m])) by (instance)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_thread_cpu_seconds_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [
            {
              "colorMode": "warning",
              "fill": true,
              "line": true,
              "op": "gt",
              "value": 1
            },
            {
              "colorMode": "critical",
              "fill": true,
              "line": true,
              "op": "gt",
              "value": 4
            }
          ],
          "timeFrom": null,
          "timeShift": null,
          "title": "RocksDB CPU",
          "tooltip": {
            "msResolution": false,
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 105,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"grpc.*\"}[1m])) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "gRPC poll CPU",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Thread CPU",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": 250,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 2108,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "current",
            "sortDesc": false,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 2,
          "points": true,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(tikv_threads_state{instance=~\"$instance\"}) by (instance, state)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{`{{`}}instance}}-{{`{{`}}state}}",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "sum(tikv_threads_state{instance=~\"$instance\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-total",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Threads state",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "none",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 2258,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 2,
          "points": true,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_threads_io_bytes_total{instance=~\"$instance\"}[30s])) by (name, io) > 1024",
              "format": "time_series",
              "hide": false,
              "interval": "",
              "intervalFactor": 1,
              "legendFormat": "{{`{{`}}name}}-{{`{{`}}io}}",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Threads IO",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "Bps",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Threads",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": "300",
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 138,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_memtable_efficiency{instance=~\"$instance\", db=\"$db\", type=\"memtable_hit\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "memtable",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_cache_efficiency{instance=~\"$instance\", db=\"$db\", type=~\"block_cache_data_hit|block_cache_filter_hit\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "block_cache",
              "metric": "",
              "refId": "E",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_get_served{instance=~\"$instance\", db=\"$db\", type=\"get_hit_l0\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "l0",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_get_served{instance=~\"$instance\", db=\"$db\", type=\"get_hit_l1\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "l1",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_get_served{instance=~\"$instance\", db=\"$db\", type=\"get_hit_l2_and_up\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "l2_and_up",
              "refId": "F",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Get operations",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "id": 82,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "minSpan": null,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_get_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"get_max\"})",
              "intervalFactor": 2,
              "legendFormat": "max",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_get_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"get_percentile99\"})",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_get_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"get_percentile95\"})",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_get_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"get_average\"})",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Get duration",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "µs",
              "label": null,
              "logBase": 10,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 129,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_locate{instance=~\"$instance\", db=\"$db\", type=\"number_db_seek\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "seek",
              "metric": "",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_locate{instance=~\"$instance\", db=\"$db\", type=\"number_db_seek_found\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "seek_found",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_locate{instance=~\"$instance\", db=\"$db\", type=\"number_db_next\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "next",
              "metric": "",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_locate{instance=~\"$instance\", db=\"$db\", type=\"number_db_next_found\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "next_found",
              "metric": "",
              "refId": "D",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_locate{instance=~\"$instance\", db=\"$db\", type=\"number_db_prev\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "prev",
              "metric": "",
              "refId": "E",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_locate{instance=~\"$instance\", db=\"$db\", type=\"number_db_prev_found\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "prev_found",
              "metric": "",
              "refId": "F",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Seek operations",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "id": 125,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "minSpan": null,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_seek_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"seek_max\"})",
              "intervalFactor": 2,
              "legendFormat": "max",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_seek_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"seek_percentile99\"})",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_seek_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"seek_percentile95\"})",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_seek_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"seek_average\"})",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Seek duration",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "µs",
              "label": null,
              "logBase": 10,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 139,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_write_served{instance=~\"$instance\", db=\"$db\", type=~\"write_done_by_self|write_done_by_other\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "done",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_write_served{instance=~\"$instance\", db=\"$db\", type=\"write_timeout\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "timeout",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_write_served{instance=~\"$instance\", db=\"$db\", type=\"write_with_wal\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "with_wal",
              "refId": "C",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Write operations",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "id": 126,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "minSpan": null,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"write_max\"})",
              "intervalFactor": 2,
              "legendFormat": "max",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"write_percentile99\"})",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"write_percentile95\"})",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"write_average\"})",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Write duration",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "µs",
              "label": null,
              "logBase": 10,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 137,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_wal_file_synced{instance=~\"$instance\", db=\"$db\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "sync",
              "metric": "",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "WAL sync operations",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "id": 135,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "minSpan": 6,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_wal_file_sync_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"wal_file_sync_max\"})",
              "intervalFactor": 2,
              "legendFormat": "max",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_wal_file_sync_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"wal_file_sync_percentile99\"})",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_wal_file_sync_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"wal_file_sync_percentile95\"})",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_wal_file_sync_micro_seconds{instance=~\"$instance\", db=\"$db\",type=\"wal_file_sync_average\"})",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "WAL sync duration",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "µs",
              "label": null,
              "logBase": 10,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 128,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_event_total{instance=~\"$instance\", db=\"$db\"}[1m])) by (type)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "metric": "tikv_engine_event_total",
              "refId": "B",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Compaction operations",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "id": 136,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "minSpan": null,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_compaction_time{instance=~\"$instance\", db=\"$db\",type=\"compaction_time_max\"})",
              "intervalFactor": 2,
              "legendFormat": "max",
              "metric": "",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_compaction_time{instance=~\"$instance\", db=\"$db\",type=\"compaction_time_percentile99\"})",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_compaction_time{instance=~\"$instance\", db=\"$db\",type=\"compaction_time_percentile95\"})",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_compaction_time{instance=~\"$instance\", db=\"$db\",type=\"compaction_time_average\"})",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Compaction duration",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "µs",
              "label": null,
              "logBase": 10,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 140,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_sst_read_micros{instance=~\"$instance\", db=\"$db\", type=\"sst_read_micros_max\"})",
              "intervalFactor": 2,
              "legendFormat": "max",
              "metric": "",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_sst_read_micros{instance=~\"$instance\", db=\"$db\", type=\"sst_read_micros_percentile99\"})",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_sst_read_micros{instance=~\"$instance\", db=\"$db\", type=\"sst_read_micros_percentile95\"})",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "metric": "",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_sst_read_micros{instance=~\"$instance\", db=\"$db\", type=\"sst_read_micros_average\"})",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "metric": "",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "SST read duration",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ms",
              "label": null,
              "logBase": 10,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 87,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", db=\"$db\", type=\"write_stall_max\"})",
              "intervalFactor": 2,
              "legendFormat": "max",
              "metric": "",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", db=\"$db\", type=\"write_stall_percentile99\"})",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", db=\"$db\", type=\"write_stall_percentile95\"})",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "metric": "",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", db=\"$db\", type=\"write_stall_average\"})",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "metric": "",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Write stall duration",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ms",
              "label": null,
              "logBase": 10,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 103,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_memory_bytes{instance=~\"$instance\", db=\"$db\", type=\"mem-tables\"}) by (cf)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}cf}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Memtable size",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "bytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "id": 88,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "minSpan": null,
          "nullPointMode": "connected",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_memtable_efficiency{instance=~\"$instance\", db=\"$db\", type=\"memtable_hit\"}[1m])) / (sum(rate(tikv_engine_memtable_efficiency{db=\"$db\", type=\"memtable_hit\"}[1m])) + sum(rate(tikv_engine_memtable_efficiency{db=\"$db\", type=\"memtable_miss\"}[1m])))",
              "intervalFactor": 2,
              "legendFormat": "hit",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Memtable hit",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "transparent": false,
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 102,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_block_cache_size_bytes{instance=~\"$instance\", db=\"$db\"}) by(cf)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}cf}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Block cache size",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "bytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "id": 80,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "minSpan": 6,
          "nullPointMode": "connected",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_cache_efficiency{instance=~\"$instance\", db=\"$db\", type=\"block_cache_hit\"}[1m])) / (sum(rate(tikv_engine_cache_efficiency{db=\"$db\", type=\"block_cache_hit\"}[1m])) + sum(rate(tikv_engine_cache_efficiency{db=\"$db\", type=\"block_cache_miss\"}[1m])))",
              "intervalFactor": 2,
              "legendFormat": "all",
              "metric": "",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_cache_efficiency{instance=~\"$instance\", db=\"$db\", type=\"block_cache_data_hit\"}[1m])) / (sum(rate(tikv_engine_cache_efficiency{db=\"$db\", type=\"block_cache_data_hit\"}[1m])) + sum(rate(tikv_engine_cache_efficiency{db=\"$db\", type=\"block_cache_data_miss\"}[1m])))",
              "intervalFactor": 2,
              "legendFormat": "data",
              "metric": "",
              "refId": "D",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_cache_efficiency{instance=~\"$instance\", db=\"$db\", type=\"block_cache_filter_hit\"}[1m])) / (sum(rate(tikv_engine_cache_efficiency{db=\"$db\", type=\"block_cache_filter_hit\"}[1m])) + sum(rate(tikv_engine_cache_efficiency{db=\"$db\", type=\"block_cache_filter_miss\"}[1m])))",
              "intervalFactor": 2,
              "legendFormat": "filter",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_cache_efficiency{instance=~\"$instance\", db=\"$db\", type=\"block_cache_index_hit\"}[1m])) / (sum(rate(tikv_engine_cache_efficiency{db=\"$db\", type=\"block_cache_index_hit\"}[1m])) + sum(rate(tikv_engine_cache_efficiency{db=\"$db\", type=\"block_cache_index_miss\"}[1m])))",
              "intervalFactor": 2,
              "legendFormat": "index",
              "metric": "",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_bloom_efficiency{instance=~\"$instance\", db=\"$db\", type=\"bloom_prefix_useful\"}[1m])) / sum(rate(tikv_engine_bloom_efficiency{db=\"$db\", type=\"bloom_prefix_checked\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "bloom prefix",
              "metric": "",
              "refId": "E",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Block cache hit",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "transparent": false,
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "percentunit",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": false
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "height": "",
          "id": 467,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_flow_bytes{instance=~\"$instance\", db=\"$db\", type=\"block_cache_byte_read\"}[1m]))",
              "hide": false,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "total_read",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_flow_bytes{instance=~\"$instance\", db=\"$db\", type=\"block_cache_byte_write\"}[1m]))",
              "hide": false,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "total_written",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_cache_efficiency{instance=~\"$instance\", db=\"$db\", type=\"block_cache_data_bytes_insert\"}[1m]))",
              "hide": false,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "data_insert",
              "metric": "",
              "refId": "D",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_cache_efficiency{instance=~\"$instance\", db=\"$db\", type=\"block_cache_filter_bytes_insert\"}[1m]))",
              "hide": false,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "filter_insert",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_cache_efficiency{instance=~\"$instance\", db=\"$db\", type=\"block_cache_filter_bytes_evict\"}[1m]))",
              "hide": false,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "filter_evict",
              "metric": "",
              "refId": "E",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_cache_efficiency{instance=~\"$instance\", db=\"$db\", type=\"block_cache_index_bytes_insert\"}[1m]))",
              "hide": false,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "index_insert",
              "metric": "",
              "refId": "F",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_cache_efficiency{instance=~\"$instance\", db=\"$db\", type=\"block_cache_index_bytes_evict\"}[1m]))",
              "hide": false,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "index_evict",
              "metric": "",
              "refId": "G",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Block cache flow",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "Bps",
              "label": null,
              "logBase": 10,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "none",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 468,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_cache_efficiency{instance=~\"$instance\", db=\"$db\", type=\"block_cache_add\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "total_add",
              "metric": "",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_cache_efficiency{instance=~\"$instance\", db=\"$db\", type=\"block_cache_data_add\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "data_add",
              "metric": "",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_cache_efficiency{instance=~\"$instance\", db=\"$db\", type=\"block_cache_filter_add\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "filter_add",
              "metric": "",
              "refId": "D",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_cache_efficiency{instance=~\"$instance\", db=\"$db\", type=\"block_cache_index_add\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "index_add",
              "metric": "",
              "refId": "E",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_cache_efficiency{instance=~\"$instance\", db=\"$db\", type=\"block_cache_add_failures\"}[1m]))",
              "intervalFactor": 2,
              "legendFormat": "add_failures",
              "metric": "",
              "refId": "B",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Block cache operations",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "height": "",
          "id": 132,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_flow_bytes{instance=~\"$instance\", db=\"$db\", type=\"keys_read\"}[1m]))",
              "hide": false,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "read",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_flow_bytes{instance=~\"$instance\", db=\"$db\", type=\"keys_written\"}[1m]))",
              "hide": false,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "written",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_compaction_num_corrupt_keys{instance=~\"$instance\", db=\"$db\"}[1m]))",
              "hide": false,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "corrupt",
              "metric": "",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Keys flow",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 131,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(tikv_engine_estimate_num_keys{instance=~\"$instance\", db=\"$db\"}) by (cf)",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}cf}}",
              "metric": "tikv_engine_estimate_num_keys",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Total keys",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "height": "",
          "id": 85,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_flow_bytes{instance=~\"$instance\", db=\"$db\", type=\"bytes_read\"}[1m]))",
              "hide": false,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "get",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_flow_bytes{instance=~\"$instance\", db=\"$db\", type=\"iter_bytes_read\"}[1m]))",
              "hide": false,
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "scan",
              "refId": "C",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Read flow",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "Bps",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "id": 133,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "minSpan": 6,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_bytes_per_read{instance=~\"$instance\", db=\"$db\",type=\"bytes_per_read_max\"})",
              "intervalFactor": 2,
              "legendFormat": "max",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_bytes_per_read{instance=~\"$instance\", db=\"$db\",type=\"bytes_per_read_percentile99\"})",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_bytes_per_read{instance=~\"$instance\", db=\"$db\",type=\"bytes_per_read_percentile95\"})",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_bytes_per_read{instance=~\"$instance\", db=\"$db\",type=\"bytes_per_read_average\"})",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Bytes / Read",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "decbytes",
              "label": null,
              "logBase": 10,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "height": "",
          "id": 86,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_flow_bytes{instance=~\"$instance\", db=\"$db\", type=\"wal_file_bytes\"}[1m]))",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "wal",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_flow_bytes{instance=~\"$instance\", db=\"$db\", type=\"bytes_written\"}[1m]))",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "write",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Write flow",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "Bps",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "id": 134,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "minSpan": 6,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_bytes_per_write{instance=~\"$instance\", db=\"$db\",type=\"bytes_per_write_max\"})",
              "intervalFactor": 2,
              "legendFormat": "max",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_bytes_per_write{instance=~\"$instance\", db=\"$db\",type=\"bytes_per_write_percentile99\"})",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_bytes_per_write{instance=~\"$instance\", db=\"$db\",type=\"bytes_per_write_percentile95\"})",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_bytes_per_write{instance=~\"$instance\", db=\"$db\",type=\"bytes_per_write_average\"})",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Bytes / Write",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "decbytes",
              "label": null,
              "logBase": 10,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 90,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_compaction_flow_bytes{instance=~\"$instance\", db=\"$db\", type=\"bytes_read\"}[1m]))",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "read",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_compaction_flow_bytes{instance=~\"$instance\", db=\"$db\", type=\"bytes_written\"}[1m]))",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "written",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_engine_flow_bytes{instance=~\"$instance\", db=\"$db\", type=\"flush_write_bytes\"}[1m]))",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "flushed",
              "refId": "B",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Compaction flow",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "Bps",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "Bps",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 127,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_pending_compaction_bytes{instance=~\"$instance\", db=\"$db\"}[1m])) by (cf)",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}cf}}",
              "metric": "tikv_engine_pending_compaction_bytes",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Compaction pending bytes",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "bytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "Bps",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 518,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_read_amp_flow_bytes{instance=~\"$instance\", db=\"$db\", type=\"read_amp_total_read_bytes\"}[1m])) by (instance) / sum(rate(tikv_engine_read_amp_flow_bytes{db=\"$db\", type=\"read_amp_estimate_useful_bytes\"}[1m])) by (instance)",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Read amplication",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 863,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_compression_ratio{instance=~\"$instance\", db=\"$db\"}) by (level)",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "level - {{`{{`}}level}}",
              "metric": "",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Compression ratio",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 516,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "tikv_engine_num_snapshots{instance=~\"$instance\", db=\"$db\"}",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Number of snapshots",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 517,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "tikv_engine_oldest_snapshot_duration{instance=~\"$instance\", db=\"$db\"}",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tikv_engine_oldest_snapshot_duration",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Oldest snapshots duration",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "fill": 1,
          "id": 2002,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": true,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_num_files_at_level{instance=~\"$instance\", db=\"$db\"}) by (cf, level)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "cf-{{`{{`}}cf}}, level-{{`{{`}}level}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Number files at each level",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "fill": 1,
          "id": 2003,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": true,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_snapshot_ingest_sst_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "A"
            },
            {
              "expr": "sum(rate(tikv_snapshot_ingest_sst_duration_seconds_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_snapshot_ingest_sst_duration_seconds_count{instance=~\"$instance\"}[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "average",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Ingest SST duration seconds",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "fill": 1,
          "id": 2381,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideZero": true,
            "max": true,
            "min": true,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "tikv_engine_stall_conditions_changed{instance=~\"$instance\", db=\"$db\"}",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-{{`{{`}}cf}}-{{`{{`}}type}}",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Stall conditions changed of each CF",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 2451,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_compaction_reason{instance=~\"$instance\", db=\"$db\"}[1m])) by (cf, reason)",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}cf}} - {{`{{`}}reason}}",
              "metric": "",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Compression reason",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            }
          ]
        }
      ],
      "repeat": "db",
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "RocksDB - $db",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": "300",
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 95,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_grpc_msg_duration_seconds_count{instance=~\"$instance\", type!=\"kv_gc\"}[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "metric": "tikv_grpc_msg_duration_seconds_bucket",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "gRPC message count",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 107,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_grpc_msg_fail_total{instance=~\"$instance\", type!=\"kv_gc\"}[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "metric": "tikv_grpc_msg_fail_total",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "gRPC message failed",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 98,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "hideEmpty": false,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_grpc_msg_duration_seconds_bucket{instance=~\"$instance\", type!=\"kv_gc\"}[1m])) by (le, type))",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "99% gRPC messge duration",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 10,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 1844,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_grpc_msg_duration_seconds_count{instance=~\"$instance\", type=\"kv_gc\"}[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "gc message cnt",
              "metric": "tikv_grpc_msg_duration_seconds_bucket",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_grpc_msg_fail_total{instance=~\"$instance\", type=\"kv_gc\"}[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "gc failed cnt",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "gRPC GC message count",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 1845,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_grpc_msg_duration_seconds_bucket{instance=~\"$instance\", type=\"kv_gc\"}[1m])) by (le, type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "99% gRPC KV GC messge duration",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 10,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "gRPC",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": "300",
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 1069,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 350,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_pd_request_duration_seconds_count{instance=~\"$instance\"}[1m])) by (type)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}} type }}",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "PD requests",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 1070,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 350,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_pd_request_duration_seconds_sum{instance=~\"$instance\"}[1m])) by (type) / sum(rate(tikv_pd_request_duration_seconds_count{instance=~\"$instance\"}[1m])) by (type)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}} type }}",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "PD request duration (average)",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 1215,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 350,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_pd_heartbeat_message_total{instance=~\"$instance\"}[1m])) by (type)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}} type }}",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "PD heartbeats",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "opm",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "id": 1396,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 350,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_pd_validate_peer_total{instance=~\"$instance\"}[1m])) by (type)",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}} type }}",
              "metric": "",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "PD validate peers",
          "tooltip": {
            "shared": true,
            "sort": 0,
            "value_type": "individual"
          },
          "type": "graph",
          "xaxis": {
            "buckets": null,
            "mode": "time",
            "name": null,
            "show": true,
            "values": []
          },
          "yaxes": [
            {
              "format": "ops",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            },
            {
              "format": "short",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": null,
              "show": true
            }
          ]
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "PD",
      "titleSize": "h6"
    }
  ],
  "schemaVersion": 14,
  "style": "dark",
  "tags": [],
  "templating": {
    "list": [
      {
        "allValue": null,
        "current": {},
        "datasource": "tidb-cluster",
        "hide": 0,
        "includeAll": true,
        "label": "db",
        "multi": true,
        "name": "db",
        "options": [],
        "query": "label_values(tikv_engine_block_cache_size_bytes, db)",
        "refresh": 1,
        "regex": "",
        "sort": 1,
        "tagValuesQuery": "",
        "tags": [],
        "tagsQuery": "",
        "type": "query",
        "useTags": false
      },
      {
        "allValue": null,
        "current": {},
        "datasource": "tidb-cluster",
        "hide": 0,
        "includeAll": true,
        "label": "command",
        "multi": true,
        "name": "command",
        "options": [],
        "query": "label_values(tikv_storage_command_total, type)",
        "refresh": 1,
        "regex": "",
        "sort": 1,
        "tagValuesQuery": "",
        "tags": [],
        "tagsQuery": "",
        "type": "query",
        "useTags": false
      },
      {
        "allValue": ".*",
        "current": {},
        "datasource": "tidb-cluster",
        "hide": 0,
        "includeAll": true,
        "label": "Instance",
        "multi": false,
        "name": "instance",
        "options": [],
        "query": "label_values(tikv_engine_size_bytes, instance)",
        "refresh": 1,
        "regex": "",
        "sort": 1,
        "tagValuesQuery": "",
        "tags": [],
        "tagsQuery": "",
        "type": "query",
        "useTags": false
      }
    ]
  },
  "time": {
    "from": "now-1h",
    "to": "now"
  },
  "timepicker": {
    "refresh_intervals": [
      "5s",
      "10s",
      "30s",
      "1m",
      "5m",
      "15m",
      "30m",
      "1h",
      "2h",
      "1d"
    ],
    "time_options": [
      "5m",
      "15m",
      "1h",
      "6h",
      "12h",
      "24h",
      "2d",
      "7d",
      "30d"
    ]
  },
  "timezone": "browser",
  "title": "TIDB-Cluster-TiKV",
  "version": 2
}
{{- end -}}
