# TiCI Deployment on GCS: Two Auth Modes

This document describes two authentication modes for deploying TiCI on GCS with TiDB Operator:

1. Service account key via Kubernetes Secret (validated)
2. GKE Workload Identity (in validation)

Date: 2026-02-11

## Scope

- TiDB Operator deployment with TiCI, TiFlash, TiCDC
- GCS sink for TiCI/TiCDC
- Example manifests under `examples/tici/`

## Auth Mode Summary

| Mode | Manifest | Current Status | Credential Source |
| --- | --- | --- | --- |
| Secret-based | `tici-tc-gcs.yaml` | Validated | `Secret/gcs-sa` mounted file `credentials.json` |
| Workload Identity | `tici-tc-gcs-wi.yaml` | In validation | GKE metadata server + bound GSA |

## Common Prerequisites

- Kubernetes cluster with TiDB Operator installed
- StorageClass `standard` (or update manifests)
- GCS bucket created (for example: `tici-test`)
- TiDB/TiCI images available in your environment

## Mode 1: Service Account Secret (Validated)

### 1) Create bucket IAM permission

For smoke test, grant bucket-level `Storage Object Admin` to the GSA used in the key JSON.

Minimum permission set for normal write/read flow:
- `storage.objects.create`
- `storage.objects.get`
- `storage.objects.list`
- optionally `storage.objects.delete` if cleanup is needed

### 2) Create secret with fixed key name

The file name must be `credentials.json`.

```bash
kubectl -n <ns> create secret generic gcs-sa \
  --from-file=credentials.json=/path/to/your-service-account.json
```

### 3) Deploy cluster

```bash
kubectl -n <ns> apply -f examples/tici/tici-tc-gcs.yaml
```

### 4) Verify runtime credentials and components

```bash
kubectl -n <ns> get pod | grep tici-demo-gcs
kubectl -n <ns> exec tici-demo-gcs-ticdc-0 -c ticdc -- ls -l /var/secrets/google/credentials.json
kubectl -n <ns> exec tici-demo-gcs-tiflash-0 -c tiflash -- ls -l /var/secrets/google/credentials.json
kubectl -n <ns> logs tici-demo-gcs-ticdc-0 -c ticdc --tail=200
```

### 5) Verify data is written to GCS

```bash
gcloud storage ls gs://<bucket>/tici_default_prefix/cdc
```

## Mode 2: GKE Workload Identity (In Validation)

Use manifest: `examples/tici/tici-tc-gcs-wi.yaml`

This manifest binds WI ServiceAccount only to:
- `tiflash`
- `ticdc`
- `tici.meta`
- `tici.worker`

### 1) Create GSA and grant bucket access

```bash
gcloud iam service-accounts create tici-gcs-wi --project <project-id>
gcloud storage buckets add-iam-policy-binding gs://<bucket> \
  --member "serviceAccount:tici-gcs-wi@<project-id>.iam.gserviceaccount.com" \
  --role "roles/storage.objectAdmin"
```

### 2) Bind KSA to GSA for Workload Identity

```bash
gcloud iam service-accounts add-iam-policy-binding \
  tici-gcs-wi@<project-id>.iam.gserviceaccount.com \
  --project <project-id> \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:<project-id>.svc.id.goog:<namespace>/tici-gcs-wi-sa"
```

### 3) Update placeholder in manifest and deploy

In `tici-tc-gcs-wi.yaml`, replace:
- `tici-gcs-wi@YOUR_GCP_PROJECT_ID.iam.gserviceaccount.com`

Then deploy:

```bash
kubectl -n <ns> apply -f examples/tici/tici-tc-gcs-wi.yaml
```

### 4) Validation checklist

- Pod uses KSA `tici-gcs-wi-sa`
- TiCDC changefeed can be created
- No `GOOGLE_APPLICATION_CREDENTIALS` file dependency errors
- Objects appear in `gs://<bucket>/tici_default_prefix/cdc`

## Known Pitfalls

1. Secret mode file-name mismatch:
   - App expects `/var/secrets/google/credentials.json`
   - Secret key must be named `credentials.json`

2. Bucket name mismatch:
   - Ensure bucket name in `sinkURI` and `tici.s3.bucket` is the same
   - Use valid GCS bucket names (for example: `tici-test`)

## Current Conclusion

- Secret mode is validated and working.
- Workload Identity mode is prepared and under active validation.
