FROM debian:bookworm-slim

ENV KUBECTL_VERSION=v1.28.5
ENV HELM_VERSION=v3.11.0
ENV CERT_MANAGER_VERSION=v1.15.1

ARG TARGETARCH

RUN apt-get update && \
    apt-get install -y ca-certificates curl git openssl default-mysql-client unzip python3 && \
    apt-get autoremove -y && \
    apt-get clean -y && \
    rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    curl -fsSL "https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/${TARGETARCH}/kubectl" -o /usr/local/bin/kubectl; \
    chmod +x /usr/local/bin/kubectl; \
    curl -fsSL "https://get.helm.sh/helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz" -o "helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz"; \
    tar -zxvf "helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz"; \
    mv "linux-${TARGETARCH}/helm" /usr/local/bin/helm; \
    rm -rf "linux-${TARGETARCH}" "helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz"

RUN set -eux; \
    case "${TARGETARCH}" in \
      amd64) aws_arch=x86_64 ;; \
      arm64) aws_arch=aarch64 ;; \
      *) echo "unsupported TARGETARCH=${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-${aws_arch}.zip" -o awscliv2.zip; \
    unzip awscliv2.zip; \
    ./aws/install; \
    rm -rf aws awscliv2.zip

RUN curl -fsSL "https://github.com/jetstack/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml" -o /cert-manager.yaml

RUN set -eux; \
    kubectl version --client=true --output=yaml; \
    helm version --short; \
    aws --version; \
    test -s /cert-manager.yaml
