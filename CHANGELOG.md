> **Note:**
>
> * Starting from September 2, 2020, all release notes of TiDB Operator will be maintained in [pingcap/docs-tidb-operator](https://github.com/pingcap/docs-tidb-operator).
> * You can read the release notes of all versions of TiDB Operator at [PingCAP Docs](https://docs.pingcap.com/tidb-in-kubernetes/stable/release-1.0.7).

# TiDB Operator v1.0.6 Release Notes

## v1.0.6 What's New

Action required: Users should migrate the configs in `values.yaml` of previous chart releases to the new `values.yaml` of the new chart. Otherwise, the monitor pods might fail when you upgrade the monitor with the new chart.

For example, configs in the old `values.yaml` file:

```
monitor:
  ...
  initializer:
    image: pingcap/tidb-monitor-initializer:v3.0.5
    imagePullPolicy: IfNotPresent
  ...
```

After migration, configs in the new `values.yaml` file should be as follows:

```
monitor:
  ...
  initializer:
    image: pingcap/tidb-monitor-initializer:v3.0.5
    imagePullPolicy: Always
    config:
      K8S_PROMETHEUS_URL: http://prometheus-k8s.monitoring.svc:9090
  ...
```

### Monitor

- Enable alert rule persistence ([#898](https://github.com/pingcap/tidb-operator/pull/898))
- Add node & pod info in TiDB Grafana ([#885](https://github.com/pingcap/tidb-operator/pull/885))

### TiDB Scheduler

- Refine scheduler error messages ([#1373](https://github.com/pingcap/tidb-operator/pull/1373))

### Compatibility

- Fix the compatibility issue in Kubernetes v1.17 ([#1241](https://github.com/pingcap/tidb-operator/pull/1241))
- Bind the `system:kube-scheduler` ClusterRole to the `tidb-scheduler` service account ([#1355](https://github.com/pingcap/tidb-operator/pull/1355))

### TiKV Importer

- Fix the default `tikv-importer` configuration ([#1415](https://github.com/pingcap/tidb-operator/pull/1415))

### E2E

- Ensure pods unaffected when upgrading ([#955](https://github.com/pingcap/tidb-operator/pull/955))

### CI

- Move the release CI script from Jenkins into the tidb-operator repository ([#1237](https://github.com/pingcap/tidb-operator/pull/1237))
- Adjust the release CI script for the `release-1.0` branch ([#1320](https://github.com/pingcap/tidb-operator/pull/1320))

# TiDB Operator v1.0.5 Release Notes

## v1.0.5 What's New

There is no action required if you are upgrading from [v1.0.4](#tidb-operator-v104-release-notes).

### Scheduled Backup

- Fix the issue that backup failed when `clusterName` is too long ([#1229](https://github.com/pingcap/tidb-operator/pull/1229))

### TiDB Binlog

- It is recommended that TiDB and Pump be deployed on the same node through the `affinity` feature and Pump be dispersed on different nodes through the `anti-affinity` feature. At most only one Pump instance is allowed on each node. We added a guide to the chart. ([#1251](https://github.com/pingcap/tidb-operator/pull/1251))

### Compatibility

- Fix `tidb-scheduler` RBAC permission in Kubernetes v1.16 ([#1282](https://github.com/pingcap/tidb-operator/pull/1282))
- Do not set `DNSPolicy` if `hostNetwork` is disabled to keep backward compatibility ([#1287](https://github.com/pingcap/tidb-operator/pull/1287))

### E2E

- Fix e2e nil point dereference ([#1221](https://github.com/pingcap/tidb-operator/pull/1221))

# TiDB Operator v1.0.4 Release Notes

## v1.0.4 What's New

### Action Required

There is no action required if you are upgrading from [v1.0.3](#tidb-operator-v103-release-notes).

### Highlights

[#1202](https://github.com/pingcap/tidb-operator/pull/1202) introduced `HostNetwork` support, which offers better performance compared to the Pod network. Check out our [benchmark report](https://pingcap.com/docs/dev/benchmark/sysbench-in-k8s/#pod-network-vs-host-network) for details.

> **Note:**
>
> Due to [this issue of Kubernetes](https://github.com/kubernetes/kubernetes/issues/78420), the Kubernetes cluster must be one of the following versions to enable `HostNetwork` of the TiDB cluster:
>  - `v1.13.11` or later
>  - `v1.14.7` or later
>  - `v1.15.4` or later
>  - any version since `v1.16.0`

[#1175](https://github.com/pingcap/tidb-operator/pull/1175) added the `podSecurityContext` support for TiDB cluster Pods. We recommend setting the namespaced kernel parameters for TiDB cluster Pods according to our [Environment Recommendation](https://pingcap.com/docs/dev/tidb-in-kubernetes/deploy/prerequisites/#the-configuration-of-kernel-parameters).

New Helm chart `tidb-lightning` brings [TiDB Lightning](https://pingcap.com/docs/stable/reference/tools/tidb-lightning/overview/) support for TiDB in Kubernetes. Check out the [document](https://pingcap.com/docs/dev/tidb-in-kubernetes/maintain/lightning/) for detailed user guide.

Another new Helm chart `tidb-drainer` brings multiple drainers support for TiDB Binlog in Kubernetes. Check out the [document](https://pingcap.com/docs/dev/tidb-in-kubernetes/maintain/tidb-binlog/#deploy-multiple-drainers) for detailed user guide.

### Improvements

- Support HostNetwork ([#1202](https://github.com/pingcap/tidb-operator/pull/1202))
- Support configuring sysctls for Pods and enable net.* ([#1175](https://github.com/pingcap/tidb-operator/pull/1175))
- Add tidb-lightning support ([#1161](https://github.com/pingcap/tidb-operator/pull/1161))
- Add new helm chart tidb-drainer to support multiple drainers ([#1160](https://github.com/pingcap/tidb-operator/pull/1160))

## Detailed Bug Fixes and Changes

- Add e2e scripts and simplify the e2e Jenkins file ([#1174](https://github.com/pingcap/tidb-operator/pull/1174))
- Fix the pump/drainer data directory to avoid data loss caused by bad configuration ([#1183](https://github.com/pingcap/tidb-operator/pull/1183))
- Add init sql case to e2e ([#1199](https://github.com/pingcap/tidb-operator/pull/1199))
- Keep the instance label of drainer same with the TiDB cluster in favor of monitoring ([#1170](https://github.com/pingcap/tidb-operator/pull/1170))
- Set `podSecuriyContext` to nil by default in favor of backward compatibility ([#1184](https://github.com/pingcap/tidb-operator/pull/1184))

## Additional Notes for Users of v1.1.0.alpha branch

For historical reasons, `v1.1.0.alpha` is a hot-fix branch and got this name by mistake. All fixes in that branch are cherry-picked to `v1.0.4` and the `v1.1.0.alpha` branch will be discarded to keep things clear.

We strongly recommend you to upgrade to `v1.0.4` if you are using any version under `v1.1.0.alpha`.

`v1.0.4` introduces the following fixes comparing to `v1.1.0.alpha.3`:

- Support HostNetwork ([#1202](https://github.com/pingcap/tidb-operator/pull/1202))
- Add the permit host option for tidb-initializer job ([#779](https://github.com/pingcap/tidb-operator/pull/779))
- Fix drainer misconfiguration in tidb-cluster chart ([#945](https://github.com/pingcap/tidb-operator/pull/945))
- Set the default `externalTrafficPolicy` to be Local for TiDB services ([#960](https://github.com/pingcap/tidb-operator/pull/960))
- Fix tidb-operator crash when users modify sts upgrade strategy improperly ([#969](https://github.com/pingcap/tidb-operator/pull/969))
- Add the `maxFailoverCount` limit to TiKV ([#976](https://github.com/pingcap/tidb-operator/pull/976))
- Fix values file customization for tidb-operator on aliyun ([#983](https://github.com/pingcap/tidb-operator/pull/983))
- Do not limit failover count when maxFailoverCount = 0 ([#978](https://github.com/pingcap/tidb-operator/pull/978))
- Suspend the `ReplaceUnhealthy` process for TiKV auto-scaling-group on AWS ([#1027](https://github.com/pingcap/tidb-operator/pull/1027))
- Fix the issue that the `create_tidb_cluster_release` variable does not work ([#1066](https://github.com/pingcap/tidb-operator/pull/1066)))
- Add `v1` to statefulset apiVersions ([#1056](https://github.com/pingcap/tidb-operator/pull/1056))
- Add timezone support ([#1126](https://github.com/pingcap/tidb-operator/pull/1027))

# TiDB Operator v1.0.3 Release Notes

## v1.0.3 What's New

### Action Required

ACTION REQUIRED: This release upgrades default TiDB version to `v3.0.5` which fixed a serious [bug](https://github.com/pingcap/tidb/pull/12597) in TiDB. So if you are using TiDB `v3.0.4` or prior versions, you **must** upgrade to `v3.0.5`.

ACTION REQUIRED: This release adds the `timezone` support for [all charts](https://github.com/pingcap/tidb-operator/tree/master/charts).

For existing TiDB clusters. If the `timezone` in `tidb-cluster/values.yaml` has been customized to other timezones instead of the default `UTC`, then upgrading tidb-operator will trigger a rolling update for the related pods.

The related pods include `pump`, `drainer`, `dicovery`, `monitor`, `scheduled backup`, `tidb-initializer`, and `tikv-importer`.

The time zone for all images maintained by `tidb-operator` should be `UTC`. If you use your own images, you need to make sure that the corresponding time zones are `UTC`.

### Improvements

- Add the `timezone` support for all containers of the TiDB cluster
- Support configuring resource requests and limits for all containers of the TiDB cluster

## Detailed Bug Fixes and Changes

- Upgrade default TiDB version to `v3.0.5` ([#1132](https://github.com/pingcap/tidb-operator/pull/1132))
- Add the `timezone` support for all containers of the TiDB cluster ([#1122](https://github.com/pingcap/tidb-operator/pull/1122))
- Support configuring resource requests and limits for all containers of the TiDB cluster ([#853](https://github.com/pingcap/tidb-operator/pull/853))

# TiDB Operator v1.0.2 Release Notes

## v1.0.2 What's New

### Action Required

The AWS Terraform script uses auto-scaling-group for all components (PD/TiKV/TiDB/monitor). When an ec2 instance fails the health check, the instance will be replaced. This is helpful for those applications that are stateless or use EBS volumes to store data.

But a TiKV Pod uses instance store to store its data. When an instance is replaced, all the data on its store will be lost. TiKV has to resync all data to the newly added instance. Though TiDB is a distributed database and can work when a node fails, resyncing data can cost much if the dataset is large. Besides, the ec2 instance may be recovered to a healthy state by rebooting.

So we disabled the auto-scaling-group's replacing behavior in `v1.0.2`.

Auto-scaling-group scaling process can also be suspended according to its [documentation](https://docs.aws.amazon.com/autoscaling/ec2/userguide/as-suspend-resume-processes.html) if you are using `v1.0.1` or prior versions.

### Improvements

- Suspend ReplaceUnhealthy process for AWS TiKV auto-scaling-group
- Add a new VM manager `qm` in stability test
- Add `tikv.maxFailoverCount` limit to TiKV
- Set the default `externalTrafficPolicy` to be `Local` for TiDB service in AWS/GCP/Aliyun
- Add provider and module versions for AWS

### Bug fixes

- Fix the issue that tkctl version does not work when the release name is un-wanted
- Migrate statefulsets apiVersion to `app/v1` which fixes compatibility with Kubernetes 1.16 and above versions
- Fix the issue that the `create_tidb_cluster_release` variable in AWS Terraform script does not work
- Fix compatibility issues by adding `v1beta1` to statefulset apiVersions
- Fix the issue that TiDB Loadbalancer is empty in Terraform output
- Fix a compatibility issue of TiKV `maxFailoverCount`
- Fix Terraform providers version constraint issues for GCP and Aliyun
- Fix values file customization for tidb-operator on Aliyun
- Fix tidb-operator crash when users modify statefulset upgrade strategy improperly
- Fix drainer misconfiguration

## Detailed Bug Fixes and Changes

- Fix the issue that tkctl version does not work when the release name is un-wanted ([#1065](https://github.com/pingcap/tidb-operator/pull/1065))
- Fix the issue that the `create_tidb_cluster_release` variable in AWS terraform script does not work ([#1062](https://github.com/pingcap/tidb-operator/pull/1062))
- Fix compatibility issues for ([#1012](https://github.com/pingcap/tidb-operator/pull/1012)): add `v1beta1` to statefulset apiVersions ([#1054](https://github.com/pingcap/tidb-operator/pull/1054))
- Enable ConfigMapRollout by default in stability test ([#1036](https://github.com/pingcap/tidb-operator/pull/1036))
- Fix the issue that TiDB Loadbalancer is empty in Terraform output ([#1045](https://github.com/pingcap/tidb-operator/pull/1045))
- Migrate statefulsets apiVersion to `app/v1` which fixes compatibility with Kubernetes 1.16 and above versions ([#1012](https://github.com/pingcap/tidb-operator/pull/1012))
- Only expect TiDB cluster upgrade to be complete when rolling back wrong configuration in stability test ([#1030](https://github.com/pingcap/tidb-operator/pull/1030))
- Suspend ReplaceUnhealthy process for AWS TiKV auto-scaling-group ([#1014](https://github.com/pingcap/tidb-operator/pull/1014))
- Add a new VM manager `qm` in stability test ([#896](https://github.com/pingcap/tidb-operator/pull/896))
- Fix provider versions constraint issues for GCP and Aliyun ([#959](https://github.com/pingcap/tidb-operator/pull/959))
- Fix values file customization for tidb-operator on Aliyun ([#971](https://github.com/pingcap/tidb-operator/pull/971))
- Fix a compatibility issue of TiKV `tikv.maxFailoverCount` ([#977](https://github.com/pingcap/tidb-operator/pull/977))
- Add `tikv.maxFailoverCount` limit to TiKV ([#965](https://github.com/pingcap/tidb-operator/pull/965))
- Fix tidb-operator crash when users modify statefulset upgrade strategy improperly ([#912](https://github.com/pingcap/tidb-operator/pull/912))
- Set the default `externalTrafficPolicy` to be `Local` for TiDB service in AWS/GCP/Aliyun ([#947](https://github.com/pingcap/tidb-operator/pull/947))
- Add note about setting PV reclaim policy to retain ([#911](https://github.com/pingcap/tidb-operator/pull/911))
- Fix drainer misconfiguration ([#939](https://github.com/pingcap/tidb-operator/pull/939))
- Add provider and module versions for AWS ([#926](https://github.com/pingcap/tidb-operator/pull/926))

# TiDB Operator v1.0.1 Release Notes

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

### Bug fixes

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

# TiDB Operator v1.0.0 Release Notes

## v1.0.0 What's New

### Action Required

- ACTION REQUIRED: `tikv.storeLabels` was removed from `values.yaml`. You can directly set it with `location-labels` in `pd.config`.
- ACTION REQUIRED: the `--features` flag of tidb-scheduler has been updated to the `key={true,false}` format. You can enable the feature by appending `=true`.
- ACTION REQUIRED: you need to change the configurations in `values.yaml` of previous chart releases to the new `values.yaml` of the new chart. Otherwise, the configurations will be ignored when upgrading the TiDB cluster with the new chart.

The `pd` section in old `values.yaml`:

```
pd:
  logLevel: info
  maxStoreDownTime: 30m
  maxReplicas: 3
```

The `pd` section in new `values.yaml`:

```
pd:
  config: |
    [log]
    level = "info"
    [schedule]
    max-store-down-time = "30m"
    [replication]
    max-replicas = 3
```

The `tikv` section in old `values.yaml`:

```
tikv:
  logLevel: info
  syncLog: true
  readpoolStorageConcurrency: 4
  readpoolCoprocessorConcurrency: 8
  storageSchedulerWorkerPoolSize: 4
```

The `tikv` section in new `values.yaml`:

```
tikv:
  config: |
    log-level = "info"
    [server]
    status-addr = "0.0.0.0:20180"
    [raftstore]
    sync-log = true
    [readpool.storage]
    high-concurrency = 4
    normal-concurrency = 4
    low-concurrency = 4
    [readpool.coprocessor]
    high-concurrency = 8
    normal-concurrency = 8
    low-concurrency = 8
    [storage]
    scheduler-worker-pool-size = 4
```

The `tidb` section in old `values.yaml`:

```
tidb:
  logLevel: info
  preparedPlanCacheEnabled: false
  preparedPlanCacheCapacity: 100
  txnLocalLatchesEnabled: false
  txnLocalLatchesCapacity: "10240000"
  tokenLimit: "1000"
  memQuotaQuery: "34359738368"
  txnEntryCountLimit: "300000"
  txnTotalSizeLimit: "104857600"
  checkMb4ValueInUtf8: true
  treatOldVersionUtf8AsUtf8mb4: true
  lease: 45s
  maxProcs: 0
```

The `tidb` section in new `values.yaml`:

```
tidb:
  config: |
    token-limit = 1000
    mem-quota-query = 34359738368
    check-mb4-value-in-utf8 = true
    treat-old-version-utf8-as-utf8mb4 = true
    lease = "45s"
    [log]
    level = "info"
    [prepared-plan-cache]
    enabled = false
    capacity = 100
    [txn-local-latches]
    enabled = false
    capacity = 10240000
    [performance]
    txn-entry-count-limit = 300000
    txn-total-size-limit = 104857600
    max-procs = 0
```

The `monitor` section in old `values.yaml`:

```
monitor:
  create: true
  ...
```

The `monitor` section in new `values.yaml`:

```
monitor:
  create: true
   initializer:
     image: pingcap/tidb-monitor-initializer:v3.0.5
     imagePullPolicy: IfNotPresent
   reloader:
     create: true
     image: pingcap/tidb-monitor-reloader:v1.0.0
     imagePullPolicy: IfNotPresent
     service:
       type: NodePort
  ...
```

Please check [cluster configuration](https://pingcap.com/docs/v3.0/tidb-in-kubernetes/reference/configuration/tidb-cluster/) for detailed configuration.

### Stability test cases added

- Stop all etcds and kubelets

### Improvements

- Simplify GKE SSD setup
- Modularization for AWS Terraform scripts
- Turn on the automatic failover feature by default
- Enable configmap rollout by default
- Enable stable scheduling by default
- Support multiple TiDB clusters management in Alibaba Cloud
- Enable AWS NLB cross zone load balancing by default

### Bug fixes

- Fix sysbench installation on bastion machine of AWS deployment
- Fix TiKV metrics monitoring in default setup

## Detailed Bug Fixes and Changes

- Allow upgrading TiDB monitor along with TiDB version ([#666](https://github.com/pingcap/tidb-operator/pull/666))
- Specify the TiKV status address to fix monitoring ([#695](https://github.com/pingcap/tidb-operator/pull/695))
- Fix sysbench installation on bastion machine for AWS deployment ([#688](https://github.com/pingcap/tidb-operator/pull/688))
- Update the `git add upstream` command to use `https` in contributing document ([#690](https://github.com/pingcap/tidb-operator/pull/690))
- Stability cases: stop kubelet and etcd ([#665](https://github.com/pingcap/tidb-operator/pull/665))
- Limit test cover packages ([#687](https://github.com/pingcap/tidb-operator/pull/687))
- Enable nlb cross zone load balancing by default ([#686](https://github.com/pingcap/tidb-operator/pull/686))
- Add TiKV raftstore parameters ([#681](https://github.com/pingcap/tidb-operator/pull/681))
- Support multiple TiDB clusters management for Alibaba Cloud ([#658](https://github.com/pingcap/tidb-operator/pull/658))
- Adjust the `EndEvictLeader` function ([#680](https://github.com/pingcap/tidb-operator/pull/680))
- Add more logs ([#676](https://github.com/pingcap/tidb-operator/pull/676))
- Update feature gates to support `key={true,false}` syntax ([#677](https://github.com/pingcap/tidb-operator/pull/677))
- Fix the typo meke to make ([#679](https://github.com/pingcap/tidb-operator/pull/679))
- Enable configmap rollout by default and quote configmap digest suffix ([#678](https://github.com/pingcap/tidb-operator/pull/678))
- Turn automatic failover on ([#667](https://github.com/pingcap/tidb-operator/pull/667))
- Sets node count for default pool equal to total desired node count ([#673](https://github.com/pingcap/tidb-operator/pull/673))
- Upgrade default TiDB version to v3.0.1 ([#671](https://github.com/pingcap/tidb-operator/pull/671))
- Remove storeLabels ([#663](https://github.com/pingcap/tidb-operator/pull/663))
- Change the way to configure TiDB/TiKV/PD in charts ([#638](https://github.com/pingcap/tidb-operator/pull/638))
- Modularize for AWS terraform scripts ([#650](https://github.com/pingcap/tidb-operator/pull/650))
- Change the `DeferClose` function ([#653](https://github.com/pingcap/tidb-operator/pull/653))
- Increase the default storage size for Pump from 10Gi to 20Gi in response to `stop-write-at-available-space` ([#657](https://github.com/pingcap/tidb-operator/pull/657))
- Simplify local SDD setup ([#644](https://github.com/pingcap/tidb-operator/pull/644))

# TiDB Operator v1.0.0-rc.1 Release Notes

## v1.0.0-rc.1 What’s New

### Stability test cases added

- stop kube-proxy
- upgrade tidb-operator

### Improvements

- get the TS first and increase the TiKV GC life time to 3 hours before the full backup
- Add endpoints list and watch permission for controller-manager
- Scheduler image is updated to use "k8s.gcr.io/kube-scheduler" which is much smaller than "gcr.io/google-containers/hyperkube". You must pre-pull the new scheduler image into your airgap environment before upgrading.
- Full backup data can be uploaded to or downloaded from Amazon S3
- The terraform scripts support manage multiple TiDB clusters in one EKS cluster.
- Add `tikv.storeLables` setting
- on GKE one can use COS for TiKV nodes with small data for faster startup
- Support force upgrade when PD cluster is unavailable.

### Bug fixes

- fix unbound variable in the backup script
- Give kube-scheduler permission to update/patch pod status
- fix tidb user of scheduled backup script
- fix scheduled backup to ceph object storage
- Fix several usability problems for AWS terraform deployment
- fix scheduled backup bug: segmentation fault when backup user's password is empty

## Detailed Bug Fixes And Changes

- bugfix: segmentation fault when backup user's password is empty ([#649](https://github.com/pingcap/tidb-operator/pull/649))
- Small fixes for terraform aws ([#646](https://github.com/pingcap/tidb-operator/pull/646))
- TiKV upgrade bug fix ([#626](https://github.com/pingcap/tidb-operator/pull/626))
- improving the readability of some code ([#639](https://github.com/pingcap/tidb-operator/pull/639))
- support force upgrade when pd cluster is unavailable ([#631](https://github.com/pingcap/tidb-operator/pull/631))
- Add new terraform version requirement to AWS deployment ([#636](https://github.com/pingcap/tidb-operator/pull/636))
- GKE local ssd provisioner for COS ([#612](https://github.com/pingcap/tidb-operator/pull/612))
- remove tidb version from build ([#627](https://github.com/pingcap/tidb-operator/pull/627))
- refactor so that using the PD API avoids unnecessary imports ([#618](https://github.com/pingcap/tidb-operator/pull/618))
- add storeLabels setting ([#527](https://github.com/pingcap/tidb-operator/pull/527))
- Update google-kubernetes-tutorial.md ([#622](https://github.com/pingcap/tidb-operator/pull/622))
- Multiple clusters management in EKS ([#616](https://github.com/pingcap/tidb-operator/pull/616))
- Add Amazon S3 support to the backup/restore features ([#606](https://github.com/pingcap/tidb-operator/pull/606))
- pass TiKV upgrade case ([#619](https://github.com/pingcap/tidb-operator/pull/619))
- separate slow log with tidb server log by default ([#610](https://github.com/pingcap/tidb-operator/pull/610))
- fix the problem of unbound variable in backup script ([#608](https://github.com/pingcap/tidb-operator/pull/608))
- fix notes of tidb-backup chart ([#595](https://github.com/pingcap/tidb-operator/pull/595))
- Give kube-scheduler ability to update/patch pod status. ([#611](https://github.com/pingcap/tidb-operator/pull/611))
- Use kube-scheduler image instead of hyperkube ([#596](https://github.com/pingcap/tidb-operator/pull/596))
- fix pull request template grammar ([#607](https://github.com/pingcap/tidb-operator/pull/607))
- local SSD provision: reduce network traffic ([#601](https://github.com/pingcap/tidb-operator/pull/601))
- {origin/HEAD} {origin/master} add operator upgrade case ([#579](https://github.com/pingcap/tidb-operator/pull/579))
- fix bug that tikv status is always upgrade ([#598](https://github.com/pingcap/tidb-operator/pull/598))
- build without debugger symbols ([#592](https://github.com/pingcap/tidb-operator/pull/592))
- improve error messages ([#591](https://github.com/pingcap/tidb-operator/pull/591))
- fix tidb user of scheduled backup script ([#594](https://github.com/pingcap/tidb-operator/pull/594))
- fix dt case bug ([#571](https://github.com/pingcap/tidb-operator/pull/571))
- GKE terraform ([#585](https://github.com/pingcap/tidb-operator/pull/585))
- fix scheduled backup to ceph object storage ([#576](https://github.com/pingcap/tidb-operator/pull/576))
- Add stop kube-scheduler/kube-controller-manager test cases ([#583](https://github.com/pingcap/tidb-operator/pull/583))
- Add endpoints list and watch permission for controller-manager ([#590](https://github.com/pingcap/tidb-operator/pull/590))
- refine fullbackup ([#570](https://github.com/pingcap/tidb-operator/pull/570))
- Make sure go modules files are always tidy and up to date ([#588](https://github.com/pingcap/tidb-operator/pull/588))
- Local SSD on GKE ([#577](https://github.com/pingcap/tidb-operator/pull/577))
- stability-test: stop kube-proxy case ([#556](https://github.com/pingcap/tidb-operator/pull/556))
- fix resource unit ([#573](https://github.com/pingcap/tidb-operator/pull/573))
- Give local-volume-provisioner pod a QoS of Guaranteed ([#569](https://github.com/pingcap/tidb-operator/pull/569))
- Check PD enpoints status when it's unhealthy. ([#545](https://github.com/pingcap/tidb-operator/pull/545))

# TiDB Operator v1.0.0-beta.3 Release Notes

## v1.0.0-beta.3 What’s New

### Action Required
- ACTION REQUIRED: `nodeSelectorRequired` was removed from values.yaml.
- ACTION REQUIRED:  Comma-separated values support in `nodeSelector` has been dropped, please use new-added `affinity` field which has a more expressive syntax.

### A lot of stability cases added
- ConfigMap rollout
- One PD replicas
- Stop TiDB Operator itself
- TiDB stable scheduling
- Disaster tolerance and data regions disaster tolerance
- Fix many bugs of stability test

### Features added
- Introduce ConfigMap rollout management. With the feature gate open, configuration file changes will be automatically applied to the cluster via a rolling update. Currently, the `scheduler` and `replication` configurations of PD can not be changed via ConfigMap rollout. You can use `pd-ctl` to change these values instead, see [#487](https://github.com/pingcap/tidb-operator/pull/487) for details.
- Support stable scheduling for pods of TiDB members in tidb-scheduler.
- Support adding additional pod annotations for PD/TiKV/TiDB,  e.g. [fluentbit.io/parser](https://docs.fluentbit.io/manual/filter/kubernetes#kubernetes-annotations).
- Support the affinity feature of k8s which can define the rule of assigning pods to nodes
- Allow pausing during TiDB upgrade

### Documentation improved
- GCP one-command deployment
- Refine user guides
- Improve GKE, AWS, Aliyun guide

### Pass User Acceptance Tests

### Other improvements
- Upgrade default TiDB version to v3.0.0-rc.1
- fix bug in reporting assigned nodes of tidb members
- `tkctl get` can show cpu usage correctly now
- Adhoc backup now appends the start time to the PVC name by default.
- add the privileged option for TiKV pod
- `tkctl upinfo` can show nodeIP podIP port now
- get TS and use it before full backup using mydumper
- Fix capabilities issue for `tkctl debug` command

## Detailed Bug Fixes And Changes

- Add capabilities and privilege mode for debug container ([#537](https://github.com/pingcap/tidb-operator/pull/537))
- docs: note helm versions in deployment docs ([#553](https://github.com/pingcap/tidb-operator/pull/553))
- deploy/aws: split public and private subnets when using existing vpc ([#530](https://github.com/pingcap/tidb-operator/pull/530))
- release v1.0.0-beta.3 ([#557](https://github.com/pingcap/tidb-operator/pull/557))
- Gke terraform upgrade to 0.12 and fix bastion instance zone to be region agnostic ([#554](https://github.com/pingcap/tidb-operator/pull/554))
- get TS and use it before full backup using mydumper ([#534](https://github.com/pingcap/tidb-operator/pull/534))
- Add port podip nodeip to tkctl upinfo ([#538](https://github.com/pingcap/tidb-operator/pull/538))
- fix disaster tolerance of stability test ([#543](https://github.com/pingcap/tidb-operator/pull/543))
- add privileged option for tikv pod template ([#550](https://github.com/pingcap/tidb-operator/pull/550))
- use staticcheck instead of megacheck ([#548](https://github.com/pingcap/tidb-operator/pull/548))
- Refine backup and restore documentation ([#518](https://github.com/pingcap/tidb-operator/pull/518))
- Fix stability tidb pause case ([#542](https://github.com/pingcap/tidb-operator/pull/542))
- Fix tkctl get cpu info rendering ([#536](https://github.com/pingcap/tidb-operator/pull/536))
- Fix aliyun tf output rendering and refine documents ([#511](https://github.com/pingcap/tidb-operator/pull/511))
- make webhook configurable ([#529](https://github.com/pingcap/tidb-operator/pull/529))
- Add pods disaster tolerance and data regions disaster tolerance test cases ([#497](https://github.com/pingcap/tidb-operator/pull/497))
- Remove helm hook annotation for initializer job ([#526](https://github.com/pingcap/tidb-operator/pull/526))
- stability test: Add stable scheduling e2e test case ([#524](https://github.com/pingcap/tidb-operator/pull/524))
- upgrade tidb version in related documentations ([#532](https://github.com/pingcap/tidb-operator/pull/532))
- stable scheduling: fix bug in reporting assigned nodes of tidb members ([#531](https://github.com/pingcap/tidb-operator/pull/531))
- reduce wait time and fix stablity test ([#525](https://github.com/pingcap/tidb-operator/pull/525))
- tidb-operator: fix documentation usability issues in GCP document ([#519](https://github.com/pingcap/tidb-operator/pull/519))
- stability cases added: pd replicas 1 and stop tidb-operator ([#496](https://github.com/pingcap/tidb-operator/pull/496))
- pause-upgrade stability test ([#521](https://github.com/pingcap/tidb-operator/pull/521))
- fix restore script bug ([#510](https://github.com/pingcap/tidb-operator/pull/510))
- stability: retry truncating sst files upon failure ([#484](https://github.com/pingcap/tidb-operator/pull/484))
- upgrade default tidb to v3.0.0-rc.1 ([#520](https://github.com/pingcap/tidb-operator/pull/520))
- add --namespace when create backup secret ([#515](https://github.com/pingcap/tidb-operator/pull/515))
- New stability test case for ConfigMap rollout ([#499](https://github.com/pingcap/tidb-operator/pull/499))
- docs: Fix issues found in Queeny's test ([#507](https://github.com/pingcap/tidb-operator/pull/507))
- Pause rolling-upgrade process of tidb statefulset ([#470](https://github.com/pingcap/tidb-operator/pull/470))
- Gke terraform and guide ([#493](https://github.com/pingcap/tidb-operator/pull/493))
- support the affinity feature of k8s which define the rule of assigning pods to nodes ([#475](https://github.com/pingcap/tidb-operator/pull/475))
- Support adding additional pod annotations for PD/TiKV/TiDB ([#500](https://github.com/pingcap/tidb-operator/pull/500))
- Document about PD configuration issue ([#504](https://github.com/pingcap/tidb-operator/pull/504))
- Refine aliyun and aws cloud tidb configurations ([#492](https://github.com/pingcap/tidb-operator/pull/492))
- tidb-operator: update wording and add note ([#502](https://github.com/pingcap/tidb-operator/pull/502))
- Support stable scheduling for TiDB ([#477](https://github.com/pingcap/tidb-operator/pull/477))
- fix `make lint` ([#495](https://github.com/pingcap/tidb-operator/pull/495))
- Support updating configuraion on the fly ([#479](https://github.com/pingcap/tidb-operator/pull/479))
- docs/aws: update AWS deploy docs after testing ([#491](https://github.com/pingcap/tidb-operator/pull/491))
- add release-note to pull_request_template.md ([#490](https://github.com/pingcap/tidb-operator/pull/490))
- Design proposal of stable scheduling in TiDB ([#466](https://github.com/pingcap/tidb-operator/pull/466))
- Update DinD image to make it possible to configure HTTP proxies ([#485](https://github.com/pingcap/tidb-operator/pull/485))
- readme: fix a broken link ([#489](https://github.com/pingcap/tidb-operator/pull/489))
- Fixed typo ([#483](https://github.com/pingcap/tidb-operator/pull/483))

# TiDB Operator v1.0.0-beta.2 Release Notes

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
- Fix bug when using shareinformer without deepcopy

### Other improvements
- Remove pushgateway from TiKV pod
- Add GitHub templates for issue reporting and PR
- Automatically set the scheduler K8s version
- Swith to go module
- Support slow log of TiDB

## Detailed Bug Fixes And Changes

- Don't initialize when there is no tidb.password ([#282](https://github.com/pingcap/tidb-operator/pull/282))
- fix join script ([#285](https://github.com/pingcap/tidb-operator/pull/285))
- Document tool setup and e2e test detail in Contributing.md ([#288](https://github.com/pingcap/tidb-operator/pull/288))
- Update setup.md ([#281](https://github.com/pingcap/tidb-operator/pull/281))
- Support slow log tailing sidcar for tidb instance ([#290](https://github.com/pingcap/tidb-operator/pull/290))
- Flexible tidb initializer job with secret set outside of helm ([#286](https://github.com/pingcap/tidb-operator/pull/286))
- Ensure SLOW_LOG_FILE env variable is always set ([#298](https://github.com/pingcap/tidb-operator/pull/298))
- fix setup document description ([#300](https://github.com/pingcap/tidb-operator/pull/300))
- refactor backup ([#301](https://github.com/pingcap/tidb-operator/pull/301))
- Abandon vendor and refresh go.sum ([#311](https://github.com/pingcap/tidb-operator/pull/311))
- set the SLOW_LOG_FILE in the startup script ([#307](https://github.com/pingcap/tidb-operator/pull/307))
- automatically set the scheduler K8s version ([#313](https://github.com/pingcap/tidb-operator/pull/313))
- tidb stability test main function ([#306](https://github.com/pingcap/tidb-operator/pull/306))
- stability: add fault-trigger server ([#312](https://github.com/pingcap/tidb-operator/pull/312))
- Yinliang/backup and restore add adhoc backup and restore functison ([#316](https://github.com/pingcap/tidb-operator/pull/316))
- stability: add scale & upgrade case functions ([#309](https://github.com/pingcap/tidb-operator/pull/309))
- add slack ([#318](https://github.com/pingcap/tidb-operator/pull/318))
- log dump when test failed ([#317](https://github.com/pingcap/tidb-operator/pull/317))
- stability: add fault-trigger client ([#326](https://github.com/pingcap/tidb-operator/pull/326))
- monitor checker ([#320](https://github.com/pingcap/tidb-operator/pull/320))
- stability: add blockWriter case for inserting data ([#321](https://github.com/pingcap/tidb-operator/pull/321))
- add scheduled-backup test case ([#322](https://github.com/pingcap/tidb-operator/pull/322))
- stability: port ddl test as a workload ([#328](https://github.com/pingcap/tidb-operator/pull/328))
- stability: use fault-trigger at e2e tests and add some log ([#330](https://github.com/pingcap/tidb-operator/pull/330))
- add binlog deploy and check process ([#329](https://github.com/pingcap/tidb-operator/pull/329))
- fix e2e can not make ([#331](https://github.com/pingcap/tidb-operator/pull/331))
- multi tidb cluster testing ([#334](https://github.com/pingcap/tidb-operator/pull/334))
- fix bakcup test bugs ([#335](https://github.com/pingcap/tidb-operator/pull/335))
- delete blockWrite.go use blockwrite.go instead ([#333](https://github.com/pingcap/tidb-operator/pull/333))
- remove vendor ([#344](https://github.com/pingcap/tidb-operator/pull/344))
- stability: add more checks for scale & upgrade ([#327](https://github.com/pingcap/tidb-operator/pull/327))
- stability: support more fault injection ([#345](https://github.com/pingcap/tidb-operator/pull/345))
- rewrite e2e ([#346](https://github.com/pingcap/tidb-operator/pull/346))
- stability: add failover test ([#349](https://github.com/pingcap/tidb-operator/pull/349))
- fix ha when replcias is less than 3 ([#351](https://github.com/pingcap/tidb-operator/pull/351))
- stability: add fault-trigger service file ([#353](https://github.com/pingcap/tidb-operator/pull/353))
- fix dind doc ([#352](https://github.com/pingcap/tidb-operator/pull/352))
- Add additionalPrintColumns for TidbCluster CRD ([#361](https://github.com/pingcap/tidb-operator/pull/361))
- refactor stability main function ([#363](https://github.com/pingcap/tidb-operator/pull/363))
- enable admin privelege for prom ([#360](https://github.com/pingcap/tidb-operator/pull/360))
- Updated Readme with New Info ([#365](https://github.com/pingcap/tidb-operator/pull/365))
- Build CLI ([#357](https://github.com/pingcap/tidb-operator/pull/357))
- add extraLabels variable in tidb-cluster chart ([#373](https://github.com/pingcap/tidb-operator/pull/373))
- fix tikv failover ([#368](https://github.com/pingcap/tidb-operator/pull/368))
- Separate and ensure setup before e2e-build ([#375](https://github.com/pingcap/tidb-operator/pull/375))
- Fix codegen.sh and lock related depedencies ([#371](https://github.com/pingcap/tidb-operator/pull/371))
- stability: add sst-file-corruption case ([#382](https://github.com/pingcap/tidb-operator/pull/382))
- use release name as default clusterName ([#354](https://github.com/pingcap/tidb-operator/pull/354))
- Add util class to support to add annotations to Grafana ([#378](https://github.com/pingcap/tidb-operator/pull/378))
- Use grafana provisioning to replace dashboard installer ([#388](https://github.com/pingcap/tidb-operator/pull/388))
- ensure test env is ready before cases running ([#386](https://github.com/pingcap/tidb-operator/pull/386))
- remove monitor config job check ([#390](https://github.com/pingcap/tidb-operator/pull/390))
- Update local-pv documentation ([#383](https://github.com/pingcap/tidb-operator/pull/383))
- Update Jenkins links in README.md ([#395](https://github.com/pingcap/tidb-operator/pull/395))
- fix e2e workflow in CONTRIBUTING.md ([#392](https://github.com/pingcap/tidb-operator/pull/392))
- Support running stability test out of cluster ([#397](https://github.com/pingcap/tidb-operator/pull/397))
- update tidb secret docs and charts ([#398](https://github.com/pingcap/tidb-operator/pull/398))
- Enable blockWriter write pressure in stability test ([#399](https://github.com/pingcap/tidb-operator/pull/399))
- Support debug and ctop commands in CLI ([#387](https://github.com/pingcap/tidb-operator/pull/387))
- marketplace update ([#380](https://github.com/pingcap/tidb-operator/pull/380))
- dashboard:update editable value from true to false ([#394](https://github.com/pingcap/tidb-operator/pull/394))
- add fault inject for kube proxy ([#384](https://github.com/pingcap/tidb-operator/pull/384))
- use `ioutil.TempDir()` create charts and operator repo's directories ([#405](https://github.com/pingcap/tidb-operator/pull/405))
- Improve workflow in docs/google-kubernetes-tutorial.md ([#400](https://github.com/pingcap/tidb-operator/pull/400))
- Support plugin start argument for tidb instance ([#412](https://github.com/pingcap/tidb-operator/pull/412))
- Replace govet with official vet tool ([#416](https://github.com/pingcap/tidb-operator/pull/416))
- allocate 24 PVs by default (after 2 clusters are scaled to ([#407](https://github.com/pingcap/tidb-operator/pull/407))
- refine stability ([#422](https://github.com/pingcap/tidb-operator/pull/422))
- Record event as grafana annotation in stability test ([#414](https://github.com/pingcap/tidb-operator/pull/414))
- add GitHub templates for issue reporting and PR ([#420](https://github.com/pingcap/tidb-operator/pull/420))
- add TiDBUpgrading func ([#423](https://github.com/pingcap/tidb-operator/pull/423))
- fix operator chart issue ([#419](https://github.com/pingcap/tidb-operator/pull/419))
- fix stability issues ([#433](https://github.com/pingcap/tidb-operator/pull/433))
- change cert generate method and add pd and kv prestop webhook ([#406](https://github.com/pingcap/tidb-operator/pull/406))
- a tidb-scheduler bug fix and emit event when scheduled failed ([#427](https://github.com/pingcap/tidb-operator/pull/427))
- Shell completion for tkctl ([#431](https://github.com/pingcap/tidb-operator/pull/431))
- Delete an duplicate import ([#434](https://github.com/pingcap/tidb-operator/pull/434))
- add etcd and kube-apiserver faults ([#367](https://github.com/pingcap/tidb-operator/pull/367))
- Fix TiDB Slack link ([#444](https://github.com/pingcap/tidb-operator/pull/444))
- fix scheduler ha bug ([#443](https://github.com/pingcap/tidb-operator/pull/443))
- add terraform script to auto deploy TiDB cluster on AWS ([#401](https://github.com/pingcap/tidb-operator/pull/401))
- Adds instructions to access Grafana in GKE tutorial ([#448](https://github.com/pingcap/tidb-operator/pull/448))
- fix label selector ([#437](https://github.com/pingcap/tidb-operator/pull/437))
- no need to set ClusterIP when syncing headless service ([#432](https://github.com/pingcap/tidb-operator/pull/432))
- docs on how to deploy tidb cluster with tidb-operator in minikube ([#451](https://github.com/pingcap/tidb-operator/pull/451))
- add slack notify ([#439](https://github.com/pingcap/tidb-operator/pull/439))
- fix local dind env ([#440](https://github.com/pingcap/tidb-operator/pull/440))
- Add terraform scripts to support alibaba cloud ACK deployment ([#436](https://github.com/pingcap/tidb-operator/pull/436))
- Fix backup data compare logic ([#454](https://github.com/pingcap/tidb-operator/pull/454))
- stability test: async emit annotations ([#438](https://github.com/pingcap/tidb-operator/pull/438))
- Use TiDB v2.1.8 by default & remove pushgateway ([#435](https://github.com/pingcap/tidb-operator/pull/435))
- Fix bug use shareinformer without copy ([#462](https://github.com/pingcap/tidb-operator/pull/462))
- Add version command for tkctl ([#456](https://github.com/pingcap/tidb-operator/pull/456))
- Add tkctl user manual ([#452](https://github.com/pingcap/tidb-operator/pull/452))
- Fix binlog problem on large scale ([#460](https://github.com/pingcap/tidb-operator/pull/460))
- Copy kubernetes.io/hostname label to PVs ([#464](https://github.com/pingcap/tidb-operator/pull/464))
- AWS EKS tutorial change to new terraform script ([#463](https://github.com/pingcap/tidb-operator/pull/463))
- docs/minikube: update documentation of minikube installation ([#471](https://github.com/pingcap/tidb-operator/pull/471))
- docs/dind: update documentation of DinD installation ([#458](https://github.com/pingcap/tidb-operator/pull/458))
- docs/minikube: add instructions to access Grafana ([#476](https://github.com/pingcap/tidb-operator/pull/476))
- support-multi-version-dashboard ([#473](https://github.com/pingcap/tidb-operator/pull/473))
- docs/aliyun: update aliyun deploy docs after testing ([#474](https://github.com/pingcap/tidb-operator/pull/474))
- GKE local SSD size warning ([#467](https://github.com/pingcap/tidb-operator/pull/467))
- update roadmap ([#376](https://github.com/pingcap/tidb-operator/pull/376))
