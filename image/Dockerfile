# syntax=docker.io/docker/dockerfile:1.8.1@sha256:e87caa74dcb7d46cd820352bfea12591f3dba3ddc4285e19c7dcd13359f7cefd
FROM --platform=$BUILDPLATFORM gcr.io/pingcap-public/third-party/baseimage/golang:1.23@sha256:080fb3462a60d926829ddcf0626acf0533eeeee934fea6c160d85565281ddc0a AS builder

ARG TARGETPLATFORM
ARG TARGET

WORKDIR /

ENV GOMODCACHE=/go/pkg/mod
ENV GOCACHE=/go/cache

COPY go.mod .
COPY go.sum .
COPY api/go.mod api/go.mod
COPY api/go.sum api/go.sum

RUN --mount=type=cache,target=/go/pkg/mod/ \
    go mod download -x
RUN --mount=type=cache,target=/go/pkg/mod/ \
    cd api && go mod download -x

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target=/go/cache \
    V_PLATFORMS=$TARGETPLATFORM ./hack/build.sh $TARGET


FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.10.0@sha256:af691c27330cb8e018add3dd16e5403dcd881b23eae815920ddb2c6ae87cbd9b AS tidb-operator

ARG TARGETPLATFORM

USER pingcap:pingcap

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder ./_output/$TARGETPLATFORM/bin/tidb-operator tidb-operator

ENTRYPOINT ["/tidb-operator"]

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.10.0@sha256:af691c27330cb8e018add3dd16e5403dcd881b23eae815920ddb2c6ae87cbd9b AS prestop-checker

ARG TARGETPLATFORM

USER pingcap:pingcap

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder ./_output/$TARGETPLATFORM/bin/prestop-checker prestop-checker

ENTRYPOINT ["/prestop-checker"]

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.10.0@sha256:af691c27330cb8e018add3dd16e5403dcd881b23eae815920ddb2c6ae87cbd9b AS testing-workload

ARG TARGETPLATFORM

USER pingcap:pingcap

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder ./_output/$TARGETPLATFORM/bin/testing-workload testing-workload


ENTRYPOINT ["/testing-workload"]

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.10.0@sha256:af691c27330cb8e018add3dd16e5403dcd881b23eae815920ddb2c6ae87cbd9b AS tidb-backup-manager

ARG TARGETPLATFORM

USER pingcap:pingcap

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder ./_output/$TARGETPLATFORM/bin/tidb-backup-manager tidb-backup-manager

ENTRYPOINT ["/tidb-backup-manager"]
