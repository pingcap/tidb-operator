# syntax=docker.io/docker/dockerfile:1.8.1@sha256:e87caa74dcb7d46cd820352bfea12591f3dba3ddc4285e19c7dcd13359f7cefd
FROM --platform=$BUILDPLATFORM golang:1.27.3@sha256:f20d83a83e18dbd2a4714b8ffd4b30b287b6243962dd75608e86abcc63ba4efc AS builder

ARG TARGETPLATFORM
ARG TARGET

WORKDIR /data

SHELL ["/bin/bash", "-c"]

ENV GOMODCACHE=/go/pkg/mod
ENV GOCACHE=/go/cache

COPY go.mod .
COPY go.sum .
COPY api/go.mod api/go.mod
COPY api/go.sum api/go.sum

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod/ \
    go mod download -x
RUN --mount=type=cache,target=/go/pkg/mod/ \
    cd api && go mod download -x

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target=/go/cache \
    V_PLATFORMS=$TARGETPLATFORM ./hack/build.sh $TARGET

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.12.5@sha256:715150b855957b5a007b006240b2735b3e645c5a471d2391ad9ffa4aafdb93fd AS tidb-operator

ARG TARGETPLATFORM

# Use numeric UID:GID instead of username for Kubernetes compatibility.
# Base image defines: PINGCAP_UID=1000, PINGCAP_GID=2000 (pingcap:pingcap)
# Kubernetes runAsNonRoot requires numeric user to verify non-root execution.
# Reference: https://github.com/PingCAP-QE/artifacts/blob/main/dockerfiles/bases/pingcap-base/Dockerfile
USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/tidb-operator tidb-operator

ENTRYPOINT ["/tidb-operator"]

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.12.5@sha256:715150b855957b5a007b006240b2735b3e645c5a471d2391ad9ffa4aafdb93fd AS prestop-checker

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/prestop-checker prestop-checker

ENTRYPOINT ["/prestop-checker"]

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.12.5@sha256:715150b855957b5a007b006240b2735b3e645c5a471d2391ad9ffa4aafdb93fd AS testing-workload

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/testing-workload testing-workload


ENTRYPOINT ["/testing-workload"]


FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.12.5@sha256:715150b855957b5a007b006240b2735b3e645c5a471d2391ad9ffa4aafdb93fd AS tidb-backup-manager

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/tidb-backup-manager tidb-backup-manager

ENTRYPOINT ["/tidb-backup-manager"]


FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.12.5@sha256:715150b855957b5a007b006240b2735b3e645c5a471d2391ad9ffa4aafdb93fd AS resource-syncer

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/resource-syncer resource-syncer

ENTRYPOINT ["/resource-syncer"]
