{{- define "tidb_dashboard_v3" -}}
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
      "version": "1.0.0"
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
  "graphTooltip": 0,
  "hideControls": false,
  "id": null,
  "links": [],
  "refresh": "30s",
  "rows": [
    {
      "collapse": false,
      "height": "240",
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "fill": 1,
          "id": 80,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.999, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "999",
              "refId": "A"
            },
            {
              "expr": "histogram_quantile(0.99, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99",
              "refId": "B"
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95",
              "refId": "C"
            },
            {
              "expr": "histogram_quantile(0.80, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "80",
              "refId": "D"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Duration",
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
            "alignAsTable": false,
            "avg": false,
            "current": true,
            "hideEmpty": true,
            "hideZero": false,
            "max": true,
            "min": false,
            "rightSide": false,
            "show": true,
            "sideWidth": 250,
            "sort": "max",
            "sortDesc": false,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "minSpan": 12,
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
              "expr": "sum(rate(tidb_server_query_total[1m])) by (result)",
              "format": "time_series",
              "instant": false,
              "intervalFactor": 2,
              "legendFormat": "query {{`{{`}}result}}",
              "refId": "A",
              "step": 60
            },
            {
              "expr": "sum(rate(tidb_server_query_total{result=\"OK\"}[1m]  offset 1d))",
              "format": "time_series",
              "hide": true,
              "instant": false,
              "intervalFactor": 3,
              "legendFormat": "yesterday",
              "refId": "B",
              "step": 90
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "QPS",
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
          "decimals": null,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 21,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
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
              "expr": "sum(rate(tidb_executor_statement_total[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 30
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Statement OPS",
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
              "logBase": 2,
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
            "sort": "max",
            "sortDesc": true,
            "total": false,
            "values": true
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "minSpan": 12,
          "nullPointMode": "null as zero",
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
              "expr": "rate(tidb_server_query_total[1m])",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}} {{`{{`}}type}} {{`{{`}}result}}",
              "refId": "A",
              "step": 30
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "QPS By Instance",
          "tooltip": {
            "msResolution": true,
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
          "id": 92,
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
          "minSpan": 12,
          "nullPointMode": "null as zero",
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
          "span": 12,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(tidb_server_execute_error_total[1m])) by (type, instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": " {{`{{`}}type}}-{{`{{`}}instance}}",
              "refId": "A",
              "step": 30
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Failed Query OPM",
          "tooltip": {
            "msResolution": true,
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
              "logBase": 2,
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
          "fill": 1,
          "id": 112,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 12,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.90, sum(rate(tidb_server_slow_query_process_duration_seconds_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "all_proc",
              "refId": "A"
            },
            {
              "expr": "histogram_quantile(0.90, sum(rate(tidb_server_slow_query_cop_duration_seconds_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "all_cop_proc",
              "refId": "B"
            },
            {
              "expr": "histogram_quantile(0.90, sum(rate(tidb_server_slow_query_wait_duration_seconds_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "all_cop_wait",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Slow query",
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
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "fill": 1,
          "id": 136,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.999, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le,sql_type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}sql_type}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "999 Duration",
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
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "fill": 1,
          "id": 134,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le,sql_type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}sql_type}}",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "99 Duration",
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
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "fill": 1,
          "id": 132,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.95, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le,sql_type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}sql_type}}",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "95 Duration",
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
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "fill": 1,
          "id": 130,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.80, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le,sql_type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}sql_type}}",
              "refId": "D"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeRegions": [],
          "timeShift": null,
          "title": "80 Duration",
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
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Query Summary",
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
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 23,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": false
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
              "expr": "histogram_quantile(0.80, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le, instance))",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "B",
              "step": 60
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Duration 80 By Instance",
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
              "format": "s",
              "label": null,
              "logBase": 2,
              "max": null,
              "min": "0.001",
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
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 1,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "hideEmpty": true,
            "hideZero": true,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": false
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
              "expr": "histogram_quantile(0.95, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le, instance))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}} instance }}",
              "refId": "B",
              "step": 60
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Duration 95 By Instance",
          "tooltip": {
            "msResolution": true,
            "shared": false,
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
            "values": [
              "max"
            ]
          },
          "yaxes": [
            {
              "format": "s",
              "label": null,
              "logBase": 2,
              "max": null,
              "min": "0.001",
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
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 25,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": false
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
              "expr": "histogram_quantile(0.99, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le, instance))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 60
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Duration 99 By Instance",
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
              "format": "s",
              "label": null,
              "logBase": 2,
              "max": null,
              "min": "0.001",
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
          "id": 81,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": false
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
              "expr": "histogram_quantile(0.999, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le, instance))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 60
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Duration 999 By Instance",
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
              "format": "s",
              "label": null,
              "logBase": 2,
              "max": null,
              "min": "0.001",
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
          "id": 94,
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
              "expr": "increase(tidb_server_execute_error_total[1m])",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}} @ {{`{{`}}instance}}",
              "refId": "A",
              "step": 60
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Failed Query OPM Detail",
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
              "format": "none",
              "label": null,
              "logBase": 2,
              "max": null,
              "min": "0.001",
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
          "description": "The internal SQL is used by TiDB itself.",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 68,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": false,
            "total": false,
            "values": false
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
              "expr": "sum(rate(tidb_session_restricted_sql_total[30s]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Internal SQL OPS",
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
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Query Detail",
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
          "id": 8,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": true,
            "max": false,
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
          "seriesOverrides": [
            {
              "alias": "total",
              "fill": 0,
              "lines": false
            }
          ],
          "spaceLength": 10,
          "span": 6,
          "stack": true,
          "steppedLine": true,
          "targets": [
            {
              "expr": "tidb_server_connections",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A",
              "step": 40
            },
            {
              "expr": "sum(tidb_server_connections)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "total",
              "refId": "B",
              "step": 40
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Connection Count",
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
          "editable": false,
          "error": false,
          "fill": 0,
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
            "sideWidth": null,
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
              "expr": "go_memstats_heap_inuse_bytes{job=~\"tidb.*\"}",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "go_memstats_heap_inuse_bytes",
              "refId": "B",
              "step": 40
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Heap Memory Usage",
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
              "format": "bytes",
              "label": "",
              "logBase": 1,
              "max": null,
              "min": "0",
              "show": true
            },
            {
              "format": "short",
              "label": "",
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
          "fill": 1,
          "id": 61,
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
              "expr": " go_goroutines{job=~\"tidb.*\"}",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Goroutine Count",
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
          "description": "TiDB Server events.",
          "fill": 1,
          "id": 49,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "increase(tidb_server_event_total[10m])",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-{{`{{`}}type}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Events OPM",
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
          "id": 54,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "increase(tidb_server_panic_total[1m])",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "panic-{{`{{`}}instance}}",
              "refId": "A"
            },
            {
              "expr": "increase(tidb_server_critical_error_total[1m])",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "critical-{{`{{`}}instance}}",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Uncommon Error OPM",
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
          "id": 82,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(tidb_monitor_keep_alive_total[1m])) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Keep Alive OPM",
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
          "description": "Duration (us) for getting token, it should be small until concurrency limit is reached.",
          "fill": 1,
          "gridPos": {
            "h": 9,
            "w": 12,
            "x": 0,
            "y": 24
          },
          "id": 111,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tidb_server_get_token_duration_seconds_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 1,
              "legendFormat": "99",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Get Token Duration",
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
              "format": "µs",
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
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Server",
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
          "editable": false,
          "error": false,
          "fill": 1,
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
              "expr": "sum(rate(tidb_session_transaction_total[1m])) by (type, sql_type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}-{{`{{`}}sql_type}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Transaction OPS",
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
          "fill": 1,
          "id": 72,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tidb_session_transaction_duration_seconds_bucket[1m])) by (le, sql_type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99-{{`{{`}}sql_type}}",
              "refId": "A"
            },
            {
              "expr": "histogram_quantile(0.95, sum(rate(tidb_session_transaction_duration_seconds_bucket[1m])) by (le, sql_type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "95-{{`{{`}}sql_type}}",
              "refId": "B"
            },
            {
              "expr": "histogram_quantile(0.80, sum(rate(tidb_session_transaction_duration_seconds_bucket[1m])) by (le, sql_type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "80-{{`{{`}}sql_type}}",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Duration",
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
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "fill": 1,
          "id": 74,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tidb_session_transaction_statement_num_bucket[30s])) by (le, sql_type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99-{{`{{`}}sql_type}}",
              "refId": "A"
            },
            {
              "expr": "histogram_quantile(0.80, sum(rate(tidb_session_transaction_statement_num_bucket[30s])) by (le, sql_type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "80-{{`{{`}}sql_type}}",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Transaction Statement Num",
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
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 36,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tidb_session_retry_error_total[30s])) by (type, sql_type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}-{{`{{`}}sql_type}}",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Session Retry Error OPS",
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
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 67,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": false
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(1.0, sum(rate(tidb_session_retry_num_bucket[30s])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "100",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "histogram_quantile(0.99, sum(rate(tidb_session_retry_num_bucket[30s])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99",
              "refId": "B"
            },
            {
              "expr": "histogram_quantile(0.90, sum(rate(tidb_session_retry_num_bucket[30s])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "90",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Transaction Retry Num",
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
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Transaction",
      "titleSize": "h6"
    },
    {
      "collapse": true,
      "height": 301,
      "panels": [
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "decimals": null,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 76,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
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
              "expr": "sum(rate(tidb_executor_expensive_total[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 30
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Expensive Executors OPS",
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
          "decimals": null,
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 91,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": true,
            "hideEmpty": true,
            "hideZero": true,
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
              "expr": "sum(rate(tidb_server_plan_cache_total[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 30
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Queries Using Plan Cache OPS",
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
              "logBase": 2,
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
      "title": "Executor",
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
          "id": 12,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 2,
          "links": [
            {
              "type": "dashboard"
            }
          ],
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
              "expr": "histogram_quantile(0.999, sum(rate(tidb_distsql_handle_query_duration_seconds_bucket[1m])) by (le, type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "999-{{`{{`}}type}}",
              "refId": "D"
            },
            {
              "expr": "histogram_quantile(0.99, sum(rate(tidb_distsql_handle_query_duration_seconds_bucket[1m])) by (le, type))",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "legendFormat": "99-{{`{{`}}type}}",
              "metric": "tidb_distsql_handle_query_duration_seconds_bucket",
              "refId": "A",
              "step": 4
            },
            {
              "expr": "histogram_quantile(0.90, sum(rate(tidb_distsql_handle_query_duration_seconds_bucket[1m])) by (le, type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "90-{{`{{`}}type}}",
              "refId": "B"
            },
            {
              "expr": "histogram_quantile(0.50, sum(rate(tidb_distsql_handle_query_duration_seconds_bucket[1m])) by (le, type))",
              "format": "time_series",
              "interval": "",
              "intervalFactor": 2,
              "legendFormat": "50-{{`{{`}}type}}",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Distsql Duration",
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
              "logBase": 2,
              "max": null,
              "min": "0.0005",
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
          "id": 14,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": false,
            "show": false,
            "total": false,
            "values": false
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
              "expr": "sum(rate(tidb_distsql_handle_query_duration_seconds_count[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "",
              "metric": "tidb_distsql_query_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Distsql QPS",
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
              "min": 0,
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
          "id": 60,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": false,
            "show": false,
            "total": false,
            "values": false
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tidb_distsql_scan_keys_partial_num_count[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "",
              "metric": "tidb_distsql_query_total",
              "refId": "A",
              "step": 4
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Distsql Partial QPS",
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
              "min": 0,
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
          "id": 57,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(1, sum(rate(tidb_distsql_scan_keys_num_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "100",
              "refId": "A"
            },
            {
              "expr": "histogram_quantile(0.90, sum(rate(tidb_distsql_scan_keys_num_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "90",
              "refId": "B"
            },
            {
              "expr": "histogram_quantile(0.50, sum(rate(tidb_distsql_scan_keys_num_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "50",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scan Keys Num",
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
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "fill": 1,
          "id": 58,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(1, sum(rate(tidb_distsql_scan_keys_partial_num_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "100",
              "refId": "A"
            },
            {
              "expr": "histogram_quantile(0.90, sum(rate(tidb_distsql_scan_keys_partial_num_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "90",
              "refId": "B"
            },
            {
              "expr": "histogram_quantile(0.80, sum(rate(tidb_distsql_scan_keys_partial_num_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "50",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Scan Keys Partial Num",
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
        },
        {
          "aliasColors": {},
          "bars": false,
          "dashLength": 10,
          "dashes": false,
          "datasource": "tidb-cluster",
          "fill": 1,
          "id": 59,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(1, sum(rate(tidb_distsql_partial_num_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "100",
              "refId": "A"
            },
            {
              "expr": "histogram_quantile(0.90, sum(rate(tidb_distsql_partial_num_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "90",
              "refId": "B"
            },
            {
              "expr": "histogram_quantile(0.50, sum(rate(tidb_distsql_partial_num_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "50",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Partial Num",
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
            "current": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": "max",
            "sortDesc": true,
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.999, sum(rate(tidb_tikvclient_cop_duration_seconds_bucket[1m])) by (le, instance))",
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
          "title": "Coprocessor Seconds 999",
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
      "title": "Distsql",
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
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 6,
          "legend": {
            "alignAsTable": false,
            "avg": false,
            "current": false,
            "max": true,
            "min": false,
            "rightSide": false,
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
              "expr": "histogram_quantile(0.9999, sum(rate(tidb_tikvclient_backoff_seconds_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "9999",
              "refId": "A",
              "step": 40
            },
            {
              "expr": "histogram_quantile(0.99, sum(rate(tidb_tikvclient_backoff_seconds_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99",
              "refId": "B"
            },
            {
              "expr": "histogram_quantile(0.80, sum(rate(tidb_tikvclient_backoff_seconds_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "80",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "KV Retry Duration",
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
          "fill": 0,
          "grid": {},
          "id": 11,
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
              "expr": "sum(rate(tidb_tikvclient_region_err_total[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "metric": "tidb_server_session_execute_parse_duration_count",
              "refId": "A",
              "step": 40
            },
            {
              "expr": "sum(rate(tidb_tikvclient_region_err_total{type=\"server_is_busy\"}[1m]))",
              "format": "time_series",
              "hide": true,
              "intervalFactor": 2,
              "legendFormat": "sum",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "TiClient Region Error OPS",
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
          "fill": 1,
          "id": 53,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "hideEmpty": true,
            "hideZero": true,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": true,
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
              "expr": "sum(rate(tidb_tikvclient_backoff_total[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "KV Backoff OPS",
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
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 32,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "hideEmpty": true,
            "hideZero": true,
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
              "expr": "sum(rate(tidb_tikvclient_lock_resolver_actions_total[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "metric": "tidb_tikvclient_lock_resolver_actions_total",
              "refId": "A",
              "step": 40
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Lock Resolve OPS",
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
          "id": 84,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "hideEmpty": true,
            "hideZero": true,
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
              "expr": "sum(rate(tidb_tikvclient_lock_cleanup_task_total[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "cleanup_secondary_failure_{{`{{`}}type}}",
              "metric": "tidb_tikvclient_lock_resolver_actions_total",
              "refId": "A",
              "step": 40
            },
            {
              "expr": "sum(rate(tidb_tikvclient_load_safepoint_total{type=\"fail\"}[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "load_safepoint_failure",
              "refId": "B"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Other Errors OPS",
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
      "title": "KV Errors",
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
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 48,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
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
              "expr": "histogram_quantile(0.9999, sum(rate(tidb_tikvclient_request_seconds_bucket{type!=\"GC\"}[1m])) by (le, store))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "store-{{`{{`}}store}}",
              "refId": "A",
              "step": 40
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "KV Request Duration 999 by store",
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
          "id": 30,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
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
              "expr": "histogram_quantile(0.999, sum(rate(tidb_tikvclient_request_seconds_bucket{type!=\"GC\"}[1m])) by (le,type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 40
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "KV Request Duration 999 by type",
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
          "id": 22,
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
              "expr": "histogram_quantile(0.999, sum(rate(tidb_tikvclient_txn_cmd_duration_seconds_bucket[1m])) by (le, type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 40
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "KV Cmd Duration 9999",
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
              "expr": "histogram_quantile(0.99, sum(rate(tidb_tikvclient_txn_cmd_duration_seconds_bucket[1m])) by (le, type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 40
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "KV Cmd Duration 99",
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
      "title": "KV Duration",
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
          "id": 5,
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
            "sort": "max",
            "sortDesc": true,
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tidb_tikvclient_txn_cmd_duration_seconds_count[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 40
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "KV Cmd OPS",
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
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 4,
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
          "nullPointMode": "null as zero",
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
              "expr": "sum(rate(tidb_tikvclient_txn_total[1m])) by (instance)",
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
          "title": "KV Txn OPS",
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
          "id": 44,
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
          "nullPointMode": "null as zero",
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
              "expr": "histogram_quantile(0.90, sum(rate(tidb_tikvclient_txn_regions_num_bucket[1m])) by (le, instance))",
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
          "title": "Txn Regions Num 90",
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
          "id": 34,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": "avg",
            "sortDesc": true,
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(1, sum(rate(tidb_tikvclient_txn_write_size_bytes_bucket[1m])) by (le, instance))",
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
          "title": "Txn Write Size Bytes 100",
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
              "format": "bytes",
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
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 33,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": "avg",
            "sortDesc": true,
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(1, sum(rate(tidb_tikvclient_txn_write_kv_num_bucket[1m])) by (le, instance))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "100 {{`{{`}}instance}}",
              "refId": "B",
              "step": 40
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Txn Write KV Num 100",
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
          "id": 83,
          "legend": {
            "alignAsTable": true,
            "avg": true,
            "current": false,
            "max": true,
            "min": false,
            "rightSide": true,
            "show": true,
            "sort": "avg",
            "sortDesc": true,
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tidb_tikvclient_load_safepoint_total{type=\"ok\"}[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "B",
              "step": 40
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Load Safepoint OPS",
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
      "title": "KV Count",
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
          "id": 20,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": false
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(pd_client_cmd_handle_cmds_duration_seconds_count{type!=\"tso\"}[1m])) by (type)",
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
          "title": "PD Client CMD OPS",
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
          "description": "",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 35,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": false
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.999, sum(rate(pd_client_cmd_handle_cmds_duration_seconds_bucket{type!=\"tso\"}[1m])) by (le, type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "999-{{`{{`}}type}}",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "histogram_quantile(0.99, sum(rate(pd_client_cmd_handle_cmds_duration_seconds_bucket{type!=\"tso\"}[1m])) by (le, type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99-{{`{{`}}type}}",
              "refId": "B"
            },
            {
              "expr": "histogram_quantile(0.90, sum(rate(pd_client_cmd_handle_cmds_duration_seconds_bucket{type!=\"tso\"}[1m])) by (le, type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "90-{{`{{`}}type}}",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "PD Client CMD Duration",
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
          "id": 43,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": false
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(pd_client_cmd_handle_failed_cmds_duration_seconds_count[1m])) by (type)",
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
          "title": "PD Client CMD Fail OPS",
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
          "description": "The duration of a client calling GetTSAsync until received the TS result.",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 79,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(pd_client_cmd_handle_cmds_duration_seconds_count{type=\"tso\"}[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "cmd",
              "refId": "C"
            },
            {
              "expr": "sum(rate(pd_client_request_handle_requests_duration_seconds_count{type=\"tso\"}[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "request",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "PD TSO OPS",
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
          "description": "The duration of a client calling GetTSAsync until received the TS result.",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 77,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.999, sum(rate(pd_client_cmd_handle_cmds_duration_seconds_bucket{type=\"tso\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "999",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "histogram_quantile(0.99, sum(rate(pd_client_cmd_handle_cmds_duration_seconds_bucket{type=\"tso\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99",
              "refId": "B"
            },
            {
              "expr": "histogram_quantile(0.90, sum(rate(pd_client_cmd_handle_cmds_duration_seconds_bucket{type=\"tso\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "90",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "PD TSO Wait Duration",
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
          "description": "The duration of a client sending TSO request until received the response.",
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 78,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
            "total": false,
            "values": false
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.999, sum(rate(pd_client_request_handle_requests_duration_seconds_bucket{type=\"tso\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "999",
              "refId": "A",
              "step": 10
            },
            {
              "expr": "histogram_quantile(0.99, sum(rate(pd_client_request_handle_requests_duration_seconds_bucket{type=\"tso\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99",
              "refId": "B"
            },
            {
              "expr": "histogram_quantile(0.90, sum(rate(pd_client_request_handle_requests_duration_seconds_bucket{type=\"tso\"}[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "90",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "PD TSO RPC Duration",
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
      "title": "PD Client",
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
          "editable": false,
          "error": false,
          "fill": 1,
          "grid": {},
          "id": 27,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": false
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tidb_domain_load_schema_duration_seconds_bucket[1m])) by (le, instance))",
              "format": "time_series",
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
          "title": "Load Schema Duration",
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
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 28,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": false
          },
          "lines": true,
          "linewidth": 1,
          "links": [],
          "nullPointMode": "null as zero",
          "percentage": false,
          "pointradius": 5,
          "points": false,
          "renderer": "flot",
          "seriesOverrides": [
            {
              "alias": "/.*failed/",
              "bars": true
            }
          ],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tidb_domain_load_schema_total[1m])) by (instance,type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-{{`{{`}}type}}",
              "metric": "tidb_domain_load_schema_duration_count",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Load Schema OPS",
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
          "decimals": 2,
          "editable": false,
          "error": false,
          "fill": 0,
          "grid": {},
          "id": 29,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "hideEmpty": true,
            "hideZero": true,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": false
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(tidb_session_schema_lease_error_total[1m])) by (instance)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}",
              "metric": "tidb_server_",
              "refId": "A",
              "step": 10
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Schema Lease Error OPM",
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
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "Schema Load",
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
          "id": 9,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
            "total": false,
            "values": false
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
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.95, sum(rate(tidb_ddl_handle_job_duration_seconds_bucket[1m])) by (le, type))",
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
          "title": "DDL Duration 95",
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
          "fill": 1,
          "id": 63,
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
              "expr": "histogram_quantile(1, sum(rate(tidb_ddl_batch_add_idx_duration_seconds_bucket[1m])) by (le, type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Batch Add Index Duration 100",
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
          "fill": 1,
          "id": 62,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "hideEmpty": true,
            "hideZero": true,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "tidb_ddl_waiting_jobs",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-{{`{{`}}type}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "DDL Waiting Jobs Count",
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
          "fill": 1,
          "id": 55,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "increase(tidb_ddl_worker_operation_total[1m])",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}instance}}-{{`{{`}}type}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "DDL META OPM",
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
          "id": 56,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "hideEmpty": true,
            "hideZero": true,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 6,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(increase(tidb_ddl_worker_operation_duration_seconds_bucket[1m])) by (le, type, action, result))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}-{{`{{`}}action}}-{{`{{`}}result}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "DDL Worker Duration 99",
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
          "id": 64,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(1, sum(rate(tidb_ddl_deploy_syncer_duration_seconds_bucket[2m])) by (le, type, result))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}-{{`{{`}}result}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Deploy Syncer Duration",
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
          "id": 65,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(1, sum(rate(tidb_ddl_owner_handle_syncer_duration_seconds_bucket[2m])) by (le, type, result))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}-{{`{{`}}result}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Owner Handle Syncer Duration",
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
          "id": 66,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(1, sum(rate(tidb_ddl_update_self_ver_duration_seconds_bucket[2m])) by (le, result))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}result}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Update Self Version Duration",
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
        }
      ],
      "repeat": null,
      "repeatIteration": null,
      "repeatRowId": null,
      "showTitle": true,
      "title": "DDL",
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
          "id": 46,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.95, sum(rate(tidb_statistics_auto_analyze_duration_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "auto analyze duration",
              "refId": "A",
              "step": 30
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Auto Analyze Duration 95",
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
          "id": 47,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tidb_statistics_auto_analyze_total[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 30
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Auto Analyze QPS",
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
          "id": 70,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tidb_statistics_stats_inaccuracy_rate_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99",
              "refId": "A",
              "step": 30
            },
            {
              "expr": "histogram_quantile(0.90, sum(rate(tidb_statistics_stats_inaccuracy_rate_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "90",
              "refId": "B"
            },
            {
              "expr": "histogram_quantile(0.50, sum(rate(tidb_statistics_stats_inaccuracy_rate_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "50",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Stats Inaccuracy Rate",
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
          "id": 71,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tidb_statistics_pseudo_estimation_total[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "A",
              "step": 30
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Pseudo Estimation OPS",
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
          "id": 92,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tidb_statistics_dump_feedback_total[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 30
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Dump Feedback OPS",
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
          "id": 93,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tidb_statistics_update_stats_total[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A",
              "step": 30
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Update Stats OPS",
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
          "description": "Counter of query feedback whose actual count is much different than calculated by current statistics",
          "fill": 1,
          "id": 113,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tidb_statistics_high_error_rate_feedback_total[1m]))",
              "format": "time_series",
              "hide": false,
              "intervalFactor": 2,
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Significant Feedback",
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
      "title": "Statistics",
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
          "id": 50,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(rate(tidb_autoid_operation_duration_seconds_count[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "AutoID QPS",
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
          "id": 51,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tidb_autoid_operation_duration_seconds_bucket[1m])) by (le, type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "99-{{`{{`}}type}}",
              "refId": "B"
            },
            {
              "expr": "histogram_quantile(0.80, sum(rate(tidb_autoid_operation_duration_seconds_bucket[1m])) by (le, type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "80-{{`{{`}}type}}",
              "refId": "C"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "AutoID Duration",
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
              "logBase": 2,
              "max": null,
              "min": "0.001",
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
          "id": 52,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tidb_meta_operation_duration_seconds_bucket[1m])) by (le, type))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Meta Operations Duration 99",
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
      "title": "Meta",
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
          "id": 85,
          "legend": {
            "alignAsTable": true,
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "rightSide": true,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(tidb_tikvclient_gc_worker_actions_total[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Worker Action OPM",
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
          "id": 86,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "histogram_quantile(0.99, sum(rate(tidb_tikvclient_gc_seconds_bucket[1m])) by (le))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Duration 99",
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
          "id": 87,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "max(tidb_tikvclient_gc_config) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Config",
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
          "id": 88,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(tidb_tikvclient_gc_failure[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "GC Failure OPM",
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
          "id": 89,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(tidb_tikvclient_gc_action_result[1m])) by (type)",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "{{`{{`}}type}}",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Action Result OPM",
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
          "id": 90,
          "legend": {
            "avg": false,
            "current": false,
            "max": false,
            "min": false,
            "show": true,
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
          "seriesOverrides": [],
          "spaceLength": 10,
          "span": 4,
          "stack": false,
          "steppedLine": false,
          "targets": [
            {
              "expr": "sum(increase(tidb_tikvclient_gc_region_too_many_locks[1m]))",
              "format": "time_series",
              "intervalFactor": 2,
              "legendFormat": "",
              "refId": "A"
            }
          ],
          "thresholds": [],
          "timeFrom": null,
          "timeShift": null,
          "title": "Too Many Locks Error OPM",
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
      "title": "GC",
      "titleSize": "h6"
    }
  ],
  "schemaVersion": 14,
  "style": "dark",
  "tags": [],
  "templating": {
    "list": []
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
  "title": "TIDB-Cluster-TiDB",
  "version": 3
}
{{- end -}}