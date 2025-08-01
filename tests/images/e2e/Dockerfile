FROM debian:bookworm-slim

ENV KUBECTL_VERSION=v1.28.5
ENV HELM_VERSION=v3.11.0

ARG TARGETARCH

# python is required by gcloud
RUN apt-get update && \
    apt-get install -y ca-certificates curl git openssl default-mysql-client unzip && \
    apt-get install -y python3 && \
    apt-get autoremove -y && \
    apt-get clean -y

RUN curl https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/${TARGETARCH}/kubectl \
    -o /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl && \
    curl https://get.helm.sh/helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz \
    -o helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz && \
    tar -zxvf helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz && \
    mv linux-${TARGETARCH}/helm /usr/local/bin/helm && \
    rm -rf linux-${TARGETARCH} && \
    rm helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && \
    unzip awscliv2.zip && \
    ./aws/install && \
    rm -r aws awscliv2.zip
RUN curl -L "https://github.com/jetstack/cert-manager/releases/download/v1.15.1/cert-manager.yaml" -o "/cert-manager.yaml"
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
