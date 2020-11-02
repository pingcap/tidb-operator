---
title: Enable Admission Controller in TiDB Operator
summary: Learn how to enable the admission controller in TiDB Operator and the functionality of the admission controller.
aliases: ['/docs/tidb-in-kubernetes/dev/enable-admission-webhook/']
---

# Enable Admission Controller in TiDB Operator

Kubernetes v1.9 introduces the [dynamic admission control](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) to modify and validate resources. TiDB Operator also supports the dynamic admission control to modify, validate, and maintain resources. This document describes how to enable the admission controller and introduces the functionality of the admission controller.

## Prerequisites

Unlike those of most products on Kubernetes, the admission controller of TiDB Operator consists of two mechanisms: [extension API-server](https://kubernetes.io/docs/tasks/access-kubernetes-api/setup-extension-api-server/) and [Webhook Configuration](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#configure-admission-webhooks-on-the-fly).

To use the admission controller, you need to enable the aggregation layer feature of the Kubernetes cluster. The feature is enabled by default. To check whether it is enabled, see [Enable Kubernetes Apiserver flags](https://kubernetes.io/docs/tasks/extend-kubernetes/configure-aggregation-layer/#enable-kubernetes-apiserver-flags).

## Enable the admission controller

With a default installation, TiDB Operator disables the admission controller. Take the following steps to manually turn it on.

1. Edit the `values.yaml` file in TiDB Operator.

    Enable the `Operator Webhook` feature:

    ```yaml
    admissionWebhook:
      create: true
    ```

    * If your Kubernetes cluster version >= v1.13.0, enable the Webhook feature by using the configuration above.

    * If your Kubernetes cluster version < v1.13.0, run the following command and configure the `admissionWebhook.cabundle` in `values.yaml` as the return value:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n'
        ```

        ```yaml
        admissionWebhook:
          # Configure the value of `admissionWebhook.cabundle` as the return value of the command above
          cabundle: <cabundle>
        ```

2. Configure the failure policy.

    Prior to Kubernetes v1.15, the management mechanism of the dynamic admission control is coarser-grained and is inconvenient to use. To prevent the impact of the dynamic admission control on the global cluster, you need to configure the [Failure Policy](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#failure-policy).

    * For Kubernetes versions earlier than v1.15, it is recommended to set the `failurePolicy` of TiDB Operator to `Ignore`. This avoids the influence on the global cluster in case of `admission webhook` exception in TiDB Operator.

        ```yaml
        ......
        failurePolicy:
            validation: Ignore
            mutation: Ignore
        ```

    * For Kubernetes v1.15 and later versions, it is recommended to set the `failurePolicy` of TiDB Operator to `Failure`. The exception occurs in `admission webhook` does not affect the whole cluster, because the dynamic admission control supports the label-based filtering mechanism.

        ```yaml
        ......
        failurePolicy:
            validation: Fail
            mutation: Fail
        ```

3. Install or update TiDB Operator.

    To install or update TiDB Operator, see [Deploy TiDB Operator in Kubernetes](deploy-tidb-operator.md).

## Set the TLS certificate for the admission controller

By default, the admission controller and Kubernetes api-server skip the [TLS verification](https://kubernetes.io/docs/tasks/access-kubernetes-api/configure-aggregation-layer/#contacting-the-extension-apiserver). To manually enable and configure the TLS verification between the admission controller and Kubernetes api-server, take the following steps:

1. Generate the custom certificate.

    To generate the custom CA (client auth) file, refer to Step 1 to Step 4 in [Generate certificates using `cfssl`](enable-tls-between-components.md#using-cfssl).

    Use the following configuration in `ca-config.json`:

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

    After executing Step 4, run the `ls` command. The following files should be listed in the `cfssl` folder:

    ```bash
    ca-config.json    ca-csr.json    ca-key.pem    ca.csr    ca.pem
    ```

2. Generate the certificate for the admission controller.

    1. Create the default `webhook-server.json` file:

        {{< copyable "shell-regular" >}}

        ```shell
        cfssl print-defaults csr > webhook-server.json
        ```

    2. Modify the `webhook-server.json` file as follows:

        ```json
        {
            "CN": "TiDB Operator Webhook",
            "hosts": [
                "tidb-admission-webhook.<namespace>",
                "tidb-admission-webhook.<namespace>.svc",
                "tidb-admission-webhook.<namespace>.svc.cluster",
                "tidb-admission-webhook.<namespace>.svc.cluster.local"
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

        `<namespace>` is the namespace which TiDB Operator is deployed in.

    3. Generate the server-side certificate for TiDB Operator Webhook:

        {{< copyable "shell-regular" >}}

        ```shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=server webhook-server.json | cfssljson -bare webhook-server
        ```

    4. Run the `ls | grep webhook-server` command. The following files should be listed:

        ```bash
        webhook-server-key.pem
        webhook-server.csr
        webhook-server.json
        webhook-server.pem
        ```

3. Create a secret in the Kubernetes cluster:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic <secret-name> --namespace=<namespace> --from-file=tls.crt=~/cfssl/webhook-server.pem --from-file=tls.key=~/cfssl/webhook-server-key.pem --from-file=ca.crt=~/cfssl/ca.pem
    ```

4. Modify `values.yaml`, and install or upgrade TiDB Operator.

    Get the value of `ca.crt`:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get secret <secret-name> --namespace=<release-namespace> -o=jsonpath='{.data.ca\.crt}'
    ```

    Configure the items in `values.yaml` as described below:

    ```yaml
    admissionWebhook:
      apiservice:
        insecureSkipTLSVerify: false # Enable TLS verification
        tlsSecret: "<secret-name>" # The name of the secret created in Step 3
        caBundle: "<caBundle>" # The value of `ca.crt` obtained in the above step
    ```

    After configuring the items, install or upgrade TiDB Operator. For installation, see [Deploy TiDB Operator](deploy-tidb-operator.md). For upgrade, see [Upgrade TiDB Operator](upgrade-tidb-operator.md).

## Functionality of the admission controller

TiDB Operator implements many functions using the admission controller. This section introduces the admission controller for each resource and its corresponding functions.

* Admission controller for Pod validation

    The admission controller for Pod validation guarantees the safe logon and safe logoff of the PD/TiKV/TiDB component. You can [restart a TiDB cluster in Kubernetes](restart-a-tidb-cluster.md) using this controller. The component is enabled by default if the admission controller is enabled.

    ```yaml
    admissionWebhook:
      validation:
        pods: true
    ```

* Admission controller for StatefulSet validation

    The admission controller for StatefulSet validation supports the gated launch of the TiDB/TiKV component in a TiDB cluster. The component is disabled by default if the admission controller is enabled.

    ```yaml
    admissionWebhook:
      validation:
        statefulSets: false
    ```

    You can control the gated launch of the TiDB/TiKV component in a TiDB cluster through two annotations, `tidb.pingcap.com/tikv-partition` and `tidb.pingcap.com/tidb-partition`. To set the gated launch of the TiKV component in a TiDB cluster, execute the following commands. The effect of `partition=2` is the same as that of [StatefulSet Partitions](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#partitions).

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl annotate tidbcluster ${name} -n ${namespace} tidb.pingcap.com/tikv-partition=2 &&
    tidbcluster.pingcap.com/${name} annotated
    ```

    Execute the following commands to unset the gated launch:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl annotate tidbcluster ${name} -n ${namespace} tidb.pingcap.com/tikv-partition- &&
    tidbcluster.pingcap.com/${name} annotated
    ```

    This also applies to the TiDB component.

* Admission controller for TiDB Operator resources validation

    The admission controller for TiDB Operator resources validation supports validating customized resources such as `TidbCluster` and `TidbMonitor` in TiDB Operator. The component is disabled by default if the admission controller is enabled.

    ```yaml
    admissionWebhook:
      validation:
        pingcapResources: false
    ```

    For example, regarding `TidbCluster` resources, the admission controller for TiDB Operator resources validation checks the required fields of the `spec` field. When you create or update `TidbCluster`, if the check is not passed (for example, neither of the `spec.pd.image` filed and the `spec.pd.baseImage` field is defined), this admission controller refuses the request.

* Admission controller for Pod modification

    The admission controller for Pod modification supports the hotspot scheduling of TiKV in the auto-scaling scenario. To [enable TidbCluster auto-scaling](enable-tidb-cluster-auto-scaling.md), you need to enable this controller. The component is enabled by default if the admission controller is enabled.

    ```yaml
    admissionWebhook:
      mutation:
        pods: true
    ```

* Admission controller for TiDB Operator resources modification

    The admission controller for TiDB Operator resources modification supports filling in the default values of customized resources, such as `TidbCluster` and `TidbMonitor` in TiDB Operator. The component is enabled by default if the admission controller is enabled.

    ```yaml
    admissionWebhook:
      mutation:
        pingcapResources: true
    ```
