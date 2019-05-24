# TiDB Backup Configuration Reference

`TiDB-Backup` is a helm chart designed for TiDB cluster backup and restore via the [`mydumper`](https://www.pingcap.com/docs/dev/reference/tools/mydumper/) and [`loader`](https://www.pingcap.com/docs-cn/tools/loader/). This documentation will explain the configurations of `TiDB-Backup`, you may refer to [Restore and Backup TiDB cluster](#tidb-backup-configuration-reference) for user guide with example.

## Configurations

### `mode`

- To choose the operation, backup or restore, required
- Default: "backup"

### `clusterName`

- The TiDB cluster name that should backup from or restore to, required
- Default: "demo"

### `name`

- The backup name
- Default: "fullbackup-${date}", date is the start time of backup, accurate to minute

### `secretName`

- The name of the secret which stores user and password used for backup/restore
- Default: "backup-secret"
- You can create the secret by `kubectl create secret generic backup-secret --from-literal=user=root --from-literal=password=<password>`

### `storage.className`

- The storageClass used to store the backup data
- Default: "local-storage"

### `storage.size`

- The storage size of PersistenceVolume
- Default: "100Gi"

### `backupOptions`

- The options that passed to [`mydumper`](https://github.com/maxbube/mydumper/blob/master/docs/mydumper_usage.rst#options)
- Default: "--chunk-filesize=100"

### `restoreOptions`

- The options that passed to [`loader`](https://www.pingcap.com/docs-cn/tools/loader/)
- Default: "-t 16"

### `gcp.bucket`

- The GCP bucket name to store backup data

> **Note**: Once you set any variables under `gcp` section, the backup data will be uploaded to Google Cloud Storage, namely, you have to keep the configuration intact.

### `gcp.secretName`

- The name of the secret which stores the gcp service account credentials json file
- You can create the secret by `kubectl create secret generic gcp-backup-secret --from-file=./credentials.json`. To download credentials json, refer to [Google Cloud Documentation](https://cloud.google.com/docs/authentication/production#obtaining_and_providing_service_account_credentials_manually)

### `ceph.endpoint`

- The endpoint of ceph object storage

> **Note**: Once you set any variables under `ceph` section, the backup data will be uploaded to ceph object storage, namely, you have to keep the configuration intact.

### `ceph.bucket`

- The bucket name of ceph object storage

### `ceph.secretName`

- The name of the secret which stores ceph object store access key and secret key
- You can create the secret by:

```shell
$ kubectl create secret generic ceph-backup-secret --from-literal=access_key=<access-key> --from-literal=secret_key=<secret-key>
```
