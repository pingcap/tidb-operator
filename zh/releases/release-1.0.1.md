---
title: TiDB Operator 1.0.1 Release Notes
---

# TiDB Operator 1.0.1 Release Notes

Release date: September 17, 2019

TiDB Operator version: 1.0.1

## v1.0.1 What's New

### Action Required

- ACTION REQUIRED: We fixed a serious bug ([#878](https://github.com/pingcap/tidb-operator/pull/878)) that could cause all `PD` and `TiKV` pods to be accidentally deleted when `kube-apiserver` fails. This would cause TiDB service outage. So if you are using `v1.0.0` or prior versions, you **must** upgrade to `v1.0.1`.
- ACTION REQUIRED: The backup tool image [pingcap/tidb-cloud-backup](https://hub.docker.com/r/pingcap/tidb-cloud-backup) uses a forked version of [`Mydumper`](https://github.com/pingcap/mydumper). The current version `pingcap/tidb-cloud-backup:20190610` contains a serious bug that could result in a missing column in the exported data. This is fixed in [#29](https://github.com/pingcap/mydumper/pull/29). And the default image used now contains this fixed version. So if you are using the old version image for backup, you **must** upgrade to use `pingcap/tidb-cloud-backup:201908028` and do a new full backup to avoid potential data inconsistency.

### Improvements

- Modularize GCP Terraform
- Add a script to remove orphaned k8s disks
- Support `binlog.pump.config`, `binlog.drainer.config` configurations for Pump and Drainer
- Set the resource limit for the `tidb-backup` job
- Add `affinity` to Pump and Drainer configurations
- Upgrade local-volume-provisioner to `v2.3.2`
- Reduce e2e run time from `60m` to `20m`
- Prevent the Pump process from exiting with `0` if the Pump becomes `offline`
- Support expanding cloud storage PV dynamically by increasing PVC storage size
- Add the `tikvGCLifeTime` option to do backup
- Add important parameters to `tikv.config` and `tidb.config` in `values.yaml`
- Support restoring the TiDB cluster from a specified scheduled backup directory
- Enable cloud storage volume expansion & label local volume
- Document and improve HA algorithm
- Support specifying the permit host in the `values.tidb.permitHost` chart
- Add the zone label and reserved resources arguments to kubelet
- Update the default backup image to `pingcap/tidb-cloud-backup:20190828`

### Bug Fixes

- Fix the TiKV scale-in failure in some cases after the TiKV failover
- Fix error handling for UpdateService
- Fix some orphan pods cleaner bugs
- Fix the bug of setting the `StatefulSet` partition
- Fix ad-hoc full backup failure due to incorrect `claimName`
- Fix the offline Pump: the Pump process will exit with `0` if going offline
- Fix an incorrect condition judgment

## Detailed Bug Fixes and Changes

- Clean up `tidb.pingcap.com/pod-scheduling` annotation when the pod is scheduled ([#790](https://github.com/pingcap/tidb-operator/pull/790))
- Update tidb-cloud-backup image tag ([#846](https://github.com/pingcap/tidb-operator/pull/846))
- Add the TiDB permit host option ([#779](https://github.com/pingcap/tidb-operator/pull/779))
- Add the zone label and reserved resources for nodes ([#871](https://github.com/pingcap/tidb-operator/pull/871))
- Fix some orphan pods cleaner bugs ([#878](https://github.com/pingcap/tidb-operator/pull/878))
- Fix the bug of setting the `StatefulSet` partition ([#830](https://github.com/pingcap/tidb-operator/pull/830))
- Add the `tikvGCLifeTime` option ([#835](https://github.com/pingcap/tidb-operator/pull/835))
- Add recommendations options to Mydumper ([#828](https://github.com/pingcap/tidb-operator/pull/828))
- Fix ad-hoc full backup failure due to incorrect `claimName` ([#836](https://github.com/pingcap/tidb-operator/pull/836))
- Improve `tkctl get` command output ([#822](https://github.com/pingcap/tidb-operator/pull/822))
- Add important parameters to TiKV and TiDB configurations ([#786](https://github.com/pingcap/tidb-operator/pull/786))
- Fix the issue that `binlog.drainer.config` is not supported in v1.0.0 ([#775](https://github.com/pingcap/tidb-operator/pull/775))
- Support restoring the TiDB cluster from a specified scheduled backup directory ([#804](https://github.com/pingcap/tidb-operator/pull/804))
- Fix `extraLabels` description in `values.yaml` ([#763](https://github.com/pingcap/tidb-operator/pull/763))
- Fix tkctl log output exception ([#797](https://github.com/pingcap/tidb-operator/pull/797))
- Add a script to remove orphaned K8s disks ([#745](https://github.com/pingcap/tidb-operator/pull/745))
- Enable cloud storage volume expansion & label local volume ([#772](https://github.com/pingcap/tidb-operator/pull/772))
- Prevent the Pump process from exiting with `0` if the Pump becomes `offline` ([#769](https://github.com/pingcap/tidb-operator/pull/769))
- Modularize GCP Terraform ([#717](https://github.com/pingcap/tidb-operator/pull/717))
- Support `binlog.pump.config` configurations for Pump and Drainer ([#693](https://github.com/pingcap/tidb-operator/pull/693))
- Remove duplicate key values ([#758](https://github.com/pingcap/tidb-operator/pull/758))
- Fix some typos ([#738](https://github.com/pingcap/tidb-operator/pull/738))
- Extend the waiting time of the `CheckManualPauseTiDB` process ([#752](https://github.com/pingcap/tidb-operator/pull/752))
- Set the resource limit for the `tidb-backup` job  ([#729](https://github.com/pingcap/tidb-operator/pull/729))
- Fix e2e test compatible with v1.0.0 ([#757](https://github.com/pingcap/tidb-operator/pull/757))
- Make incremental backup test work ([#764](https://github.com/pingcap/tidb-operator/pull/764))
- Add retry logic for `LabelNodes` function ([#735](https://github.com/pingcap/tidb-operator/pull/735))
- Fix the TiKV scale-in failure in some cases ([#726](https://github.com/pingcap/tidb-operator/pull/726))
- Add affinity to Pump and Drainer ([#741](https://github.com/pingcap/tidb-operator/pull/741))
- Refine cleanup logic ([#719](https://github.com/pingcap/tidb-operator/pull/719))
- Inject a failure by pod annotation ([#716](https://github.com/pingcap/tidb-operator/pull/716))
- Update README links to point to correct `pingcap.com/docs` URLs for English and Chinese ([#732](https://github.com/pingcap/tidb-operator/pull/732))
- Document and improve HA algorithm ([#670](https://github.com/pingcap/tidb-operator/pull/670))
- Fix an incorrect condition judgment ([#718](https://github.com/pingcap/tidb-operator/pull/718))
- Upgrade local-volume-provisioner to v2.3.2 ([#696](https://github.com/pingcap/tidb-operator/pull/696))
- Reduce e2e test run time ([#713](https://github.com/pingcap/tidb-operator/pull/713))
- Fix Terraform GKE scale-out issues ([#711](https://github.com/pingcap/tidb-operator/pull/711))
- Update wording and fix format for v1.0.0 ([#709](https://github.com/pingcap/tidb-operator/pull/709))
- Update documents ([#705](https://github.com/pingcap/tidb-operator/pull/705))
