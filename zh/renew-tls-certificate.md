---
title: 更新和替换 TLS 证书
summary: 介绍如何更新和替换 TiDB 组件间的 TLS 证书。
---

# 更新和替换 TLS 证书

本文以更新和替换 TiDB 集群中 PD、TiKV、TiDB 组件间的 TLS 证书为例，介绍在证书过期之前，如何更新和替换相应组件的证书。

如需要更新和替换集群中其他组件间的证书、TiDB Server 端证书或 MySQL Client 端证书，可使用类似的步骤进行操作。

本文的更新和替换操作假定原证书尚未过期。若原证书已经过期或失效，可参考[为 TiDB 组件间开启 TLS](enable-tls-between-components.md) 或[为 MySQL 客户端开启 TLS](enable-tls-for-mysql-client.md) 生成新的证书并重启集群。

## 更新和替换 `cfssl` 系统颁发的证书

如原 TLS 证书是[使用 `cfssl` 系统颁发的证书](enable-tls-between-components.md#使用-cfssl-系统颁发证书)，且原证书尚未过期，可按如下步骤更新和替换 PD、TiKV、TiDB 组件间的证书。

### 更新和替换 CA 证书

> **注意：**
>
> 若无需更新 CA 证书，可跳过本节中的操作，直接按[更新和替换组件间证书](#更新和替换组件间证书)进行操作。

1. 备份原 CA 证书与密钥。

    {{< copyable "shell-regular" >}}

    ```bash
    mv ca.pem ca.old.pem && \
    mv ca-key.pem ca-key.old.pem
    ```

2. 基于原 CA 证书的配置与证书签名请求 (CSR)，生成新的 CA 证书和密钥。

    {{< copyable "shell-regular" >}}

    ```bash
    cfssl gencert -initca ca-csr.json | cfssljson -bare ca -
    ```

    > **注意：**
    >
    > 配置文件与 CSR 中的 `expiry` 如有需要可进行更新。

3. 备份新 CA 证书与密钥，并基于原有 CA 证书及新的 CA 证书生成组合 CA 证书。

    {{< copyable "shell-regular" >}}

    ```bash
    mv ca.pem ca.new.pem && \
    mv ca-key.pem ca-key.new.pem && \
    cat ca.new.pem ca.old.pem > ca.pem
    ```

4. 基于组合 CA 证书更新各相应的 Kubernetes Secret 对象。

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl create secret generic ${cluster_name}-pd-cluster-secret --namespace=${namespace} --from-file=tls.crt=pd-server.pem --from-file=tls.key=pd-server-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-tikv-cluster-secret --namespace=${namespace} --from-file=tls.crt=tikv-server.pem --from-file=tls.key=tikv-server-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-tidb-cluster-secret --namespace=${namespace} --from-file=tls.crt=tidb-server.pem --from-file=tls.key=tidb-server-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-cluster-client-secret --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    ```

    其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 TiDB 集群部署的命名空间。

    > **注意：**
    >
    > 上述示例命令中仅更新了 PD、TiKV、TiDB 的组件间 Server 端 CA 证书与 Client 端 CA 证书，如需更新其他如 TiCDC、TiFlash 等的 Server 端 CA 证书，可使用类似命令进行更新。

5. 参考[滚动重启 TiDB 集群](restart-a-tidb-cluster.md)对需要加载组合 CA 证书的组件进行滚动重启。

    滚动重启完成后，基于组合 CA 证书，各组件将能同时接受由原 CA 证书与新 CA 证书签发的证书。

### 更新和替换组件间证书

> **注意：**
>
> 在更新和替换组件间证书前，请确保更新前后的组件间证书均能被 CA 证书验证为有效。如已[更新和替换 CA 证书](#更新和替换-ca-证书)，请确保 TiDB 集群已基于新的 CA 证书完成重启。

1. 基于各组件原配置信息，生成新的 Server 端与 Client 端证书。

    {{< copyable "shell-regular" >}}

    ```bash
    cfssl gencert -ca=ca.new.pem -ca-key=ca-key.new.pem -config=ca-config.json -profile=internal pd-server.json | cfssljson -bare pd-server
    cfssl gencert -ca=ca.new.pem -ca-key=ca-key.new.pem -config=ca-config.json -profile=internal tikv-server.json | cfssljson -bare tikv-server
    cfssl gencert -ca=ca.new.pem -ca-key=ca-key.new.pem -config=ca-config.json -profile=internal tidb-server.json | cfssljson -bare tidb-server
    cfssl gencert -ca=ca.new.pem -ca-key=ca-key.new.pem -config=ca-config.json -profile=client client.json | cfssljson -bare client
    ```

    > **注意：**
    >
    > - 上述示例命令中假定已参考[更新和替换 CA 证书](#更新和替换-ca-证书)中的步骤备份了新的 CA 证书与密钥为 `ca.new.pem` 与 `ca-key.new.pem`。如未更新 CA 证书与密钥，请修改示例命令中的对应参数为 `ca.pem` 与 `ca-key.pem`。
    > - 上述示例命令中仅生成了 PD、TiKV、TiDB 的组件间 Server 端证书与 Client 端证书，如需生成其他如 TiCDC、TiFlash 等的 Server 端证书，可使用类似命令进行生成。

2. 基于新生成的 Server 端与 Client 端证书，更新相应的 Kubernetes Secret 对象。

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl create secret generic ${cluster_name}-pd-cluster-secret --namespace=${namespace} --from-file=tls.crt=pd-server.pem --from-file=tls.key=pd-server-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-tikv-cluster-secret --namespace=${namespace} --from-file=tls.crt=tikv-server.pem --from-file=tls.key=tikv-server-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-tidb-cluster-secret --namespace=${namespace} --from-file=tls.crt=tidb-server.pem --from-file=tls.key=tidb-server-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-cluster-client-secret --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    ```

    其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 TiDB 集群部署的命名空间。

    > **注意：**
    >
    > 上述示例命令中仅更新了 PD、TiKV、TiDB 的组件间 Server 端证书与 Client 端证书，如需更新其他如 TiCDC、TiFlash 等的 Server 端证书，可使用类似命令进行更新。

3. 参考[滚动重启 TiDB 集群](restart-a-tidb-cluster.md)对需要加载新证书的组件进行滚动重启。

    滚动重启完成后，各组件将使用新的证书进行 TLS 通信。如有参考[更新和替换 CA 证书](#更新和替换-ca-证书)并使各组件加载了组合 CA 证书，则其仍能接受由原 CA 证书签发的证书。

### 可选：移除组合 CA 证书中的原 CA 证书

若同时参考[更新和替换 CA 证书](#更新和替换-ca-证书)与[更新和替换组件间证书](#更新和替换组件间证书)更新与替换了组合 CA 证书与 Server 端、Client 端组件证书，且计划移除原 CA 证书（如原 CA 证书已过期或原 CA 证书的密钥被盗），则可按如下步骤移除原 CA 证书。

1. 基于新 CA 证书更新 Kubernetes Secret 对象。

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl create secret generic ${cluster_name}-pd-cluster-secret --namespace=${namespace} --from-file=tls.crt=pd-server.pem --from-file=tls.key=pd-server-key.pem --from-file=ca.crt=ca.new.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-tikv-cluster-secret --namespace=${namespace} --from-file=tls.crt=tikv-server.pem --from-file=tls.key=tikv-server-key.pem --from-file=ca.crt=ca.new.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-tidb-cluster-secret --namespace=${namespace} --from-file=tls.crt=tidb-server.pem --from-file=tls.key=tidb-server-key.pem --from-file=ca.crt=ca.new.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-cluster-client-secret --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.new.pem --dry-run=client -o yaml | kubectl apply -f -
    ```

    其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 TiDB 集群部署的命名空间。

    > **注意：**
    >
    > 上述示例命令中假定已参考[更新和替换 CA 证书](#更新和替换-ca-证书)中的步骤备份了新的 CA 证书为 `ca.new.pem`

2. 参考[滚动重启 TiDB 集群](restart-a-tidb-cluster.md)对需要加载新证书的组件进行滚动重启。

    滚动重启完成后，各组件将仅能接受由新 CA 证书签发的证书。

## 更新和替换 `cert-manager` 颁发的证书

如原 TLS 证书是[使用 `cert-manager` 系统颁发的证书](enable-tls-between-components.md#使用-cert-manager-系统颁发证书)，且原证书尚未过期，根据是否需要更新 CA 证书需要分别处理。

### 更新和替换 CA 证书及组件间证书

使用 cert-manager 颁发证书时，通过指定 `Certificate` 资源的 `spec.renewBefore` 可由 cert-manager 在证书过期之前自动进行更新。

但 cert-manager 虽然能自动更新 CA 证书及对应的 Kubernetes Secret 对象，但目前并不支持将新旧 CA 证书合并为组合 CA 证书以同时接受新旧 CA 证书签发的证书。因此，在更新和替换 CA 证书的过程中，会出现集群组件间 TLS 无法互相认证的问题。

> **警告：**
>
> 由于组件间无法同时接受新旧 CA 签发的证书，因此在更新和替换证书的过程中需要重建部分组件的 Pod，这可能会引起部分访问 TiDB 集群的请求失败。

相应的更新和替换 PD、TiKV、TiDB 的 CA 证书及组件间证书的步骤如下。

1. 由 cert-manager 在证书过期之前自动更新 CA 证书及 Kubernetes Secret 对象 `${cluster_name}-ca-secret`。

    其中 `${cluster_name}` 为集群的名字。

    若要手动更新 CA 证书，可直接删除相应的 Kubernetes Secret 对象后触发 cert-manager 重新生成 CA 证书。

2. 删除各组件对应证书的 Kubernetes Secret 对象。

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl delete secret ${cluster_name}-pd-cluster-secret --namespace=${namespace}
    kubectl delete secret ${cluster_name}-tikv-cluster-secret --namespace=${namespace}
    kubectl delete secret ${cluster_name}-tidb-cluster-secret --namespace=${namespace}
    kubectl delete secret ${cluster_name}-cluster-client-secret --namespace=${namespace}
    ```

    其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 TiDB 集群部署的命名空间。

3. 等待 cert-manager 基于新的 CA 证书为各组件颁发新的证书。

    观察 `kubectl get secret --namespace=${namespace}` 的输出，直到所有组件对应的 Kubernetes Secret 对象都被创建。

4. 依次强制重建 PD、TiKV 与 TiDB 组件 Pod。

    由于 cert-manager 不支持组合 CA 证书，若尝试滚动升级各组件，则使用新旧不同 CA 签发证书的 Pod 间将无法基于 TLS 正常通信。因此需要强制删除 Pod 并通过基于新 CA 签发的证书重建 Pod。

    {{< copyable "shell-regular" >}}

    ```
    kubectl delete -n ${namespace} pod ${pod_name}
    ```

    其中 `${namespace}` 为 TiDB 集群部署的命名空间，`${pod_name}` 为 PD、TiKV 与 TiDB 各 replica 的 Pod 名称。

### 仅更新和替换组件间证书

1. 由 cert-manager 在证书过期之前自动更新各组件的证书及 Kubernetes Secret 对象。

    对于 PD、TiKV 及 TiDB 组件，在 TiDB 集群部署的命名空间下包含以下 Kubernetes Secret 对象：

    {{< copyable "shell-regular" >}}

    ```
    ${cluster_name}-pd-cluster-secret
    ${cluster_name}-tikv-cluster-secret
    ${cluster_name}-tidb-cluster-secret
    ${cluster_name}-cluster-client-secret
    ```

    其中 `${cluster_name}` 为集群的名字。

    若要手动更新组件间证书，可直接删除相应的 Kubernetes Secret 对象后触发 cert-manager 重新生成组件间证书。

2. 对于各组件间的证书，各组件会在之后新建连接时自动重新加载新的证书，无需手动操作。

    > **注意：**
    >
    > - 各组件目前[暂不支持 CA 证书的自动重新加载](https://docs.pingcap.com/zh/tidb/stable/enable-tls-between-components#证书重加载)，需要参考[更新和替换 CA 证书及组件间证书](#更新和替换-ca-证书及组件间证书)进行处理。
    > - 对于 TiDB Server 端证书，可参考以下任意方式进行手动重加载：
    >     - 参考[重加载证书、密钥和 CA](https://docs.pingcap.com/zh/tidb/stable/enable-tls-between-clients-and-servers#重加载证书密钥和-ca)。
    >     - 参考[滚动重启 TiDB 集群](restart-a-tidb-cluster.md)对 TiDB Server 进行滚动重启。
