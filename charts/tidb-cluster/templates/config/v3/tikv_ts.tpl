{{- define "tikv_ts_dashboard_v3" -}}
{
  "__requires": [
    {
      "type": "grafana",
      "id": "grafana",
      "name": "Grafana",
      "version": ""
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
  "id": null,
  "iteration": 1556501107870,
  "links": [],
  "panels": [
    {
      "collapsed": true,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 0
      },
      "id": 2796,
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
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 0,
            "y": 1
          },
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
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 12,
            "y": 1
          },
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
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 0,
            "y": 9
          },
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
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 12,
            "y": 9
          },
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
          "fill": 1,
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 0,
            "y": 17
          },
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
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 12,
            "y": 17
          },
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
        }
      ],
      "repeat": null,
      "title": "Hot Read",
      "type": "row"
    },
    {
      "collapsed": true,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 1
      },
      "id": 2797,
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
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 0,
            "y": 18
          },
          "id": 2763,
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
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 12,
            "y": 18
          },
          "id": 2764,
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
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 0,
            "y": 26
          },
          "id": 2765,
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
          "fill": 1,
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 12,
            "y": 26
          },
          "id": 2783,
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
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 0,
            "y": 34
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"raftstore_.*\"}[1m])) by (instance)",
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
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 12,
            "y": 34
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"apply_[0-9]+\"}[1m])) by (instance)",
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
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 0,
            "y": 42
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"storage_schedul.*\"}[1m])) by (instance)",
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
        }
      ],
      "repeat": null,
      "title": "Hot Write",
      "type": "row"
    },
    {
      "collapsed": true,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 2
      },
      "id": 2798,
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 43
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(tikv_raftstore_region_count{instance=~\"$instance\", type=\"leader\"}) by (instance)",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "delta(tikv_raftstore_region_count{instance=~\"$instance\", type=\"leader\"}[30s]) < -10",
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
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 43
          },
          "id": 2786,
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": " 99%",
              "metric": "",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_raftstore_append_log_duration_seconds_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_raftstore_append_log_duration_seconds_count{instance=~\"$instance\"}[1m]))",
              "format": "time_series",
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 50
          },
          "id": 2787,
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99999, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le, instance))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} ",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "99.999% Append log duration per server",
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
          "fill": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 50
          },
          "id": 2789,
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"raft\",type=\"write_max\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "max",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"raft\",type=\"write_percentile99\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"raft\",type=\"write_percentile95\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"raft\",type=\"write_average\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Raft RocksDB write duration",
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
              "logBase": 2,
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
      "title": "Leader Drop",
      "type": "row"
    },
    {
      "collapsed": true,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 3
      },
      "id": 2799,
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
          "fill": 3,
          "grid": {},
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 0,
            "y": 51
          },
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
          "thresholds": [],
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
          "fill": 1,
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 12,
            "y": 51
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", db=\"raft\", type=\"write_stall_max\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "max",
              "metric": "",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", db=\"raft\", type=\"write_stall_percentile99\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", db=\"raft\", type=\"write_stall_percentile95\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "metric": "",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", db=\"raft\", type=\"write_stall_average\"})",
              "format": "time_series",
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
          "title": "Raft RocksDB write stall duration",
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
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 0,
            "y": 59
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": " 99%",
              "metric": "",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_raftstore_append_log_duration_seconds_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_raftstore_append_log_duration_seconds_count{instance=~\"$instance\"}[1m]))",
              "format": "time_series",
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
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 12,
            "y": 59
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le, instance))",
              "format": "time_series",
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
      "title": "Channel Full",
      "type": "row"
    },
    {
      "collapsed": true,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 4
      },
      "id": 2800,
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
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 0,
            "y": 60
          },
          "id": 2777,
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
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 12,
            "y": 60
          },
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
          "minSpan": 12,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
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
          "thresholds": [],
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
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 0,
            "y": 68
          },
          "id": 2779,
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
          "thresholds": [],
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
          "fill": 1,
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 12,
            "y": 68
          },
          "id": 2785,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", type=\"write_stall_max\"}) by (db)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}db}} max",
              "metric": "",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", type=\"write_stall_percentile99\"}) by (db)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}db}} 99%",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", type=\"write_stall_percentile95\"}) by (db)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}db}} 95%",
              "metric": "",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", type=\"write_stall_average\"}) by (db)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}db}} avg",
              "metric": "",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "RocksDB write stall duration",
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
        }
      ],
      "repeat": null,
      "title": "Server Is Busy",
      "type": "row"
    },
    {
      "collapsed": true,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 5
      },
      "id": 2801,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 6
          },
          "id": 2768,
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
          "timeRegions": [],
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 6
          },
          "id": 2782,
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
          "timeRegions": [],
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 13
          },
          "id": 2762,
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
          "timeRegions": [],
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 13
          },
          "id": 2761,
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
          "timeRegions": [],
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 20
          },
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
              "expr": "histogram_quantile(0.95, sum(rate(tikv_coprocessor_request_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le,req))",
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
          "timeRegions": [],
          "timeShift": null,
          "title": "Coprocessor request duration",
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 20
          },
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
              "expr": "histogram_quantile(0.95, sum(rate(tikv_coprocessor_request_wait_seconds_bucket{instance=~\"$instance\"}[1m])) by (le,req))",
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
          "timeRegions": [],
          "timeShift": null,
          "title": "Coprocessor wait duration",
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 27
          },
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
          "timeRegions": [],
          "timeShift": null,
          "title": "Coprocessor scan keys",
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 27
          },
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
          "timeRegions": [],
          "timeShift": null,
          "title": "95% Coprocessor wait duration by store",
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 34
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_get_micro_seconds{instance=~\"$instance\", db=\"kv\",type=\"get_max\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "max",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_get_micro_seconds{instance=~\"$instance\", db=\"kv\",type=\"get_percentile99\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_get_micro_seconds{instance=~\"$instance\", db=\"kv\",type=\"get_percentile95\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_get_micro_seconds{instance=~\"$instance\", db=\"kv\",type=\"get_average\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "RocksDB get duration",
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
              "logBase": 2,
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 34
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_seek_micro_seconds{instance=~\"$instance\", db=\"kv\",type=\"seek_max\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "max",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_seek_micro_seconds{instance=~\"$instance\", db=\"kv\",type=\"seek_percentile99\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_seek_micro_seconds{instance=~\"$instance\", db=\"kv\",type=\"seek_percentile95\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_seek_micro_seconds{instance=~\"$instance\", db=\"kv\",type=\"seek_average\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "RocksDB seek duration",
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
              "logBase": 2,
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "repeat": null,
      "title": "Read Too Slow",
      "type": "row"
    },
    {
      "collapsed": true,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 6
      },
      "id": 2802,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 7
          },
          "id": 2781,
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
          "timeRegions": [],
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 7
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{instance=~\"$instance\", type=\"write\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{instance=~\"$instance\", type=\"write\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_storage_engine_async_request_duration_seconds_sum{instance=~\"$instance\", type=\"write\"}[1m])) / sum(rate(tikv_storage_engine_async_request_duration_seconds_count{instance=~\"$instance\", type=\"write\"}[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 14
          },
          "id": 2753,
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{instance=~\"$instance\", type=\"prewrite\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "metric": "",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{instance=~\"$instance\", type=\"prewrite\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_scheduler_latch_wait_duration_seconds_sum{instance=~\"$instance\", type=\"prewrite\"}[1m])) / sum(rate(tikv_scheduler_latch_wait_duration_seconds_count{instance=~\"$instance\", type=\"prewrite\"}[1m])) ",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "metric": "",
              "refId": "C",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Prewrite latch wait duration",
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 14
          },
          "id": 2774,
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{instance=~\"$instance\", type=\"commit\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "metric": "",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{instance=~\"$instance\", type=\"commit\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_scheduler_latch_wait_duration_seconds_sum{instance=~\"$instance\", type=\"commit\"}[1m])) / sum(rate(tikv_scheduler_latch_wait_duration_seconds_count{instance=~\"$instance\", type=\"commit\"}[1m])) ",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "metric": "",
              "refId": "C",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Commit latch wait duration",
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 21
          },
          "id": 2788,
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": " 99%",
              "metric": "",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_raftstore_append_log_duration_seconds_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_raftstore_append_log_duration_seconds_count{instance=~\"$instance\"}[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 21
          },
          "id": 2790,
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le, instance))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} ",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 28
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_apply_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": " 99%",
              "metric": "",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_raftstore_apply_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_raftstore_apply_log_duration_seconds_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_raftstore_apply_log_duration_seconds_count{instance=~\"$instance\"}[1m])) ",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 28
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_apply_log_duration_seconds_bucket{instance=~\"$instance\"}[1m])) by (le, instance))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": " 99% {{`{{`}}instance}}",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 35
          },
          "id": 2794,
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_request_wait_time_duration_secs_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": " 99%",
              "metric": "",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_raftstore_request_wait_time_duration_secs_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_raftstore_request_wait_time_duration_secs_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_raftstore_request_wait_time_duration_secs_count{instance=~\"$instance\"}[1m])) ",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 35
          },
          "id": 2795,
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_apply_wait_time_duration_secs_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": " 99%",
              "metric": "",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_raftstore_apply_wait_time_duration_secs_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "B",
              "step": 4
            },
            {
              "expr": "sum(rate(tikv_raftstore_apply_wait_time_duration_secs_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_raftstore_apply_wait_time_duration_secs_count{instance=~\"$instance\"}[1m])) ",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "C",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Apply wait duration",
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 42
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"kv\",type=\"write_max\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} max",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"kv\",type=\"write_percentile99\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} 99%",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"kv\",type=\"write_percentile95\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} 95%",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"kv\",type=\"write_average\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} avg",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "KV RocksDB write duration",
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
              "logBase": 2,
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 42
          },
          "id": 2776,
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"raft\",type=\"write_max\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} max",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"raft\",type=\"write_percentile99\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} 99%",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"raft\",type=\"write_percentile95\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} 95%",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_micro_seconds{instance=~\"$instance\", db=\"raft\",type=\"write_average\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} avg",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Raft RocksDB write duration",
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
              "logBase": 2,
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 49
          },
          "id": 2791,
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
          "timeRegions": [],
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 0,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 49
          },
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
          "minSpan": 12,
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_wal_file_sync_micro_seconds{instance=~\"$instance\", db=\"raft\",type=\"wal_file_sync_max\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "max",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_wal_file_sync_micro_seconds{instance=~\"$instance\", db=\"raft\",type=\"wal_file_sync_percentile99\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_wal_file_sync_micro_seconds{instance=~\"$instance\", db=\"raft\",type=\"wal_file_sync_percentile95\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_wal_file_sync_micro_seconds{instance=~\"$instance\", db=\"raft\",type=\"wal_file_sync_average\"})",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Raft RocksDB WAL sync duration",
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 56
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_engine_wal_file_synced{instance=~\"$instance\", db=\"raft\"}[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "sync",
              "metric": "",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Raft RocksDB WAL sync operations",
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 56
          },
          "id": 2793,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", type=\"write_stall_max\"}) by (db)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}db}} max",
              "metric": "",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", type=\"write_stall_percentile99\"}) by (db)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}db}} 99%",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", type=\"write_stall_percentile95\"}) by (db)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}db}} 95%",
              "metric": "",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", type=\"write_stall_average\"}) by (db)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}db}} avg",
              "metric": "",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "RocksDB write stall duration",
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "repeat": null,
      "title": "Write Too Slow",
      "type": "row"
    },
    {
      "collapsed": true,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 7
      },
      "id": 2806,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 8
          },
          "id": 2810,
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
          "scopedVars": {
            "db": {
              "selected": false,
              "text": "kv",
              "value": "kv"
            }
          },
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_num_files_at_level{instance=~\"$instance\", db=\"kv\", level=\"0\"}) by (instance, cf)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{`{{`}}instance}} {{`{{`}}cf}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Level0 SST file number",
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 8
          },
          "id": 2811,
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
          "scopedVars": {
            "db": {
              "selected": false,
              "text": "kv",
              "value": "kv"
            }
          },
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_num_immutable_mem_table{instance=~\"$instance\", db=\"$db\"}) by (instance, cf)",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "{{`{{`}}instance}} {{`{{`}}cf}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Immutable mem-table number",
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 15
          },
          "id": 2808,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "scopedVars": {
            "db": {
              "selected": false,
              "text": "kv",
              "value": "kv"
            }
          },
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_pending_compaction_bytes{instance=~\"$instance\", db=\"$db\"}) by (instance, cf)",
              "format": "time_series",
              "instant": false,
              "intervalFactor": 1,
              "legendFormat": "{{`{{`}}instance}} {{`{{`}}cf}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "Pending compaction bytes",
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
              "format": "bytes",
              "label": null,
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": false
            }
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 15
          },
          "id": 2812,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sideWidth": 300,
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
          "scopedVars": {
            "db": {
              "selected": false,
              "text": "kv",
              "value": "kv"
            }
          },
          "seriesOverrides": [],
          "spaceLength": 10,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", type=\"write_stall_max\", db=\"$db\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} max",
              "metric": "",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", type=\"write_stall_percentile99\", db=\"$db\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} 99%",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", type=\"write_stall_percentile95\", db=\"$db\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} 95%",
              "metric": "",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "avg(tikv_engine_write_stall{instance=~\"$instance\", type=\"write_stall_average\", db=\"$db\"}) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} avg",
              "metric": "",
              "refId": "D",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "RocksDB write stall duration",
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "repeat": "db",
      "title": "Write Stall - $db",
      "type": "row"
    },
    {
      "collapsed": true,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 8
      },
      "id": 2803,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": 1,
          "fill": 1,
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 0,
            "y": 8
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "avg(tikv_engine_block_cache_size_bytes{instance=~\"$instance\", db=\"kv\"}) by(cf)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}cf}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 8,
            "w": 12,
            "x": 12,
            "y": 8
          },
          "id": 2770,
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
          "timeRegions": [],
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "repeat": null,
      "title": "OOM",
      "type": "row"
    },
    {
      "collapsed": true,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 9
      },
      "id": 2804,
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 0,
            "y": 17
          },
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tikv_raftstore_region_size_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99%",
              "metric": "",
              "refId": "B",
              "step": 10
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tikv_raftstore_region_size_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95%",
              "metric": "",
              "refId": "C",
              "step": 10
            },
            {
              "expr": "sum(rate(tikv_raftstore_region_size_sum{instance=~\"$instance\"}[1m])) / sum(rate(tikv_raftstore_region_size_count{instance=~\"$instance\"}[1m])) ",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "avg",
              "metric": "",
              "refId": "D",
              "step": 10
            },
            {
              "expr": "histogram_quantile(0.999999, sum(rate(tikv_raftstore_region_size_bucket{instance=~\"$instance\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99.9999%",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
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
          "gridPos": {
            "h": 7,
            "w": 12,
            "x": 12,
            "y": 17
          },
          "id": 2792,
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
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tikv_thread_cpu_seconds_total{instance=~\"$instance\", name=~\"split_check\"}[1m])) by (instance)",
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
          "timeRegions": [],
          "timeShift": null,
          "title": "Split checker CPU",
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
          ],
          "yaxis": {
            "align": false,
            "alignLevel": null
          }
        }
      ],
      "repeat": null,
      "title": "Huge Region",
      "type": "row"
    }
  ],
  "refresh": "1m",
  "schemaVersion": 16,
  "style": "dark",
  "tags": [],
  "templating": {
    "list": [
      {
        "allValue": null,
        "current": {},
        "datasource": "tidb-cluster",
        "definition": "",
        "hide": 0,
        "includeAll": true,
        "label": "db",
        "multi": true,
        "name": "db",
        "options": [],
        "query": "label_values(tikv_engine_block_cache_size_bytes, db)",
        "refresh": 1,
        "regex": "",
        "skipUrlSync": false,
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
        "definition": "",
        "hide": 0,
        "includeAll": true,
        "label": "command",
        "multi": true,
        "name": "command",
        "options": [],
        "query": "label_values(tikv_storage_command_total, type)",
        "refresh": 1,
        "regex": "",
        "skipUrlSync": false,
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
        "definition": "",
        "hide": 0,
        "includeAll": true,
        "label": "Instance",
        "multi": false,
        "name": "instance",
        "options": [],
        "query": "label_values(tikv_engine_size_bytes, instance)",
        "refresh": 1,
        "regex": "",
        "skipUrlSync": false,
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
    "from": "now-5m",
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
  "title": "TIDB-Cluster-TiKV-Trouble-Shooting",
  "uid": "Lg4wiEkZz",
  "version": 2
}
{{- end -}}