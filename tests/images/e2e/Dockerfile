FROM debian:bookworm-slim

ENV KUBECTL_VERSION=v1.28.5
ENV HELM_VERSION=v3.11.0

ARG TARGETARCH
ARG CURL_RETRY_OPTS="--fail --location --connect-timeout 30 --max-time 180"

# python is required by gcloud
RUN apt-get update && \
    apt-get install -y ca-certificates curl git openssl default-mysql-client unzip && \
    apt-get install -y python3 && \
    apt-get autoremove -y && \
    apt-get clean -y

RUN for i in 1 2 3 4 5; do \
        curl ${CURL_RETRY_OPTS} https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/${TARGETARCH}/kubectl \
            -o /usr/local/bin/kubectl && break; \
        if [ "$i" = "5" ]; then exit 1; fi; \
        sleep 5; \
    done && \
    chmod +x /usr/local/bin/kubectl && \
    for url in https://get.helm.sh/helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz https://mirrors.huaweicloud.com/helm/${HELM_VERSION}/helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz; do \
        for i in 1 2 3; do \
            curl ${CURL_RETRY_OPTS} "$url" -o helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz && helm_downloaded=1 && break 2; \
            sleep 5; \
        done; \
    done && \
    test "${helm_downloaded:-}" = "1" && \
    tar -zxvf helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz && \
    mv linux-${TARGETARCH}/helm /usr/local/bin/helm && \
    rm -rf linux-${TARGETARCH} && \
    rm helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz
RUN for i in 1 2 3 4 5; do \
        curl ${CURL_RETRY_OPTS} "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && break; \
        if [ "$i" = "5" ]; then exit 1; fi; \
        sleep 5; \
    done && \
    unzip awscliv2.zip && \
    ./aws/install && \
    rm -r aws awscliv2.zip
RUN for i in 1 2 3 4 5; do \
        curl ${CURL_RETRY_OPTS} "https://github.com/jetstack/cert-manager/releases/download/v1.15.1/cert-manager.yaml" -o "/cert-manager.yaml" && break; \
        if [ "$i" = "5" ]; then exit 1; fi; \
        sleep 5; \
    done
ADD minio /minio

ADD tidb-operator /charts/e2e/tidb-operator
ADD tidb-drainer /charts/e2e/tidb-drainer
ADD manifests /manifests/e2e

ADD bin/ginkgo /usr/local/bin/
ADD bin/e2e.test /usr/local/bin/
ADD bin/webhook /usr/local/bin/
ADD bin/blockwriter /usr/local/bin/

COPY tikv.toml /etc/tikv.toml

ADD entrypoint.sh /usr/local/bin
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
