FROM alpine:3.10

RUN apk add tzdata --no-cache
ADD bin/tidb-scheduler /usr/local/bin/tidb-scheduler
ADD bin/tidb-discovery /usr/local/bin/tidb-discovery
ADD bin/tidb-controller-manager /usr/local/bin/tidb-controller-manager
ADD bin/tidb-apiserver /usr/local/bin/tidb-apiserver
ADD bin/tidb-admission-webhook /usr/local/bin/tidb-admission-webhook
