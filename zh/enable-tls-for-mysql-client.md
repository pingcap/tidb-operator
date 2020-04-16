---
title: 为 MySQL 客户端开启 TLS
summary: 在 Kubernetes 上如何为 TiDB 集群的 MySQL 客户端开启 TLS。
category: how-to
---

# 为 MySQL 客户端开启 TLS

本文主要描述了在 Kubernetes 上如何为 TiDB 集群的 MySQL 客户端开启 TLS。TiDB Operator 从 v1.1 开始已经支持为 Kubernetes 上 TiDB 集群开启 MySQL 客户端 TLS。开启步骤为：

1. 为 TiDB Server 颁发一套 Server 端证书，为 MySQL Client 颁发一套 Client 端证书。并创建两个 Secret 对象，Secret 名字分别为：`${cluster_name}-tidb-server-secret` 和  `${cluster_name}-tidb-client-secret`，分别包含前面创建的两套证书；
2. 部署集群，设置 `.spec.tidb.tlsClient.enabled` 属性为 `true`；
3. 配置 MySQL 客户端使用加密连接。

其中，颁发证书的方式有多种，本文档提供两种方式，用户也可以根据需要为 TiDB 集群颁发证书，这两种方式分别为：

- 使用 `cfssl` 系统颁发证书；
- 使用 `cert-manager` 系统颁发证书；

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
          "*.${cluster_name}-tidb.${namespace}.svc"
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
    kubectl create secret generic ${cluster_name}-tidb-server-secret --namespace=${namespace} --from-file=tls.crt=~/cfssl/server.pem --from-file=tls.key=~/cfssl/server-key.pem --from-file=ca.crt=~/cfssl/ca.pem
    kubectl create secret generic ${cluster_name}-tidb-client-secret --namespace=${namespace} --from-file=tls.crt=~/cfssl/client.pem --from-file=tls.key=~/cfssl/client-key.pem --from-file=ca.crt=~/cfssl/ca.pem
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
    mkdir -p ~/cert-manager
    cd ~/cert-manager
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
    kubectl apply -f ~/cert-manager/tidb-server-issuer.yaml
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
    - `ipAddresses` 需要填写这两个 IP ，根据需要可以填写其他 IP：
      - `127.0.0.1`
      - `::1`
    - `issuerRef` 请填写上面创建的 Issuer；
    - 其他属性请参考 [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)。

    通过执行下面的命令来创建证书：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f ~/cert-manager/tidb-server-cert.yaml
    ```

    创建这个对象以后，cert-manager 会生成一个名字为 `${cluster_name}-tidb-server-secret` 的 Secret 对象供 TiDB Server 使用。

4. 创建 Client 端证书。

    创建一个 `tidb-client-cert.yaml` 文件，并输入以下内容：

    {{< copyable "shell-regular" >}}

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
    kubectl apply -f ~/cert-manager/tidb-client-cert.yaml
    ```

    创建这个对象以后，cert-manager 会生成一个名字为 `${cluster_name}-tidb-client-secret` 的 Secret 对象供 TiDB Client 使用。

用户可以生成多套 Client 端证书，并且至少要生成一套 Client 证书供 TiDB Operator 内部组件访问 TiDB Server（目前有 TidbInitializer 会访问 TiDB Server 来设置密码或者一些初始化操作）。

> **注意：**
>
> TiDB Server 的 TLS 兼容 MySQL 协议。当证书内容发生改变后，需要管理员手动执行 SQL 语句 `alter instance reload tls` 进行刷新。

## 第二步：部署 TiDB 集群

接下来将会通过两个 CR 对象来创建一个 TiDB 集群，并且执行以下步骤：

- 开启 MySQL 客户端 TLS；
- 对集群进行初始化（这里创建了一个数据库 `app`）。

``` yaml
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
 name: ${cluster_name}
 namespace: ${namespace}
spec:
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
```

其中 `${cluster_name}` 为集群的名字，`${namespace}` 为 TiDB 集群部署的命名空间。通过设置 `spec.tidb.tlsClient.enabled` 属性为 `true` 来开启 MySQL 客户端 TLS。

将上面文件保存为 `cr.yaml`，然后使用 `kubectl apply -f cr.yaml` 来创建 TiDB 集群。

## 第三步：配置 MySQL 客户端使用加密连接

可以根据[官网文档](https://pingcap.com/docs-cn/stable/how-to/secure/enable-tls-clients/#配置-mysql-客户端使用加密连接)提示，使用上面创建的 Client 证书，通过下面的方法连接 TiDB 集群：

1. 通过 `cfssl` 颁发证书，连接 TiDB Server 的方法是：

    {{< copyable "shell-regular" >}}

    ``` shell
    mysql -uroot -p -P 4000 -h ${tidb_host} --ssl-cert=~/cfssl/client.pem --ssl-key=~/cfssl/client-key.pem --ssl-ca=~/cfssl/ca.pe
    ```

2. 通过 `cert-manager` 颁发证书，获取 Client 证书的方式并连接 TiDB Server 的方法是：

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl get secret -n ${namespace} ${cluster_name}-tidb-client-secret  -ojsonpath='{.data.tls\.crt}' | base64 --decode > ~/cert-manager/client-tls.crt
    kubectl get secret -n ${namespace} ${cluster_name}-tidb-client-secret  -ojsonpath='{.data.tls\.key}' | base64 --decode > ~/cert-manager/client-tls.key
    kubectl get secret -n ${namespace} ${cluster_name}-tidb-client-secret  -ojsonpath='{.data.ca\.crt}' | base64 --decode >  ~/cert-manager/client-ca.crt
    ```

    {{< copyable "shell-regular" >}}

    ``` shell
    mysql -uroot -p -P 4000 -h ${tidb_host} --ssl-cert=~/cert-manager/client-tls.crt --ssl-key=~/cert-manager/client-tls.key --ssl-ca=~/cert-manager/client-ca.crt
    ```

最后请参考 [官网文档](https://pingcap.com/docs-cn/v3.0/how-to/secure/enable-tls-clients/#检查当前连接是否是加密连接) 来验证是否正确开启了 TLS。
