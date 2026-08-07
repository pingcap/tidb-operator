FROM ghcr.io/pingcap-qe/bases/pingcap-base:v1.12.2@sha256:07772021ca1a9e31f6f89c745c15a577a1cd752da3c6df0f64e9453b8091cfb9
ARG TARGETARCH
RUN dnf install -y bind-utils tzdata && dnf clean all
ADD bin/${TARGETARCH}/br-federation-manager /usr/local/bin/br-federation-manager
