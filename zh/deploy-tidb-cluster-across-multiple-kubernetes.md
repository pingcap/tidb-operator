---
title: 跨多个 Kubernetes 集群部署 TiDB 集群
summary: 本文档介绍如何实现跨多个 Kubernetes 集群部署 TiDB 集群
---

> **警告：**
>
> 当前该功能为实验特性，不建议在生产环境中使用。

# 跨多个 Kubernetes 集群部署 TiDB 集群

跨多个 Kubernetes 集群部署 TiDB 集群，是指在多个网络互通的 Kubernetes 集群上部署**一个** TiDB 集群，集群各个组件分布在多个 Kubernetes 集群上，实现在 Kubernetes 集群间容灾。所谓 Kubernetes 集群网络互通，是指 Pod IP 在任意集群内和集群间可以被互相访问，Pod FQDN 记录在集群内和集群间均可被解析。

## 前置条件

需要配置 Kubernetes 的网络和 DNS，使得 Kubernetes 集群满足以下条件：

- 各 Kubernetes 集群上的 TiDB 组件有能力访问集群内和集群间所有 TiDB 组件的 Pod IP。
- 各 Kubernetes 集群上的 TiDB 组件有能力解析集群内和集群间所有 TiDB 组件的 Pod FQDN。

## 支持场景

目前支持场景：

- 新部署跨多个 Kubernetes 集群的 TiDB 集群。
- 在其他 Kubernetes 集群上部署开启此功能的新集群加入同样开启此功能的集群。

实验性支持场景：

- 对已有数据的集群从未开启此功能状态变为开启此功能状态，如需在生产环境中使用，建议通过数据迁移完成此需求。

不支持场景：

- 两个已有数据集群互相连通，对于这一场景应通过数据迁移完成。

## 跨多个 Kubernetes 集群部署集群

部署跨多个 Kubernetes 集群的 TiDB 集群，默认你已部署好此场景所需要的 Kubernetes 集群，在此基础上进行下面的部署工作。

下面以部署两个集群为例进行介绍，其中集群 1 为初始集群，按照下面给出的配置进行创建，集群 1 正常运行后，按照下面给出配置创建集群 2，等集群完成创建和部署工作后，两个集群正常运行。

### 部署初始集群

