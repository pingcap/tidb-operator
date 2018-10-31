# pump Configuration.

# addr(i.e. 'host:port') to listen on for client traffic
addr = "0.0.0.0:8250"

# addr(i.e. 'host:port') to advertise to the public
# advertise-addr = ""

# a integer value to control expiry date of the binlog data, indicates for how long (in days) the binlog data would be stored.
# must bigger than 0
gc = {{ .Values.binlog.pump.gc | default 7 }}

# path to the data directory of pump's data
data-dir = "/data"

# number of seconds between heartbeat ticks (in 2 seconds)
heartbeat-interval = {{ .Values.binlog.pump.heartbeatInterval | default 2 }}

# a comma separated list of PD endpoints
pd-urls = "http://{{ .Values.clusterName }}-pd:2379"

#[security]
# Path of file that contains list of trusted SSL CAs for connection with cluster components.
# ssl-ca = "/path/to/ca.pem"
# Path of file that contains X509 certificate in PEM format for connection with cluster components.
# ssl-cert = "/path/to/drainer.pem"
# Path of file that contains X509 key in PEM format for connection with cluster components.
# ssl-key = "/path/to/drainer-key.pem"
