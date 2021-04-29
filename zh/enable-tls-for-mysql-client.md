---
title: 为 MySQL 客户端开启 TLS
summary: 在 Kubernetes 上如何为 TiDB 集群的 MySQL 客户端开启 TLS。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/enable-tls-for-mysql-client/']
---

# 为 MySQL 客户端开启 TLS

本文主要描述了在 Kubernetes 上如何为 TiDB 集群的 MySQL 客户端开启 TLS。TiDB Operator 从 v1.1 开始已经支持为 Kubernetes 上 TiDB 集群开启 MySQL 客户端 TLS。开启步骤为：

1. 为 TiDB Server 颁发一套 Server 端证书，为 MySQL Client 颁发一套 Client 端证书。并创建两个 Secret 对象，Secret 名字分别为：`${cluster_name}-tidb-server-secret` 和  `${cluster_name}-tidb-client-secret`，分别包含前面创建的两套证书；
2. 部署集群，设置 `.spec.tidb.tlsClient.enabled` 属性为 `true`；
3. 配置 MySQL 客户端使用加密连接。

其中，颁发证书的方式有多种，本文档提供两种方式，用户也可以根据需要为 TiDB 集群颁发证书，这两种方式分别为：

- 使用 `cfssl` 系统颁发证书；
- 使用 `cert-manager` 系统颁发证书；

当需要更新已有 TLS 证书时，可参考[更新和替换 TLS 证书](renew-tls-certificate.md)。

## 第一步：为 TiDB 集群颁发两套证书

### 使用 `cfssl` 系统颁发证书

1. 首先下载 `cfssl` 软件并初始化证书颁发机构：

    {{< copyable "shell-regular" >}}

    ``` shell
    mkdir -p ~/bin
    curl -s -L -o ~/bin/cfssl https://pkg.cfssl.org/R1.2/cfssl_linux-amd64
    curl -s -L -o ~/bin/cfssljson https://pkg.cfssl.org/R1.2/cfssljson_linux-amd64
    chmod +x ~/bin/{cfssl,cfssljson}
    export PATH=$PATH:~/bin

    mkdir -p cfssl
    cd cfssl
    cfssl print-defaults config > ca-config.json
    cfssl print-defaults csr > ca-csr.json
    ```

2. 在 `ca-config.json` 配置文件中配置 CA 选项：

    ``` json
    {
        "signing": {
            "default": {
                "expiry": "8760h"
            },
            "profiles": {
                "server": {
                    "expiry": "8760h",
                    "usages": [
                        "signing",
                        "key encipherment",
                        "server auth"
                    ]
                },
                "client": {
                    "expiry": "8760h",
                    "usages": [
                        "signing",
                        "key encipherment",
                        "client auth"
                    ]
                }
            }
        }
    }
    ```

3. 您还可以修改 `ca-csr.json` 证书签名请求 (CSR)：

    ``` json
    {
        "CN": "TiDB Server",
        "CA": {
            "expiry": "87600h"
        },
        "key": {
            "algo": "rsa",
            "size": 2048
        },
        "names": [
            {
                "C": "US",
                "L": "CA",
                "O": "PingCAP",
                "ST": "Beijing",
                "OU": "TiDB"
            }
        ]
    }
    ```

4. 使用定义的选项生成 CA：

    {{< copyable "shell-regular" >}}

    ``` shell
    cfssl gencert -initca ca-csr.json | cfssljson -bare ca -
    ```

5. 生成 Server 端证书。

    首先生成默认的 `server.json` 文件：

    {{< copyable "shell-regular" >}}

    ``` shell
    cfssl print-defaults csr > server.json
    ```

    然后编辑这个文件，修改 `CN`，`hosts` 属性：

    ``` json
    ...
        "CN": "TiDB Server",
        "hosts": [
          "127.0.0.1",
          "::1",
          "${cluster_name}-tidb",
          "${cluster_name}-tidb.${namespace}",
          "${cluster_name}-tidb.${namespace}.svc",
          "*.${cluster_name}-tidb",
          "*.${cluster_name}-tidb.${namespace}",
          "*.${cluster_name}-tidb.${namespace}.svc",
          "*.${cluster_name}-tidb-peer",
          "*.${cluster_name}-tidb-peer.${namespace}",
          "*.${cluster_name}-tidb-peer.${namespace}.svc"
        ],
    ...
    ```

    其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 TiDB 集群部署的命名空间，用户也可以添加自定义 `hosts`。

    最后生成 Server 端证书：

    {{< copyable "shell-regular" >}}

    ``` shell
    cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=server server.json | cfssljson -bare server
    ```

