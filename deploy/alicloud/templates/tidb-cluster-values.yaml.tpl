# Default values for tidb-cluster.
# This is a YAML-formatted file.
# Declare variables to be passed into your templates.

# Also see monitor.serviceAccount
# If you set rbac.create to false, you need to provide a value for monitor.serviceAccount
rbac:
  create: true

# clusterName is the TiDB cluster name, if not specified, the chart release name will be used
# clusterName: demo

# Add additional TidbCluster labels
# ref: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
extraLabels: {}

# schedulerName must be same with charts/tidb-operator/values#scheduler.schedulerName
schedulerName: tidb-scheduler

# timezone is the default system timzone for TiDB
timezone: UTC

# default reclaim policy of a PV
pvReclaimPolicy: Retain

# services is the service list to expose, default is ClusterIP
# can be ClusterIP | NodePort | LoadBalancer
services:
  - name: pd
    type: ClusterIP

discovery:
  image: pingcap/tidb-operator:latest
  imagePullPolicy: IfNotPresent
  resources:
    limits:
      cpu: 250m
      memory: 150Mi
    requests:
      cpu: 80m
      memory: 50Mi

pd:
  replicas: ${pd_replicas}
  image: "pingcap/pd:${cluster_version}"
  logLevel: info
  # storageClassName is a StorageClass provides a way for administrators to describe the "classes" of storage they offer.
  # different classes might map to quality-of-service levels, or to backup policies,
  # or to arbitrary policies determined by the cluster administrators.
  # refer to https://kubernetes.io/docs/concepts/storage/storage-classes
  storageClassName: ${local_storage_class}

  # Image pull policy.
  imagePullPolicy: IfNotPresent

  # maxStoreDownTime is how long a store will be considered `down` when disconnected
  # if a store is considered `down`, the regions will be migrated to other stores
  maxStoreDownTime: 30m
  # maxReplicas is the number of replicas for each region
  maxReplicas: 3
  resources:
    limits: {}
    #   cpu: 8000m
    #   memory: 8Gi
    requests:
      # cpu: 4000m
      # memory: 4Gi
      storage: ${pd_storage_size}
  # nodeSelector is used for scheduling pod,
  # if nodeSelectorRequired is true, all the following labels must be matched
  nodeSelector:
    dedicated: pd
    # kind: pd
    # # zone is comma separated availability zone list
    # zone: cn-bj1-01,cn-bj1-02
    # # region is comma separated region list
    # region: cn-bj1
  # Tolerations are applied to pods, and allow pods to schedule onto nodes with matching taints.
  # refer to https://kubernetes.io/docs/concepts/configuration/taint-and-toleration
  tolerations:
  - key: dedicated
    operator: Equal
    value: pd
    effect: "NoSchedule"

tikv:
  replicas: ${tikv_replicas}
  image: "pingcap/tikv:${cluster_version}"
  logLevel: info
  # storageClassName is a StorageClass provides a way for administrators to describe the "classes" of storage they offer.
  # different classes might map to quality-of-service levels, or to backup policies,
  # or to arbitrary policies determined by the cluster administrators.
  # refer to https://kubernetes.io/docs/concepts/storage/storage-classes
  storageClassName: ${local_storage_class}

  # Image pull policy.
  imagePullPolicy: IfNotPresent

  # syncLog is a bool value to enable or disable syc-log for raftstore, default is true
  # enable this can prevent data loss when power failure
  syncLog: true
  # size of thread pool for grpc server.
  # grpcConcurrency: 4
  resources:
    limits: {}
    #   cpu: 16000m
    #   memory: 32Gi
    #   storage: 300Gi
    requests:
      # cpu: 12000m
      # memory: 24Gi
      storage: ${tikv_storage_size}
  nodeSelector:
    dedicated: tikv
  tolerations:
  - key: dedicated
    operator: Equal
    value: tikv
    effect: "NoSchedule"

  # block-cache used to cache uncompressed blocks, big block-cache can speed up read.
  # in normal cases should tune to 30%-50% tikv.resources.limits.memory
  defaultcfBlockCacheSize: "${tikv_defaultcf_block_cache_size}"

  # in normal cases should tune to 10%-30% tikv.resources.limits.memory
  writecfBlockCacheSize: "${tikv_writecf_block_cache_size}"

  # size of thread pool for high-priority/normal-priority/low-priority operations
  # readpoolStorageConcurrency: 4

  # Notice: if tikv.resources.limits.cpu > 8, default thread pool size for coprocessors
  # will be set to tikv.resources.limits.cpu * 0.8.
  # readpoolCoprocessorConcurrency: 8

  # scheduler's worker pool size, should increase it in heavy write cases,
  # also should less than total cpu cores.
  # storageSchedulerWorkerPoolSize: 4

