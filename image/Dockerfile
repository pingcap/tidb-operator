# syntax=docker.io/docker/dockerfile:1.8.1@sha256:e87caa74dcb7d46cd820352bfea12591f3dba3ddc4285e19c7dcd13359f7cefd
FROM --platform=$BUILDPLATFORM gcr.io/pingcap-public/third-party/baseimage/golang:1.23 AS builder

ARG TARGETPLATFORM

WORKDIR /

ENV GOMODCACHE=/go/pkg/mod
ENV GOCACHE=/go/cache

COPY go.mod .
COPY go.sum .
COPY apis/core/go.mod apis/core/go.mod
COPY apis/core/go.sum apis/core/go.sum

RUN --mount=type=cache,target=/go/pkg/mod/ \
    go mod download -x
RUN --mount=type=cache,target=/go/pkg/mod/ \
    cd apis/core && go mod download -x

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target=/go/cache \
    V_PLATFORMS=$TARGETPLATFORM ./hack/build.sh

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.9.2 AS operator

ARG TARGETPLATFORM

WORKDIR /

COPY --from=builder ./_output/$TARGETPLATFORM/bin/operator operator

# nonroot user of distroless
USER 65532:65532

ENTRYPOINT ["/operator"]

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.9.2 AS prestop-checker

ARG TARGETPLATFORM

WORKDIR /

COPY --from=builder ./_output/$TARGETPLATFORM/bin/prestop-checker prestop-checker

# nonroot user of distroless
USER 65532:65532

ENTRYPOINT ["/prestop-checker"]

FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.9.2 AS testing-workload

ARG TARGETPLATFORM

WORKDIR /

COPY --from=builder ./_output/$TARGETPLATFORM/bin/testing-workload testing-workload

# nonroot user of distroless
USER 65532:65532

ENTRYPOINT ["/testing-workload"]