6. 生成 Client 端证书。

    首先生成默认的 `client.json` 文件：

    {{< copyable "shell-regular" >}}

    ``` shell
    cfssl print-defaults csr > client.json
    ```

    然后编辑这个文件，修改 `CN`，`hosts` 属性，`hosts` 可以留空：

    ``` json
    ...
        "CN": "TiDB Client",
        "hosts": [],
    ...
    ```

    最后生成 Client 端证书：

    {{< copyable "shell-regular" >}}

    ``` shell
    cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=client client.json | cfssljson -bare client
    ```

7. 创建 Kubernetes Secret 对象。

    到这里假设你已经按照上述文档把两套证书都创建好了。通过下面的命令为 TiDB 集群创建 Secret 对象：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create secret generic ${cluster_name}-tidb-server-secret --namespace=${namespace} --from-file=tls.crt=server.pem --from-file=tls.key=server-key.pem --from-file=ca.crt=ca.pem
    kubectl create secret generic ${cluster_name}-tidb-client-secret --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
    ```

    这样就给 Server/Client 端证书分别创建了：

    - 一个 Secret 供 TiDB Server 启动时加载使用;
    - 另一个 Secret 供 MySQL 客户端连接 TiDB 集群时候使用。

用户可以生成多套 Client 端证书，并且至少要生成一套 Client 证书供 TiDB Operator 内部组件访问 TiDB Server（目前有 TidbInitializer 会访问 TiDB Server 来设置密码或者一些初始化操作）。

### 使用 cert-manager 颁发证书

1. 安装 cert-manager。

    请参考官网安装：[cert-manager installation in Kubernetes](https://docs.cert-manager.io/en/release-0.11/getting-started/install/kubernetes.html)。

2. 创建一个 Issuer 用于给 TiDB 集群颁发证书。

    为了配置 `cert-manager` 颁发证书，必须先创建 Issuer 资源。

    首先创建一个目录保存 `cert-manager` 创建证书所需文件：

    {{< copyable "shell-regular" >}}

    ``` shell
    mkdir -p cert-manager
    cd cert-manager
    ```

    然后创建一个 `tidb-server-issuer.yaml` 文件，输入以下内容：

    ``` yaml
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
      commonName: "TiDB CA"
      isCA: true
      duration: 87600h # 10yrs
      renewBefore: 720h # 30d
      issuerRef:
        name: ${cluster_name}-selfsigned-ca-issuer
        kind: Issuer
    ---
    apiVersion: cert-manager.io/v1alpha2
    kind: Issuer
    metadata:
      name: ${cluster_name}-tidb-issuer
      namespace: ${namespace}
    spec:
      ca:
        secretName: ${cluster_name}-ca-secret
    ```

    上面的文件创建三个对象：

    - 一个 SelfSigned 类型的 Isser 对象（用于生成 CA 类型 Issuer 所需要的 CA 证书）;
    - 一个 Certificate 对象，`isCa` 属性设置为 `true`；
    - 一个可以用于颁发 TiDB Server TLS 证书的 Issuer。

    最后执行下面的命令进行创建：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f tidb-server-issuer.yaml
    ```

