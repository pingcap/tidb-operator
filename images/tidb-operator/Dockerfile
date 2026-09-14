FROM ghcr.io/pingcap-qe/bases/pingcap-base:v1.12.3@sha256:eee0c7a72ce382ec2bca340c6ab4c996e56a5d9deb3745ebcbe43d65bec86b88

ARG TARGETARCH
RUN dnf install -y tzdata bind-utils && dnf clean all
ADD bin/${TARGETARCH}/tidb-scheduler /usr/local/bin/tidb-scheduler
ADD bin/${TARGETARCH}/tidb-discovery /usr/local/bin/tidb-discovery
ADD bin/${TARGETARCH}/tidb-controller-manager /usr/local/bin/tidb-controller-manager
ADD bin/${TARGETARCH}/tidb-admission-webhook /usr/local/bin/tidb-admission-webhook
