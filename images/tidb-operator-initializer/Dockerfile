FROM alpine:3.5

ARG KUBE_VERSION="1.11.1"

RUN apk add --update ca-certificates && \
    apk add --update -t deps curl && \
    curl -L https://storage.googleapis.com/kubernetes-release/release/v$KUBE_VERSION/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl && \
    apk del --purge deps && \
    rm /var/cache/apk/*

# install openssl
RUN apk add --update openssl && \
    rm -rf /var/cache/apk/*

RUN apk add tzdata --no-cache
ADD bin/tidb-initializer /usr/local/bin/tidb-initializer