3. 创建 Server 端证书。

    在 `cert-manager` 中，Certificate 资源表示证书接口，该证书将由上面创建的 Issuer 颁发并保持更新。

    首先来创建 Server 端证书，创建一个 `tidb-server-cert.yaml` 文件，并输入以下内容：

    ``` yaml
    apiVersion: cert-manager.io/v1alpha2
    kind: Certificate
    metadata:
      name: ${cluster_name}-tidb-server-secret
      namespace: ${namespace}
    spec:
      secretName: ${cluster_name}-tidb-server-secret
      duration: 8760h # 365d
      renewBefore: 360h # 15d
      organization:
        - PingCAP
      commonName: "TiDB Server"
      usages:
        - server auth
      dnsNames:
        - "${cluster_name}-tidb"
        - "${cluster_name}-tidb.${namespace}"
        - "${cluster_name}-tidb.${namespace}.svc"
        - "*.${cluster_name}-tidb"
        - "*.${cluster_name}-tidb.${namespace}"
        - "*.${cluster_name}-tidb.${namespace}.svc"
        - "*.${cluster_name}-tidb-peer"
        - "*.${cluster_name}-tidb-peer.${namespace}"
        - "*.${cluster_name}-tidb-peer.${namespace}.svc"
      ipAddresses:
        - 127.0.0.1
        - ::1
      issuerRef:
        name: ${cluster_name}-tidb-issuer
        kind: Issuer
        group: cert-manager.io
    ```

    其中 `${cluster_name}` 为集群的名字：

    - `spec.secretName` 请设置为 `${cluster_name}-tidb-server-secret`；
    - `usages` 请添加上 `server auth`；
    - `dnsNames` 需要填写这 6 个 DNS，根据需要可以填写其他 DNS：
      - `${cluster_name}-tidb`
      - `${cluster_name}-tidb.${namespace}`
      - `${cluster_name}-tidb.${namespace}.svc`
      - `*.${cluster_name}-tidb`
      - `*.${cluster_name}-tidb.${namespace}`
      - `*.${cluster_name}-tidb.${namespace}.svc`
      - `*.${cluster_name}-tidb-peer`
      - `*.${cluster_name}-tidb-peer.${namespace}`
      - `*.${cluster_name}-tidb-peer.${namespace}.svc`
    - `ipAddresses` 需要填写这两个 IP ，根据需要可以填写其他 IP：
      - `127.0.0.1`
      - `::1`
    - `issuerRef` 请填写上面创建的 Issuer；
    - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

    通过执行下面的命令来创建证书：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f tidb-server-cert.yaml
    ```

    创建这个对象以后，cert-manager 会生成一个名字为 `${cluster_name}-tidb-server-secret` 的 Secret 对象供 TiDB Server 使用。

4. 创建 Client 端证书。

    创建一个 `tidb-client-cert.yaml` 文件，并输入以下内容：

    ``` yaml
    apiVersion: cert-manager.io/v1alpha2
    kind: Certificate
    metadata:
      name: ${cluster_name}-tidb-client-secret
      namespace: ${namespace}
    spec:
      secretName: ${cluster_name}-tidb-client-secret
      duration: 8760h # 365d
      renewBefore: 360h # 15d
      organization:
        - PingCAP
      commonName: "TiDB Client"
      usages:
        - client auth
      issuerRef:
        name: ${cluster_name}-tidb-issuer
        kind: Issuer
        group: cert-manager.io
    ```

    其中 `${cluster_name}` 为集群的名字：

    - `spec.secretName` 请设置为 `${cluster_name}-tidb-client-secret`；
    - `usages` 请添加上 `client auth`；
    - `dnsNames` 和 `ipAddresses` 不需要填写；
    - `issuerRef` 请填写上面创建的 Issuer；
    - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

    通过执行下面的命令来创建证书：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f tidb-client-cert.yaml
    ```

    创建这个对象以后，cert-manager 会生成一个名字为 `${cluster_name}-tidb-client-secret` 的 Secret 对象供 TiDB Client 使用。

