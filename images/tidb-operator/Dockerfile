FROM alpine:3.5

ARG KUBE_VERSION=1.12.10

# install kubectl / openssl
RUN apk add --update ca-certificates && \
    apk add --update -t deps curl && \
    apk add --update openssl && \
    curl -L https://storage.googleapis.com/kubernetes-release/release/v$KUBE_VERSION/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl && \
    apk del --purge deps && \
    rm /var/cache/apk/*

RUN apk add tzdata --no-cache
ADD bin/tidb-scheduler /usr/local/bin/tidb-scheduler
ADD bin/tidb-discovery /usr/local/bin/tidb-discovery
ADD bin/tidb-admission-controller /usr/local/bin/tidb-admission-controller
ADD bin/tidb-controller-manager /usr/local/bin/tidb-controller-manager
ADD bin/tidb-apiserver /usr/local/bin/tidb-apiserver
ADD bin/tidb-operator-initializer /usr/local/bin/tidb-operator-initializer
ADD bin/cert-refresh /usr/local/bin/cert-refresh
