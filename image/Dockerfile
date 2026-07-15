# syntax=docker.io/docker/dockerfile:1.8.1@sha256:e87caa74dcb7d46cd820352bfea12591f3dba3ddc4285e19c7dcd13359f7cefd
FROM --platform=$BUILDPLATFORM golang:1.25.10@sha256:c0a2bd0756d92462a0d449124b039100ce447ebf69dc6c80a6d877503b36935e AS builder

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

<<<<<<< HEAD
FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.3@sha256:d062f058122370102e55fab5dd25b89f67a8bd27ad18c70197fc348c1653e596 AS tidb-operator
=======
FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.12.0@sha256:a9c4c9922df8ff8581fdc0ccdf77fe0bb6646d7852e85bda3ef4c80bc368f861 AS tidb-operator
>>>>>>> 110697e96 (chore(deps): bump pingcap-qe/bases/pingcap-base from v1.11.4 to v1.12.0 in /image (#6997))

ARG TARGETPLATFORM

# Use numeric UID:GID instead of username for Kubernetes compatibility.
# Base image defines: PINGCAP_UID=1000, PINGCAP_GID=2000 (pingcap:pingcap)
# Kubernetes runAsNonRoot requires numeric user to verify non-root execution.
# Reference: https://github.com/PingCAP-QE/artifacts/blob/main/dockerfiles/bases/pingcap-base/Dockerfile
USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/tidb-operator tidb-operator

ENTRYPOINT ["/tidb-operator"]

<<<<<<< HEAD
FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.3@sha256:d062f058122370102e55fab5dd25b89f67a8bd27ad18c70197fc348c1653e596 AS prestop-checker
=======
FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.12.0@sha256:a9c4c9922df8ff8581fdc0ccdf77fe0bb6646d7852e85bda3ef4c80bc368f861 AS prestop-checker
>>>>>>> 110697e96 (chore(deps): bump pingcap-qe/bases/pingcap-base from v1.11.4 to v1.12.0 in /image (#6997))

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/prestop-checker prestop-checker

ENTRYPOINT ["/prestop-checker"]

<<<<<<< HEAD
FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.3@sha256:d062f058122370102e55fab5dd25b89f67a8bd27ad18c70197fc348c1653e596 AS testing-workload
=======
FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.12.0@sha256:a9c4c9922df8ff8581fdc0ccdf77fe0bb6646d7852e85bda3ef4c80bc368f861 AS testing-workload
>>>>>>> 110697e96 (chore(deps): bump pingcap-qe/bases/pingcap-base from v1.11.4 to v1.12.0 in /image (#6997))

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/testing-workload testing-workload


ENTRYPOINT ["/testing-workload"]


<<<<<<< HEAD
FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.3@sha256:d062f058122370102e55fab5dd25b89f67a8bd27ad18c70197fc348c1653e596 AS tidb-backup-manager
=======
FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.12.0@sha256:a9c4c9922df8ff8581fdc0ccdf77fe0bb6646d7852e85bda3ef4c80bc368f861 AS tidb-backup-manager
>>>>>>> 110697e96 (chore(deps): bump pingcap-qe/bases/pingcap-base from v1.11.4 to v1.12.0 in /image (#6997))

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/tidb-backup-manager tidb-backup-manager

ENTRYPOINT ["/tidb-backup-manager"]


<<<<<<< HEAD
FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.3@sha256:d062f058122370102e55fab5dd25b89f67a8bd27ad18c70197fc348c1653e596 AS resource-syncer
=======
FROM --platform=$TARGETPLATFORM ghcr.io/pingcap-qe/bases/pingcap-base:v1.12.0@sha256:a9c4c9922df8ff8581fdc0ccdf77fe0bb6646d7852e85bda3ef4c80bc368f861 AS resource-syncer
>>>>>>> 110697e96 (chore(deps): bump pingcap-qe/bases/pingcap-base from v1.11.4 to v1.12.0 in /image (#6997))

ARG TARGETPLATFORM

USER 1000:2000

WORKDIR /

COPY --chown=pingcap:pingcap --from=builder /data/_output/$TARGETPLATFORM/bin/resource-syncer resource-syncer

ENTRYPOINT ["/resource-syncer"]