5. 创建多套 Client 端证书（可选）。

    TiDB Operator 集群内部有 4 个组件需要请求 TiDB Server，当开启 TLS 验证后，这些组件可以使用证书来请求 TiDB Server，每个组件都可以使用单独的证书。这些组件有：

    - TidbInitializer
    - PD Dashboard
    - Backup
    - Restore
      
    如需要[使用 TiDB Lightning 恢复 Kubernetes 上的集群数据](restore-data-using-tidb-lightning.md)，则也可以为其中的 TiDB Lightning 组件生成 Client 端证书。

    下面就来生成这些组件的 Client 证书。

    1. 创建一个 `tidb-components-client-cert.yaml` 文件，并输入以下内容：

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-tidb-initializer-client-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-tidb-initializer-client-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
            - PingCAP
          commonName: "TiDB Initializer client"
          usages:
            - client auth
          issuerRef:
            name: ${cluster_name}-tidb-issuer
            kind: Issuer
            group: cert-manager.io
        ---
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-pd-dashboard-client-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-pd-dashboard-client-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
            - PingCAP
          commonName: "PD Dashboard client"
          usages:
            - client auth
          issuerRef:
            name: ${cluster_name}-tidb-issuer
            kind: Issuer
            group: cert-manager.io
        ---
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-backup-client-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-backup-client-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
            - PingCAP
          commonName: "Backup client"
          usages:
            - client auth
          issuerRef:
            name: ${cluster_name}-tidb-issuer
            kind: Issuer
            group: cert-manager.io
        ---
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-restore-client-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-restore-client-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
            - PingCAP
          commonName: "Restore client"
          usages:
            - client auth
          issuerRef:
            name: ${cluster_name}-tidb-issuer
            kind: Issuer
            group: cert-manager.io
        ```

        其中 `${cluster_name}` 为集群的名字：

        - `spec.secretName` 请设置为 `${cluster_name}-${component}-client-secret`；
        - `usages` 请添加上 `client auth`；
        - `dnsNames` 和 `ipAddresses` 不需要填写；
        - `issuerRef` 请填写上面创建的 Issuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        如需要为 TiDB Lignting 组件生成 Client 端证书，则可以使用以下内容并通过在 TiDB Lightning 的 Helm Chart `values.yaml` 中设置 `tlsCluster.tlsClientSecretName` 为 `${cluster_name}-lightning-client-secret`：
        
        ```yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-lightning-client-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-lightning-client-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
            - PingCAP
          commonName: "Lightning client"
          usages:
            - client auth
          issuerRef:
            name: ${cluster_name}-tidb-issuer
            kind: Issuer
            group: cert-manager.io
        ```

    2. 通过执行下面的命令来创建证书：

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl apply -f tidb-components-client-cert.yaml
        ```

    3. 创建这些对象以后，cert-manager 会生成 4 个 Secret 对象供上面四个组件使用。

    > **注意：**
    >
    > TiDB Server 的 TLS 兼容 MySQL 协议。当证书内容发生改变后，需要管理员手动执行 SQL 语句 `alter instance reload tls` 进行刷新。

## 第二步：部署 TiDB 集群

接下来将会创建一个 TiDB 集群，并且执行以下步骤：

- 开启 MySQL 客户端 TLS；
- 对集群进行初始化（这里创建了一个数据库 `app`）;
- 创建一个 Backup 对象对集群进行备份；
- 创建一个 Restore 对象对进群进行恢复；
- TidbInitializer，PD Dashboard，Backup 以及 Restore 分别使用单独的 Client 证书（用 `tlsClientSecretName` 指定）。

