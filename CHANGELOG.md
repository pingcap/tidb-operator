# TiDB Operator v1.0.0-beta.2 Release Notes

## v1.0.0-beta.2 What’s New

Stability has been greatly enhanced
- Refactored e2e test
- Added stability test, 7x24 running

Greatly improved ease of use

- One-command deployment for AWS, Aliyun
- Minikube deployment for testing
- Tkctl cli tool
- Refactor backup chart for ease use
- Refine initializer job
- Grafana monitor dashboard improved, support multi-version
- Improved user guide
- Contributing documentation

Bug fixes

- Fix PD start script, add join file when startup
- Fix TiKV failover take too long
- Fix PD ha when replcias is less than 3
- Fix a tidb-scheduler acquireLock bug and emit event when scheduled failed
- Fix scheduler ha bug with defer deleting pods
- Fix bug when using shareinformer without deepcopy

Other improvements

- Remove pushgateway from TiKV pod
- Add GitHub templates for issue reporting and PR
- Automatically set the scheduler K8s version
- Swith to go module
- Support slow log of TiDB

## Detailed Bug Fixes And Changes

- Don't initialize when there is no tidb.password (#282)
- fix join script (#285)
- Document tool setup and e2e test detail in Contributing.md (#288)
- Update setup.md (#281)
- Support slow log tailing sidcar for tidb instance (#290)
- Flexible tidb initializer job with secret set outside of helm (#286)
- Ensure SLOW_LOG_FILE env variable is always set (#298)
- fix setup document description (#300)
- refactor backup (#301)
- Abandon vendor and refresh go.sum (#311)
- set the SLOW_LOG_FILE in the startup script (#307)
- automatically set the scheduler K8s version (#313)
- tidb stability test main function (#306)
- stability: add fault-trigger server (#312)
- Yinliang/backup and restore add adhoc backup and restore functison (#316)
- stability: add scale & upgrade case functions (#309)
- add slack (#318)
- log dump when test failed (#317)
- stability: add fault-trigger client (#326)
- monitor checker (#320)
- stability: add blockWriter case for inserting data (#321)
- add scheduled-backup test case (#322)
- stability: port ddl test as a workload (#328)
- stability: use fault-trigger at e2e tests and add some log (#330)
- add binlog deploy and check process (#329)
- fix e2e can not make (#331)
- multi tidb cluster testing (#334)
- fix bakcup test bugs (#335)
- delete blockWrite.go use blockwrite.go instead (#333)
- remove vendor (#344)
- stability: add more checks for scale & upgrade (#327)
- stability: support more fault injection (#345)
- rewrite e2e (#346)
- stability: add failover test (#349)
- fix ha when replcias is less than 3 (#351)
- stability: add fault-trigger service file (#353)
- fix dind doc (#352)
- Add additionalPrintColumns for TidbCluster CRD (#361)
- refactor stability main function (#363)
- enable admin privelege for prom (#360)
- Updated Readme with New Info (#365)
- Build CLI (#357)
- add extraLabels variable in tidb-cluster chart (#373)
- fix tikv failover (#368)
- Separate and ensure setup before e2e-build (#375)
- Fix codegen.sh and lock related depedencies (#371)
- stability: add sst-file-corruption case (#382)
- use release name as default clusterName (#354)
- Add util class to support to add annotations to Grafana (#378)
- Use grafana provisioning to replace dashboard installer (#388)
- ensure test env is ready before cases running (#386)
- remove monitor config job check (#390)
- Update local-pv documentation (#383)
- Update Jenkins links in README.md (#395)
- fix e2e workflow in CONTRIBUTING.md (#392)
- Support running stability test out of cluster (#397)
- update tidb secret docs and charts (#398)
- Enable blockWriter write pressure in stability test (#399)
- Support debug and ctop commands in CLI (#387)
- marketplace update (#380)
- dashboard:update editable value from true to false (#394)
- add fault inject for kube proxy (#384)
- use `ioutil.TempDir()` create charts and operator repo's directories (#405)
- Improve workflow in docs/google-kubernetes-tutorial.md (#400)
- Support plugin start argument for tidb instance (#412)
- Replace govet with official vet tool (#416)
- allocate 24 PVs by default (after 2 clusters are scaled to (#407)
- refine stability (#422)
- Record event as grafana annotation in stability test (#414)
- add GitHub templates for issue reporting and PR (#420)
- add TiDBUpgrading func (#423)
- fix operator chart issue (#419)
- fix stability issues (#433)
- change cert generate method and add pd and kv prestop webhook (#406)
- a tidb-scheduler bug fix and emit event when scheduled failed (#427)
- Shell completion for tkctl (#431)
- Delete an duplicate import (#434)
- add etcd and kube-apiserver faults (#367)
- Fix TiDB Slack link (#444)
- fix scheduler ha bug (#443)
- add terraform script to auto deploy TiDB cluster on AWS (#401)
- Adds instructions to access Grafana in GKE tutorial (#448)
- fix label selector (#437)
- no need to set ClusterIP when syncing headless service (#432)
- docs on how to deploy tidb cluster with tidb-operator in minikube (#451)
- add slack notify (#439)
- fix local dind env (#440)
- Add terraform scripts to support alibaba cloud ACK deployment (#436)
- Fix backup data compare logic (#454)
- stability test: async emit annotations (#438)
- Use TiDB v2.1.8 by default & remove pushgateway (#435)
- Fix bug use shareinformer without copy (#462)
- Add version command for tkctl (#456)
- Add tkctl user manual (#452)
- Fix binlog problem on large scale (#460)
- Copy kubernetes.io/hostname label to PVs (#464)
- AWS EKS tutorial change to new terraform script (#463)
- docs/minikube: update documentation of minikube installation (#471)
- docs/dind: update documentation of DinD installation (#458)
- docs/minikube: add instructions to access Grafana (#476)
- support-multi-version-dashboard (#473)
- docs/aliyun: update aliyun deploy docs after testing (#474)
- GKE local SSD size warning (#467)
- update roadmap (#376)
