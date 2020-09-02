---
title: TiDB Operator 1.0 RC.1 Release Notes
---

# TiDB Operator 1.0 RC.1 Release Notes

Release date: July 12, 2019

TiDB Operator version: 1.0.0-rc.1

## v1.0.0-rc.1 What’s New

### Stability test cases added

- Stop kube-proxy
- Upgrade tidb-operator

### Improvements

- Get the TS first and increase the TiKV GC life time to 3 hours before the full backup
- Add endpoints list and watch permission for controller-manager
- Scheduler image is updated to use "k8s.gcr.io/kube-scheduler" which is much smaller than "gcr.io/google-containers/hyperkube". You must pre-pull the new scheduler image into your airgap environment before upgrading.
- Full backup data can be uploaded to or downloaded from Amazon S3
- The terraform scripts support manage multiple TiDB clusters in one EKS cluster.
- Add `tikv.storeLables` setting
- On GKE one can use COS for TiKV nodes with small data for faster startup
- Support force upgrade when PD cluster is unavailable.

### Bug Fixes

- Fix unbound variable in the backup script
- Give kube-scheduler permission to update/patch pod status
- Fix tidb user of scheduled backup script
- Fix scheduled backup to ceph object storage
- Fix several usability problems for AWS terraform deployment
- Fix scheduled backup bug: segmentation fault when backup user's password is empty

## Detailed Bug Fixes and Changes

- Segmentation fault when backup user's password is empty ([#649](https://github.com/pingcap/tidb-operator/pull/649))
- Small fixes for terraform AWS ([#646](https://github.com/pingcap/tidb-operator/pull/646))
- TiKV upgrade bug fix ([#626](https://github.com/pingcap/tidb-operator/pull/626))
- Improve the readability of some code ([#639](https://github.com/pingcap/tidb-operator/pull/639))
- Support force upgrade when PD cluster is unavailable ([#631](https://github.com/pingcap/tidb-operator/pull/631))
- Add new terraform version requirement to AWS deployment ([#636](https://github.com/pingcap/tidb-operator/pull/636))
- GKE local ssd provisioner for COS ([#612](https://github.com/pingcap/tidb-operator/pull/612))
- Remove TiDB version from build ([#627](https://github.com/pingcap/tidb-operator/pull/627))
- Refactor so that using the PD API avoids unnecessary imports ([#618](https://github.com/pingcap/tidb-operator/pull/618))
- Add `storeLabels` setting ([#527](https://github.com/pingcap/tidb-operator/pull/527))
- Update google-kubernetes-tutorial.md ([#622](https://github.com/pingcap/tidb-operator/pull/622))
- Multiple clusters management in EKS ([#616](https://github.com/pingcap/tidb-operator/pull/616))
- Add Amazon S3 support to the backup/restore features ([#606](https://github.com/pingcap/tidb-operator/pull/606))
- Pass TiKV upgrade case ([#619](https://github.com/pingcap/tidb-operator/pull/619))
- Separate slow log with TiDB server log by default ([#610](https://github.com/pingcap/tidb-operator/pull/610))
- Fix the problem of unbound variable in backup script ([#608](https://github.com/pingcap/tidb-operator/pull/608))
- Fix notes of tidb-backup chart ([#595](https://github.com/pingcap/tidb-operator/pull/595))
- Give kube-scheduler ability to update/patch pod status. ([#611](https://github.com/pingcap/tidb-operator/pull/611))
- Use kube-scheduler image instead of hyperkube ([#596](https://github.com/pingcap/tidb-operator/pull/596))
- Fix pull request template grammar ([#607](https://github.com/pingcap/tidb-operator/pull/607))
- Local SSD provision: reduce network traffic ([#601](https://github.com/pingcap/tidb-operator/pull/601))
- Add operator upgrade case ([#579](https://github.com/pingcap/tidb-operator/pull/579))
- Fix a bug that TiKV status is always upgrade ([#598](https://github.com/pingcap/tidb-operator/pull/598))
- Build without debugger symbols ([#592](https://github.com/pingcap/tidb-operator/pull/592))
- Improve error messages ([#591](https://github.com/pingcap/tidb-operator/pull/591))
- Fix tidb user of scheduled backup script ([#594](https://github.com/pingcap/tidb-operator/pull/594))
- Fix dt case bug ([#571](https://github.com/pingcap/tidb-operator/pull/571))
- GKE terraform ([#585](https://github.com/pingcap/tidb-operator/pull/585))
- Fix scheduled backup to Ceph object storage ([#576](https://github.com/pingcap/tidb-operator/pull/576))
- Add stop kube-scheduler/kube-controller-manager test cases ([#583](https://github.com/pingcap/tidb-operator/pull/583))
- Add endpoints list and watch permission for controller-manager ([#590](https://github.com/pingcap/tidb-operator/pull/590))
- Refine fullbackup ([#570](https://github.com/pingcap/tidb-operator/pull/570))
- Make sure go modules files are always tidy and up to date ([#588](https://github.com/pingcap/tidb-operator/pull/588))
- Local SSD on GKE ([#577](https://github.com/pingcap/tidb-operator/pull/577))
- Stop kube-proxy case ([#556](https://github.com/pingcap/tidb-operator/pull/556))
- Fix resource unit ([#573](https://github.com/pingcap/tidb-operator/pull/573))
- Give local-volume-provisioner pod a QoS of Guaranteed ([#569](https://github.com/pingcap/tidb-operator/pull/569))
- Check PD endpoints status when it's unhealthy ([#545](https://github.com/pingcap/tidb-operator/pull/545))
