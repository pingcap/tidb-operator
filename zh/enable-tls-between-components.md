---
title: 为 TiDB 组件间开启 TLS
summary: 在 Kubernetes 上如何为 TiDB 集群组件间开启 TLS。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/enable-tls-between-components/']
---

# 为 TiDB 组件间开启 TLS

本文主要描述了在 Kubernetes 上如何为 TiDB 集群组件间开启 TLS。TiDB Operator 从 v1.1 开始已经支持为 Kubernetes 上 TiDB 集群组件间开启 TLS。开启步骤为：

1. 为即将被创建的 TiDB 集群的每个组件生成证书：
    - 为 PD/TiKV/TiDB/Pump/Drainer/TiFlash 组件分别创建一套 Server 端证书，保存为 Kubernetes Secret 对象：`${cluster_name}-${component_name}-cluster-secret`
    - 为它们的各种客户端创建一套共用的 Client 端证书，保存为 Kubernetes Secret 对象：`${cluster_name}-cluster-client-secret`
2. 部署集群，设置 `.spec.tlsCluster.enabled` 属性为 `true`；
3. 配置 `pd-ctl`，`tikv-ctl` 连接集群。

> **注意：**
>
> * TiDB v4.0.5, TiDB Operator v1.1.4 及以上版本支持 TiFlash 开启 TLS。
> * TiDB v4.0.3, TiDB Operator v1.1.3 及以上版本支持 TiCDC 开启 TLS。

其中，颁发证书的方式有多种，本文档提供两种方式，用户也可以根据需要为 TiDB 集群颁发证书，这两种方式分别为：

- 使用 `cfssl` 系统颁发证书；
- 使用 `cert-manager` 系统颁发证书；

