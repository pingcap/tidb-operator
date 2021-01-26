---
title: Renew and Replace the TLS Certificate
summary: Learn how to renew and replace TLS certificates between TiDB components.
---

# Renew and Replace the TLS Certificate

This document introduces how to renew and replace certificates of the corresponding components before certificates expire, taking TLS certificates between PD, TiKV, and TiDB components in the TiDB cluster as an example.

If you need to renew and replace certificates between other components in the TiDB cluster, TiDB server-side certificate, or MySQL client-side certificate, you can take similar steps to complete the operation.

The renewal and replacement operations in this document assume that the original certificates have not expired. If the original certificates expire or become invalid, to generate new certificates and restart the TiDB cluster, refer to [Enable TLS between TiDB components](enable-tls-between-components.md) or [Enable TLS for MySQL client](enable-tls-for-mysql-client.md).

## Renew and replace certificates issued by the `cfssl` system

If the original TLS certificates are issued by [the `cfssl` system](enable-tls-between-components.md#using-cfssl) and the original certificates have not expired, you can renew and replace the certificates between PD, TiKV and TiDB components as follows.

### Renew and replace the CA certificate

> **Note:**
>
> If you don't need to renew the CA certificate, you can skip the operations in this section and directly refer to [renew and replace certificates between components](#renew-and-replace-certificates-between-components).

1. Back up the original CA certificate and key.

    {{< copyable "shell-regular" >}}

    ```shell
    mv ca.pem ca.old.pem && \
    mv ca-key.pem ca-key.old.pem
    ```

2. Generate the new CA certificate and key based on the configuration of the original CA certificate and certificate signing request (CSR).

    {{< copyable "shell-regular" >}}

    ```shell
    cfssl gencert -initca ca-csr.json | cfssljson -bare ca -
    ```

    > **Note:**
    >
    > If necessary, you can update `expiry` in the configuration file and in CSR.

3. Back up the new CA certificate and key, and generate a combined CA certificate based on the original CA certificate and the new CA certificate.

    {{< copyable "shell-regular" >}}

    ```shell
    mv ca.pem ca.new.pem && \
    mv ca-key.pem ca-key.new.pem && \
    cat ca.new.pem ca.old.pem > ca.pem
    ```

4. Update each corresponding Kubernetes Secret objects based on the combined CA certificate.

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic ${cluster_name}-pd-cluster-secret --namespace=${namespace} --from-file=tls.crt=pd-server.pem --from-file=tls.key=pd-server-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-tikv-cluster-secret --namespace=${namespace} --from-file=tls.crt=tikv-server.pem --from-file=tls.key=tikv-server-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-tidb-cluster-secret --namespace=${namespace} --from-file=tls.crt=tidb-server.pem --from-file=tls.key=tidb-server-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-cluster-client-secret --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    ```

    In the above command, `${cluster_name}` is the name of the cluster, and `${namespace}` is the namespace in which the TiDB cluster is deployed.

    > **Note:**
    >
    > The above command only renews the server-side CA certificate and the client-side CA certificate between PD, TiKV, and TiDB components. If you need to renew the server-side CA certificates for other components, such as TiCDC and TiFlash, you can execute the similar command.

5. [Perform the rolling restart](restart-a-tidb-cluster.md) to components that need to load the combined CA certificate.

    After the completion of the rolling restart, based on the combined CA certificate, each component can accept the certificate issued by either the original CA certificate or the new CA certificate at the same time.

### Renew and replace certificates between components

> **Note:**
>
> Before renewing and replacing certificates between components, make sure that the CA certificate can verify the certificates between components before and after the renewal as valid. If you have [renewed and replaced the CA certificate](#renew-and-replace-the-ca-certificate), make sure that the TiDB cluster is restarted based on the new CA certificate.

1. Generate new server-side and client-side certificates based on the original configuration information of each component.

    {{< copyable "shell-regular" >}}

    ```shell
    cfssl gencert -ca=ca.new.pem -ca-key=ca-key.new.pem -config=ca-config.json -profile=internal pd-server.json | cfssljson -bare pd-server
    cfssl gencert -ca=ca.new.pem -ca-key=ca-key.new.pem -config=ca-config.json -profile=internal tikv-server.json | cfssljson -bare tikv-server
    cfssl gencert -ca=ca.new.pem -ca-key=ca-key.new.pem -config=ca-config.json -profile=internal tidb-server.json | cfssljson -bare tidb-server
    cfssl gencert -ca=ca.new.pem -ca-key=ca-key.new.pem -config=ca-config.json -profile=client client.json | cfssljson -bare client
    ```

    > **Note:**
    >
    > - The above command assumes that you have [renewed and replaced the CA certificate](#renew-and-replace-the-tls-certificate) and saved the new CA certificate as `ca.new.pem` and the new key as `ca-key.new.pem`. If you have not renewed the CA certificate and the key, modify the corresponding parameters in the command to `ca.pem` and `ca-key.pem`.
    > - The above command only generates the server-side and the client-side certificates between PD, TiKV, and TiDB components. If you need to generate the server-side CA certificates for other components, such as TiCDC and TiFlash, you can execute the similar command.

2. Update each corresponding Kubernetes Secret object based on the newly generated server-side and client-side certificates.

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic ${cluster_name}-pd-cluster-secret --namespace=${namespace} --from-file=tls.crt=pd-server.pem --from-file=tls.key=pd-server-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-tikv-cluster-secret --namespace=${namespace} --from-file=tls.crt=tikv-server.pem --from-file=tls.key=tikv-server-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-tidb-cluster-secret --namespace=${namespace} --from-file=tls.crt=tidb-server.pem --from-file=tls.key=tidb-server-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-cluster-client-secret --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.pem --dry-run=client -o yaml | kubectl apply -f -
    ```

    In the above command, `${cluster_name}` is the name of the cluster, and `${namespace}` is the namespace in which the TiDB cluster is deployed.

    > **Note:**
    >
    >  The above command only renews the server-side and the client-side certificate between PD, TiKV, and TiDB components. If you need to renew the server-side certificates for other components, such as TiCDC and TiFlash, you can execute the similar command.

3. [Perform the rolling restart](restart-a-tidb-cluster.md) to components that need to load the new certificates.

    After the completion of the rolling restart, each component use the new certificate for TLS communication. If you refer to [Renew and replace the CA certificate](#renew-and-replace-the-ca-certificate) and make each component load the combined CA certificate, each component can still accept the certificate issued by the original CA certificate.

### Optional: Remove the original CA certificate from the combined CA certificate

After you [renew and replace the combined CA certificate](#renew-and-replace-the-ca-certificate), [server-side and client-side certificates between components](#renew-and-replace-certificates-between-components), you might want to remove the original CA certificate (for example, because the CA certificate has expired or the private key is compromised). To remove the original CA certificate, take steps as follows:

1. Renew the Kubernetes Secret objects based on the new CA certificate.

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl create secret generic ${cluster_name}-pd-cluster-secret --namespace=${namespace} --from-file=tls.crt=pd-server.pem --from-file=tls.key=pd-server-key.pem --from-file=ca.crt=ca.new.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-tikv-cluster-secret --namespace=${namespace} --from-file=tls.crt=tikv-server.pem --from-file=tls.key=tikv-server-key.pem --from-file=ca.crt=ca.new.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-tidb-cluster-secret --namespace=${namespace} --from-file=tls.crt=tidb-server.pem --from-file=tls.key=tidb-server-key.pem --from-file=ca.crt=ca.new.pem --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic ${cluster_name}-cluster-client-secret --namespace=${namespace} --from-file=tls.crt=client.pem --from-file=tls.key=client-key.pem --from-file=ca.crt=ca.new.pem --dry-run=client -o yaml | kubectl apply -f -
    ```

    In the above command, `${cluster_name}` is the name of the cluster, and `${namespace}` is the namespace in which the TiDB cluster is deployed.

    > **Note:**
    >
    > - The above command assumes that you have [renewed and replaced the CA certificate](#renew-and-replace-the-tls-certificate) and saved the new CA certificate as `ca.new.pem`.

2. [Perform the rolling restart](restart-a-tidb-cluster.md) to components that need to load the new certificates.

    After the completion of the rolling restart, each component can only accept the certificate issued by the new CA certificate.

## Renew and replace the certificate issued by `cert-manager`

If the original TLS certificate is issued by [the `cert-manager` system](enable-tls-between-components.md#using-cert-manager), and the original certificate has not expired, the procedure varies with whether to renew the CA certificate.

### Renew and replace the CA certificate and certificates between components

When you use `cert-manager` to issue the certificate, if you specify the `spec.renewBefore` of the `Certificate` resource, `cert-manager` can automatically update the certificate before it expires.

Although `cert-manager` can automatically renew the CA certificate and the corresponding Kubernetes Secret objects, it currently does not support merging the old and new CA certificates into a combined CA certificate to accept certificates issued by the new and old CA certificates at the same time. Therefore, during the process of renewing and replacing the CA certificate, the cluster components cannot authenticate each other via TLS.

> **Warning:**
>
> Because the components cannot accept certificates issued by the new and old CAs at the same time, during the process of renewing and replacing certificates, some components' Pods need to be recreated. This might cause some requests to access the TiDB cluster to fail.

The steps to renew and replace the CA certificates of PD, TiKV, TiDB and certificates between components are as follows.

1. The `cert-manager` automatically renews the CA certificate and the Kubernetes Secret object `${cluster_name}-ca-secret` before the certificate expires.

    `${cluster_name}` is the name of the cluster.

    To manually renew the CA certificate, you can directly delete the corresponding Kubernetes Secret objects and trigger `cert-manager` to regenerate the CA certificate.

2. Delete the Kubernetes Secret objects corresponding to the certificate of each component.

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl delete secret ${cluster_name}-pd-cluster-secret --namespace=${namespace}
    kubectl delete secret ${cluster_name}-tikv-cluster-secret --namespace=${namespace}
    kubectl delete secret ${cluster_name}-tidb-cluster-secret --namespace=${namespace}
    kubectl delete secret ${cluster_name}-cluster-client-secret --namespace=${namespace}
    ```

    In the above command, `${cluster_name}` is the name of the cluster, and `${namespace}` is the namespace in which the TiDB cluster is deployed.

3. Wait for `cert-manager` to issue new certificates for each component based on the new CA certificate.

    Observe the output of `kubectl get secret --namespace=${namespace}` until the Kubernetes Secret objects corresponding to all components are created.

4. Forcibly recreate the Pods of the PD, TiKV, and TiDB components in sequence.

    Because `cert-manager` does not support combined CA certificates, if you try to perform a rolling update of each component, the Pods using the different CAs to issue certificates cannot communicate with each other via TLS. Therefore, you need to delete the Pods forcibly and recreate the Pods based on the certificate issued by the new CA.

    {{< copyable "shell-regular" >}}

    ```
    kubectl delete -n ${namespace} pod ${pod_name}
    ```

    In the above command, `${namespace}` is the namespace in which the TiDB cluster is deployed, and `${pod_name}` is the Pod name of each replica of PD, TiKV, and TiDB.

### Only renew and replace certificates between components

1. The `cert-manager` automatically updates the certificate of each component and the Kubernetes Secret object before the certificate expires.

    For PD, TiKV, and TiDB components, the namespace in which the TiDB cluster is deployed contains the following Kubernetes Secret objects:

    {{< copyable "shell-regular" >}}

    ```
    ${cluster_name}-pd-cluster-secret
    ${cluster_name}-tikv-cluster-secret
    ${cluster_name}-tidb-cluster-secret
    ${cluster_name}-cluster-client-secret
    ```

    In the above command, `${cluster_name}` is the name of the cluster.

    If you want to manually update the certificate between components, you can directly delete the corresponding Kubernetes Secret objects and trigger `cert-manager` to regenerate the certificate between components.

2. For certificates between components, each component automatically reloads the new certificates when creating the new connection later.

    > **Note:**
    >
    > - Currently, each component does not support [reload CA certificates](https://docs.pingcap.com/tidb/stable/enable-tls-between-components#reload-certificates) manually，you need to refer to [renew and replace the CA certificate and certificates between components](#renew-and-replace-the-ca-certificate-and-certificates-between-components).
    > - For the TiDB server-side certificate, you can manually reload by referring to any of the following methods:
    >     - Refer to [Reload certificate, key, and CA](https://docs.pingcap.com/tidb/stable/enable-tls-between-clients-and-servers#reload-certificate-key-and-ca).
    >     - Refer to [Rolling restart the TiDB Cluster](restart-a-tidb-cluster.md) to perform a rolling restart of TiDB server.
