# syntax=docker.io/docker/dockerfile:1.8.1@sha256:e87caa74dcb7d46cd820352bfea12591f3dba3ddc4285e19c7dcd13359f7cefd
FROM --platform=$BUILDPLATFORM gcr.io/pingcap-public/third-party/baseimage/golang:1.23@sha256:60deed95d3888cc5e4d9ff8a10c54e5edc008c6ae3fba6187be6fb592e19e8c0 AS builder

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

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.10.0@sha256:af691c27330cb8e018add3dd16e5403dcd881b23eae815920ddb2c6ae87cbd9b AS tidb-operator

ARG TARGETPLATFORM

# Use numeric UID:GID instead of username for Kubernetes compatibility.
# Base image defines: PINGCAP_UID=1000, PINGCAP_GID=2000 (pingcap:pingcap)
# Kubernetes runAsNonRoot requires numeric user to verify non-root execution.
# Reference: https://github.com/PingCAP-QE/artifacts/blob/main/dockerfiles/bases/pingcap-base/Dockerfile
USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/tidb-operator tidb-operator

ENTRYPOINT ["/tidb-operator"]

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.10.0@sha256:af691c27330cb8e018add3dd16e5403dcd881b23eae815920ddb2c6ae87cbd9b AS prestop-checker

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/prestop-checker prestop-checker

ENTRYPOINT ["/prestop-checker"]

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.10.0@sha256:af691c27330cb8e018add3dd16e5403dcd881b23eae815920ddb2c6ae87cbd9b AS testing-workload

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/testing-workload testing-workload


ENTRYPOINT ["/testing-workload"]


FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.10.0@sha256:af691c27330cb8e018add3dd16e5403dcd881b23eae815920ddb2c6ae87cbd9b AS tidb-backup-manager

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/tidb-backup-manager tidb-backup-manager

ENTRYPOINT ["/tidb-backup-manager"]
