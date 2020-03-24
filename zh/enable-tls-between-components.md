---
title: 为 TiDB 组件间开启 TLS
summary: 在 Kubernetes 上如何为 TiDB 集群组件间开启 TLS。
category: how-to
---

# 为 TiDB 组件间开启 TLS

本文主要描述了在 Kubernetes 上如何为 TiDB 集群组件间开启 TLS。TiDB Operator 从 v1.1 开始已经支持为 Kubernetes 上 TiDB 集群组件间开启 TLS。开启步骤为：

1. 为即将被创建的 TiDB 集群的每个组件生成证书：为 PD/TiKV/TiDB/Pump/Drainer 组件分别创建一套 Server 端证书，保存为 Kubernetes Secret 对象：`<cluster-name>-<component-name>-cluster-secret`，为它们的各种客户端创建一套共用的 Client 端证书，保存为 Kubernetes Secret 对象：`<cluster-name>-cluster-client-secret`；
2. 部署集群，设置 `.spec.tlsCluster.enabled` 属性为 `true`；
3. 配置 `pd-ctl` 连接集群。


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

    mkdir -p ~/cfssl
    cd ~/cfssl
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
    ```

    注意：这里的 `profile server` 的 `usages` 必须添加上 `"client auth"`。因为这套 Server 端证书同时也会作为 Client 端证书使用。

3. 您还可以修改 `ca-csr.json` 证书签名请求 (CSR)：

    ``` json
    {
        "CN": "TiDB Server",
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
            "CN": "PD Server",
            "hosts": [
              "127.0.0.1",
              "::1",
              "<cluster-name>-pd",
              "<cluster-name>-pd.<namespace>",
              "<cluster-name>-pd.<namespace>.svc",
              "<cluster-name>-pd-peer",
              "<cluster-name>-pd-peer.<namespace>",
              "<cluster-name>-pd-peer.<namespace>.svc",
              "*.<cluster-name>-pd-peer",
              "*.<cluster-name>-pd-peer.<namespace>",
              "*.<cluster-name>-pd-peer.<namespace>.svc"
            ],
        ...
        ```

        其中 `<cluster-name>` 为集群的名字，`<namespace>` 为 TiDB 集群部署的命名空间，用户也可以添加自定义 `hosts`。

        最后生成 PD Server 端证书：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=server pd-server.json | cfssljson -bare pd-server
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
            "CN": "TiKV Server",
            "hosts": [
              "127.0.0.1",
              "::1",
              "<cluster-name>-tikv",
              "<cluster-name>-tikv.<namespace>",
              "<cluster-name>-tikv.<namespace>.svc",
              "<cluster-name>-tikv-peer",
              "<cluster-name>-tikv-peer.<namespace>",
              "<cluster-name>-tikv-peer.<namespace>.svc",
              "*.<cluster-name>-tikv-peer",
              "*.<cluster-name>-tikv-peer.<namespace>",
              "*.<cluster-name>-tikv-peer.<namespace>.svc"
            ],
        ...
        ```

        其中 `<cluster-name>` 为集群的名字，`<namespace>` 为 TiDB 集群部署的命名空间，用户也可以添加自定义 `hosts`。

        最后生成 TiKV Server 端证书：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=server tikv-server.json | cfssljson -bare tikv-server
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
            "CN": "TiDB Server",
            "hosts": [
              "127.0.0.1",
              "::1",
              "<cluster-name>-tidb",
              "<cluster-name>-tidb.<namespace>",
              "<cluster-name>-tidb.<namespace>.svc",
              "<cluster-name>-tidb-peer",
              "<cluster-name>-tidb-peer.<namespace>",
              "<cluster-name>-tidb-peer.<namespace>.svc",
              "*.<cluster-name>-tidb-peer",
              "*.<cluster-name>-tidb-peer.<namespace>",
              "*.<cluster-name>-tidb-peer.<namespace>.svc"
            ],
        ...
        ```

        其中 `<cluster-name>` 为集群的名字，`<namespace>` 为 TiDB 集群部署的命名空间，用户也可以添加自定义 `hosts`。

        最后生成 TiDB Server 端证书：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=server tidb-server.json | cfssljson -bare tidb-server
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
            "CN": "Pump Server",
            "hosts": [
              "127.0.0.1",
              "::1",
              "*.<cluster-name>-pump",
              "*.<cluster-name>-pump.<namespace>",
              "*.<cluster-name>-pump.<namespace>.svc"
            ],
        ...
        ```

        其中 `<cluster-name>` 为集群的名字，`<namespace>` 为 TiDB 集群部署的命名空间，用户也可以添加自定义 `hosts`。

        最后生成 Pump Server 端证书：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=server pump-server.json | cfssljson -bare pump-server
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
            "CN": "Drainer Server",
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
            "CN": "Drainer Server",
            "hosts": [
              "127.0.0.1",
              "::1",
              "*.<drainer-name>",
              "*.<drainer-name>.<namespace>",
              "*.<drainer-name>.<namespace>.svc"
            ],
        ...
        ```

        如果部署的时候没有设置 `drainerName` 属性，需要这样配置 `hosts` 属性：

        ``` json
        ...
            "CN": "Drainer Server",
            "hosts": [
              "127.0.0.1",
              "::1",
              "*.<cluster-name>-<release-name>-drainer",
              "*.<cluster-name>-<release-name>-drainer.<namespace>",
              "*.<cluster-name>-<release-name>-drainer.<namespace>.svc"
            ],
        ...
        ```

        其中 `<cluster-name>` 为集群的名字，`<namespace>` 为 TiDB 集群部署的命名空间，`<release-name>` 是 `helm install` 时候填写的 `release name`，`<drainer-name>` 为 `values.yaml` 文件里的 `drainerName`，用户也可以添加自定义 `hosts`。

        最后生成 Drainer Server 端证书：

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=server drainer-server.json | cfssljson -bare drainer-server
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
        "CN": "TiDB Cluster Client",
        "hosts": [],
    ...
    ```

    最后生成 Client 端证书：

    ``` shell
    cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=client client.json | cfssljson -bare client
    ```

7. 创建 Kubernetes Secret 对象。

    到这里假设你已经按照上述文档为每个组件创建了一套 Server 端证书和 他们的各种客户端创建了一套 Client 端证书。通过下面的命令为 TiDB 集群创建这些 Secret 对象。

    PD 集群证书 Secret：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create secret generic <cluster-name>-pd-cluster-secret --namespace=<namespace> --from-file=tls.crt=~/cfssl/pd-server.pem --from-file=tls.key=~/cfssl/pd-server-key.pem --from-file=ca.crt=~/cfssl/ca.pem
    ```

    TiKV 集群证书 Secret：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create secret generic <cluster-name>-tikv-cluster-secret --namespace=<namespace> --from-file=tls.crt=~/cfssl/tikv-server.pem --from-file=tls.key=~/cfssl/tikv-server-key.pem --from-file=ca.crt=~/cfssl/ca.pem
    ```

    TiDB 集群证书 Secret：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create secret generic <cluster-name>-tidb-cluster-secret --namespace=<namespace> --from-file=tls.crt=~/cfssl/tidb-server.pem --from-file=tls.key=~/cfssl/tidb-server-key.pem --from-file=ca.crt=~/cfssl/ca.pem
    ```

    Pump 集群证书 Secret：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create secret generic <cluster-name>-pump-cluster-secret --namespace=<namespace> --from-file=tls.crt=~/cfssl/pump-server.pem --from-file=tls.key=~/cfssl/pump-server-key.pem --from-file=ca.crt=~/cfssl/ca.pem
    ```

    Drainer 集群证书 Secret：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create secret generic <cluster-name>-drainer-cluster-secret --namespace=<namespace> --from-file=tls.crt=~/cfssl/drainer-server.pem --from-file=tls.key=~/cfssl/drainer-server-key.pem --from-file=ca.crt=~/cfssl/ca.pem
    ```

    Client 证书 Secret：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl create secret generic <cluster-name>-cluster-client-secret --namespace=<namespace> --from-file=tls.crt=~/cfssl/client.pem --from-file=tls.key=~/cfssl/client-key.pem --from-file=ca.crt=~/cfssl/ca.pem
    ```

    这里给 PD/TiKV/TiDB/Pump/Drainer 的 Server 端证书分别创建了一个 Secret 供他们启动时加载使用，另外一套 Client 端证书供他们的客户端连接使用。

### 使用 `cert-manager` 系统颁发证书

1. 安装 cert-manager。

    请参考官网安装：[cert-manager installation in Kubernetes](https://docs.cert-manager.io/en/release-0.11/getting-started/install/kubernetes.html)。

2. 创建一个 ClusterIssuer 用于给 TiDB 集群颁发证书。

    为了配置 `cert-manager` 颁发证书，必须先创建 Issuer 或 ClusterIssuer 资源，这里使用 ClusterIssuer 资源可以颁发多个 `namespace` 下的证书。

    首先创建一个目录保存 `cert-manager` 创建证书所需文件：

    {{< copyable "shell-regular" >}}

    ``` shell
    mkdir -p ~/cert-manager
    cd ~/cert-manager
    ```

    然后创建一个 `tidb-cluster-issuer.yaml` 文件，输入以下内容：

    ``` yaml
    apiVersion: cert-manager.io/v1alpha2
    kind: ClusterIssuer
    metadata:
      name: tidb-selfsigned-ca-issuer
    spec:
      selfSigned: {}
    ---
    apiVersion: cert-manager.io/v1alpha2
    kind: Certificate
    metadata:
      name: tidb-cluster-issuer-cert
      namespace: cert-manager
    spec:
      secretName: tidb-cluster-issuer-cert
      commonName: "TiDB CA"
      isCA: true
      issuerRef:
        name: tidb-selfsigned-ca-issuer
        kind: ClusterIssuer
    ---
    apiVersion: cert-manager.io/v1alpha2
    kind: ClusterIssuer
    metadata:
      name: tidb-cluster-issuer
    spec:
      ca:
        secretName: tidb-cluster-issuer-cert
    ```

    上面的文件创建三个对象：

    - 一个 SelfSigned 类型的 ClusterIsser 对象（用于生成 CA 类型 ClusterIssuer 所需要的 CA 证书）;
    - 一个 Certificate 对象，`isCa` 属性设置为 `true`；
    - 一个可以用于颁发 TiDB 组件间 TLS 证书的 ClusterIssuer。

    最后执行下面的命令进行创建：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f ~/cert-manager/tidb-cluster-issuer.yaml
    ```

3. 创建 Server 端证书。

    在 `cert-manager` 中，Certificate 资源表示证书接口，该证书将由上面创建的 ClusterIssuer 颁发并保持更新。

    根据官网文档：[Enable TLS Authentication | TiDB Documentation](https://pingcap.com/docs/stable/how-to/secure/enable-tls-between-components/)，我们需要为每个组件创建一个 Server 端证书，并且为他们的 Client 创建一套公用的 Client 端证书。

    - PD 组件的 Server 端证书。

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: <cluster-name>-pd-cluster-secret
          namespace: <namespace>
        spec:
          secretName: <cluster-name>-pd-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "PD"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "<cluster-name>-pd"
          - "<cluster-name>-pd.<namespace>"
          - "<cluster-name>-pd.<namespace>.svc"
          - "<cluster-name>-pd-peer"
          - "<cluster-name>-pd-peer.<namespace>"
          - "<cluster-name>-pd-peer.<namespace>.svc"
          - "*.<cluster-name>-pd-peer"
          - "*.<cluster-name>-pd-peer.<namespace>"
          - "*.<cluster-name>-pd-peer.<namespace>.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: tidb-cluster-issuer
            kind: ClusterIssuer
            group: cert-manager.io
        ```

        其中 `<cluster-name>` 为集群的名字：

        - `spec.secretName` 请设置为 `<cluster-name>-pd-cluster-secret`；
        - `usages` 请添加上  `server auth` 和 `client auth`；
        - `dnsNames` 需要填写这些 DNS，根据需要可以填写其他 DNS：
          - `<cluster-name>-pd`
          - `<cluster-name>-pd.<namespace>`
          - `<cluster-name>-pd.<namespace>.svc`
          - `<cluster-name>-pd-pee`
          - `<cluster-name>-pd-peer.<namespace>`
          - `<cluster-name>-pd-peer.<namespace>.svc`
          - `*.<cluster-name>-pd-peer`
          - `*.<cluster-name>-pd-peer.<namespace>`
          - `*.<cluster-name>-pd-peer.<namespace>.svc`
        - `ipAddresses` 需要填写这两个 IP ，根据需要可以填写其他 IP：
          - `127.0.0.1`
          - `::1`
        - `issuerRef` 请填写上面创建的 ClusterIssuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `<cluster-name>-pd-cluster-secret` 的 Secret 对象供 TiDB 集群的 PD 组件使用。

    - TiKV 组件的 Server 端证书。

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: <cluster-name>-tikv-cluster-secret
          namespace: <namespace>
        spec:
          secretName: <cluster-name>-tikv-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiKV"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "<cluster-name>-tikv"
          - "<cluster-name>-tikv.<namespace>"
          - "<cluster-name>-tikv.<namespace>.svc"
          - "<cluster-name>-tikv-peer"
          - "<cluster-name>-tikv-peer.<namespace>"
          - "<cluster-name>-tikv-peer.<namespace>.svc"
          - "*.<cluster-name>-tikv-peer"
          - "*.<cluster-name>-tikv-peer.<namespace>"
          - "*.<cluster-name>-tikv-peer.<namespace>.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: tidb-cluster-issuer
            kind: ClusterIssuer
            group: cert-manager.io
        ```

        其中 `<cluster-name>` 为集群的名字：

        - `spec.secretName` 请设置为 `<clusterName>-tikv-cluster-secret`；
        - `usages` 请添加上  `server auth` 和 `client auth`；
        - `dnsNames` 需要填写这些 DNS，根据需要可以填写其他 DNS：
          - `<cluster-name>-tikv`
          - `<cluster-name>-tikv.<namespace>`
          - `<cluster-name>-tikv.<namespace>.svc`
          - `<cluster-name>-tikv-peer`
          - `<cluster-name>-tikv-peer.<namespace>`
          - `<cluster-name>-tikv-peer.<namespace>.svc`
          - `*.<cluster-name>-tikv-peer`
          - `*.<cluster-name>-tikv-peer.<namespace>`
          - `*.<cluster-name>-tikv-peer.<namespace>.svc`
        - `ipAddresses` 需要填写这两个 IP ，根据需要可以填写其他 IP：
          - `127.0.0.1`
          - `::1`
        - `issuerRef` 请填写上面创建的 ClusterIssuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `<cluster-name>-tikv-cluster-secret` 的 Secret 对象供 TiDB 集群的 TiKV 组件使用。

    - TiDB 组件的 Server 端证书。

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: <cluster-name>-tidb-cluster-secret
          namespace: <namespace>
        spec:
          secretName: <cluster-name>-tidb-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "<cluster-name>-tidb"
          - "<cluster-name>-tidb.<namespace>"
          - "<cluster-name>-tidb.<namespace>.svc"
          - "<cluster-name>-tidb-peer"
          - "<cluster-name>-tidb-peer.<namespace>"
          - "<cluster-name>-tidb-peer.<namespace>.svc"
          - "*.<cluster-name>-tidb-peer"
          - "*.<cluster-name>-tidb-peer.<namespace>"
          - "*.<cluster-name>-tidb-peer.<namespace>.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: tidb-cluster-issuer
            kind: ClusterIssuer
            group: cert-manager.io
        ```

        其中 `<cluster-name>` 为集群的名字：

        - `spec.secretName` 请设置为 `<clusterName>-tidb-cluster-secret`；
        - `usages` 请添加上  `server auth` 和 `client auth`；
        - `dnsNames` 需要填写这些 DNS，根据需要可以填写其他 DNS：
          - `<cluster-name>-tidb`
          - `<cluster-name>-tidb.<namespace>`
          - `<cluster-name>-tidb.<namespace>.svc`
          - `<cluster-name>-tidb-peer`
          - `<cluster-name>-tidb-peer.<namespace>`
          - `<cluster-name>-tidb-peer.<namespace>.svc`
          - `*.<cluster-name>-tidb-peer`
          - `*.<cluster-name>-tidb-peer.<namespace>`
          - `*.<cluster-name>-tidb-peer.<namespace>.svc`
        - `ipAddresses` 需要填写这两个 IP ，根据需要可以填写其他 IP：
          - `127.0.0.1`
          - `::1`
        - `issuerRef` 请填写上面创建的 ClusterIssuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `<cluster-name>-tidb-cluster-secret` 的 Secret 对象供 TiDB 集群的 TiDB 组件使用。


    - Pump 组件的 Server 端证书。

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: <cluster-name>-pump-cluster-secret
          namespace: <namespace>
        spec:
          secretName: <cluster-name>-pump-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "Pump"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "*.<cluster-name>-pump"
          - "*.<cluster-name>-pump.<namespace>"
          - "*.<cluster-name>-pump.<namespace>.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: tidb-cluster-issuer
            kind: ClusterIssuer
            group: cert-manager.io
        ```

        其中 `<cluster-name>` 为集群的名字：

        - `spec.secretName` 请设置为 `<cluster-name>-pump-cluster-secret`；
        - `usages` 请添加上  `server auth` 和 `client auth`；
        - `dnsNames` 需要填写这些 DNS，根据需要可以填写其他 DNS：
          - `*.<cluster-name>-pump`
          - `*.<cluster-name>-pump.<namespace>`
          - `*.<cluster-name>-pump.<namespace>.svc`
        - `ipAddresses` 需要填写这两个 IP ，根据需要可以填写其他 IP：
          - `127.0.0.1`
          - `::1`
        - `issuerRef` 请填写上面创建的 ClusterIssuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `<cluster-name>-pump-cluster-secret` 的 Secret 对象供 TiDB 集群的 Pump 组件使用。

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
          name: <cluster-name>-drainer-cluster-secret
          namespace: <namespace>
        spec:
          secretName: <cluster-name>-drainer-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "Drainer"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "*.<drainer-name>"
          - "*.<drainer-name>.<namespace>"
          - "*.<drainer-name>.<namespace>.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: tidb-cluster-issuer
            kind: ClusterIssuer
            group: cert-manager.io
        ```

        如果部署的时候没有设置 `drainerName` 属性，需要这样配置 `dnsNames` 属性：

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: <cluster-name>-drainer-cluster-secret
          namespace: <namespace>
        spec:
          secretName: <cluster-name>-drainer-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "Drainer"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "*.<cluster-name>-<release-name>-drainer"
          - "*.<cluster-name>-<release-name>-drainer.<namespace>"
          - "*.<cluster-name>-<release-name>-drainer.<namespace>.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: tidb-cluster-issuer
            kind: ClusterIssuer
            group: cert-manager.io
        ```

        其中 `<cluster-name>` 为集群的名字，`<namespace>` 为 TiDB 集群部署的命名空间，`<release-name>` 是 `helm install` 时候填写的 `release name`，`<drainer-name>` 为 `values.yaml` 文件里的 `drainerName`，用户也可以添加自定义 `dnsNames`。

        - `spec.secretName` 请设置为 `<cluster-name>-drainer-cluster-secret`；
        - `usages` 请添加上  `server auth` 和 `client auth`；
        - `dnsNames` 请参考上面的描述；
        - `ipAddresses` 需要填写这两个 IP ，根据需要可以填写其他 IP：
          - `127.0.0.1`
          - `::1`
        - `issuerRef` 请填写上面创建的 ClusterIssuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `<cluster-name>-drainer-cluster-secret` 的 Secret 对象供 TiDB 集群的 Drainer 组件使用。

    - 一套 TiDB 集群组件的 Client 端证书。

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: <cluster-name>-cluster-client-secret
          namespace: <namespace>
        spec:
          secretName: <cluster-name>-cluster-client-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB Components TLS Client"
          usages:
            - client auth
          issuerRef:
            name: tidb-cluster-issuer
            kind: ClusterIssuer
            group: cert-manager.io
        ```

        其中 `<cluster-name>` 为集群的名字：

        - `spec.secretName` 请设置为 `<cluster-name>-cluster-client-secret`；
        - `usages` 请添加上  `client auth`；
        - `dnsNames` 和 `ipAddresses` 不需要填写；
        - `issuerRef` 请填写上面创建的 ClusterIssuer；
        - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

        创建这个对象以后，`cert-manager` 会生成一个名字为 `<cluster-name>-cluster-client-secret` 的 Secret 对象供 TiDB 组件的 Client 使用。获取 Client 证书的方式是：

        {{< copyable "shell-regular" >}}

        ``` shell
        mkdir -p ~/<cluster-name>-cluster-client-tls
        cd ~/<cluster-name>-cluster-client-tls
        kubectl get secret -n <namespace> <cluster-name>-cluster-client-secret  -ojsonpath='{.data.tls\.crt}' | base64 --decode > tls.crt
        kubectl get secret -n <namespace> <cluster-name>-cluster-client-secret  -ojsonpath='{.data.tls\.key}' | base64 --decode > tls.key
        kubectl get secret -n <namespace> <cluster-name>-cluster-client-secret  -ojsonpath='{.data.ca\.crt}' | base64 --decode > ca.crt
        ```

## 第二步：部署 TiDB 集群

1. 创建一套 TiDB 集群。
    接下来将会通过两个 CR 对象来创建一个 TiDB 集群，并且执行以下步骤：

    - 为 TiDB 组件间开启 TLS；
    - 部署一套监控系统；
    - 部署 Pump 组件。

    创建 `tidb-cluster.yaml` 文件：

    ``` yaml
    apiVersion: pingcap.com/v1alpha1
    kind: TidbCluster
    metadata:
     name: <cluster-name>
     namespace: <namespace>
    spec:
     tlsClusster:
       enabled: true
     version: v3.0.8
     timezone: UTC
     pvReclaimPolicy: Retain
     pd:
       baseImage: pingcap/pd
       replicas: 1
       requests:
         storage: "1Gi"
       config: {}
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
     pump:
       baseImage: pingcap/tidb-binlog
       replicas: 1
       requests:
         storage: "1Gi"
       config: {}
    ---
    apiVersion: pingcap.com/v1alpha1
    kind: TidbMonitor
    metadata:
     name: <cluser-name>-monitor
     namespace: <namespace>
    spec:
     clusters:
     - name: <cluster-name>
     prometheus:
       baseImage: prom/prometheus
       version: v2.11.1
     grafana:
       baseImage: grafana/grafana
       version: 6.0.1
     initializer:
       baseImage: pingcap/tidb-monitor-initializer
       version: v3.0.8
     reloader:
       baseImage: pingcap/tidb-monitor-reloader
       version: v1.0.1
     imagePullPolicy: IfNotPresent
    ```

    然后使用 `kubectl apply -f tidb-cluster.yaml` 来创建 TiDB 集群。

2. 创建 Drainer 组件并开启 TLS。

    - 第一种方式：创建 Drainer 的时候设置 `drainerName`：

        编辑 values.yaml 文件，设置好 drainer-name，并将 TLS 功能打开：

        ``` yaml
        ...
        drainerName: <drainer-name>
        tlsCluster:
          enabled: true
        ...
        ```

        然后部署 Drainer 集群：

        {{< copyable "shell-regular" >}}

        ``` shell
        helm install charts/tidb-drainer --name=<release-name> --namespace=<namespace>
        ```

    - 第二种方式：创建 Drainer 的时候不设置 `drainerName`：

        编辑 values.yaml 文件，将 TLS 功能打开：

        ``` yaml
        ...
        tlsCluster:
          enabled: true
        ...
        ```

        然后部署 Drainer 集群：

        {{< copyable "shell-regular" >}}

        ``` shell
        helm install charts/tidb-drainer --name=<release-name> --namespace=<namespace>
        ```

3. 创建 Backup/Restore 资源对象并开启 TLS。

    - 创建 `backup.yaml` 文件，并将 TLS 功能打开：

        ``` yaml
        apiVersion: pingcap.com/v1alpha1
        kind: Backup
        metadata:
          name: <cluster-name>-backup
          namespace: <namespace>
        spec:
          backupType: full
          br:
            cluster: <cluster-name>
            clusterNamespace: <namespace>
            sendCredToTikv: true
            tlsCluster:
              enabled: true
          from:
            host: <host>
            secretName: <tidb-secret>
            port: 4000
            user: root
          s3:
            provider: aws
            region: <my-region>
            secretName: <s3-secret>
            bucket: <my-bucket>
            prefix: <my-folder>
        ````

        然后部署 Backup：

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl apply -f backup.yaml
        ```

    - 创建 `restore.yaml` 文件，并将 TLS 功能打开：

        ``` yaml
        apiVersion: pingcap.com/v1alpha1
        kind: Restore
        metadata:
          name: <cluster-name>-restore
          namespace: <namespace>
        spec:
          backupType: full
          br:
            cluster: <cluster-name>
            clusterNamespace: <namespace>
            sendCredToTikv: true
            tlsCluster:
              enabled: true
          to:
            host: <host>
            secretName: <tidb-secret>
            port: 4000
            user: root
          s3:
            provider: aws
            region: <my-region>
            secretName: <s3-secret>
            bucket: <my-bucket>
            prefix: <my-folder>

        ````

        然后部署 Restore：

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl apply -f restore.yaml
        ```

## 第三步：配置 `pd-ctl` 连接集群。

1. 下载 `pd-ctl`。

    参考官网文档：[Download TiDB installation package](https://pingcap.com/docs/stable/reference/tools/pd-control/#download-tidb-installation-package)。

2. 连接集群。

    首先需要下载 Client 端证书，Client 端证书就是上面创建的那套 Client 证书，可以直接使用或者从 K8s Secret 对象（之前创建的）里获取，这个 Secret 对象的名字是: `<cluster-name>-cluster-client-secret`。

    {{< copyable "shell-regular" >}}

    ``` shell
    mkdir -p ~/<cluster-name>-cluster-client-tls
    cd ~/<cluster-name>-cluster-client-tls
    kubectl get secret -n <namespace> <cluster-name>-cluster-client-secret  -ojsonpath='{.data.tls\.crt}' | base64 --decode > tls.crt
    kubectl get secret -n <namespace> <cluster-name>-cluster-client-secret  -ojsonpath='{.data.tls\.key}' | base64 --decode > tls.key
    kubectl get secret -n <namespace> <cluster-name>-cluster-client-secret  -ojsonpath='{.data.ca\.crt}' | base64 --decode > ca.crt
    ```

3. 使用 pd-ctl 连接 PD 集群。
    由于我们刚才在配置 PD Server 端证书的时候，自定义填写了一些 `hosts`，所以需要通过这些 `hosts` 来连接 PD 集群。

    {{< copyable "shell-regular" >}}

    ``` shell
    pd-ctl --cacert=~/<cluster-name>-cluster-client-tls/ca.crt --cert=~/<cluster-name>-cluster-client-tls/tls.crt --key=~/<cluster-name>-cluster-client-tls/tls.key -u https://<cluster-name>-pd.<namespace>.svc:2379 member
    ```
