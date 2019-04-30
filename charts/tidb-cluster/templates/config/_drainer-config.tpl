# drainer Configuration.

# addr (i.e. 'host:port') to listen on for drainer connections
# will register this addr into etcd
# addr = "127.0.0.1:8249"

# the interval time (in seconds) of detect pumps' status
detect-interval = {{ .Values.binlog.drainer.detectInterval | default 10 }}

# drainer meta data directory path
data-dir = "/data"

# a comma separated list of PD endpoints
pd-urls = "http://{{ template "cluster.name" . }}-pd:2379"

# Use the specified compressor to compress payload between pump and drainer
compressor = ""

#[security]
# Path of file that contains list of trusted SSL CAs for connection with cluster components.
# ssl-ca = "/path/to/ca.pem"
# Path of file that contains X509 certificate in PEM format for connection with cluster components.
# ssl-cert = "/path/to/pump.pem"
# Path of file that contains X509 key in PEM format for connection with cluster components.
# ssl-key = "/path/to/pump-key.pem"

# syncer Configuration.
[syncer]

# Assume the upstream sql-mode.
# If this is set , will use the same sql-mode to parse DDL statment, and set the same sql-mode at downstream when db-type is mysql.
# The default value will not set any sql-mode.
# sql-mode = "STRICT_TRANS_TABLES,NO_ENGINE_SUBSTITUTION"

# number of binlog events in a transaction batch
txn-batch = {{ .Values.binlog.drainer.txnBatch | default 20 }}

# work count to execute binlogs
# if the latency between drainer and downstream(mysql or tidb) are too high, you might want to increase this
# to get higher throughput by higher concurrent write to the downstream
worker-count = {{ .Values.binlog.drainer.workerCount | default 16 }}

disable-dispatch = {{ .Values.binlog.drainer.disableDispatch | default false }}

# safe mode will split update to delete and insert
safe-mode = {{ .Values.binlog.drainer.safeMode | default false }}

# downstream storage, equal to --dest-db-type
# valid values are "mysql", "pb", "tidb", "flash", "kafka"
db-type = "{{ .Values.binlog.drainer.destDBType }}"

# disable sync these schema
ignore-schemas = {{ .Values.binlog.drainer.ignoreSchemas | default "INFORMATION_SCHEMA,PERFORMANCE_SCHEMA,mysql" | quote }}

##replicate-do-db priority over replicate-do-table if have same db name
##and we support regex expression , start with '~' declare use regex expression.
#
#replicate-do-db = ["~^b.*","s1"]

#[[syncer.replicate-do-table]]
#db-name ="test"
#tbl-name = "log"

#[[syncer.replicate-do-table]]
#db-name ="test"
#tbl-name = "~^a.*"

# disable sync these table
#[[syncer.ignore-table]]
#db-name = "test"
#tbl-name = "log"

{{- if eq .Values.binlog.drainer.destDBType "mysql" }}
# the downstream mysql protocol database
[syncer.to]
host = {{ .Values.binlog.drainer.mysql.host | quote }}
user = {{ .Values.binlog.drainer.mysql.user | default "root" | quote }}
password = {{ .Values.binlog.drainer.mysql.password | quote }}
port = {{ .Values.binlog.drainer.mysql.port | default 3306 }}

[syncer.to.checkpoint]
# you can uncomment this to change the database to save checkpoint when the downstream is mysql or tidb
#schema = "tidb_binlog"
{{- end }}

{{- if eq .Values.binlog.drainer.destDBType "pb" }}
# Uncomment this if you want to use pb or sql as db-type.
# Compress compresses output file, like pb and sql file. Now it supports "gzip" algorithm only.
# Values can be "gzip". Leave it empty to disable compression.
[syncer.to]
# directory to save pb file, default same as data-dir(save checkpoint file) if this is not configured.
dir = "/data/pb"
compression = "gzip"
{{- end }}

{{- if eq .Values.binlog.drainer.destDBType "kafka" }}
# when db-type is kafka, you can uncomment this to config the down stream kafka, it will be the globle config kafka default
[syncer.to]
# only need config one of zookeeper-addrs and kafka-addrs, will get kafka address if zookeeper-addrs is configed.
{{- if .Values.binlog.drainer.kafka.zookeeperAddrs }}
zookeeper-addrs = {{ .Values.binlog.drainer.kafka.zookeeperAddrs | quote }}
{{- end }}
{{- if .Values.binlog.drainer.kafka.kafkaAddrs }}
kafka-addrs = {{ .Values.binlog.drainer.kafka.kafkaAddrs | quote }}
{{- end }}
kafka-version = {{ .Values.binlog.drainer.kafka.kafkaVersion | default "0.8.2.0" | quote }}
kafka-max-messages = 1024
#
#
# the topic name drainer will push msg, the default name is <cluster-id>_obinlog
# be careful don't use the same name if run multi drainer instances
# topic-name = ""
{{- end -}}
