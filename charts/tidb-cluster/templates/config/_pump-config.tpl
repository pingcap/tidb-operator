# pump Configuration.

# addr(i.e. 'host:port') to listen on for client traffic
addr = "0.0.0.0:8250"

# addr(i.e. 'host:port') to advertise to the public
advertise-addr = ""

# a integer value to control expiry date of the binlog data, indicates for how long (in days) the binlog data would be stored.
# must bigger than 0
gc = {{ .Values.binlog.pump.gc | default 7 }}

# path to the data directory of pump's data
data-dir = "/data"

# number of seconds between heartbeat ticks (in 2 seconds)
heartbeat-interval = {{ .Values.binlog.pump.heartbeatInterval | default 2 }}

# a comma separated list of PD endpoints
pd-urls = "{{ template "cluster.scheme" . }}://{{ template "cluster.name" . }}-pd:2379"

[storage]
# Set to `true` (default) for best reliability, which prevents data loss when there is a power failure.
sync-log = {{ .Values.binlog.pump.syncLog | default true }}
#
# we suggest using the default config of the embedded LSM DB now, do not change it unless you know what you are doing
# [storage.kv]
# block-cache-capacity = 8388608
# block-restart-interval = 16
# block-size = 4096
# compaction-L0-trigger = 8
# compaction-table-size = 67108864
# compaction-total-size = 536870912
# compaction-total-size-multiplier = 8.0
# write-buffer = 67108864
# write-L0-pause-trigger = 24
# write-L0-slowdown-trigger = 17
{{ if and .Values.tlsCluster .Values.tlsCluster.enabled }}
[security]
# Path of file that contains list of trusted SSL CAs for connection with cluster components.
ssl-ca = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
# Path of file that contains X509 certificate in PEM format for connection with cluster components.
ssl-cert = "/var/lib/pump-tls/tls.crt"
# Path of file that contains X509 key in PEM format for connection with cluster components.
ssl-key = "/var/lib/pump-tls/tls.key"
{{- end -}}

