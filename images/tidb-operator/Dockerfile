FROM alpine:3.5

RUN apk add tzdata --no-cache
ADD bin/tidb-controller-manager /usr/local/bin/tidb-controller-manager
ADD bin/tidb-scheduler /usr/local/bin/tidb-scheduler
ADD bin/tidb-discovery /usr/local/bin/tidb-discovery
ADD bin/tidb-admission-controller /usr/local/bin/tidb-admission-controller
