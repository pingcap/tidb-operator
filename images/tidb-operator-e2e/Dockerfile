# Builder image
FROM golang:1.10-alpine AS builder

# Uncomment these lines to hack the repositories
# RUN echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.3/main" > /etc/apk/repositories  && \
#    echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.5/community" >> /etc/apk/repositories

RUN apk update && apk add --no-cache git
RUN go get github.com/onsi/ginkgo/ginkgo

# Executable image
FROM alpine:3.5

ENV KUBECTL_VERSION=v1.10.2
ENV HELM_VERSION=v2.9.1

# Uncomment these lines to hack the repositories
# RUN echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.3/main" > /etc/apk/repositories  && \
#    echo "https://mirrors.tuna.tsinghua.edu.cn/alpine/v3.5/community" >> /etc/apk/repositories

RUN apk update && apk add --no-cache ca-certificates curl

COPY --from=builder /go/bin/ginkgo /usr/local/bin/ginkgo

RUN curl https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl \
    -o /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl && \
    curl https://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz \
    -o helm-${HELM_VERSION}-linux-amd64.tar.gz && \
    tar -zxvf helm-${HELM_VERSION}-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/local/bin/helm && \
    rm -rf linux-amd64 && \
    rm helm-${HELM_VERSION}-linux-amd64.tar.gz

ADD bin/e2e.test /usr/local/bin/e2e.test

ADD tidb-operator /charts/tidb-operator
ADD tidb-cluster /charts/tidb-cluster

ADD tidb-cluster-values.yaml /tidb-cluster-values.yaml
ADD tidb-operator-values.yaml /tidb-operator-values.yaml
