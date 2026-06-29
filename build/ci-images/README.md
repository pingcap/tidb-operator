# TiDB Operator CI base images

This directory contains Dockerfiles for pre-baked images used by `release-1.x` CI/CD.

The goal is to move flaky external binary downloads out of normal PR and release Docker builds. These images are built and pushed explicitly by `.github/workflows/ci-base-images.yaml`; normal CI should consume a pinned tag/digest instead of downloading tools during `make docker` or `make e2e-docker`.

## Images

### `backup-manager-base`

Base image for `images/tidb-backup-manager/Dockerfile` and `images/tidb-backup-manager/Dockerfile.e2e`.

Includes:

- `ghcr.io/pingcap-qe/bases/pingcap-base:v1.11.1`
- `ca-certificates`, `bind-utils`, `wget`, `nc`, `unzip`
- `rclone v1.71.2`
- `shush v1.5.5`

### `e2e-base`

Base image for `tests/images/e2e/Dockerfile`.

Includes:

- `debian:bookworm-slim`
- `ca-certificates`, `curl`, `git`, `openssl`, `default-mysql-client`, `unzip`, `python3`
- `kubectl v1.28.5`
- `helm v3.11.0`
- AWS CLI v2
- `/cert-manager.yaml` from cert-manager `v1.15.1`

## Publishing

Run the **CI Base Images** workflow manually with an immutable tag, for example:

```text
release-1.x-20260629
```

The workflow publishes:

```text
ghcr.io/pingcap/tidb-operator/backup-manager-base:<tag>
ghcr.io/pingcap/tidb-operator/e2e-base:<tag>
```

Before updating consuming Dockerfiles, verify the images are pullable by CI/Prow and record the resulting digest.

## Update policy

Only rebuild these images when one of the pinned inputs changes:

- base image or OS package set
- `rclone` version
- `shush` version
- `kubectl` version
- `helm` version
- AWS CLI installer behavior/version
- cert-manager manifest version

Do not use mutable `latest` tags in consuming Dockerfiles.
