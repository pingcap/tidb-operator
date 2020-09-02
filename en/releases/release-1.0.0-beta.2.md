---
title: TiDB Operator 1.0 Beta.2 Release Notes
---

# TiDB Operator 1.0 Beta.2 Release Notes

Release date: May 10, 2019

TiDB Operator version: 1.0.0-beta.2

## v1.0.0-beta.2 What’s New

### Stability has been greatly enhanced

- Refactored e2e test
- Added stability test, 7x24 running

### Greatly improved ease of use

- One-command deployment for AWS, Aliyun
- Minikube deployment for testing
- Tkctl cli tool
- Refactor backup chart for ease use
- Refine initializer job
- Grafana monitor dashboard improved, support multi-version
- Improved user guide
- Contributing documentation

### Bug fixes

- Fix PD start script, add join file when startup
- Fix TiKV failover take too long
- Fix PD ha when replcias is less than 3
- Fix a tidb-scheduler acquireLock bug and emit event when scheduled failed
- Fix scheduler ha bug with defer deleting pods
- Fix a bug when using shareinformer without deepcopy

### Other improvements

- Remove pushgateway from TiKV pod
- Add GitHub templates for issue reporting and PR
- Automatically set the scheduler K8s version
- Switch to go module
- Support slow log of TiDB

## Detailed Bug Fixes and Changes

