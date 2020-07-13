---
title: Common Network Issues of TiDB in Kubernetes
summary: Learn the common network issues of TiDB in Kubernetes and their solutions.
---

# Common Network Issues of TiDB in Kubernetes

This document describes the common network issues of TiDB in Kubernetes and their solutions.

## Network connection failure between Pods

In a TiDB cluster, you can access most Pods by using the Pod's domain name (allocated by the Headless Service). The exception is when TiDB Operator collects the cluster information or issues control commands, it accesses the PD (Placement Driver) cluster using the `service-name` of the PD service.

When you find some network connection issues among Pods from the log or monitoring metrics, or when you find the network connection among Pods might be abnormal according to the problematic condition, follow the following process to diagnose and narrow down the problem:

1. Confirm that the endpoints of the Service and Headless Service are normal:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl -n ${namespace} get endpoints ${cluster_name}-pd
    kubectl -n ${namespace} get endpoints ${cluster_name}-tidb
    kubectl -n ${namespace} get endpoints ${cluster_name}-pd-peer
    kubectl -n ${namespace} get endpoints ${cluster_name}-tikv-peer
    kubectl -n ${namespace} get endpoints ${cluster_name}-tidb-peer
    ```

    The `ENDPOINTS` field shown in the above command must be a comma-separated list of `cluster_ip:port`. If the field is empty or incorrect, check the health of the Pod and whether `kube-controller-manager` is working properly.

2. Enter the Pod's Network Namespace to diagnose network problems:

    {{< copyable "shell-regular" >}}

    ```shell
    tkctl debug -n ${namespace} ${pod_name}
    ```

    After the remote shell is started, use the `dig` command to diagnose the DNS resolution. If the DNS resolution is abnormal, refer to [Debugging DNS Resolution](https://kubernetes.io/docs/tasks/administer-cluster/dns-debugging-resolution/) for troubleshooting.

    {{< copyable "shell-regular" >}}

    ```shell
    dig ${HOSTNAME}
    ```

    Use the `ping` command to diagnose the connection with the destination IP (the Pod IP resolved using `dig`):

    {{< copyable "shell-regular" >}}

    ```shell
    ping ${TARGET_IP}
    ```

    - If the `ping` check fails, refer to [Debugging Kubernetes Networking](https://www.praqma.com/stories/debugging-kubernetes-networking/) for troubleshooting.

    - If the `ping` check succeeds, continue to check whether the target port is open by using `telnet`:

        {{< copyable "shell-regular" >}}

        ```shell
        telnet ${TARGET_IP} ${TARGET_PORT}
        ```

        If the `telnet` check fails, check whether the port corresponding to the Pod is correctly exposed and whether the port of the application is correctly configured:

        {{< copyable "shell-regular" >}}

        ```shell
        # Checks whether the ports are consistent.
        kubectl -n ${namespace} get po ${pod_name} -ojson | jq '.spec.containers[].ports[].containerPort'

        # Checks whether the application is correctly configured to serve the specified port.
        # The default port of PD is 2379 when not configured.
        kubectl -n ${namespace} -it exec ${pod_name} -- cat /etc/pd/pd.toml | grep client-urls
        # The default port of PD is 20160 when not configured.
        kubectl -n ${namespace} -it exec ${pod_name} -- cat /etc/tikv/tikv.toml | grep addr
        # The default port of TiDB is 4000 when not configured.
        kubectl -n ${namespace} -it exec ${pod_name} -- cat /etc/tidb/tidb.toml | grep port
        ```

## Unable to access the TiDB service

If you cannot access the TiDB service, first check whether the TiDB service is deployed successfully using the following method:

1. Check whether all components of the cluster are up and the status of each component is `Running`.

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get po -n ${namespace}
    ```

2. Check whether the TiDB service correctly generates the endpoint object:

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl get endpoints -n ${namespaces} ${cluster_name}-tidb
    ```

3. Check the log of TiDB components to see whether errors are reported.

    {{< copyable "shell-regular" >}}

    ```shell
    kubectl logs -f ${pod_name} -n ${namespace} -c tidb
    ```

If the cluster is successfully deployed, check the network using the following steps:

1. If you cannot access the TiDB service using `NodePort`, try to access the TiDB service using the `clusterIP` on the node. If the `clusterIP` works, the network within the Kubernetes cluster is normal. Then the possible issues are as follows:

    - Network failure exists between the client and the node.
    - Check whether the `externalTrafficPolicy` attribute of the TiDB service is `Local`. If it is `Local`, you must access the client using the IP of the node where the TiDB Pod is located.

2. If you still cannot access the TiDB service using the `clusterIP`, connect using `<PodIP>:4000` on the TiDB service backend. If the `PodIP` works, you can confirm that the problem is in the connection between `clusterIP` and `PodIP`. Check the following items:

    - Check whether `kube-proxy` on each node is working.

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl get po -n kube-system -l k8s-app=kube-proxy
        ```

    - Check whether the TiDB service rule is correct in the `iptables` rules.

        {{< copyable "shell-regular" >}}

        ```shell
        iptables-save -t nat |grep ${clusterIP}
        ```

    - Check whether the corresponding endpoint is correct:

        {{< copyable "shell-regular" >}}

        ```shell
        kubectl get endpoints -n ${namespaces} ${cluster_name}-tidb
        ```

3. If you cannot access the TiDB service even using `PodIP`, the problem is on the Pod level network. Check the following items:

    - Check whether the relevant route rules on the node are correct.
    - Check whether the network plugin service works well.
    - Refer to [network connection failure between Pods](#network-connection-failure-between-pods) section.
