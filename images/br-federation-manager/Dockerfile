FROM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.1@sha256:24ecefceae6b114e11f6ae675232af3fa4942c53011f807983ac5bff7a62e3d9
ARG TARGETARCH
RUN dnf install -y bind-utils tzdata && dnf clean all
ADD bin/${TARGETARCH}/br-federation-manager /usr/local/bin/br-federation-manager
