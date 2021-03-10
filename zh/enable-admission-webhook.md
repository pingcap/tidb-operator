---
title: 开启 TiDB Operator 准入控制器
summary: 介绍如何开启 TiDB Operator 准入控制器以及它的作用。
aliases: ['/docs-cn/tidb-in-kubernetes/dev/enable-admission-webhook/']
---

# TiDB Operator 准入控制器

Kubernetes 在 1.9 版本引入了 [动态准入机制](https://kubernetes.io/zh/docs/reference/access-authn-authz/extensible-admission-controllers/)，从而使得拥有对 Kubernetes 中的各类资源进行修改与验证的功能。 在 TiDB Operator 中，我们也同样使用了动态准入机制来帮助我们进行相关资源的修改、验证与运维。

## 先置条件

TiDB Operator 准入控制器与大部分 Kubernetes 平台上产品的准入控制器较为不同，TiDB Operator 通过[扩展 API-Server](https://kubernetes.io/docs/tasks/access-kubernetes-api/setup-extension-api-server/) 与 [WebhookConfiguration](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#configure-admission-webhooks-on-the-fly) 的两个机制组合而成。所以需要 Kubernetes 集群启用聚合层功能，通常情况下这个功能已经默认开启。如需查看是否开启聚合层功能，请参考[启用 Kubernetes Apiserver 标志](https://kubernetes.io/zh/docs/tasks/extend-kubernetes/configure-aggregation-layer/#启用-kubernetes-apiserver-标志)。

## 开启 TiDB Operator 准入控制器

TiDB Operator 在默认安装情况下不会开启准入控制器，你需要手动开启:

1. 修改 Operator 的 `values.yaml`

    开启 Operator Webhook 特性:

    ```yaml
    admissionWebhook:
      create: true
    ```

    默认情况下，如果你的 Kubernetes 集群版本大于等于 v1.13.0，你可以通过上述配置直接开启 Webhook 功能。

    如果你的 Kubernetes 集群版本小于 v1.13.0，你需要执行以下命令，将得到的返回值配置在 `values.yaml` 中的 `admissionWebhook.cabundle`：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n'
    ```

    ```yaml
    admissionWebhook:
      # 将上述命令的返回值填写到 admissionWebhook.cabundle 中
      cabundle: ${cabundle}
    ```

2. 配置失败策略

    在 Kubernetes 1.15 版本之前，动态准入机制的管理机制的粒度较粗并且并不方便去使用。所以为了防止 TiDB Operator 的动态准入机制影响全局集群，我们需要配置[失败策略](https://kubernetes.io/zh/docs/reference/access-authn-authz/extensible-admission-controllers/#failure-policy)。

    对于 Kubernetes 1.15 以下的版本，我们推荐将 TiDB Operator 失败策略配置为 `Ignore`，从而防止 TiDB Operator 的 admission webhook 出现异常时影响整个集群。

    ```yaml
    ......
    failurePolicy:
        validation: Ignore
        mutation: Ignore
    ```

    对于 Kubernetes 1.15 及以上的版本，我们推荐给 TiDB Operator 失败策略配置为 `Failure`，由于 Kubernetes 1.15 及以上的版本中，动态准入机制已经有了基于 Label 的筛选机制，所以不会由于 TiDB Operator 的 admission webhook 出现异常而影响整个集群。

    ```yaml
    ......
    failurePolicy:
        validation: Fail
        mutation: Fail
    ```

3. 安装/更新 TiDB Operator

    修改完 `values.yaml` 文件中的上述配置项以后，进行 TiDB Operator 部署或者更新。安装与更新 TiDB Operator 请参考[在 Kubernetes 上部署 TiDB Operator](deploy-tidb-operator.md)。

## 为 TiDB Operator 准入控制器设置 TLS 证书

在默认情况下，TiDB Operator 准入控制器与 Kubernetes api-server 之间跳过了 [TLS 验证环节](https://kubernetes.io/docs/tasks/access-kubernetes-api/configure-aggregation-layer/#contacting-the-extension-apiserver)，你可以通过以下步骤手动开启并配置 TiDB Operator 准入控制器与 Kubernetes api-server 之间的 TLS 验证。

1. 生成自定义证书

    参考[使用 `cfssl` 系统颁发证书](enable-tls-between-components.md#使用-cfssl-系统颁发证书)的第一步至第四步，生成自定义 CA 文件。对于 `ca-config.json`，我们使用如下配置:

    ```json
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
                }
            }
        }
    }
    ```

    当执行至第四步以后，通过 `ls` 命令执行，`cfssl` 文件夹下应该有以下文件:

    ```bash
    ca-config.json    ca-csr.json    ca-key.pem    ca.csr    ca.pem
    ```

2. 生成 TiDB Operator 准入控制器证书

    首先生成默认的 webhook-server.json 文件:

    {{< copyable "shell-regular" >}}

    ```shell
    cfssl print-defaults csr > webhook-server.json
    ```

    然后将 `webhook-server.json` 文件的内容修改如下:

    ```json
    {
        "CN": "TiDB Operator Webhook",
        "hosts": [
            "tidb-admission-webhook.${namespace}",
            "tidb-admission-webhook.${namespace}.svc",
            "tidb-admission-webhook.${namespace}.svc.cluster",
            "tidb-admission-webhook.${namespace}.svc.cluster.local"
        ],
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

    其中 `${namespace}` 为 TiDB Operator 部署的命名空间。

    然后生成 TiDB Operator Webhook Server 端证书:

    {{< copyable "shell-regular" >}}

    ```shell
    cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=server webhook-server.json | cfssljson -bare webhook-server
    ```

    执行完上述命令后，通过 `ls | grep webhook-server` 命令应该能查询到以下文件:

    ```bash
    webhook-server-key.pem
    webhook-server.csr
    webhook-server.json
    webhook-server.pem
    ```

3. 在 Kubernetes 集群中创建 Secret

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic ${secret_name} --namespace=${namespace} --from-file=tls.crt=~/cfssl/webhook-server.pem --from-file=tls.key=~/cfssl/webhook-server-key.pem --from-file=ca.crt=~/cfssl/ca.pem
    ```

4. 修改 values.yaml 并安装或升级 TiDB Operator

    获取 `ca.crt` 的值：

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get secret ${secret_name} --namespace=${namespace} -o=jsonpath='{.data.ca\.crt}'
    ```

    将 `values.yaml` 中下述配置按说明来进行配置:

    ```yaml
    admissionWebhook:
      apiservice:
        insecureSkipTLSVerify: false # 开启 TLS 验证
        tlsSecret: "${secret_name}" # 将上文中所创建的 secret 的 name 填写在这里
        caBundle: "${caBundle}" # 将上文中 ca.crt 的值填入此处
    ```

    修改完 `values.yaml` 文件中上述配置项以后进行 TiDB Operator 部署或者更新。安装 TiDB Operator 请参考[在 Kubernetes 上部署 TiDB Operator](deploy-tidb-operator.md)，升级 TiDB Operator 请参考[升级 TiDB Operator](upgrade-tidb-operator.md)

## TiDB Operator 准入控制器功能

TiDB Operator 通过准入控制器的帮助实现了许多功能。我们将在这里介绍各个资源的准入控制器与其相对应的功能。

* Pod 验证准入控制器:

    Pod 准入控制器提供了对 PD/TiKV/TiDB 组件安全下线与安全上线的保证，通过 Pod 验证准入控制器，我们可以实现[重启 Kubernetes 上的 TiDB 集群](restart-a-tidb-cluster.md)。该组件在准入控制器开启的情况下默认开启。

    ```yaml
    admissionWebhook:
      validation:
        pods: true
    ```

* StatefulSet 验证准入控制器:

    StatefulSet 验证准入控制器帮助实现 TiDB 集群中 TiDB/TiKV 组件的灰度发布，该组件在准入控制器开启的情况下默认关闭。

    ```yaml
    admissionWebhook:
      validation:
        statefulSets: false
    ```

    通过 `tidb.pingcap.com/tikv-partition` 和 `tidb.pingcap.com/tidb-partition` 这两个 annotation, 你可以控制 TiDB 集群中 TiDB 与 TiKV 组件的灰度发布。你可以通过以下方式对 TiDB 集群的 TiKV 组件设置灰度发布，其中 `partition=2` 的效果等同于 [StatefulSet 分区](https://kubernetes.io/zh/docs/concepts/workloads/controllers/statefulset/#%E5%88%86%E5%8C%BA)

    {{< copyable "shell-regular" >}}

    ```shell
    $  kubectl annotate tidbcluster ${name} -n ${namespace} tidb.pingcap.com/tikv-partition=2
    tidbcluster.pingcap.com/${name} annotated
    ```

    执行以下命令取消灰度发布设置：

    {{< copyable "shell-regular" >}}

    ```shell
    $  kubectl annotate tidbcluster ${name} -n ${namespace} tidb.pingcap.com/tikv-partition-
    tidbcluster.pingcap.com/${name} annotated
    ```

    以上设置同样适用于 TiDB 组件。

* TiDB Operator 资源验证准入控制器:

    TiDB Operator 资源验证准入控制器帮助实现针对 `TidbCluster`、`TidbMonitor` 等 TiDB Operator 自定义资源的验证，该组件在准入控制器开启的情况下默认关闭。

    ```yaml
    admissionWebhook:
      validation:
        pingcapResources: false
    ```

    举个例子，对于 `TidbCluster` 资源，TiDB Operator 资源验证准入控制器将会检查其 `spec` 字段中的必要字段。如果在 `TidbCluster` 创建或者更新时发现检查不通过，比如同时没有定义 `spec.pd.image` 或者 `spec.pd.baseImage` 字段，TiDB Operator 资源验证准入控制器将会拒绝这个请求。

* Pod 修改准入控制器:

    Pod 修改准入控制器帮助我们在弹性伸缩场景下实现 TiKV 的热点调度功能，在[启用 TidbCluster 弹性伸缩](enable-tidb-cluster-auto-scaling.md)中需要开启该控制器。该组件在准入控制器开启的情况下默认开启。

    ```yaml
    admissionWebhook:
      mutation:
        pods: true
    ```

* TiDB Operator 资源修改准入控制器:

    TiDB Operator 资源修改准入控制器帮助我们实现 TiDB Operator 相关自定义资源的默认值填充工作，如 `TidbCluster`，`TidbMonitor` 等。该组件在准入控制器开启的情况下默认开启。

    ```yaml
    admissionWebhook:
      mutation:
        pingcapResources: true
    ```