## 第一步：为 TiDB 集群各个组件生成证书

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
    ```

2. 生成 `ca-config.json` 配置文件：

    ```shell
    cat << EOF > ca-config.json
    {
        "signing": {
            "default": {
                "expiry": "8760h"
            },
            "profiles": {
                "internal": {
                    "expiry": "8760h",
                    "usages": [
                        "signing",
                        "key encipherment",
                        "server auth",
                        "client auth"
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
    EOF
    ```

3. 生成 `ca-csr.json` 配置文件：

    ```shell
    cat << EOF > ca-csr.json
    {
        "CN": "TiDB",
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
    EOF
    ```

4. 使用定义的选项生成 CA：

    {{< copyable "shell-regular" >}}

    ``` shell
    cfssl gencert -initca ca-csr.json | cfssljson -bare ca -
    ```

5. 生成 Server 端证书。

    这里需要为每个 TiDB 集群的组件生成一套 Server 端证书。

    - PD Server 端证书

        首先生成默认的 `pd-server.json` 文件：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl print-defaults csr > pd-server.json
        ```

        然后编辑这个文件，修改 `CN`，`hosts` 属性：

        ``` json
        ...
            "CN": "TiDB",
            "hosts": [
              "127.0.0.1",
              "::1",
              "${cluster_name}-pd",
              "${cluster_name}-pd.${namespace}",
              "${cluster_name}-pd.${namespace}.svc",
              "${cluster_name}-pd-peer",
              "${cluster_name}-pd-peer.${namespace}",
              "${cluster_name}-pd-peer.${namespace}.svc",
              "*.${cluster_name}-pd-peer",
              "*.${cluster_name}-pd-peer.${namespace}",
              "*.${cluster_name}-pd-peer.${namespace}.svc"
            ],
        ...
        ```

        其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 TiDB 集群部署的命名空间，用户也可以添加自定义 `hosts`。

        最后生成 PD Server 端证书：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal pd-server.json | cfssljson -bare pd-server
        ```

    - TiKV Server 端证书

        首先生成默认的 `tikv-server.json` 文件：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl print-defaults csr > tikv-server.json
        ```

        然后编辑这个文件，修改 `CN`，`hosts` 属性：

        ``` json
        ...
            "CN": "TiDB",
            "hosts": [
              "127.0.0.1",
              "::1",
              "${cluster_name}-tikv",
              "${cluster_name}-tikv.${namespace}",
              "${cluster_name}-tikv.${namespace}.svc",
              "${cluster_name}-tikv-peer",
              "${cluster_name}-tikv-peer.${namespace}",
              "${cluster_name}-tikv-peer.${namespace}.svc",
              "*.${cluster_name}-tikv-peer",
              "*.${cluster_name}-tikv-peer.${namespace}",
              "*.${cluster_name}-tikv-peer.${namespace}.svc"
            ],
        ...
        ```

        其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 TiDB 集群部署的命名空间，用户也可以添加自定义 `hosts`。

        最后生成 TiKV Server 端证书：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal tikv-server.json | cfssljson -bare tikv-server
        ```

    - TiDB Server 端证书

        首先生成默认的 `tidb-server.json` 文件：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl print-defaults csr > tidb-server.json
        ```

        然后编辑这个文件，修改 `CN`，`hosts` 属性：

        ``` json
        ...
            "CN": "TiDB",
            "hosts": [
              "127.0.0.1",
              "::1",
              "${cluster_name}-tidb",
              "${cluster_name}-tidb.${namespace}",
              "${cluster_name}-tidb.${namespace}.svc",
              "${cluster_name}-tidb-peer",
              "${cluster_name}-tidb-peer.${namespace}",
              "${cluster_name}-tidb-peer.${namespace}.svc",
              "*.${cluster_name}-tidb-peer",
              "*.${cluster_name}-tidb-peer.${namespace}",
              "*.${cluster_name}-tidb-peer.${namespace}.svc"
            ],
        ...
        ```

        其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 TiDB 集群部署的命名空间，用户也可以添加自定义 `hosts`。

        最后生成 TiDB Server 端证书：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal tidb-server.json | cfssljson -bare tidb-server
        ```

    - Pump Server 端证书

        首先生成默认的 `pump-server.json` 文件：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl print-defaults csr > pump-server.json
        ```

        然后编辑这个文件，修改 `CN`，`hosts` 属性：

        ``` json
        ...
            "CN": "TiDB",
            "hosts": [
              "127.0.0.1",
              "::1",
              "*.${cluster_name}-pump",
              "*.${cluster_name}-pump.${namespace}",
              "*.${cluster_name}-pump.${namespace}.svc"
            ],
        ...
        ```

        其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 TiDB 集群部署的命名空间，用户也可以添加自定义 `hosts`。

        最后生成 Pump Server 端证书：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal pump-server.json | cfssljson -bare pump-server
        ```

    - Drainer Server 端证书

        首先生成默认的 `drainer-server.json` 文件：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl print-defaults csr > drainer-server.json
        ```

        然后编辑这个文件，修改 `CN`，`hosts` 属性：

        ``` json
        ...
            "CN": "TiDB",
            "hosts": [
              "127.0.0.1",
              "::1",
              "<hosts 列表请参考下面描述>"
            ],
        ...
        ```

        现在 Drainer 组件是通过 Helm 来部署的，根据 `values.yaml` 文件配置方式不同，所需要填写的 `hosts` 字段也不相同。

        如果部署的时候设置 `drainerName` 属性，像下面这样：

        ``` yaml
        ...
        # Change the name of the statefulset and pod
        # The default is clusterName-ReleaseName-drainer
        # Do not change the name of an existing running drainer: this is unsupported.
        drainerName: my-drainer
        ...
        ```

        那么就这样配置 `hosts` 属性：

        ``` json
        ...
            "CN": "TiDB",
            "hosts": [
              "127.0.0.1",
              "::1",
              "*.${drainer_name}",
              "*.${drainer_name}.${namespace}",
              "*.${drainer_name}.${namespace}.svc"
            ],
        ...
        ```

        如果部署的时候没有设置 `drainerName` 属性，需要这样配置 `hosts` 属性：

        ``` json
        ...
            "CN": "TiDB",
            "hosts": [
              "127.0.0.1",
              "::1",
              "*.${cluster_name}-${release_name}-drainer",
              "*.${cluster_name}-${release_name}-drainer.${namespace}",
              "*.${cluster_name}-${release_name}-drainer.${namespace}.svc"
            ],
        ...
        ```

        其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 TiDB 集群部署的命名空间，`${release_name}` 是 `helm install` 时候填写的 `release name`，`${drainer_name}` 为 `values.yaml` 文件里的 `drainerName`，用户也可以添加自定义 `hosts`。

        最后生成 Drainer Server 端证书：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal drainer-server.json | cfssljson -bare drainer-server
        ```

    - TiCDC Server 端证书

        首先生成默认的 `ticdc-server.json` 文件：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl print-defaults csr > ticdc-server.json
        ```

        然后编辑这个文件，修改 `CN`，`hosts` 属性：

        ``` json
        ...
            "CN": "TiDB",
            "hosts": [
              "127.0.0.1",
              "::1",
              "${cluster_name}-ticdc",
              "${cluster_name}-ticdc.${namespace}",
              "${cluster_name}-ticdc.${namespace}.svc",
              "${cluster_name}-ticdc-peer",
              "${cluster_name}-ticdc-peer.${namespace}",
              "${cluster_name}-ticdc-peer.${namespace}.svc",
              "*.${cluster_name}-ticdc-peer",
              "*.${cluster_name}-ticdc-peer.${namespace}",
              "*.${cluster_name}-ticdc-peer.${namespace}.svc"
            ],
        ...
        ```

        其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 TiDB 集群部署的命名空间，用户也可以添加自定义 `hosts`。

        最后生成 TiCDC Server 端证书：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal ticdc-server.json | cfssljson -bare ticdc-server
        ```

    - TiFlash Server 端证书

        首先生成默认的 `tiflash-server.json` 文件：

        {{< copyable "shell-regular" >}}

        ```shell
        cfssl print-defaults csr > tiflash-server.json
        ```

        然后编辑这个文件，修改 `CN`、`hosts` 属性：

        ```json
        ...
            "CN": "TiDB",
            "hosts": [
              "127.0.0.1",
              "::1",
              "${cluster_name}-tiflash",
              "${cluster_name}-tiflash.${namespace}",
              "${cluster_name}-tiflash.${namespace}.svc",
              "${cluster_name}-tiflash-peer",
              "${cluster_name}-tiflash-peer.${namespace}",
              "${cluster_name}-tiflash-peer.${namespace}.svc",
              "*.${cluster_name}-tiflash-peer",
              "*.${cluster_name}-tiflash-peer.${namespace}",
              "*.${cluster_name}-tiflash-peer.${namespace}.svc"
            ],
        ...
        ```

        其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 TiDB 集群部署的命名空间，用户也可以添加自定义 `hosts`。

        最后生成 TiFlash Server 端证书：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal tiflash-server.json | cfssljson -bare tiflash-server
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
        "CN": "TiDB",
        "hosts": [],
    ...
    ```

    最后生成 Client 端证书：

    ``` shell
    cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=client client.json | cfssljson -bare client
    ```

7. 创建 Kubernetes Secret 对象。

    假设你已经按照上述文档为每个组件创建了一套 Server 端证书，并为各个客户端创建了一套 Client 端证书。通过下面的命令为 TiDB 集群创建这些 Secret 对象：

    PD 集群证书 Secret：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create secret generic ${cluster_name}-pd-cluster-secret --namespace=${namespace} --from-file=tls.crt=pd-server.pem --from-file=tls.key=pd-server-key.pem --from-file=ca.crt=ca.pem
    ```

    TiKV 集群证书 Secret：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create secret generic ${cluster_name}-tikv-cluster-secret --namespace=${namespace} --from-file=tls.crt=tikv-server.pem --from-file=tls.key=tikv-server-key.pem --from-file=ca.crt=ca.pem
    ```

    TiDB 集群证书 Secret：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create secret generic ${cluster_name}-tidb-cluster-secret --namespace=${namespace} --from-file=tls.crt=tidb-server.pem --from-file=tls.key=tidb-server-key.pem --from-file=ca.crt=ca.pem
    ```

    Pump 集群证书 Secret：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create secret generic ${cluster_name}-pump-cluster-secret --namespace=${namespace} --from-file=tls.crt=pump-server.pem --from-file=tls.key=pump-server-key.pem --from-file=ca.crt=ca.pem
    ```

    Drainer 集群证书 Secret：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create secret generic ${cluster_name}-drainer-cluster-secret --namespace=${namespace} --from-file=tls.crt=drainer-server.pem --from-file=tls.key=drainer-server-key.pem --from-file=ca.crt=ca.pem
    ```

    TiCDC 集群证书 Secret：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create secret generic ${cluster_name}-ticdc-cluster-secret --namespace=${namespace} --from-file=tls.crt=ticdc-server.pem --from-file=tls.key=ticdc-server-key.pem --from-file=ca.crt=ca.pem
    ```

    TiFlash 集群证书 Secret：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create secret generic ${cluster_name}-tiflash-cluster-secret --namespace=${namespace} --from-file=tls.crt=tiflash-server.pem --from-file=tls.key=tiflash-server-key.pem --from-file=ca.crt=ca.pem
    ```

    Client 证书 Secret：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create secret generic ${cluster_name}-cluster-client-secret --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
    ```

    这里给 PD/TiKV/TiDB/Pump/Drainer 的 Server 端证书分别创建了一个 Secret 供他们启动时加载使用，另外一套 Client 端证书供他们的客户端连接使用。

### 使用 `cert-manager` 系统颁发证书

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

    然后创建一个 `tidb-cluster-issuer.yaml` 文件，输入以下内容：

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
      commonName: "TiDB"
      isCA: true
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

    其中 `${cluster_name}` 为集群的名字，上面的文件创建三个对象：

    - 一个 SelfSigned 类型的 Isser 对象（用于生成 CA 类型 Issuer 所需要的 CA 证书）;
    - 一个 Certificate 对象，`isCa` 属性设置为 `true`；
    - 一个可以用于颁发 TiDB 组件间 TLS 证书的 Issuer。

    最后执行下面的命令进行创建：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f tidb-cluster-issuer.yaml
    ```

3. 创建 Server 端证书。

    在 `cert-manager` 中，Certificate 资源表示证书接口，该证书将由上面创建的 Issuer 颁发并保持更新。

    根据官网文档：[Enable TLS Authentication | TiDB Documentation](https://pingcap.com/docs/v3.0/how-to/secure/enable-tls-between-components/)，我们需要为每个组件创建一个 Server 端证书，并且为他们的 Client 创建一套公用的 Client 端证书。

    - PD 组件的 Server 端证书。

        ``` yaml
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
          - "${cluster_name}-pd-peer"
          - "${cluster_name}-pd-peer.${namespace}"
          - "${cluster_name}-pd-peer.${namespace}.svc"
          - "*.${cluster_name}-pd-peer"
          - "*.${cluster_name}-pd-peer.${namespace}"
          - "*.${cluster_name}-pd-peer.${namespace}.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: ${cluster_name}-tidb-issuer
            kind: Issuer
            group: cert-manager.io
        ```

        其中 `${cluster_name}` 为集群的名字：

        - `spec.secretName` 请设置为 `${cluster_name}-pd-cluster-secret`；
        - `usages` 请添加上  `server auth` 和 `client auth`；
        - `dnsNames` 需要填写这些 DNS，根据需要可以填写其他 DNS：
          - `${cluster_name}-pd`
          - `${cluster_name}-pd.${namespace}`
          - `${cluster_name}-pd.${namespace}.svc`
          - `${cluster_name}-pd-pee`
          - `${cluster_name}-pd-peer.${namespace}`
          - `${cluster_name}-pd-peer.${namespace}.svc`
          - `*.${cluster_name}-pd-peer`
          - `*.${cluster_name}-pd-peer.${namespace}`
          - `*.${cluster_name}-pd-peer.${namespace}.svc`
        - `ipAddresses` 需要填写这两个 IP ，根据需要可以填写其他 IP：
          - `127.0.0.1`
          - `::1`
        - `issuerRef` 请填写上面创建的 Issuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `${cluster_name}-pd-cluster-secret` 的 Secret 对象供 TiDB 集群的 PD 组件使用。

    - TiKV 组件的 Server 端证书。

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-tikv-cluster-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-tikv-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "${cluster_name}-tikv"
          - "${cluster_name}-tikv.${namespace}"
          - "${cluster_name}-tikv.${namespace}.svc"
          - "${cluster_name}-tikv-peer"
          - "${cluster_name}-tikv-peer.${namespace}"
          - "${cluster_name}-tikv-peer.${namespace}.svc"
          - "*.${cluster_name}-tikv-peer"
          - "*.${cluster_name}-tikv-peer.${namespace}"
          - "*.${cluster_name}-tikv-peer.${namespace}.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: ${cluster_name}-tidb-issuer
            kind: Issuer
            group: cert-manager.io
        ```

        其中 `${cluster_name}` 为集群的名字：

        - `spec.secretName` 请设置为 `${cluster_name}-tikv-cluster-secret`；
        - `usages` 请添加上  `server auth` 和 `client auth`；
        - `dnsNames` 需要填写这些 DNS，根据需要可以填写其他 DNS：
          - `${cluster_name}-tikv`
          - `${cluster_name}-tikv.${namespace}`
          - `${cluster_name}-tikv.${namespace}.svc`
          - `${cluster_name}-tikv-peer`
          - `${cluster_name}-tikv-peer.${namespace}`
          - `${cluster_name}-tikv-peer.${namespace}.svc`
          - `*.${cluster_name}-tikv-peer`
          - `*.${cluster_name}-tikv-peer.${namespace}`
          - `*.${cluster_name}-tikv-peer.${namespace}.svc`
        - `ipAddresses` 需要填写这两个 IP ，根据需要可以填写其他 IP：
          - `127.0.0.1`
          - `::1`
        - `issuerRef` 请填写上面创建的 Issuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `${cluster_name}-tikv-cluster-secret` 的 Secret 对象供 TiDB 集群的 TiKV 组件使用。

    - TiDB 组件的 Server 端证书。

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-tidb-cluster-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-tidb-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "${cluster_name}-tidb"
          - "${cluster_name}-tidb.${namespace}"
          - "${cluster_name}-tidb.${namespace}.svc"
          - "${cluster_name}-tidb-peer"
          - "${cluster_name}-tidb-peer.${namespace}"
          - "${cluster_name}-tidb-peer.${namespace}.svc"
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

        - `spec.secretName` 请设置为 `${cluster_name}-tidb-cluster-secret`；
        - `usages` 请添加上  `server auth` 和 `client auth`；
        - `dnsNames` 需要填写这些 DNS，根据需要可以填写其他 DNS：
          - `${cluster_name}-tidb`
          - `${cluster_name}-tidb.${namespace}`
          - `${cluster_name}-tidb.${namespace}.svc`
          - `${cluster_name}-tidb-peer`
          - `${cluster_name}-tidb-peer.${namespace}`
          - `${cluster_name}-tidb-peer.${namespace}.svc`
          - `*.${cluster_name}-tidb-peer`
          - `*.${cluster_name}-tidb-peer.${namespace}`
          - `*.${cluster_name}-tidb-peer.${namespace}.svc`
        - `ipAddresses` 需要填写这两个 IP ，根据需要可以填写其他 IP：
          - `127.0.0.1`
          - `::1`
        - `issuerRef` 请填写上面创建的 Issuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `${cluster_name}-tidb-cluster-secret` 的 Secret 对象供 TiDB 集群的 TiDB 组件使用。

    - Pump 组件的 Server 端证书。

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-pump-cluster-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-pump-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "*.${cluster_name}-pump"
          - "*.${cluster_name}-pump.${namespace}"
          - "*.${cluster_name}-pump.${namespace}.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: ${cluster_name}-tidb-issuer
            kind: Issuer
            group: cert-manager.io
        ```

        其中 `${cluster_name}` 为集群的名字：

        - `spec.secretName` 请设置为 `${cluster_name}-pump-cluster-secret`；
        - `usages` 请添加上  `server auth` 和 `client auth`；
        - `dnsNames` 需要填写这些 DNS，根据需要可以填写其他 DNS：
          - `*.${cluster_name}-pump`
          - `*.${cluster_name}-pump.${namespace}`
          - `*.${cluster_name}-pump.${namespace}.svc`
        - `ipAddresses` 需要填写这两个 IP ，根据需要可以填写其他 IP：
          - `127.0.0.1`
          - `::1`
        - `issuerRef` 请填写上面创建的 Issuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `${cluster_name}-pump-cluster-secret` 的 Secret 对象供 TiDB 集群的 Pump 组件使用。

    - Drainer 组件的 Server 端证书。

        现在 Drainer 组件是通过 Helm 来部署的，根据 `values.yaml` 文件配置方式不同，所需要填写的 `dnsNames` 字段也不相同。

        如果部署的时候设置 `drainerName` 属性，像下面这样：

        ``` yaml
        ...
        # Change the name of the statefulset and pod
        # The default is clusterName-ReleaseName-drainer
        # Do not change the name of an existing running drainer: this is unsupported.
        drainerName: my-drainer
        ...
        ```

        那么就需要这样配置证书：

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-drainer-cluster-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-drainer-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "*.${drainer_name}"
          - "*.${drainer_name}.${namespace}"
          - "*.${drainer_name}.${namespace}.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: ${cluster_name}-tidb-issuer
            kind: Issuer
            group: cert-manager.io
        ```

        如果部署的时候没有设置 `drainerName` 属性，需要这样配置 `dnsNames` 属性：

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-drainer-cluster-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-drainer-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "*.${cluster_name}-${release_name}-drainer"
          - "*.${cluster_name}-${release_name}-drainer.${namespace}"
          - "*.${cluster_name}-${release_name}-drainer.${namespace}.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: ${cluster_name}-tidb-issuer
            kind: Issuer
            group: cert-manager.io
        ```

        其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 TiDB 集群部署的命名空间，`${release_name}` 是 `helm install` 时候填写的 `release name`，`${drainer_name}` 为 `values.yaml` 文件里的 `drainerName`，用户也可以添加自定义 `dnsNames`。

        - `spec.secretName` 请设置为 `${cluster_name}-drainer-cluster-secret`；
        - `usages` 请添加上  `server auth` 和 `client auth`；
        - `dnsNames` 请参考上面的描述；
        - `ipAddresses` 需要填写这两个 IP ，根据需要可以填写其他 IP：
          - `127.0.0.1`
          - `::1`
        - `issuerRef` 请填写上面创建的 Issuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `${cluster_name}-drainer-cluster-secret` 的 Secret 对象供 TiDB 集群的 Drainer 组件使用。

    - TiCDC 组件的 Server 端证书。

      TiCDC 从 v4.0.3 版本开始支持 TLS，TiDB Operator v1.1.3 版本同步支持 TiCDC 开启 TLS 功能。

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-ticdc-cluster-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-ticdc-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "${cluster_name}-ticdc"
          - "${cluster_name}-ticdc.${namespace}"
          - "${cluster_name}-ticdc.${namespace}.svc"
          - "${cluster_name}-ticdc-peer"
          - "${cluster_name}-ticdc-peer.${namespace}"
          - "${cluster_name}-ticdc-peer.${namespace}.svc"
          - "*.${cluster_name}-ticdc-peer"
          - "*.${cluster_name}-ticdc-peer.${namespace}"
          - "*.${cluster_name}-ticdc-peer.${namespace}.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: ${cluster_name}-tidb-issuer
            kind: Issuer
            group: cert-manager.io
        ```

        其中 `${cluster_name}` 为集群的名字：

        - `spec.secretName` 请设置为 `${cluster_name}-ticdc-cluster-secret`；
        - `usages` 请添加上 `server auth` 和 `client auth`；
        - `dnsNames` 需要填写这些 DNS，根据需要可以填写其他 DNS：
          - `${cluster_name}-ticdc`
          - `${cluster_name}-ticdc.${namespace}`
          - `${cluster_name}-ticdc.${namespace}.svc`
          - `${cluster_name}-ticdc-peer`
          - `${cluster_name}-ticdc-peer.${namespace}`
          - `${cluster_name}-ticdc-peer.${namespace}.svc`
          - `*.${cluster_name}-ticdc-peer`
          - `*.${cluster_name}-ticdc-peer.${namespace}`
          - `*.${cluster_name}-ticdc-peer.${namespace}.svc`
        - `ipAddresses` 需要填写这两个 IP，根据需要可以填写其他 IP：
          - `127.0.0.1`
          - `::1`
        - `issuerRef` 请填写上面创建的 Issuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `${cluster_name}-ticdc-cluster-secret` 的 Secret 对象供 TiDB 集群的 TiCDC 组件使用。

    - TiFlash 组件的 Server 端证书。

        ```yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-tiflash-cluster-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-tiflash-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "${cluster_name}-tiflash"
          - "${cluster_name}-tiflash.${namespace}"
          - "${cluster_name}-tiflash.${namespace}.svc"
          - "${cluster_name}-tiflash-peer"
          - "${cluster_name}-tiflash-peer.${namespace}"
          - "${cluster_name}-tiflash-peer.${namespace}.svc"
          - "*.${cluster_name}-tiflash-peer"
          - "*.${cluster_name}-tiflash-peer.${namespace}"
          - "*.${cluster_name}-tiflash-peer.${namespace}.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: ${cluster_name}-tidb-issuer
            kind: Issuer
            group: cert-manager.io
        ```

        其中 `${cluster_name}` 为集群的名字：

        - `spec.secretName` 请设置为 `${cluster_name}-tiflash-cluster-secret`；
        - `usages` 请添加上 `server auth` 和 `client auth`；
        - `dnsNames` 需要填写这些 DNS，根据需要可以填写其他 DNS：
            - `${cluster_name}-tiflash`
            - `${cluster_name}-tiflash.${namespace}`
            - `${cluster_name}-tiflash.${namespace}.svc`
            - `${cluster_name}-tiflash-peer`
            - `${cluster_name}-tiflash-peer.${namespace}`
            - `${cluster_name}-tiflash-peer.${namespace}.svc`
            - `*.${cluster_name}-tiflash-peer`
            - `*.${cluster_name}-tiflash-peer.${namespace}`
            - `*.${cluster_name}-tiflash-peer.${namespace}.svc`
        - `ipAddresses` 需要填写这两个 IP ，根据需要可以填写其他 IP：
            - `127.0.0.1`
            - `::1`
        - `issuerRef` 请填写上面创建的 Issuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `${cluster_name}-tiflash-cluster-secret` 的 Secret 对象供 TiDB 集群的 TiFlash 组件使用。

    - 一套 TiDB 集群组件的 Client 端证书。

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-cluster-client-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-cluster-client-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB"
          usages:
            - client auth
          issuerRef:
            name: ${cluster_name}-tidb-issuer
            kind: Issuer
            group: cert-manager.io
        ```

        其中 `${cluster_name}` 为集群的名字：

        - `spec.secretName` 请设置为 `${cluster_name}-cluster-client-secret`；
        - `usages` 请添加上  `client auth`；
        - `dnsNames` 和 `ipAddresses` 不需要填写；
        - `issuerRef` 请填写上面创建的 Issuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `${cluster_name}-cluster-client-secret` 的 Secret 对象供 TiDB 组件的 Client 使用。

## 第二步：部署 TiDB 集群

在部署 TiDB 集群时，可以开启集群间的 TLS，同时可以设置 `cert-allowed-cn` 配置项（TiDB 为 `cluster-verify-cn`），用来验证集群间各组件证书的 CN (Common Name)。

> **注意：**
>
> 目前 PD 的 `cert-allowed-cn` 配置项只能设置一个值。因此所有 `Certificate` 对象的 `commonName` 都要设置成同样一个值。

在这一步中，需要完成以下操作：

- 创建一套 TiDB 集群
- 为 TiDB 组件间开启 TLS，并开启 CN 验证
- 部署一套监控系统
- 部署 Pump 组件，并开启 CN 验证

1. 创建一套 TiDB 集群（监控系统和 Pump 组件已包含在内）：

    创建 `tidb-cluster.yaml` 文件：

    ``` yaml
    apiVersion: pingcap.com/v1alpha1
    kind: TidbCluster
    metadata:
     name: ${cluster_name}
     namespace: ${namespace}
    spec:
     tlsCluster:
       enabled: true
     version: v4.0.4
     timezone: UTC
     pvReclaimPolicy: Retain
     pd:
       baseImage: pingcap/pd
       replicas: 1
       requests:
         storage: "1Gi"
       config:
         security:
           cert-allowed-cn:
             - TiDB
     tikv:
       baseImage: pingcap/tikv
       replicas: 1
       requests:
         storage: "1Gi"
       config:
         security:
           cert-allowed-cn:
             - TiDB
     tidb:
       baseImage: pingcap/tidb
       replicas: 1
       service:
         type: ClusterIP
       config:
         security:
           cluster-verify-cn:
             - TiDB
     pump:
       baseImage: pingcap/tidb-binlog
       replicas: 1
       requests:
         storage: "1Gi"
       config:
         security:
           cert-allowed-cn:
             - TiDB
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: TidbMonitor
    metadata:
     name: ${cluster_name}
     namespace: ${namespace}
    spec:
     clusters:
     - name: ${cluster_name}
     prometheus:
       baseImage: prom/prometheus
       version: v2.11.1
     grafana:
       baseImage: grafana/grafana
       version: 6.0.1
     initializer:
       baseImage: pingcap/tidb-monitor-initializer
       version: v4.0.4
     reloader:
       baseImage: pingcap/tidb-monitor-reloader
       version: v1.0.1
     imagePullPolicy: IfNotPresent
    ```

    然后使用 `kubectl apply -f tidb-cluster.yaml` 来创建 TiDB 集群。

2. 创建 Drainer 组件并开启 TLS 以及 CN 验证。

    - 第一种方式：创建 Drainer 的时候设置 `drainerName`：

        编辑 values.yaml 文件，设置好 drainer-name，并将 TLS 功能打开：

        ``` yaml
        ...
        drainerName: ${drainer_name}
        tlsCluster:
          enabled: true
          certAllowedCN:
            - TiDB
        ...
        ```

        然后部署 Drainer 集群：

        {{< copyable "shell-regular" >}}

        ``` shell
        helm install pingcap/tidb-drainer --name=${release_name} --namespace=${namespace} --version=${helm_version} -f values.yaml
        ```

    - 第二种方式：创建 Drainer 的时候不设置 `drainerName`：

        编辑 values.yaml 文件，将 TLS 功能打开：

        ``` yaml
        ...
        tlsCluster:
          enabled: true
          certAllowedCN:
            - TiDB
        ...
        ```

        然后部署 Drainer 集群：

        {{< copyable "shell-regular" >}}

        ``` shell
        helm install pingcap/tidb-drainer --name=${release_name} --namespace=${namespace} --version=${helm_version} -f values.yaml
        ```

3. 创建 Backup/Restore 资源对象。

    - 创建 `backup.yaml` 文件：

        ``` yaml
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
          s3:
            provider: aws
            region: ${my_region}
            secretName: ${s3_secret}
            bucket: ${my_bucket}
            prefix: ${my_folder}
        ````

        然后部署 Backup：

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl apply -f backup.yaml
        ```

    - 创建 `restore.yaml` 文件：

        ``` yaml
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
          s3:
            provider: aws
            region: ${my_region}
            secretName: ${s3_secret}
            bucket: ${my_bucket}
            prefix: ${my_folder}

        ````

        然后部署 Restore：

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl apply -f restore.yaml
        ```

## 第三步：配置 `pd-ctl` 连接集群

1. 下载 `pd-ctl`。

    参考官网文档：[Download TiDB installation package](https://pingcap.com/docs/v3.0/reference/tools/pd-control/#download-tidb-installation-package)。

2. 下载 Client 端证书。

    Client 端证书就是[第一步](#第一步为-tidb-集群各个组件生成证书)中创建的那套 Client 证书，可以直接使用或者从 Kubernetes Secret 对象（之前创建的）里获取，这个 Secret 对象的名字是: `${cluster_name}-cluster-client-secret`。

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl get secret -n ${namespace} ${cluster_name}-cluster-client-secret  -ojsonpath='{.data.tls\.crt}' | base64 --decode > client-tls.crt
    kubectl get secret -n ${namespace} ${cluster_name}-cluster-client-secret  -ojsonpath='{.data.tls\.key}' | base64 --decode > client-tls.key
    kubectl get secret -n ${namespace} ${cluster_name}-cluster-client-secret  -ojsonpath='{.data.ca\.crt}'  | base64 --decode > client-ca.crt
    ```

3. 使用 `pd-ctl`，`tikv-ctl` 连接集群。

    由于我们刚才在配置 PD/TiKV Server 端证书的时候，自定义填写了一些 `hosts`，所以需要通过这些 `hosts` 来连接 PD/TiKV 集群。

    - 连接 PD 集群:

        {{< copyable "shell-regular" >}}

        ``` shell
        pd-ctl --cacert=client-ca.crt --cert=client-tls.crt --key=client-tls.key -u https://${cluster_name}-pd.${namespace}.svc:2379 member
        ```
    
    - 连接 TiKV 集群：

        {{< copyable "shell-regular" >}}

        ``` shell
        tikv-ctl --ca-path=client-ca.crt --cert-path=client-tls.crt --key-path=client-tls.key --host ${cluster_name}-tikv-0.${cluster_name}-tikv-peer.${namespace}:20160 cluster
        ```
