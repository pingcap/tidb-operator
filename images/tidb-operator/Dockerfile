FROM ghcr.io/pingcap-qe/bases/pingcap-base:v1.12.2@sha256:07772021ca1a9e31f6f89c745c15a577a1cd752da3c6df0f64e9453b8091cfb9

ARG TARGETARCH
RUN dnf install -y tzdata bind-utils && dnf clean all
ADD bin/${TARGETARCH}/tidb-scheduler /usr/local/bin/tidb-scheduler
ADD bin/${TARGETARCH}/tidb-discovery /usr/local/bin/tidb-discovery
ADD bin/${TARGETARCH}/tidb-controller-manager /usr/local/bin/tidb-controller-manager
ADD bin/${TARGETARCH}/tidb-admission-webhook /usr/local/bin/tidb-admission-webhook
