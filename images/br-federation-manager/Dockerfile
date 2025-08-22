FROM ghcr.io/pingcap-qe/bases/pingcap-base:v1.10.0
ARG TARGETARCH
RUN dnf install -y bind-utils tzdata && dnf clean all
ADD bin/${TARGETARCH}/br-federation-manager /usr/local/bin/br-federation-manager