tidb:
  replicas: ${tidb_replicas}
  # The secret name of root password, you can create secret with following command:
  # kubectl create secret generic tidb-secret --from-literal=root=<root-password> --namespace=<namespace>
  # If unset, the root password will be empty and you can set it after connecting
  # passwordSecretName: tidb-secret
  # initSql is the SQL statements executed after the TiDB cluster is bootstrapped.
  # initSql: |-
  #   create database app;
  image: "pingcap/tidb:${cluster_version}"
  # Image pull policy.
  imagePullPolicy: IfNotPresent
  logLevel: info
  preparedPlanCacheEnabled: false
  preparedPlanCacheCapacity: 100
  # Enable local latches for transactions. Enable it when
  # there are lots of conflicts between transactions.
  txnLocalLatchesEnabled: false
  txnLocalLatchesCapacity: "10240000"
  # The limit of concurrent executed sessions.
  tokenLimit: "1000"
  # Set the memory quota for a query in bytes. Default: 32GB
  memQuotaQuery: "34359738368"
  # The limitation of the number for the entries in one transaction.
  # If using TiKV as the storage, the entry represents a key/value pair.
  # WARNING: Do not set the value too large, otherwise it will make a very large impact on the TiKV cluster.
  # Please adjust this configuration carefully.
  txnEntryCountLimit: "300000"
  # The limitation of the size in byte for the entries in one transaction.
  # If using TiKV as the storage, the entry represents a key/value pair.
  # WARNING: Do not set the value too large, otherwise it will make a very large impact on the TiKV cluster.
  # Please adjust this configuration carefully.
  txnTotalSizeLimit: "104857600"
  # enableBatchDml enables batch commit for the DMLs
  enableBatchDml: false
  # check mb4 value in utf8 is used to control whether to check the mb4 characters when the charset is utf8.
  checkMb4ValueInUtf8: true
  # treat-old-version-utf8-as-utf8mb4 use for upgrade compatibility. Set to true will treat old version table/column UTF8 charset as UTF8MB4.
  treatOldVersionUtf8AsUtf8mb4: true
  # lease is schema lease duration, very dangerous to change only if you know what you do.
  lease: 45s
  # Max CPUs to use, 0 use number of CPUs in the machine.
  maxProcs: 0
  resources:
    limits: {}
    #   cpu: 16000m
    #   memory: 16Gi
    requests: {}
    #   cpu: 12000m
    #   memory: 12Gi
  nodeSelector:
    dedicated: tidb
    # kind: tidb
    # zone: cn-bj1-01,cn-bj1-02
    # region: cn-bj1
  tolerations:
  - key: dedicated
    operator: Equal
    value: tidb
    effect: "NoSchedule"
  maxFailoverCount: 3
  service:
    type: LoadBalancer
    exposeStatus: true
    annotations:
      service.beta.kubernetes.io/alicloud-loadbalancer-address-type: intranet
      service.beta.kubernetes.io/alicloud-loadbalancer-slb-network-type: vpc
  # separateSlowLog: true
  slowLogTailer:
    image: busybox:1.26.2
    resources:
      limits:
        cpu: 100m
        memory: 50Mi
      requests:
        cpu: 20m
        memory: 5Mi

  # tidb plugin configuration
  plugin:
    # enable plugin or not
    enable: false
    # the start argument to specify the folder containing
    directory: /plugins
    # the start argument to specify the plugin id (name "-" version) that needs to be loaded, e.g. 'conn_limit-1'.
    list: ["whitelist-1"]

