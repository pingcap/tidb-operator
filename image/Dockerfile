# syntax=docker.io/docker/dockerfile:1.8.1@sha256:e87caa74dcb7d46cd820352bfea12591f3dba3ddc4285e19c7dcd13359f7cefd
FROM --platform=$BUILDPLATFORM golang:1.25.9@sha256:a95d3d115ec2643a014c9539962dcecba297fb4ed5e5cbfab84cf726327e8b8c AS builder

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

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.2@sha256:4ac362fe93a1067f406b13345b4ab6929f1fdbf3ded3cbecccb7a57730a5ce0f AS tidb-operator

ARG TARGETPLATFORM

# Use numeric UID:GID instead of username for Kubernetes compatibility.
# Base image defines: PINGCAP_UID=1000, PINGCAP_GID=2000 (pingcap:pingcap)
# Kubernetes runAsNonRoot requires numeric user to verify non-root execution.
# Reference: https://github.com/PingCAP-QE/artifacts/blob/main/dockerfiles/bases/pingcap-base/Dockerfile
USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/tidb-operator tidb-operator

ENTRYPOINT ["/tidb-operator"]

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.2@sha256:4ac362fe93a1067f406b13345b4ab6929f1fdbf3ded3cbecccb7a57730a5ce0f AS prestop-checker

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/prestop-checker prestop-checker

ENTRYPOINT ["/prestop-checker"]

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.2@sha256:4ac362fe93a1067f406b13345b4ab6929f1fdbf3ded3cbecccb7a57730a5ce0f AS testing-workload

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/testing-workload testing-workload


ENTRYPOINT ["/testing-workload"]


FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.2@sha256:4ac362fe93a1067f406b13345b4ab6929f1fdbf3ded3cbecccb7a57730a5ce0f AS tidb-backup-manager

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/tidb-backup-manager tidb-backup-manager

ENTRYPOINT ["/tidb-backup-manager"]


FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.2@sha256:4ac362fe93a1067f406b13345b4ab6929f1fdbf3ded3cbecccb7a57730a5ce0f AS resource-syncer

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/resource-syncer resource-syncer

ENTRYPOINT ["/resource-syncer"]