- Don't initialize when there is no tidb.password ([#282](https://github.com/pingcap/tidb-operator/pull/282))
- Fix join script ([#285](https://github.com/pingcap/tidb-operator/pull/285))
- Document tool setup and e2e test detail in `CONTRIBUTING.md` ([#288](https://github.com/pingcap/tidb-operator/pull/288))
- Update setup.md ([#281](https://github.com/pingcap/tidb-operator/pull/281))
- Support slow log tailing sidcar for TiDB instance ([#290](https://github.com/pingcap/tidb-operator/pull/290))
- Flexible tidb initializer job with secret set outside of helm ([#286](https://github.com/pingcap/tidb-operator/pull/286))
- Ensure SLOW_LOG_FILE env variable is always set ([#298](https://github.com/pingcap/tidb-operator/pull/298))
- Fix setup document description ([#300](https://github.com/pingcap/tidb-operator/pull/300))
- Refactor backup ([#301](https://github.com/pingcap/tidb-operator/pull/301))
- Abandon vendor and refresh go.sum ([#311](https://github.com/pingcap/tidb-operator/pull/311))
- Set the SLOW_LOG_FILE in the startup script ([#307](https://github.com/pingcap/tidb-operator/pull/307))
- Automatically set the scheduler K8s version ([#313](https://github.com/pingcap/tidb-operator/pull/313))
- TiDB stability test main function ([#306](https://github.com/pingcap/tidb-operator/pull/306))
- Add fault-trigger server ([#312](https://github.com/pingcap/tidb-operator/pull/312))
- Add ad-hoc backup and restore function ([#316](https://github.com/pingcap/tidb-operator/pull/316))
- Add scale & upgrade case functions ([#309](https://github.com/pingcap/tidb-operator/pull/309))
- Add slack ([#318](https://github.com/pingcap/tidb-operator/pull/318))
- Log dump when test failed ([#317](https://github.com/pingcap/tidb-operator/pull/317))
- Add fault-trigger client ([#326](https://github.com/pingcap/tidb-operator/pull/326))
- Monitor checker ([#320](https://github.com/pingcap/tidb-operator/pull/320))
- Add blockWriter case for inserting data ([#321](https://github.com/pingcap/tidb-operator/pull/321))
- Add scheduled-backup test case ([#322](https://github.com/pingcap/tidb-operator/pull/322))
- Port ddl test as a workload ([#328](https://github.com/pingcap/tidb-operator/pull/328))
- Use fault-trigger at e2e tests and add some log ([#330](https://github.com/pingcap/tidb-operator/pull/330))
- Add binlog deploy and check process ([#329](https://github.com/pingcap/tidb-operator/pull/329))
- Fix e2e can not make ([#331](https://github.com/pingcap/tidb-operator/pull/331))
- Multi TiDB cluster testing ([#334](https://github.com/pingcap/tidb-operator/pull/334))
- Fix backup test bugs ([#335](https://github.com/pingcap/tidb-operator/pull/335))
- Delete `blockWrite.go` and use `blockwrite.go` instead ([#333](https://github.com/pingcap/tidb-operator/pull/333))
- Remove vendor ([#344](https://github.com/pingcap/tidb-operator/pull/344))
- Add more checks for scale & upgrade ([#327](https://github.com/pingcap/tidb-operator/pull/327))
- Support more fault injection ([#345](https://github.com/pingcap/tidb-operator/pull/345))
- Rewrite e2e ([#346](https://github.com/pingcap/tidb-operator/pull/346))
- Add failover test ([#349](https://github.com/pingcap/tidb-operator/pull/349))
- Fix HA when the number of replicas are less than 3 ([#351](https://github.com/pingcap/tidb-operator/pull/351))
- Add fault-trigger service file ([#353](https://github.com/pingcap/tidb-operator/pull/353))
- Fix dind doc ([#352](https://github.com/pingcap/tidb-operator/pull/352))
- Add additionalPrintColumns for TidbCluster CRD ([#361](https://github.com/pingcap/tidb-operator/pull/361))
- Refactor stability main function ([#363](https://github.com/pingcap/tidb-operator/pull/363))
- Enable admin privilege for prom ([#360](https://github.com/pingcap/tidb-operator/pull/360))
- Update `README.md` with new info ([#365](https://github.com/pingcap/tidb-operator/pull/365))
- Build CLI ([#357](https://github.com/pingcap/tidb-operator/pull/357))
- Add extraLabels variable in tidb-cluster chart ([#373](https://github.com/pingcap/tidb-operator/pull/373))
- Fix TiKV failover ([#368](https://github.com/pingcap/tidb-operator/pull/368))
- Separate and ensure setup before e2e-build ([#375](https://github.com/pingcap/tidb-operator/pull/375))
- Fix `codegen.sh` and lock related dependencies ([#371](https://github.com/pingcap/tidb-operator/pull/371))
- Add sst-file-corruption case ([#382](https://github.com/pingcap/tidb-operator/pull/382))
- Use release name as default clusterName ([#354](https://github.com/pingcap/tidb-operator/pull/354))
- Add util class to support adding annotations to Grafana ([#378](https://github.com/pingcap/tidb-operator/pull/378))
- Use Grafana provisioning to replace dashboard installer ([#388](https://github.com/pingcap/tidb-operator/pull/388))
- Ensure test env is ready before cases running ([#386](https://github.com/pingcap/tidb-operator/pull/386))
- Remove monitor config job check ([#390](https://github.com/pingcap/tidb-operator/pull/390))
- Update local-pv documentation ([#383](https://github.com/pingcap/tidb-operator/pull/383))
- Update Jenkins links in `README.md` ([#395](https://github.com/pingcap/tidb-operator/pull/395))
- Fix e2e workflow in `CONTRIBUTING.md` ([#392](https://github.com/pingcap/tidb-operator/pull/392))
- Support running stability test out of cluster ([#397](https://github.com/pingcap/tidb-operator/pull/397))
- Update TiDB secret docs and charts ([#398](https://github.com/pingcap/tidb-operator/pull/398))
- Enable blockWriter write pressure in stability test ([#399](https://github.com/pingcap/tidb-operator/pull/399))
- Support debug and ctop commands in CLI ([#387](https://github.com/pingcap/tidb-operator/pull/387))
- Marketplace update ([#380](https://github.com/pingcap/tidb-operator/pull/380))
- Update editable value from true to false ([#394](https://github.com/pingcap/tidb-operator/pull/394))
- Add fault inject for kube proxy ([#384](https://github.com/pingcap/tidb-operator/pull/384))
- Use `ioutil.TempDir()` create charts and operator repo's directories ([#405](https://github.com/pingcap/tidb-operator/pull/405))
- Improve workflow in docs/google-kubernetes-tutorial.md ([#400](https://github.com/pingcap/tidb-operator/pull/400))
- Support plugin start argument for TiDB instance ([#412](https://github.com/pingcap/tidb-operator/pull/412))
- Replace govet with official vet tool ([#416](https://github.com/pingcap/tidb-operator/pull/416))
- Allocate 24 PVs by default (after 2 clusters are scaled to ([#407](https://github.com/pingcap/tidb-operator/pull/407))
- Refine stability ([#422](https://github.com/pingcap/tidb-operator/pull/422))
- Record event as grafana annotation in stability test ([#414](https://github.com/pingcap/tidb-operator/pull/414))
- Add GitHub templates for issue reporting and PR ([#420](https://github.com/pingcap/tidb-operator/pull/420))
- Add TiDBUpgrading func ([#423](https://github.com/pingcap/tidb-operator/pull/423))
- Fix operator chart issue ([#419](https://github.com/pingcap/tidb-operator/pull/419))
- Fix stability issues ([#433](https://github.com/pingcap/tidb-operator/pull/433))
- Change cert generate method and add pd and kv prestop webhook ([#406](https://github.com/pingcap/tidb-operator/pull/406))
- A tidb-scheduler bug fix and emit event when scheduled failed ([#427](https://github.com/pingcap/tidb-operator/pull/427))
- Shell completion for tkctl ([#431](https://github.com/pingcap/tidb-operator/pull/431))
- Delete an duplicate import ([#434](https://github.com/pingcap/tidb-operator/pull/434))
- Add etcd and kube-apiserver faults ([#367](https://github.com/pingcap/tidb-operator/pull/367))
- Fix TiDB Slack link ([#444](https://github.com/pingcap/tidb-operator/pull/444))
- Fix scheduler ha bug ([#443](https://github.com/pingcap/tidb-operator/pull/443))
- Add terraform script to auto deploy TiDB cluster on AWS ([#401](https://github.com/pingcap/tidb-operator/pull/401))
- Add instructions to access Grafana in GKE tutorial ([#448](https://github.com/pingcap/tidb-operator/pull/448))
- Fix label selector ([#437](https://github.com/pingcap/tidb-operator/pull/437))
- No need to set ClusterIP when syncing headless service ([#432](https://github.com/pingcap/tidb-operator/pull/432))
- Document how to deploy TiDB cluster with tidb-operator in minikube ([#451](https://github.com/pingcap/tidb-operator/pull/451))
- Add slack notify ([#439](https://github.com/pingcap/tidb-operator/pull/439))
- Fix local dind env ([#440](https://github.com/pingcap/tidb-operator/pull/440))
- Add terraform scripts to support alibaba cloud ACK deployment ([#436](https://github.com/pingcap/tidb-operator/pull/436))
- Fix backup data compare logic ([#454](https://github.com/pingcap/tidb-operator/pull/454))
- Async emit annotations ([#438](https://github.com/pingcap/tidb-operator/pull/438))
- Use TiDB v2.1.8 by default & remove pushgateway ([#435](https://github.com/pingcap/tidb-operator/pull/435))
- Fix a bug that uses shareinformer without copy ([#462](https://github.com/pingcap/tidb-operator/pull/462))
- Add version command for tkctl ([#456](https://github.com/pingcap/tidb-operator/pull/456))
- Add tkctl user manual ([#452](https://github.com/pingcap/tidb-operator/pull/452))
- Fix binlog problem on large scale ([#460](https://github.com/pingcap/tidb-operator/pull/460))
- Copy kubernetes.io/hostname label to PVs ([#464](https://github.com/pingcap/tidb-operator/pull/464))
- AWS EKS tutorial change to new terraform script ([#463](https://github.com/pingcap/tidb-operator/pull/463))
- Update documentation of minikube installation ([#471](https://github.com/pingcap/tidb-operator/pull/471))
- Update documentation of DinD installation ([#458](https://github.com/pingcap/tidb-operator/pull/458))
- Add instructions to access Grafana ([#476](https://github.com/pingcap/tidb-operator/pull/476))
- Support-multi-version-dashboard ([#473](https://github.com/pingcap/tidb-operator/pull/473))
- Update Aliyun deploy docs after testing ([#474](https://github.com/pingcap/tidb-operator/pull/474))
- GKE local SSD size warning ([#467](https://github.com/pingcap/tidb-operator/pull/467))
- Update roadmap ([#376](https://github.com/pingcap/tidb-operator/pull/376))
