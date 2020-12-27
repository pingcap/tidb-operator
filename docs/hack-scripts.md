# Hack Scripts

This documentation describes usage of common scripts under `hack/` dir.


## `gen-tls-certs.sh`

This script generates `cfssl` configs and certs according to [Enable TLS for the MySQL Client](https://docs.pingcap.com/tidb-in-kubernetes/stable/enable-tls-for-mysql-client) and [Enable TLS between TiDB Components](https://docs.pingcap.com/tidb-in-kubernetes/stable/enable-tls-between-components).
The generated artifacts are under `output/tls`, which is ignored by git.

There are several configurable variables at the top of the script, in which the most commonly changed ones are `TIDB_CLUSTER` and `NAMESPACE`.

Internally, this script will generate CA, certs for tidb cluster components mTLS, and certs for tidb server/client mTLS. Then it will create several secret objects using `kubectl`, so that `TidbCluster` components can use them afterwards.
