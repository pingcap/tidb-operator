---
title: Enable TLS between TiDB Components
summary: Learn how to enable TLS between TiDB components on Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/enable-tls-between-components/']
---

# Enable TLS between TiDB Components

This document describes how to enable Transport Layer Security (TLS) between components of the TiDB cluster in Kubernetes, which is supported since TiDB Operator v1.1.

To enable TLS between TiDB components, perform the following steps:

1. Generate certificates for each component of the TiDB cluster to be created:

   - A set of server-side certificates for the PD/TiKV/TiDB/Pump/Drainer component, saved as the Kubernetes Secret objects: `${cluster_name}-${component_name}-cluster-secret`
   - A set of shared client-side certificates for the various clients of each component, saved as the Kubernetes Secret objects: `${cluster_name}-cluster-client-secret`.

2. Deploy the cluster, and set `.spec.tlsCluster.enabled` to `true`.
3. Configure `pd-ctl` and `tikv-ctl` to connect to the cluster.

Certificates can be issued in multiple methods. This document describes two methods. You can choose either of them to issue certificates for the TiDB cluster:

- [Using the `cfssl` system](#using-cfssl)
- [Using the `cert-manager` system](#using-cert-manager)

## Generate certificates for components of the TiDB cluster

This section describes how to issue certificates using two methods: `cfssl` and `cert-manager`.

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
    ```

    > **Note:**
    >
    > Add `"client auth"` in `profiles` - `internal` - `usages`, because this server-side certificate is also used as the client-side certificate.

3. Change the certificate signing request (CSR) of `ca-csr.json`:

    ``` json
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
    ```

4. Generate CA by the configured option:

    {{< copyable "shell-regular" >}}

    ```shell
    cfssl gencert -initca ca-csr.json | cfssljson -bare ca -
    ```

5. Generate the server-side certificates:

    In this step, a set of server-side certificate is created for each component of the TiDB cluster.

    - PD

        First, generate the default `pd-server.json` file:

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl print-defaults csr > pd-server.json
        ```

        Then, edit this file to change the `CN` and `hosts` attributes:

        ```json
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

        `${cluster_name}` is the name of the cluster. `${namespace}` is the namespace in which the TiDB cluster is deployed. You can also add your customized `hosts`.

        Finally, generate the PD server-side certificate:

        {{< copyable "shell-regular" >}}

        ```shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal pd-server.json | cfssljson -bare pd-server
        ```

    - TiKV

        First, generate the default `tikv-server.json` file:

        {{< copyable "shell-regular" >}}

        ``` shell
        cfssl print-defaults csr > tikv-server.json
        ```

        Then, edit this file to change the `CN` and `hosts` attributes:

        ```json
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

        `${cluster_name}` is the name of the cluster. `${namespace}` is the namespace in which the TiDB cluster is deployed. You can also add your customized `hosts`.

        Finally, generate the TiKV server-side certificate:

        {{< copyable "shell-regular" >}}

        ```shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal tikv-server.json | cfssljson -bare tikv-server
        ```

    - TiDB

        First, create the default `tidb-server.json` file:

        {{< copyable "shell-regular" >}}

        ```shell
        cfssl print-defaults csr > tidb-server.json
        ```

        Then, edit this file to change the `CN`, `hosts` attributes:

        ```json
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

        `${cluster_name}` is the name of the cluster. `${namespace}` is the namespace in which the TiDB cluster is deployed. You can also add your customized `hosts`.

        Finally, generate the TiDB server-side certificate:

        {{< copyable "shell-regular" >}}

        ```shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal tidb-server.json | cfssljson -bare tidb-server
        ```

    - Pump

        First, create the default `pump-server.json` file:

        {{< copyable "shell-regular" >}}

        ```shell
        cfssl print-defaults csr > pump-server.json
        ```

        Then, edit this file to change the `CN`, `hosts` attributes:

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

        `${cluster_name}` is the name of the cluster. `${namespace}` is the namespace in which the TiDB cluster is deployed. You can also add your customized `hosts`.

        Finally, generate the Pump server-side certificate:

        {{< copyable "shell-regular" >}}

        ```shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal pump-server.json | cfssljson -bare pump-server
        ```

    - Drainer

        First, generate the default `drainer-server.json` file:

        {{< copyable "shell-regular" >}}

        ```shell
        cfssl print-defaults csr > drainer-server.json
        ```

        Then, edit this file to change the `CN`, `hosts` attributes:

        ```json
        ...
            "CN": "TiDB",
            "hosts": [
              "127.0.0.1",
              "::1",
              "<for hosts list, see the following instructions>"
            ],
        ...
        ```

        Drainer is deployed using Helm. The `hosts` field varies with different configuration of the `values.yaml` file.

        If you have set the `drainerName` attribute when deploying Drainer as follows:

        ```yaml
        ...
        # Changes the names of the statefulset and Pod.
        # The default value is clusterName-ReleaseName-drainer.
        # Does not change the name of an existing running Drainer, which is unsupported.
        drainerName: my-drainer
        ...
        ```

        Then you can set the `hosts` attribute as described below:

        ```json
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

        If you have not set the `drainerName` attribute when deploying Drainer, configure the `hosts` attribute as follows:

        ```json
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

        `${cluster_name}` is the name of the cluster. `${namespace}` is the namespace in which the TiDB cluster is deployed. `${release_name}` is the `release name` you set when `helm install` is executed. `${drainer_name}` is `drainerName` in the `values.yaml` file. You can also add your customized `hosts`.

        Finally, generate the Drainer server-side certificate:

        {{< copyable "shell-regular" >}}

        ```shell
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=internal drainer-server.json | cfssljson -bare drainer-server
        ```

6. Generate the client-side certificate:

    First, create the default `client.json` file:

    {{< copyable "shell-regular" >}}

    ```shell
    cfssl print-defaults csr > client.json
    ```

    Then, edit this file to change the `CN`, `hosts` attributes. You can leave the `hosts` empty:

    ```json
    ...
        "CN": "TiDB",
        "hosts": [],
    ...
    ```

    Finally, generate the client-side certificate:

    ```shell
    cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=client client.json | cfssljson -bare client
    ```

7. Create the Kubernetes Secret object:

    If you have already generated a set of certificates for each component and a set of client-side certificate for each client as described in the above steps, create the Secret objects for the TiDB cluster by executing the following command:

    - The PD cluster certificate Secret:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl create secret generic ${cluster_name}-pd-cluster-secret --namespace=${namespace} --from-file=tls.crt=pd-server.pem --from-file=tls.key=pd-server-key.pem --from-file=ca.crt=ca.pem
        ```

    - The TiKV cluster certificate Secret:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl create secret generic ${cluster_name}-tikv-cluster-secret --namespace=${namespace} --from-file=tls.crt=tikv-server.pem --from-file=tls.key=tikv-server-key.pem --from-file=ca.crt=ca.pem
        ```

    - The TiDB cluster certificate Secret:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl create secret generic ${cluster_name}-tidb-cluster-secret --namespace=${namespace} --from-file=tls.crt=tidb-server.pem --from-file=tls.key=tidb-server-key.pem --from-file=ca.crt=ca.pem
        ```

    - The Pump cluster certificate Secret:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl create secret generic ${cluster_name}-pump-cluster-secret --namespace=${namespace} --from-file=tls.crt=pump-server.pem --from-file=tls.key=pump-server-key.pem --from-file=ca.crt=ca.pem
        ```

    - The Drainer cluster certificate Secret:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl create secret generic ${cluster_name}-drainer-cluster-secret --namespace=${namespace} --from-file=tls.crt=drainer-server.pem --from-file=tls.key=drainer-server-key.pem --from-file=ca.crt=ca.pem
        ```

    - The client certificate Secret:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl create secret generic ${cluster_name}-cluster-client-secret --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem
        ```

    You have created two Secret objects:

    - One Secret object for each PD/TiKV/TiDB/Pump/Drainer server-side certificate to load when the server is started;
    - One Secret object for their clients to connect.

### Using `cert-manager`

1. Install `cert-manager`.

    Refer to [cert-manager installation in Kubernetes](https://docs.cert-manager.io/en/release-0.11/getting-started/install/kubernetes.html) for details.

2. Create an Issuer to issue certificates to the TiDB cluster.

    To configure `cert-manager`, create the Issuer resources.

    First, create a directory which saves the files that `cert-manager` needs to create certificates:

    {{< copyable "shell-regular" >}}

    ```shell
    mkdir -p cert-manager
    cd cert-manager
    ```

    Then, create a `tidb-cluster-issuer.yaml` file with the following content:

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

    `${cluster_name}` is the name of the cluster. The above YAML file creates three objects:

    - An Issuer object of the SelfSigned type, used to generate the CA certificate needed by Issuer of the CA type;
    - A Certificate object, whose `isCa` is set to `true`.
    - An Issuer, used to issue TLS certificates between TiDB components.

    Finally, execute the following command to create an Issuer:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl apply -f tidb-cluster-issuer.yaml
    ```

3. Generate the server-side certificate.

    In `cert-manager`, the Certificate resource represents the certificate interface. This certificate is issued and updated by the Issuer created in Step 2.

    According to [Enable TLS Authentication | TiDB Documentation](https://pingcap.com/docs/stable/how-to/secure/enable-tls-between-components/), each component needs a server-side certificate, and all components need a shared client-side certificate for their clients.

    - PD

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

        `${cluster_name}` is the name of the cluster. Configure the items as follows:

        - Set `spec.secretName` to `${cluster_name}-pd-cluster-secret`.
        - Add `server auth` and `client auth` in `usages`.
        - Add the following DNSs in `dnsNames`. You can also add other DNSs according to your needs:
            - `${cluster_name}-pd`
            - `${cluster_name}-pd.${namespace}`
            - `${cluster_name}-pd.${namespace}.svc`
            - `${cluster_name}-pd-peer`
            - `${cluster_name}-pd-peer.${namespace}`
            - `${cluster_name}-pd-peer.${namespace}.svc`
            - `*.${cluster_name}-pd-peer`
            - `*.${cluster_name}-pd-peer.${namespace}`
            - `*.${cluster_name}-pd-peer.${namespace}.svc`
        - Add the following two IPs in `ipAddresses`. You can also add other IPs according to your needs:
            - `127.0.0.1`
            - `::1`
        - Add the Issuer created above in `issuerRef`.
        - For other attributes, refer to [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec).

        After the object is created, `cert-manager` generates a `${cluster_name}-pd-cluster-secret` Secret object to be used by the PD component of the TiDB server.

    - TiKV

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

        `${cluster_name}` is the name of the cluster. Configure the items as follows:

        - Set `spec.secretName` to `${cluster_name}-tikv-cluster-secret`.
        - Add `server auth` and `client auth` in `usages`.
        - Add the following DNSs in `dnsNames`. You can also add other DNSs according to your needs:

            - `${cluster_name}-tikv`
            - `${cluster_name}-tikv.${namespace}`
            - `${cluster_name}-tikv.${namespace}.svc`
            - `${cluster_name}-tikv-peer`
            - `${cluster_name}-tikv-peer.${namespace}`
            - `${cluster_name}-tikv-peer.${namespace}.svc`
            - `*.${cluster_name}-tikv-peer`
            - `*.${cluster_name}-tikv-peer.${namespace}`
            - `*.${cluster_name}-tikv-peer.${namespace}.svc`

        - Add the following 2 IPs in `ipAddresses`. You can also add other IPs according to your needs:
            - `127.0.0.1`
            - `::1`
        - Add the Issuer created above in `issuerRef`.
        - For other attributes, refer to [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec).

        After the object is created, `cert-manager` generates a `${cluster_name}-tikv-cluster-secret` Secret object to be used by the TiKV component of the TiDB server.

    - TiDB

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

        `${cluster_name}` is the name of the cluster. Configure the items as follows:

        - Set `spec.secretName` to `${cluster_name}-tidb-cluster-secret`
        - Add `server auth` and `client auth` in `usages`
        - Add the following DNSs in `dnsNames`. You can also add other DNSs according to your needs:

            - `${cluster_name}-tidb`
            - `${cluster_name}-tidb.${namespace}`
            - `${cluster_name}-tidb.${namespace}.svc`
            - `${cluster_name}-tidb-peer`
            - `${cluster_name}-tidb-peer.${namespace}`
            - `${cluster_name}-tidb-peer.${namespace}.svc`
            - `*.${cluster_name}-tidb-peer`
            - `*.${cluster_name}-tidb-peer.${namespace}`
            - `*.${cluster_name}-tidb-peer.${namespace}.svc`

        - Add the following 2 IPs in `ipAddresses`. You can also add other IPs according to your needs:
            - `127.0.0.1`
            - `::1`
        - Add the Issuer created above in `issuerRef`.
        - For other attributes, refer to [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec).

        After the object is created, `cert-manager` generates a `${cluster_name}-tidb-cluster-secret` Secret object to be used by the TiDB component of the TiDB server.

    - Pump

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

        `${cluster_name}` is the name of the cluster. Configure the items as follows:

        - Set `spec.secretName` to `${cluster_name}-pump-cluster-secret`
        - Add `server auth` and `client auth` in `usages`
        - Add the following DNSs in `dnsNames`. You can also add other DNSs according to your needs:

            - `*.${cluster_name}-pump`
            - `*.${cluster_name}-pump.${namespace}`
            - `*.${cluster_name}-pump.${namespace}.svc`

        - Add the following 2 IPs in `ipAddresses`. You can also add other IPs according to your needs:
            - `127.0.0.1`
            - `::1`
        - Add the Issuer created above in the `issuerRef`
        - For other attributes, refer to [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec).

        After the object is created, `cert-manager` generates a `${cluster_name}-pump-cluster-secret` Secret object to be used by the Pump component of the TiDB server.

    - Drainer

        Drainer is deployed using Helm. The `dnsNames` field varies with different configuration of the `values.yaml` file.

        If you set the `drainerName` attributes when deploying Drainer as follows:

        ```yaml
        ...
        # Changes the name of the statefulset and Pod.
        # The default value is clusterName-ReleaseName-drainer
        # Does not change the name of an existing running Drainer, which is unsupported.
        drainerName: my-drainer
        ...
        ```

        Then you need to configure the certificate as described below:

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

        If you didn't set the `drainerName` attribute when deploying Drainer, configure the `dnsNames` attributes as follows:

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

        `${cluster_name}` is the name of the cluster. `${namespace}` is the namespace in which the TiDB cluster is deployed. `${release_name}` is the `release name` you set when `helm install` is executed. `${drainer_name}` is `drainerName` in the `values.yaml` file. You can also add your customized `dnsNames`.

        - Set `spec.secretName` to `${cluster_name}-drainer-cluster-secret`.
        - Add `server auth` and `client auth` in `usages`.
        - See the above descriptions for `dnsNames`.
        - Add the following 2 IPs in `ipAddresses`. You can also add other IPs according to your needs:
            - `127.0.0.1`
            - `::1`
        - Add the Issuer created above in `issuerRef`.
        - For other attributes, refer to [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec).

        After the object is created, `cert-manager` generates a `${cluster_name}-drainer-cluster-secret` Secret object to be used by the Drainer component of the TiDB server.

4. Generate the client-side certificate for components of the TiDB cluster.

    ```yaml
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

    `${cluster_name}` is the name of the cluster. Configure the items as follows:

    - Set `spec.secretName` to `${cluster_name}-cluster-client-secret`.
    - Add `client auth` in `usages`.
    - You can leave `dnsNames` and `ipAddresses` empty.
    - Add the Issuer created above in `issuerRef`.
    - For other attributes, refer to [cert-manager API](https://cert-manager.io/docs/reference/api-docs/#cert-manager.io/v1alpha2.CertificateSpec).

    After the object is created, `cert-manager` generates a `${cluster_name}-cluster-client-secret` Secret object to be used by the clients of the TiDB components.

## Deploy the TiDB cluster

When you deploy a TiDB cluster, you can enable TLS between TiDB components, and set the `cert-allowed-cn` configuration item (for TiDB, the configuration item is `cluster-verify-cn`) to verify the CN (Common Name) of each component's certificate.

> **Note:**
>
> Currently, you can set only one value for the `cert-allowed-cn` configuration item of PD. Therefore, the `commonName` of all `Certificate` objects must be the same.

In this step, you need to perform the following operations:

- Create a TiDB cluster
- Enable TLS between the TiDB components, and enable CN verification
- Deploy a monitoring system
- Deploy the Pump component, and enable CN verification

1. Create a TiDB cluster:

    Create the `tidb-cluster.yaml` file:

    ``` yaml
    apiVersion: pingcap.com/v1alpha1
    kind: TidbCluster
    metadata:
     name: ${cluster_name}
     namespace: ${namespace}
    spec:
     tlsCluster:
       enabled: true
     version: v3.1.0
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
       version: v3.1.0
     reloader:
       baseImage: pingcap/tidb-monitor-reloader
       version: v1.0.1
     imagePullPolicy: IfNotPresent
    ```

    Execute `kubectl apply -f tidb-cluster.yaml` to create a TiDB cluster.

    This operation also includes deploying a monitoring system and the Pump component.

2. Create a Drainer component and enable TLS and CN verification:

    - Method 1: Set `drainerName` when you create Drainer.

        Edit the `values.yaml` file, set `drainer-name`, and enable the TLS feature:

        ``` yaml
        ...
        drainerName: ${drainer_name}
        tlsCluster:
          enabled: true
          certAllowedCN:
            - TiDB
        ...
        ```

        Deploy the Drainer cluster:

        {{< copyable "shell-regular" >}}

        ``` shell
        helm install pingcap/tidb-drainer --name=${release_name} --namespace=${namespace} --version=${helm_version} -f values.yaml
        ```

    - Method 2: Do not set `drainerName` when you create Drainer.

        Edit the `values.yaml` file, and enable the TLS feature:

        ``` yaml
        ...
        tlsCluster:
          enabled: true
          certAllowedCN:
            - TiDB
        ...
        ```

        Deploy the Drainer cluster:

        {{< copyable "shell-regular" >}}

        ``` shell
        helm install pingcap/tidb-drainer --name=${release_name} --namespace=${namespace} --version=${helm_version} -f values.yaml
        ```

3. Create the Backup/Restore resource object:

    - Create the `backup.yaml` file:

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

        Deploy Backup:

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl apply -f backup.yaml
        ```

    - Create the `restore.yaml` file:

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
        ```

        Deploy Restore:

        {{< copyable "shell-regular" >}}

        ``` shell
        kubectl apply -f restore.yaml
        ```

## Configure `pd-ctl` and connect to the cluster

1. Download `pd-ctl`:

    Refer to [Download TiDB installation package](https://pingcap.com/docs/stable/reference/tools/pd-control/#download-tidb-installation-package).

2. Download the client-side certificate:

    The client-side certificate is the client certificate you have created in [Generate certificates for components](#generate-certificates-for-components-of-the-tidb-cluster). You can directly use the certificate, or obtain it from the `${cluster_name}-cluster-client-secret` Kubernetes Secret object created before.

    {{< copyable "shell-regular" >}}

    ``` shell
    kubectl get secret -n ${namespace} ${cluster_name}-cluster-client-secret  -ojsonpath='{.data.tls\.crt}' | base64 --decode > client-tls.crt
    kubectl get secret -n ${namespace} ${cluster_name}-cluster-client-secret  -ojsonpath='{.data.tls\.key}' | base64 --decode > client-tls.key
    kubectl get secret -n ${namespace} ${cluster_name}-cluster-client-secret  -ojsonpath='{.data.ca\.crt}'  | base64 --decode > client-ca.crt
    ```

3. Connect to the PD and TiKV cluster using `pd-ctl` and `tikv-ctl`:

    When you deploy the server-side certificate for the PD and TiKV component, some `hosts` are customized, so you need to use these `hosts` to connect to the PD and TiKV cluster.

    - Connect to the PD cluster:

        {{< copyable "shell-regular" >}}

        ```shell
        pd-ctl --cacert=client-ca.crt --cert=client-tls.crt --key=client-tls.key -u https://${cluster_name}-pd.${namespace}.svc:2379 member
        ```

    - Connect to the TiKV cluster:

        {{< copyable "shell-regular" >}}

        ```shell
        tikv-ctl --ca-path=client-ca.crt --cert-path=client-tls.crt --key-path=client-tls.key --host ${cluster_name}-tikv-0.${cluster_name}-tikv-peer.${namespace}:20160 cluster
        ```