# mysqlClient is used to set password for TiDB
# it must has Python MySQL client installed
mysqlClient:
  image: tnir/mysqlclient
  imagePullPolicy: IfNotPresent

monitor:
  create: true
  # Also see rbac.create
  # If you set rbac.create to false, you need to provide a value here.
  # If you set rbac.create to true, you should leave this empty.
  # serviceAccount:
  persistent: true
  storageClassName: ${monitor_storage_class}
  storage: ${monitor_storage_size}
  grafana:
    create: true
    image: grafana/grafana:6.0.1
    imagePullPolicy: IfNotPresent
    logLevel: info
    resources:
      limits: {}
      #   cpu: 8000m
      #   memory: 8Gi
      requests: {}
      #   cpu: 4000m
      #   memory: 4Gi
    username: admin
    password: admin
    config:
      # Configure Grafana using environment variables except GF_PATHS_DATA, GF_SECURITY_ADMIN_USER and GF_SECURITY_ADMIN_PASSWORD
      # Ref https://grafana.com/docs/installation/configuration/#using-environment-variables
      GF_AUTH_ANONYMOUS_ENABLED: "true"
      GF_AUTH_ANONYMOUS_ORG_NAME: "Main Org."
      GF_AUTH_ANONYMOUS_ORG_ROLE: "Viewer"
      # if grafana is running behind a reverse proxy with subpath http://foo.bar/grafana
      # GF_SERVER_DOMAIN: foo.bar
      # GF_SERVER_ROOT_URL: "%(protocol)s://%(domain)s/grafana/"
    service:
      type: LoadBalancer
      annotations:
        service.beta.kubernetes.io/alicloud-loadbalancer-address-type: ${monitor_slb_network_type}
  prometheus:
    image: prom/prometheus:v2.2.1
    imagePullPolicy: IfNotPresent
    logLevel: info
    resources:
      limits: {}
      #   cpu: 8000m
      #   memory: 8Gi
      requests: {}
      #   cpu: 4000m
      #   memory: 4Gi
    service:
      type: NodePort
    reserveDays: ${monitor_reserve_days}
    # alertmanagerURL: ""
  nodeSelector: {}
    # kind: monitor
    # zone: cn-bj1-01,cn-bj1-02
    # region: cn-bj1
  tolerations: []
  # - key: node-role
  #   operator: Equal
  #   value: tidb
  #   effect: "NoSchedule"