1. 创建三个 `.yaml` 文件：

    - `tidb-cluster.yaml`:

        ``` yaml
        apiVersion: pingcap.com/v1alpha1
        kind: TidbCluster
        metadata:
         name: ${cluster_name}
         namespace: ${namespace}
        spec:
         version: v5.0.1
         timezone: UTC
         pvReclaimPolicy: Retain
         pd:
           baseImage: pingcap/pd
           replicas: 1
           requests:
             storage: "1Gi"
           config: {}
           tlsClientSecretName: ${cluster_name}-pd-dashboard-client-secret
         tikv:
           baseImage: pingcap/tikv
           replicas: 1
           requests:
             storage: "1Gi"
           config: {}
         tidb:
           baseImage: pingcap/tidb
           replicas: 1
           service:
             type: ClusterIP
           config: {}
           tlsClient:
             enabled: true
        ---
        apiVersion: pingcap.com/v1alpha1
        kind: TidbInitializer
        metadata:
         name: ${cluster_name}-init
         namespace: ${namespace}
        spec:
         image: tnir/mysqlclient
         cluster:
           namespace: ${namespace}
           name: ${cluster_name}
         initSql: |-
           create database app;
         tlsClientSecretName: ${cluster_name}-tidb-initializer-client-secret
        ```

    - `backup.yaml`:

        ```
        apiVersion: pingcap.com/v1alpha1
        kind: Backup
        metadata:
          name: ${cluster_name}-backup
          namespace: ${namespace}
        spec:
          backupType: full
          br:
            cluster: ${cluster_name}
            clusterNamespace: ${namespace}
            sendCredToTikv: true
          from:
            host: ${host}
            secretName: ${tidb_secret}
            port: 4000
            user: root
            tlsClientSecretName: ${cluster_name}-backup-client-secret
          s3:
            provider: aws
            region: ${my_region}
            secretName: ${s3_secret}
            bucket: ${my_bucket}
            prefix: ${my_folder}
        ```

    - `restore.yaml`:

        ```
        apiVersion: pingcap.com/v1alpha1
        kind: Restore
        metadata:
          name: ${cluster_name}-restore
          namespace: ${namespace}
        spec:
          backupType: full
          br:
            cluster: ${cluster_name}
            clusterNamespace: ${namespace}
            sendCredToTikv: true
          to:
            host: ${host}
            secretName: ${tidb_secret}
            port: 4000
            user: root
            tlsClientSecretName: ${cluster_name}-restore-client-secret
          s3:
            provider: aws
            region: ${my_region}
            secretName: ${s3_secret}
            bucket: ${my_bucket}
            prefix: ${my_folder}
        ```

    其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 TiDB 集群部署的命名空间。通过设置 `spec.tidb.tlsClient.enabled` 属性为 `true` 来开启 MySQL 客户端 TLS。

2. 部署 TiDB 集群：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f tidb-cluster.yaml
    ```

3. 集群备份：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f backup.yaml
    ```

4. 集群恢复：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f restore.yaml
    ```

## 第三步：配置 MySQL 客户端使用加密连接

可以根据[官网文档](https://pingcap.com/docs-cn/stable/how-to/secure/enable-tls-clients/#配置-mysql-客户端使用加密连接)提示，使用上面创建的 Client 证书，通过下面的方法连接 TiDB 集群：

获取 Client 证书的方式并连接 TiDB Server 的方法是：

{{< copyable "shell-regular" >}}

``` shell
kubectl get secret -n ${namespace} ${cluster_name}-tidb-client-secret  -ojsonpath='{.data.tls\.crt}' | base64 --decode > client-tls.crt
kubectl get secret -n ${namespace} ${cluster_name}-tidb-client-secret  -ojsonpath='{.data.tls\.key}' | base64 --decode > client-tls.key
kubectl get secret -n ${namespace} ${cluster_name}-tidb-client-secret  -ojsonpath='{.data.ca\.crt}'  | base64 --decode > client-ca.crt
```

{{< copyable "shell-regular" >}}

``` shell
mysql -uroot -p -P 4000 -h ${tidb_host} --ssl-cert=client-tls.crt --ssl-key=client-tls.key --ssl-ca=client-ca.crt
```

> **注意：**
>
> [MySQL 8.0 默认认证插件](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_default_authentication_plugin)从 `mysql_native_password` 更新为 `caching_sha2_password`，因此如果使用 MySQL 8.0 客户端访问 TiDB 服务（TiDB 版本 < v4.0.7），并且用户账户有配置密码，需要显示指定 `--default-auth=mysql_native_password` 参数。

最后请参考 [官网文档](https://pingcap.com/docs-cn/stable/enable-tls-between-clients-and-servers/#检查当前连接是否是加密连接) 来验证是否正确开启了 TLS。