根据实际情况设置以下环境变量，实际使用中需要根据你的实际情况设置 `cluster1_name` 和 `cluster1_cluster_domain` 变量的内容，其中 `cluster1_name` 为集群 1 的集群名称，`cluster1_cluster_domain` 为集群 1 的 [Cluster Domain](https://kubernetes.io/docs/tasks/administer-cluster/dns-custom-nameservers/#introduction), `cluster1_namespace` 为集群 1 的命名空间。

{{< copyable "shell-regular" >}}

```bash
cluster1_name="cluster1"
cluster1_cluster_domain="cluster1.com"
cluster1_namespace="pingcap"
```

执行以下指令：

{{< copyable "shell-regular" >}}

```bash
cat << EOF | kubectl apply -n ${cluster1_namespace} -f -
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: "${cluster1_name}"
spec:
  version: v5.2.0
  timezone: UTC
  pvReclaimPolicy: Delete
  enableDynamicConfiguration: true
  configUpdateStrategy: RollingUpdate
  clusterDomain: "${cluster1_cluster_domain}"
  discovery: {}
  pd:
    baseImage: pingcap/pd
    replicas: 1
    requests:
      storage: "10Gi"
    config: {}
  tikv:
    baseImage: pingcap/tikv
    replicas: 1
    requests:
      storage: "10Gi"
    config: {}
  tidb:
    baseImage: pingcap/tidb
    replicas: 1
    service:
      type: ClusterIP
    config: {}
EOF
```

### 部署新集群加入初始集群

等待集群 1 完成部署后，创建集群 2。在实际使用中，新创建的集群 2 可以加入多集群内的任意一个已有集群。

可以参考下面的范例，根据实际情况设置填入集群 1 和集群 2 的 `Name`、`Cluster Domain`、`Namespace` 等相关信息：

{{< copyable "shell-regular" >}}

```bash
cluster1_name="cluster1"
cluster1_cluster_domain="cluster1.com"
cluster1_namespace="pingcap"

cluster2_name="cluster2"
cluster2_cluster_domain="cluster2.com"
cluster2_namespace="pingcap"
```

执行以下指令：

{{< copyable "shell-regular" >}}

```bash
cat << EOF | kubectl apply -n ${cluster2_namespace} -f -
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: "${cluster2_name}"
spec:
  version: v5.2.0
  timezone: UTC
  pvReclaimPolicy: Delete
  enableDynamicConfiguration: true
  configUpdateStrategy: RollingUpdate
  clusterDomain: "${cluster2_cluster_domain}"
  cluster:
    name: "${cluster1_name}"
    namespace: "${cluster1_namespace}"
    clusterDomain: "${cluster1_clusterdomain}"
  discovery: {}
  pd:
    baseImage: pingcap/pd
    replicas: 1
    requests:
      storage: "10Gi"
    config: {}
  tikv:
    baseImage: pingcap/tikv
    replicas: 1
    requests:
      storage: "10Gi"
    config: {}
  tidb:
    baseImage: pingcap/tidb
    replicas: 1
    service:
      type: ClusterIP
    config: {}
EOF
```

## 跨多个 Kubernetes 集群部署开启组件间 TLS 的 TiDB 集群

可以按照以下步骤为跨多个 Kubernetes 集群部署的 TiDB 集群开启组件间 TLS。

### 签发根证书

#### 使用 cfssl 系统签发根证书

如果你使用 `cfssl`，签发 CA 证书的过程与一般签发过程没有差别，需要保存好第一次创建的 CA 证书，并且在后面为 TiDB 组件签发证书时都使用这个 CA 证书，即在为其他集群创建组件证书时，不需要再次创建 CA 证书，你只需要完成一次[为 TiDB 组件间开启 TLS](enable-tls-between-components.md#使用-cfssl-系统颁发证书) 文档中 1 ~ 4 步操作，完成 CA 证书签发，为其他集群组件间证书签发操作从第 5 步开始即可。

#### 使用 cert-manager 系统签发根证书

如果你使用 `cert-manager`，只需要在初始集群创建 `CA Issuer` 和创建 `CA Certificate`，并导出 `CA Secret` 给其他准备加入的新集群，其他集群只需要创建组件证书签发 `Issuer`（在 [TLS 文档](enable-tls-between-components.md#使用-cert-manager-系统颁发证书)中指名字为 `${cluster_name}-tidb-issuer` 的 `Issuer`），配置 `Issuer` 使用该 CA，具体过程如下：

1. 在初始集群上创建 `CA Issuer` 和创建 `CA Certificate`。

    根据实际情况设置以下环境变量：

    {{< copyable "shell-regular" >}}

    ```bash
    cluster_name="cluster1"
    namespace="pingcap"
    ```

    执行以下指令：

    {{< copyable "shell-regular" >}}

    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: cert-manager.io/v1alpha2
    kind: Issuer
    metadata:
      name: ${cluster_name}-selfsigned-ca-issuer
      namespace: ${namespace}
    spec:
      selfSigned: {}
    ---
    apiVersion: cert-manager.io/v1alpha2
    kind: Certificate
    metadata:
      name: ${cluster_name}-ca
      namespace: ${namespace}
    spec:
      secretName: ${cluster_name}-ca-secret
      commonName: "TiDB"
      isCA: true
      duration: 87600h # 10yrs
      renewBefore: 720h # 30d
      issuerRef:
        name: ${cluster_name}-selfsigned-ca-issuer
        kind: Issuer
    EOF
    ```

2. 导出 CA 并删除无关信息。

    首先需要导出存放 CA 的 `Secret`，`Secret` 的名字可以由第一步 `Certificate` 的 `.spec.secretName` 得到。

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl get secret cluster1-ca-secret -n ${namespace} -o yaml > ca.yaml
    ```

    删除 Secret YAML 文件中无关信息，删除后 YAML 文件如下所示，其中 `data` 内信息已省略：

    ```yaml
    apiVersion: v1
    data:
      ca.crt: LS0...LQo=
      tls.crt: LS0t....LQo=
      tls.key: LS0t...tCg==
    kind: Secret
    metadata:
      name: cluster1-ca-secret
    type: kubernetes.io/tls
    ```

3. 将导出的 CA 导入到其他集群。

    你需要配置这里的 `namespace` 使得相关组件可以访问到 CA 证书：

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl apply -f ca.yaml -n ${namespace}
    ```

4. 在初始集群和新集群创建组件证书签发 `Issuer`，使用该 CA。

    1. 在初始集群上，创建组件间证书签发 `Issuer`。

        根据实际情况设置以下环境变量：

        {{< copyable "shell-regular" >}}

        ```bash
        cluster_name="cluster1"
        namespace="pingcap"
        ca_secret_name="cluster1-ca-secret"
        ```

        执行以下指令：

        {{< copyable "shell-regular" >}}

        ```bash
        cat << EOF | kubectl apply -f -
        apiVersion: cert-manager.io/v1alpha2
        kind: Issuer
        metadata:
          name: ${cluster_name}-tidb-issuer
          namespace: ${namespace}
        spec:
          ca:
            secretName: ${ca_secret_name}
        EOF
        ```

    2. 在新集群上，创建组件间证书签发 `Issuer`。

       根据实际情况设置以下环境变量，其中 `ca_secret_name` 需要指向你刚才导入的存放 `CA` 的 `Secret`，`cluster_name` 和 `namespace` 在下面的操作中需要用到：

       {{< copyable "shell-regular" >}}

       ```bash
       cluster_name="cluster2"
       namespace="pingcap"
       ca_secret_name="cluster1-ca-secret"
       ```
       
       执行以下指令：

       {{< copyable "shell-regular" >}}

       ```bash
       cat << EOF | kubectl apply -f -
       apiVersion: cert-manager.io/v1alpha2
       kind: Issuer
       metadata:
         name: ${cluster_name}-tidb-issuer
         namespace: ${namespace}
       spec:
         ca:
           secretName: ${ca_secret_name}
       EOF
       ```

### 为各个 Kubernetes 集群的 TiDB 组件签发证书

你需要为每个 Kubernetes 集群上的 TiDB 组件都签发组件证书。在签发组件证书时，需要在 hosts 中加上以 `.${cluster_domain}` 结尾的授权记录， 例如 `${cluster_name}-pd.${namespace}.svc.${cluster_domain}`。

#### 使用 cfssl 系统为 TiDB 组件签发证书

如果使用 `cfssl`，以创建 PD 组件所使用的证书为例，`pd-server.json` 文件如下所示。

根据实际情况设置以下环境变量：

{{< copyable "shell-regular" >}}

```bash
cluster_name=cluster2
cluster_domain=cluster2.com
namespace=pingcap
```

`pd-server.json`可以通过如下指令创建：

{{< copyable "shell-regular" >}}

```bash
cat << EOF > pd-server.json
{
    "CN": "TiDB",
    "hosts": [
      "127.0.0.1",
      "::1",
      "${cluster_name}-pd",
      "${cluster_name}-pd.${namespace}",
      "${cluster_name}-pd.${namespace}.svc",
      "${cluster_name}-pd.${namespace}.svc.${cluster_domain}",
      "${cluster_name}-pd-peer",
      "${cluster_name}-pd-peer.${namespace}",
      "${cluster_name}-pd-peer.${namespace}.svc",
      "${cluster_name}-pd-peer.${namespace}.svc.${cluster_domain}",
      "*.${cluster_name}-pd-peer",
      "*.${cluster_name}-pd-peer.${namespace}",
      "*.${cluster_name}-pd-peer.${namespace}.svc",
      "*.${cluster_name}-pd-peer.${namespace}.svc.${cluster_domain}"
    ],
    "key": {
        "algo": "ecdsa",
        "size": 256
    },
    "names": [
        {
            "C": "US",
            "L": "CA",
            "ST": "San Francisco"
        }
    ]
}
EOF
```

#### 使用 cert-manager 系统为 TiDB 组件签发证书

如果使用 `cert-manager`，以创建 PD 组件所使用的证书为例，`Certifcates` 如下所示。

通过以下指令设置环境变量：

{{< copyable "shell-regular" >}}

```bash
cluster_name="cluster2"
namespace="pingcap"
cluster_domain="cluster2.com"
```

执行以下指令：

{{< copyable "shell-regular" >}}

```bash
cat << EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: ${cluster_name}-pd-cluster-secret
  namespace: ${namespace}
spec:
  secretName: ${cluster_name}-pd-cluster-secret
  duration: 8760h # 365d
  renewBefore: 360h # 15d
  organization:
  - PingCAP
  commonName: "TiDB"
  usages:
    - server auth
    - client auth
  dnsNames:
    - "${cluster_name}-pd"
    - "${cluster_name}-pd.${namespace}"
    - "${cluster_name}-pd.${namespace}.svc"
    - "${cluster_name}-pd.${namespace}.svc.${cluster_domain}"
    - "${cluster_name}-pd-peer"
    - "${cluster_name}-pd-peer.${namespace}"
    - "${cluster_name}-pd-peer.${namespace}.svc"
    - "${cluster_name}-pd-peer.${namespace}.svc.${cluster_domain}"
    - "*.${cluster_name}-pd-peer"
    - "*.${cluster_name}-pd-peer.${namespace}"
    - "*.${cluster_name}-pd-peer.${namespace}.svc"
    - "*.${cluster_name}-pd-peer.${namespace}.svc.${cluster_domain}"
  ipAddresses:
  - 127.0.0.1
  - ::1
  issuerRef:
    name: ${cluster_name}-tidb-issuer
    kind: Issuer
    group: cert-manager.io
EOF
```

需要参考 TLS 相关文档，为组件签发对应的证书，并在相应 Kubernetes 集群中创建 Secret。

其他 TLS 相关信息，可参考以下文档：

- [为 TiDB 组件间开启 TLS](enable-tls-between-components.md)
- [为 MySQL 客户端开启 TLS](enable-tls-for-mysql-client.md)

### 部署初始集群

通过如下命令部署初始化集群，实际使用中需要根据你的实际情况设置 `cluster1_name` 和 `cluster1_cluster_domain` 变量的内容，其中 `cluster1_name` 为集群 1 的集群名称，`cluster1_cluster_domain` 为集群 1 的 `Cluster Domain`，`cluster1_namespace` 为集群 1 的命名空间。下面的 YAML 文件已经开启了 TLS 功能，并通过配置 `cert-allowed-cn`，使得各个组件开始验证由 `CN` 为 `TiDB` 的 `CA` 所签发的证书。

根据实际情况设置以下环境变量：

{{< copyable "shell-regular" >}}

```bash
cluster1_name="cluster1"
cluster1_cluster_domain="cluster1.com"
cluster1_namespace="pingcap"
```

执行以下指令：

```
cat << EOF | kubectl apply -n ${cluster1_namespace} -f -
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: "${cluster1_name}"
spec:
  version: v5.2.0
  timezone: UTC
  tlsCluster:
   enabled: true
  pvReclaimPolicy: Delete
  enableDynamicConfiguration: true
  configUpdateStrategy: RollingUpdate
  clusterDomain: "${cluster1_cluster_domain}"
  discovery: {}
  pd:
    baseImage: pingcap/pd
    replicas: 1
    requests:
      storage: "10Gi"
    config: 
      security:
        cert-allowed-cn:
          - TiDB
  tikv:
    baseImage: pingcap/tikv
    replicas: 1
    requests:
      storage: "10Gi"
    config: 
      security:
       cert-allowed-cn:
         - TiDB
  tidb:
    baseImage: pingcap/tidb
    replicas: 1
    service:
      type: ClusterIP
    tlsClient:
      enabled: true
    config: 
      security:
       cert-allowed-cn:
         - TiDB
EOF
```

### 部署新集群加入初始集群

等待集群 1 完成部署，完成部署后，创建集群 2，相关命令如下。在实际使用中，集群 1 并不一定是初始集群，可以指定多集群内的任一集群加入。

根据实际情况设置以下环境变量：

{{< copyable "shell-regular" >}}

```bash
cluster1_name="cluster1"
cluster1_cluster_domain="cluster1.com"
cluster1_namespace="pingcap"

cluster2_name="cluster2"
cluster2_cluster_domain="cluster2.com"
cluster2_namespace="pingcap"
```

执行以下指令：

{{< copyable "shell-regular" >}}

```bash
cat << EOF | kubectl apply -n ${cluster2_namespace} -f -
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: "${cluster2_name}"
spec:
  version: v5.2.0
  timezone: UTC
  tlsCluster:
   enabled: true
  pvReclaimPolicy: Delete
  enableDynamicConfiguration: true
  configUpdateStrategy: RollingUpdate
  clusterDomain: "${cluster2_cluster_domain}"
  cluster:
    name: "${cluster1_name}"
    namespace: "${cluster1_namespace}"
    clusterDomain: "${cluster1_clusterdomain}"
  discovery: {}
  pd:
    baseImage: pingcap/pd
    replicas: 1
    requests:
      storage: "10Gi"
    config: 
      security:
        cert-allowed-cn:
          - TiDB
  tikv:
    baseImage: pingcap/tikv
    replicas: 1
    requests:
      storage: "10Gi"
    config: 
      security:
       cert-allowed-cn:
         - TiDB
  tidb:
    baseImage: pingcap/tidb
    replicas: 1
    service:
      type: ClusterIP
    tlsClient:
      enabled: true
    config: 
      security:
       cert-allowed-cn:
         - TiDB
EOF
```

## 退出和回收已加入集群

当你需要让一个集群从所加入的跨 Kubernetes 部署的 TiDB 集群退出并回收资源时，可以通过缩容流程来实现上述需求。在此场景下，需要满足缩容的一些限制，限制如下：

- 缩容后，集群中 TiKV 副本数应大于 PD 中设置的 `max-replicas` 数量，默认情况下 TiKV 副本数量需要大于 3。

以上面文档创建的集群 2 为例，先将 PD、TiKV、TiDB 的副本数设置为 0，如果开启了 TiFlash、TiCDC、Pump 等其他组件，也请一并将其副本数设为 0：

{{< copyable "shell-regular" >}}

```bash
kubectl patch tc cluster2 --type merge -p '{"spec":{"pd":{"replicas":0},"tikv":{"replicas":0},"tidb":{"replicas":0}}}'
```

等待集群 2 状态变为 `Ready`，相关组件此时应被缩容到 0 副本：

{{< copyable "shell-regular" >}}

```bash
kubectl get pods -l app.kubernetes.io/instance=cluster2 -n pingcap
```

Pod 列表显示为 `No resources found.`，此时 Pod 已经被全部缩容，集群 2 已经退出集群，查看集群 2 的集群状态：

{{< copyable "shell-regular" >}}

```bash
kubectl get tc cluster2
```

结果显示集群 2 为 `Ready` 状态，此时可以删除该对象，对相关资源进行回收。

{{< copyable "shell-regular" >}}

```bash
kubectl delete tc cluster2
```

通过上述步骤完成已加入集群的退出和资源回收。

## 已有数据集群开启跨多个 Kubernetes 集群功能并作为 TiDB 集群的初始集群

> **警告：**
>
> 目前此场景属于实验性支持，可能会造成数据丢失，请谨慎使用。

1. 更新 `.spec.clusterDomain` 配置：

    根据你的 Kubernetes 集群信息中的 `clusterDomain` 配置下面的参数：

    > **警告：**
    > 
    > 目前需要你使用正确的信息配置 `clusterDomain`，配置修改后无法再次修改。

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl patch tidbcluster cluster1 --type merge -p '{"spec":{"clusterDomain":"cluster1.com"}}'
    ```

    修改完成后，TiDB 集群进入滚动更新状态。

2. 更新 PD 的 `PeerURL` 信息：

    滚动更新结束后，需要使用 `port-forward` 暴露 PD 的 API 接口，使用 PD 的 API 接口更新 PD 的 `PeerURL`。

    1. 使用 `port-forward` 暴露 PD 的 API 接口：

        {{< copyable "shell-regular" >}}

        ```bash
        kubectl port-forward pods/cluster1-pd-0 2380:2380 2379:2379 -n pingcap
        ```

    2. 访问 `PD API`，获取 `members` 信息，注意使用 `port-forward` 后，终端会被占用，需要在另一个终端执行下列操作：

        {{< copyable "shell-regular" >}}

        ```bash
        curl http://127.0.0.1:2379/v2/members
        ```

        > **注意：**
        >
        > 如果集群开启了 TLS，使用 curl 命令时需要配置证书。例如：
        > 
        > `curl --cacert /var/lib/pd-tls/ca.crt --cert /var/lib/pd-tls/tls.crt --key /var/lib/pd-tls/tls.key https://127.0.0.1:2379/v2/members`

        执行后输出如下结果：

        ```output
        {"members":[{"id":"6ed0312dc663b885","name":"cluster1-pd-0.cluster1-pd-peer.pingcap.svc.cluster1.com","peerURLs":["http://cluster1-pd-0.cluster1-pd-peer.pingcap.svc:2380"],"clientURLs":["http://cluster1-pd-0.cluster1-pd-peer.pingcap.svc.cluster1.com:2379"]},{"id":"bd9acd3d57e24a32","name":"cluster1-pd-1.cluster1-pd-peer.pingcap.svc.cluster1.com","peerURLs":["http://cluster1-pd-1.cluster1-pd-peer.pingcap.svc:2380"],"clientURLs":["http://cluster1-pd-1.cluster1-pd-peer.pingcap.svc.cluster1.com:2379"]},{"id":"e04e42cccef60246","name":"cluster1-pd-2.cluster1-pd-peer.pingcap.svc.cluster1.com","peerURLs":["http://cluster1-pd-2.cluster1-pd-peer.pingcap.svc:2380"],"clientURLs":["http://cluster1-pd-2.cluster1-pd-peer.pingcap.svc.cluster1.com:2379"]}]}
        ```

    3. 记录各个 PD 实例的 `id`，使用 `id` 依次更新每个成员的 `peerURL`：

        {{< copyable "shell-regular" >}}

        ```bash
        member_ID="6ed0312dc663b885"
        member_peer_url="http://cluster1-pd-0.cluster1-pd-peer.pingcap.svc.cluster1.com:2380"

        curl http://127.0.0.1:2379/v2/members/${member_ID} -XPUT \
        -H "Content-Type: application/json" -d '{"peerURLs":["${member_peer_url}"]}'
        ```

更多示例信息以及开发信息，请参阅 [`multi-cluster`](https://github.com/pingcap/tidb-operator/tree/master/examples/multi-cluster)。
