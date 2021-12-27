---
title: Enable TLS for DM
summary: Learn how to enable TLS for DM in Kubernetes.
---

# Enable TLS for DM

This document describes how to enable TLS between components of the DM cluster in Kubernetes and how to use DM to migrate data between MySQL/TiDB databases that enable TLS for the MySQL client.

## Enable TLS between DM components

Starting from v1.2, TiDB Operator supports enabling TLS between components of the DM cluster in Kubernetes.

To enable TLS between components of the DM cluster, perform the following steps:

1. Generate certificates for each component of the DM cluster to be created:
    - A set of server-side certificates for the DM-master/DM-worker component, saved as the Kubernetes Secret objects: `${cluster_name}-${component_name}-cluster-secret`
    - A set of shared client-side certificates for the various clients of each component, saved as the Kubernetes Secret objects: `${cluster_name}-dm-client-secret`.

    > **Note:**
    >
    > The Secret objects you created must follow the above naming convention. Otherwise, the deployment of the DM cluster will fail.

2. Deploy the cluster, and set `.spec.tlsCluster.enabled` to `true`.
3. Configure `dmctl` to connect to the cluster.

Certificates can be issued in multiple methods. This document describes two methods. You can choose either of them to issue certificates for the DM cluster:

- [Using the `cfssl` system](#using-cfssl)
- [Using the `cert-manager` system](#using-cert-manager)

If you need to renew the existing TLS certificate, refer to [Renew and Replace the TLS Certificate](renew-tls-certificate.md).

### Generate certificates for components of the DM cluster

This section describes how to issue certificates using two methods: `cfssl` and `cert-manager`.

#### Using `cfssl`

1. Download `cfssl` and initialize the certificate issuer:

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

2. Generate the `ca-config.json` configuration file:

    ```bash
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

3. Generate the `ca-csr.json` configuration file:

    ```bash
    cat << EOF > ca-csr.json
    {
        "CN": "TiDB",
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
    EOF
    ```

4. Generate CA by the configured option:

    {{< copyable "shell-regular" >}}

    ``` shell
    cfssl gencert -initca ca-csr.json | cfssljson -bare ca -
    ```

5. Generate the server-side certificates:

    In this step, a set of server-side certificate is created for each component of the DM cluster.

    - DM-master

        First, generate the default `dm-master-server.json` file:

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl print-defaults csr > dm-master-server.json
        ```

        Then, edit this file to change the `CN` and `hosts` attributes:

        ``` json
        ...
            "CN": "TiDB",
            "hosts": [
              "127.0.0.1",
              "::1",
              "${cluster_name}-dm-master",
              "${cluster_name}-dm-master.${namespace}",
              "${cluster_name}-dm-master.${namespace}.svc",
              "${cluster_name}-dm-master-peer",
              "${cluster_name}-dm-master-peer.${namespace}",
              "${cluster_name}-dm-master-peer.${namespace}.svc",
              "*.${cluster_name}-dm-master-peer",
              "*.${cluster_name}-dm-master-peer.${namespace}",
              "*.${cluster_name}-dm-master-peer.${namespace}.svc"
            ],
        ...
        ```

        `${cluster_name}` is the name of the DM cluster. `${namespace}` is the namespace in which the DM cluster is deployed. You can also add your customized `hosts`.

        Finally, generate the DM-master server-side certificate:

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal dm-master-server.json | cfssljson -bare dm-master-server
        ```

    - DM-worker

        First, generate the default `dm-worker-server.json` file:

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl print-defaults csr > dm-worker-server.json
        ```

        Then, edit this file to change the `CN` and `hosts` attributes:

        ``` json
        ...
            "CN": "TiDB",
            "hosts": [
              "127.0.0.1",
              "::1",
              "${cluster_name}-dm-worker",
              "${cluster_name}-dm-worker.${namespace}",
              "${cluster_name}-dm-worker.${namespace}.svc",
              "${cluster_name}-dm-worker-peer",
              "${cluster_name}-dm-worker-peer.${namespace}",
              "${cluster_name}-dm-worker-peer.${namespace}.svc",
              "*.${cluster_name}-dm-worker-peer",
              "*.${cluster_name}-dm-worker-peer.${namespace}",
              "*.${cluster_name}-dm-worker-peer.${namespace}.svc"
            ],
        ...
        ```

        `${cluster_name}` is the name of the cluster. `${namespace}` is the namespace in which the DM cluster is deployed. You can also add your customized `hosts`.

        Finally, generate the DM-worker server-side certificate:

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal dm-worker-server.json | cfssljson -bare dm-worker-server
        ```

6. Generate the client-side certificates:

    First, generate the default `client.json` file:

    {{< copyable "shell-regular" >}}

    ``` shell
    cfssl print-defaults csr > client.json
    ```

    Then, edit this file to change the `CN`, `hosts` attributes. You can leave the `hosts` empty:

    ``` json
    ...
        "CN": "TiDB",
        "hosts": [],
    ...
    ```

    Finally, generate the client-side certificate:

    {{< copyable "shell-regular" >}}

    ``` shell
    cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=client client.json | cfssljson -bare client
    ```

7. Create the Kubernetes Secret object:

    If you have already generated a set of certificates for each component and a set of client-side certificate for each client as described in the above steps, create the Secret objects for the DM cluster by executing the following command:

    * The DM-master cluster certificate Secret:

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl create secret generic ${cluster_name}-dm-master-cluster-secret --namespace=${namespace} --from-file=tls.crt=dm-master-server.pem --from-file=tls.key=dm-master-server-key.pem --from-file=ca.crt=ca.pem
        ```

    * The DM-worker cluster certificate Secret：

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl create secret generic ${cluster_name}-dm-worker-cluster-secret --namespace=${namespace} --from-file=tls.crt=dm-worker-server.pem --from-file=tls.key=dm-worker-server-key.pem --from-file=ca.crt=ca.pem
        ```

    * Client certificate Secret：

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl create secret generic ${cluster_name}-dm-client-secret --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
        ```

    You have created two Secret objects:

    - One Secret object for each DM-master/DM-worker server-side certificate to load when the server is started;
    - One Secret object for their clients to connect.

#### Using `cert-manager`

1. Install `cert-manager`.

    Refer to [cert-manager installation in Kubernetes](https://docs.cert-manager.io/en/release-0.11/getting-started/install/kubernetes.html) for details.

2. Create an Issuer to issue certificates to the DM cluster.

    To configure `cert-manager`, create the Issuer resources.

    First, create a directory which saves the files that `cert-manager` needs to create certificates:

    {{< copyable "shell-regular" >}}

    ``` shell
    mkdir -p cert-manager
    cd cert-manager
    ```

    Then, create a `dm-cluster-issuer.yaml` file with the following content:

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
      duration: 87600h # 10yrs
      renewBefore: 720h # 30d
      issuerRef:
        name: ${cluster_name}-selfsigned-ca-issuer
        kind: Issuer
    ---
    apiVersion: cert-manager.io/v1alpha2
    kind: Issuer
    metadata:
      name: ${cluster_name}-dm-issuer
      namespace: ${namespace}
    spec:
      ca:
        secretName: ${cluster_name}-ca-secret
    ```

    `${cluster_name}` is the name of the cluster. The above YAML file creates three objects:

    - An Issuer object of the SelfSigned type, used to generate the CA certificate needed by Issuer of the CA type;
    - A Certificate object, whose `isCa` is set to `true`.
    - An Issuer, used to issue TLS certificates between components of the DM cluster.

    Finally, execute the following command to create an Issuer:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f dm-cluster-issuer.yaml
    ```

3. Generate the server-side certificate.

    In `cert-manager`, the Certificate resource represents the certificate interface. This certificate is issued and updated by the Issuer created in Step 2.

    Each component needs a server-side certificate, and all components need a shared client-side certificate for their clients.

    - The DM-master server-side certificate

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-dm-master-cluster-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-dm-master-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "${cluster_name}-dm-master"
          - "${cluster_name}-dm-master.${namespace}"
          - "${cluster_name}-dm-master.${namespace}.svc"
          - "${cluster_name}-dm-master-peer"
          - "${cluster_name}-dm-master-peer.${namespace}"
          - "${cluster_name}-dm-master-peer.${namespace}.svc"
          - "*.${cluster_name}-dm-master-peer"
          - "*.${cluster_name}-dm-master-peer.${namespace}"
          - "*.${cluster_name}-dm-master-peer.${namespace}.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: ${cluster_name}-dm-issuer
            kind: Issuer
            group: cert-manager.io
        ```

        `${cluster_name}` is the name of the cluster. Configure the items as follows:

        - Set `spec.secretName` to `${cluster_name}-dm-master-cluster-secret`.
        - Add `server auth` and `client auth` in `usages`.
        - Add the following DNSs in `dnsNames`. You can also add other DNSs according to your needs:
            - "${cluster_name}-dm-master"
            - "${cluster_name}-dm-master.${namespace}"
            - "${cluster_name}-dm-master.${namespace}.svc"
            - "${cluster_name}-dm-master-peer"
            - "${cluster_name}-dm-master-peer.${namespace}"
            - "${cluster_name}-dm-master-peer.${namespace}.svc"
            - "*.${cluster_name}-dm-master-peer"
            - "*.${cluster_name}-dm-master-peer.${namespace}"
            - "*.${cluster_name}-dm-master-peer.${namespace}.svc"
        - Add the following two IPs in `ipAddresses`. You can also add other IPs according to your needs:
            - `127.0.0.1`
            - `::1`
        - Add the Issuer created above in `issuerRef`.
        - For other attributes, refer to [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec).

        After the object is created, `cert-manager` generates a `${cluster_name}-dm-master-cluster-secret` Secret object to be used by the DM-master component of the DM cluster.

    - The DM-worker server-side certificate

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-dm-worker-cluster-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-dm-worker-cluster-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB"
          usages:
            - server auth
            - client auth
          dnsNames:
          - "${cluster_name}-dm-worker"
          - "${cluster_name}-dm-worker.${namespace}"
          - "${cluster_name}-dm-worker.${namespace}.svc"
          - "${cluster_name}-dm-worker-peer"
          - "${cluster_name}-dm-worker-peer.${namespace}"
          - "${cluster_name}-dm-worker-peer.${namespace}.svc"
          - "*.${cluster_name}-dm-worker-peer"
          - "*.${cluster_name}-dm-worker-peer.${namespace}"
          - "*.${cluster_name}-dm-worker-peer.${namespace}.svc"
          ipAddresses:
          - 127.0.0.1
          - ::1
          issuerRef:
            name: ${cluster_name}-dm-issuer
            kind: Issuer
            group: cert-manager.io
        ```

        `${cluster_name}` is the name of the cluster. Configure the items as follows:

        - Set `spec.secretName` to `${cluster_name}-dm-worker-cluster-secret`.
        - Add `server auth` and `client auth` in `usages`.
        - Add the following DNSs in `dnsNames`. You can also add other DNSs according to your needs:
            - "${cluster_name}-dm-worker"
            - "${cluster_name}-dm-worker.${namespace}"
            - "${cluster_name}-dm-worker.${namespace}.svc"
            - "${cluster_name}-dm-worker-peer"
            - "${cluster_name}-dm-worker-peer.${namespace}"
            - "${cluster_name}-dm-worker-peer.${namespace}.svc"
            - "*.${cluster_name}-dm-worker-peer"
            - "*.${cluster_name}-dm-worker-peer.${namespace}"
            - "*.${cluster_name}-dm-worker-peer.${namespace}.svc"
        - Add the following two IPs in `ipAddresses`. You can also add other IPs according to your needs:
            - `127.0.0.1`
            - `::1`
        - Add the Issuer created above in `issuerRef`.
        - For other attributes, refer to [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec).

        After the object is created, `cert-manager` generates a `${cluster_name}-dm-cluster-secret` Secret object to be used by the DM-worker component of the DM cluster.

    - A set of client-side certificates of DM cluster components.

        ``` yaml
        apiVersion: cert-manager.io/v1alpha2
        kind: Certificate
        metadata:
          name: ${cluster_name}-dm-client-secret
          namespace: ${namespace}
        spec:
          secretName: ${cluster_name}-dm-client-secret
          duration: 8760h # 365d
          renewBefore: 360h # 15d
          organization:
          - PingCAP
          commonName: "TiDB"
          usages:
            - client auth
          issuerRef:
            name: ${cluster_name}-dm-issuer
            kind: Issuer
            group: cert-manager.io
        ```

        `${cluster_name}` is the name of the cluster. The above YAML file creates three objects:

        - Set `spec.secretName` to `${cluster_name}-dm-master-cluster-secret`.
        - Add `server auth` and `client auth` in `usages`.
        - `dnsNames` and `ipAddresses` are not required.
        - Add the Issuer created above in the `issuerRef`
        - For other attributes, refer to [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)

        After the object is created, `cert-manager` generates a `${cluster_name}-cluster-client-secret` Secret object to be used by the clients of the DM components.

### Deploy the DM cluster

When you deploy a DM cluster, you can enable TLS between DM components, and set the `cert-allowed-cn` configuration item to verify the CN (Common Name) of each component's certificate.

> **Note:**
>
> Currently, you can set only one value for the `cert-allowed-cn` configuration item of DM-master. Therefore, the `commonName` of all `Certificate` objects must be the same.

- Create the `dm-cluster.yaml` file:

```yaml
apiVersion: pingcap.com/v1alpha1
kind: DMCluster
metadata:
  name: ${cluster_name}
  namespace: ${namespace}
spec:
  tlsCluster:
    enabled: true
  version: v2.0.7
  pvReclaimPolicy: Retain
  discovery: {}
  master:
    baseImage: pingcap/dm
    maxFailoverCount: 0
    replicas: 1
    storageSize: "1Gi"
    config:
      cert-allowed-cn:
        - TiDB
  worker:
    baseImage: pingcap/dm
    maxFailoverCount: 0
    replicas: 1
    storageSize: "1Gi"
    config:
      cert-allowed-cn:
        - TiDB
```

Use the `kubectl apply -f dm-cluster.yaml` file to create a DM cluster.

### Configure `dmctl` and connect to the cluster

Get into the DM-master Pod:

{{< copyable "shell-regular" >}}

``` shell
kubectl exec -it ${cluster_name}-dm-master-0 -n ${namespace} sh
```

Use `dmctl`:

{{< copyable "shell-regular" >}}

``` shell
cd /var/lib/dm-master-tls
/dmctl --ssl-ca=ca.crt --ssl-cert=tls.crt --ssl-key=tls.key --master-addr 127.0.0.1:8261 list-member
```

## Use DM to migrate data between MySQL/TiDB databases that enable TLS for the MySQL client

This section describes how to configure DM to migrate data between MySQL/TiDB databases that enable TLS for the MySQL client.

To learn how to enable TLS for the MySQL client of TiDB, refer to [Enable TLS for the MySQL Client](enable-tls-for-mysql-client.md).

### Step 1: Create the Kubernetes Secret object for each TLS-enabled MySQL

Suppose you have deployed a MySQL/TiDB database with TLS-enabled for the MySQL client. To create Secret objects for the TiDB cluster, execute the following command:

{{< copyable "shell-regular" >}}

```bash
kubectl create secret generic ${mysql_secret_name1} --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
kubectl create secret generic ${tidb_secret_name} --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
```

### Step 2: Mount the Secret objects to the DM cluster

After creating the Kubernetes Secret objects for the upstream and downstream databases, you need to set `spec.tlsClientSecretNames` so that you can mount the Secret objects to the Pod of DM-master/DM-worker.

```yaml
apiVersion: pingcap.com/v1alpha1
kind: DMCluster
metadata:
  name: ${cluster_name}
  namespace: ${namespace}
spec:
  version: v2.0.7
  pvReclaimPolicy: Retain
  discovery: {}
  tlsClientSecretNames:
    - ${mysql_secret_name1}
    - ${tidb_secret_name}
  master:
    ...
```

### Step 3: Modify the data source and migration task configuration

After configuring `spec.tlsClientSecretNames`, TiDB Operator will mount the Secret objects `${secret_name}` to the path `/var/lib/source-tls/${secret_name}`.

1. Configure `from.security` in the `source1.yaml` file as described in the [data source configuration](use-tidb-dm.md#create-data-source):

    ``` yaml
    source-id: mysql-replica-01
    relay-dir: /var/lib/dm-worker/relay
    from:
      host: ${mysql_host1}
      user: dm
      password: ""
      port: 3306
      security:
        ssl-ca: /var/lib/source-tls/${mysql_secret_name1}/ca.crt
        ssl-cert: /var/lib/source-tls/${mysql_secret_name1}/tls.crt
        ssl-key: /var/lib/source-tls/${mysql_secret_name1}/tls.key
    ```

2. Configure `target-database.security` in the `task.yaml` file as described in the [Configure Migration Tasks](use-tidb-dm.md#configure-migration-tasks):

    ``` yaml
    name: test
    task-mode: all
    is-sharding: false

    target-database:
      host: ${tidb_host}
      port: 4000
      user: "root"
      password: ""
      security:
        ssl-ca: /var/lib/source-tls/${tidb_secret_name}/ca.crt
        ssl-cert: /var/lib/source-tls/${tidb_secret_name}/tls.crt
        ssl-key: /var/lib/source-tls/${tidb_secret_name}/tls.key
    
    mysql-instances:
    - source-id: "replica-01"
      loader-config-name: "global"
      
    loaders:
      global:
        dir: "/var/lib/dm-worker/dumped_data"    
    ```

### Step 4: Start the migration tasks

Refer to [Start the migration tasks](use-tidb-dm.md#startcheckstop-the-migration-tasks).
