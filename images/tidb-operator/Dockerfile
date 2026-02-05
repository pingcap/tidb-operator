FROM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.1@sha256:24ecefceae6b114e11f6ae675232af3fa4942c53011f807983ac5bff7a62e3d9

ARG TARGETARCH
RUN dnf install -y tzdata bind-utils && dnf clean all
ADD bin/${TARGETARCH}/tidb-scheduler /usr/local/bin/tidb-scheduler
ADD bin/${TARGETARCH}/tidb-discovery /usr/local/bin/tidb-discovery
ADD bin/${TARGETARCH}/tidb-controller-manager /usr/local/bin/tidb-controller-manager
ADD bin/${TARGETARCH}/tidb-admission-webhook /usr/local/bin/tidb-admission-webhook
