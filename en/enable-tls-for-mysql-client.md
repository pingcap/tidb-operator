---
title: Enable TLS for the MySQL Client
summary: Learn how to enable TLS for MySQL client of the TiDB cluster on Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/enable-tls-for-mysql-client/']
---

# Enable TLS for the MySQL Client

This document describes how to enable TLS for MySQL client of the TiDB cluster on Kubernetes. Starting from TiDB Operator v1.1, TLS for the MySQL client of the TiDB cluster on Kubernetes is supported.

To enable TLS for the MySQL client, perform the following steps:

1. [Issue two sets of certificates](#issue-two-sets-of-certificates-for-the-tidb-cluster): a set of server-side certificates for TiDB server, and a set of client-side certificates for MySQL client. Create two Secret objects, `${cluster_name}-tidb-server-secret` and `${cluster_name}-tidb-client-secret`, respectively including these two sets of certificates.

    > **Note:**
    >
    > The Secret objects you created must follow the above naming convention. Otherwise, the deployment of the TiDB cluster will fail.

    Certificates can be issued in multiple methods. This document describes two methods. You can choose either of them to issue certificates for the TiDB cluster:

    - [Using the `cfssl` system](#using-cfssl)
    - [Using the `cert-manager` system](#using-cert-manager)

2. [Deploy the cluster](#deploy-the-tidb-cluster), and set `.spec.tidb.tlsClient.enabled` to `true`.

3. [Configure the MySQL client to use an encrypted connection](#configure-the-mysql-client-to-use-an-encrypted-connection).

If you need to renew the existing TLS certificate, refer to [Renew and Replace the TLS Certificate](renew-tls-certificate.md).

## Issue two sets of certificates for the TiDB cluster

This section describes how to issue certificates for the TiDB cluster using two methods: `cfssl` and `cert-manager`.

### Using `cfssl`

1. Download `cfssl` and initialize the certificate issuer:

    {{< copyable "shell-regular" >}}

    ```shell
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

2. Configure the client auth (CA) option in `ca-config.json`:

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

3. Change the certificate signing request (CSR) of `ca-csr.json`:

    ```json
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

4. Generate CA by the configured option:

    {{< copyable "shell-regular" >}}

    ```shell
    cfssl gencert -initca ca-csr.json | cfssljson -bare ca -
    ```

5. Generate the server-side certificate:

    First, create the default `server.json` file:

    {{< copyable "shell-regular" >}}

    ``` shell
    cfssl print-defaults csr > server.json
    ```

    Then, edit this file to change the `CN`, `hosts` attributes:

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

    `${cluster_name}` is the name of the cluster. `${namespace}` is the namespace in which the TiDB cluster is deployed. You can also add your customized `hosts`.

    Finally, generate the server-side certificate:

    {{< copyable "shell-regular" >}}

    ``` shell
    cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=server server.json | cfssljson -bare server
    ```

6. Generate the client-side certificate:

    First, create the default `client.json` file:

    {{< copyable "shell-regular" >}}

    ``` shell
    cfssl print-defaults csr > client.json
    ```

    Then, edit this file to change the `CN`, `hosts` attributes. You can leave the `hosts` empty:

    ``` json
    ...
        "CN": "TiDB Client",
        "hosts": [],
    ...
    ```

    Finally, generate the client-side certificate:

    {{< copyable "shell-regular" >}}

    ``` shell
    cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=client client.json | cfssljson -bare client
    ```

7. Create the Kubernetes Secret object.

    If you have already generated two sets of certificates as described in the above steps, create the Secret object for the TiDB cluster by the following command:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic ${cluster_name}-tidb-server-secret --namespace=${namespace} --from-file=tls.crt=server.pem --from-file=tls.key=server-key.pem --from-file=ca.crt=ca.pem
    kubectl create secret generic ${cluster_name}-tidb-client-secret --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
    ```

    You have created two Secret objects for the server-side and client-side certificates:

    - The TiDB server loads one Secret object when it starts
    - The MySQL client uses another Secret object when it connects to the TiDB cluster

You can generate multiple sets of client-side certificates. At least one set of client-side certificates is needed for the internal components of TiDB Operator to access the TiDB server. Currently, `TidbInitializer` accesses the TiDB server to set the password or perform initialization.

### Using `cert-manager`

1. Install `cert-manager`.

    Refer to [cert-manager installation in Kubernetes](https://docs.cert-manager.io/en/release-0.11/getting-started/install/kubernetes.html).

2. Create an Issuer to issue certificates for the TiDB cluster.

    To configure `cert-manager`, create the Issuer resources.

    First, create a directory which saves the files that `cert-manager` needs to create certificates:

    {{< copyable "shell-regular" >}}

    ``` shell
    mkdir -p cert-manager
    cd cert-manager
    ```

    Then, create a `tidb-server-issuer.yaml` file with the following content:

    ```yaml
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

    This `.yaml` file creates three objects:

    - An Issuer object of SelfSigned class, used to generate the CA certificate needed by Issuer of CA class
    - A Certificate object, whose `isCa` is set to `true`
    - An Issuer, used to issue TLS certificates for the TiDB server

    Finally, execute the following command to create an Issuer:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f tidb-server-issuer.yaml
    ```

3. Generate the server-side certificate.

    In `cert-manager`, the Certificate resource represents the certificate interface. This certificate is issued and updated by the Issuer created in Step 2.

    First, create a `tidb-server-cert.yaml` file with the following content:

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

    `${cluster_name}` is the name of the cluster. Configure the items as follows:

    - Set `spec.secretName` to `${cluster_name}-tidb-server-secret`
    - Add `server auth` in `usages`
    - Add the following 6 DNSs in `dnsNames`. You can also add other DNSs according to your needs:
        - `${cluster_name}-tidb`
        - `${cluster_name}-tidb.${namespace}`
        - `${cluster_name}-tidb.${namespace}.svc`
        - `*.${cluster_name}-tidb`
        - `*.${cluster_name}-tidb.${namespace}`
        - `*.${cluster_name}-tidb.${namespace}.svc`
        - `*.${cluster_name}-tidb-peer`
        - `*.${cluster_name}-tidb-peer.${namespace}`
        - `*.${cluster_name}-tidb-peer.${namespace}.svc`
    - Add the following 2 IPs in `ipAddresses`. You can also add other IPs according to your needs:
        - `127.0.0.1`
        - `::1`
    - Add the Issuer created above in the `issuerRef`
    - For other attributes, refer to [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec).

    Execute the following command to generate the certificate:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f tidb-server-cert.yaml
    ```

    After the object is created, cert-manager generates a `${cluster_name}-tidb-server-secret` Secret object to be used by the TiDB server.

4. Generate the client-side certificate:

    Create a `tidb-client-cert.yaml` file with the following content:

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

    `${cluster_name}` is the name of the cluster. Configure the items as follows:

    - Set `spec.secretName` to `${cluster_name}-tidb-client-secret`
    - Add `client auth` in `usages`
    - `dnsNames` and `ipAddresses` are not required
    - Add the Issuer created above in the `issuerRef`
    - For other attributes, refer to [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec)

    Execute the following command to generate the certificate:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f tidb-client-cert.yaml
    ```

    After the object is created, cert-manager generates a `${cluster_name}-tidb-client-secret` Secret object to be used by the TiDB client.

5. Create multiple sets of client-side certificates (optional).

    Four components in the TiDB Operator cluster need to request the TiDB server. When TLS is enabled, these components can use certificates to request the TiDB server, each with a separate certificate. The four components are listed as follows:

    - TidbInitializer
    - PD Dashboard
    - Backup
    - Restore

    If you need to [restore data using TiDB Lightning](restore-data-using-tidb-lightning.md), you need to generate a server-side certificate for the TiDB Lightning component.

    To create certificates for these components, take the following steps:

    1. Create a `tidb-components-client-cert.yaml` file with the following content:

        ```yaml
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

        In the `.yaml` file above, `${cluster_name}` is the name of the cluster. Configure the items as follows:

        - Set the value of `spec.secretName` to `${cluster_name}-${component}-client-secret`.
        - Add `client auth` in `usages`.
        - `dnsNames` and `ipAddresses` are not required.
        - Add the Issuer created above in the `issuerRef`.
        - For other attributes, refer to [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec).

        To generate a client-side certificate for TiDB Lightning, use the following content and set `tlsCluster.tlsClientSecretName` to `${cluster_name}-lightning-client-secret` in TiDB Lightning's `values.yaml` file.

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

    2. Create the certificate by running the following command:

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl apply -f tidb-components-client-cert.yaml
        ```

    3. After creating these objects, cert-manager will generate four secret objects for the four components.

    > **Note:**
    >
    > TiDB server's TLS is compatible with the MySQL protocol. When the certificate content is changed, the administrator needs to manually execute the SQL statement `alter instance reload tls` to refresh the content.

## Deploy the TiDB cluster

In this step, you create a TiDB cluster and perform the following operations:

- Enable TLS for the MySQL client
- Initialize the cluster (an `app` database is created for demonstration)
- Create a Backup object to back up the cluster
- Create a Restore object to restore the cluster
- Use separate client-side certificates for `TidbInitializer`, PD Dashboard, Backup, and Restore (specified by `tlsClientSecretName`)

1. Create three `.yaml` files:

    - `tidb-cluster.yaml` file:

        ```yaml
        apiVersion: pingcap.com/v1alpha1
        kind: TidbCluster
        metadata:
          name: ${cluster_name}
          amespace: ${namespace}
        spec:
          version: v5.2.0
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

    In the above file, `${cluster_name}` is the name of the cluster, and `${namespace}` is the namespace in which the TiDB cluster is deployed. To enable TLS for the MySQL client, set `spec.tidb.tlsClient.enabled` to `true`.

2. Deploy the TiDB cluster:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f tidb-cluster.yaml
    ```

3. Back up the cluster:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f backup.yaml
    ```

4. Restore the cluster:

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl apply -f restore.yaml
    ```

## Configure the MySQL client to use an encrypted connection

To connect the MySQL client with the TiDB cluster, use the client-side certificate created above and take the following methods. For details, refer to [Configure the MySQL client to use encrypted connections](https://pingcap.com/docs/stable/how-to/secure/enable-tls-clients/#configure-the-mysql-client-to-use-encrypted-connections).

Execute the following command to acquire the client-side certificate and connect to the TiDB server:

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

> **Note:**
>
> [The default authentication plugin of MySQL 8.0](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_default_authentication_plugin) is updated from `mysql_native_password` to `caching_sha2_password`. Therefore, if you use MySQL client from MySQL 8.0 to access the TiDB service (TiDB version < v4.0.7), and if the user account has a password, you need to explicitly specify the `--default-auth=mysql_native_password` parameter.

Finally, to verify whether TLS is successfully enabled, refer to [checking the current connection](https://pingcap.com/docs/stable/enable-tls-between-clients-and-servers/#check-whether-the-current-connection-uses-encryption).