binlog:
  pump:
    create: false
    replicas: 1
    image: "pingcap/tidb-binlog:${cluster_version}"
    imagePullPolicy: IfNotPresent
    logLevel: info
    # storageClassName is a StorageClass provides a way for administrators to describe the "classes" of storage they offer.
    # different classes might map to quality-of-service levels, or to backup policies,
    # or to arbitrary policies determined by the cluster administrators.
    # refer to https://kubernetes.io/docs/concepts/storage/storage-classes
    storageClassName: ${local_storage_class}
    storage: 10Gi
    syncLog: true
    # a integer value to control expiry date of the binlog data, indicates for how long (in days) the binlog data would be stored.
    # must bigger than 0
    gc: 7
    # number of seconds between heartbeat ticks (in 2 seconds)
    heartbeatInterval: 2

  drainer:
    create: false
    image: "pingcap/tidb-binlog:${cluster_version}"
    imagePullPolicy: IfNotPresent
    logLevel: info
    # storageClassName is a StorageClass provides a way for administrators to describe the "classes" of storage they offer.
    # different classes might map to quality-of-service levels, or to backup policies,
    # or to arbitrary policies determined by the cluster administrators.
    # refer to https://kubernetes.io/docs/concepts/storage/storage-classes
    storageClassName: ${local_storage_class}
    storage: 10Gi
    # parallel worker count (default 16)
    workerCount: 16
    # the interval time (in seconds) of detect pumps' status (default 10)
    detectInterval: 10
    # disbale detect causality
    disableDetect: false
    # disable dispatching sqls that in one same binlog; if set true, work-count and txn-batch would be useless
    disableDispatch: false
    # # disable sync these schema
    ignoreSchemas: "INFORMATION_SCHEMA,PERFORMANCE_SCHEMA,mysql,test"
    # if drainer donesn't have checkpoint, use initial commitTS to initial checkpoint
    initialCommitTs: 0
    # enable safe mode to make syncer reentrant
    safeMode: false
    # number of binlog events in a transaction batch (default 20)
    txnBatch: 20
    # downstream storage, equal to --dest-db-type
    # valid values are "mysql", "pb", "kafka"
    destDBType: pb
    mysql: {}
      # host: "127.0.0.1"
      # user: "root"
      # password: ""
      # port: 3306
      # # Time and size limits for flash batch write
      # timeLimit: "30s"
      # sizeLimit: "100000"
    kafka: {}
      # only need config one of zookeeper-addrs and kafka-addrs, will get kafka address if zookeeper-addrs is configed.
      # zookeeperAddrs: "127.0.0.1:2181"
      # kafkaAddrs: "127.0.0.1:9092"
      # kafkaVersion: "0.8.2.0"

scheduledBackup:
  create: false
  binlogImage: "pingcap/tidb-binlog:${cluster_version}"
  binlogImagePullPolicy: IfNotPresent
  # https://github.com/tennix/tidb-cloud-backup
  mydumperImage: pingcap/tidb-cloud-backup:latest
  mydumperImagePullPolicy: IfNotPresent
  # storageClassName is a StorageClass provides a way for administrators to describe the "classes" of storage they offer.
  # different classes might map to quality-of-service levels, or to backup policies,
  # or to arbitrary policies determined by the cluster administrators.
  # refer to https://kubernetes.io/docs/concepts/storage/storage-classes
  storageClassName: ${local_storage_class}
  storage: 100Gi
  # https://kubernetes.io/docs/tasks/job/automated-tasks-with-cron-jobs/#schedule
  schedule: "0 0 * * *"
  # https://kubernetes.io/docs/tasks/job/automated-tasks-with-cron-jobs/#suspend
  suspend: false
  # https://kubernetes.io/docs/tasks/job/automated-tasks-with-cron-jobs/#jobs-history-limits
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1
  # https://kubernetes.io/docs/tasks/job/automated-tasks-with-cron-jobs/#starting-deadline
  startingDeadlineSeconds: 3600
  # https://github.com/maxbube/mydumper/blob/master/docs/mydumper_usage.rst#options
  options: "--chunk-filesize=100"
  # secretName is the name of the secret which stores user and password used for backup
  # Note: you must give the user enough privilege to do the backup
  # you can create the secret by:
  # kubectl create secret generic backup-secret --from-literal=user=root --from-literal=password=<password>
  secretName: backup-secret
  # backup to gcp
  gcp: {}
  # bucket: ""
  # secretName is the name of the secret which stores the gcp service account credentials json file
  # The service account must have read/write permission to the above bucket.
  # Read the following document to create the service account and download the credentials file as credentials.json:
  # https://cloud.google.com/docs/authentication/production#obtaining_and_providing_service_account_credentials_manually
  # And then create the secret by: kubectl create secret generic gcp-backup-secret --from-file=./credentials.json
  # secretName: gcp-backup-secret

  # backup to ceph object storage
  ceph: {}
  # endpoint: ""
  # bucket: ""
  # secretName is the name of the secret which stores ceph object store access key and secret key
  # You can create the secret by:
  # kubectl create secret generic ceph-backup-secret --from-literal=access_key=<access-key> --from-literal=secret_key=<secret-key>
  # secretName: ceph-backup-secret

metaInstance: "{{ $labels.instance }}"
metaType: "{{ $labels.type }}"
metaValue: "{{ $value }}"
