# syntax=docker.io/docker/dockerfile:1.8.1@sha256:e87caa74dcb7d46cd820352bfea12591f3dba3ddc4285e19c7dcd13359f7cefd
FROM --platform=$BUILDPLATFORM golang:1.26.1@sha256:c7e98cc0fd4dfb71ee7465fee6c9a5f079163307e4bf141b336bb9dae00159a5 AS builder

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

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.1@sha256:bf0bb17bc184d2726c8eb32d9da3b70c316858ad916a3ab3f1fcf90cf9eccd16 AS tidb-operator

ARG TARGETPLATFORM

# Use numeric UID:GID instead of username for Kubernetes compatibility.
# Base image defines: PINGCAP_UID=1000, PINGCAP_GID=2000 (pingcap:pingcap)
# Kubernetes runAsNonRoot requires numeric user to verify non-root execution.
# Reference: https://github.com/PingCAP-QE/artifacts/blob/main/dockerfiles/bases/pingcap-base/Dockerfile
USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/tidb-operator tidb-operator

ENTRYPOINT ["/tidb-operator"]

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.1@sha256:bf0bb17bc184d2726c8eb32d9da3b70c316858ad916a3ab3f1fcf90cf9eccd16 AS prestop-checker

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/prestop-checker prestop-checker

ENTRYPOINT ["/prestop-checker"]

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.1@sha256:bf0bb17bc184d2726c8eb32d9da3b70c316858ad916a3ab3f1fcf90cf9eccd16 AS testing-workload

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/testing-workload testing-workload


ENTRYPOINT ["/testing-workload"]


FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.1@sha256:bf0bb17bc184d2726c8eb32d9da3b70c316858ad916a3ab3f1fcf90cf9eccd16 AS tidb-backup-manager

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/tidb-backup-manager tidb-backup-manager

ENTRYPOINT ["/tidb-backup-manager"]


FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.1@sha256:bf0bb17bc184d2726c8eb32d9da3b70c316858ad916a3ab3f1fcf90cf9eccd16 AS resource-syncer

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/resource-syncer resource-syncer

ENTRYPOINT ["/resource-syncer"]
