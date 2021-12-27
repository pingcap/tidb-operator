---
title: Access the TiDB Cluster in Kubernetes
summary: Learn how to access the TiDB cluster in Kubernetes.
aliases: ['/docs/tidb-in-kubernetes/dev/access-tidb/']
---

# Access the TiDB Cluster

This document describes how to access the TiDB cluster.

You can configure Service with different types according to the scenarios, such as `ClusterIP`, `NodePort`, `LoadBalancer`, etc., and use different access methods for different types. 

You can obtain TiDB Service information by running the following command:

{{< copyable "shell-regular" >}}

```bash
kubectl get svc ${serviceName} -n ${namespace}
```

For example:

```
# kubectl get svc basic-tidb -n default
NAME         TYPE       CLUSTER-IP     EXTERNAL-IP   PORT(S)                          AGE
basic-tidb   NodePort   10.233.6.240   <none>        4000:32498/TCP,10080:30171/TCP   61d
```

The above example describes the information of the `basic-tidb` service in the `default` namespace. The type is `NodePort`, ClusterIP is `10.233.6.240`, ServicePort is `4000` and `10080`, and the corresponding NodePort is `32498` and `30171`.

> **Note:**
>
> [The default authentication plugin of MySQL 8.0](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_default_authentication_plugin) is updated from `mysql_native_password` to `caching_sha2_password`. Therefore, if you use MySQL client from MySQL 8.0 to access the TiDB service (TiDB version earlier than v4.0.7), and if the user account has a password, you need to explicitly specify the `--default-auth=mysql_native_password` parameter.

## ClusterIP

`ClusterIP` exposes services through the internal IP of the cluster. When selecting this type of service, you can only access it within the cluster by the following methods:

* ClusterIP + ServicePort
* Service domain name (`${serviceName}.${namespace}`) + ServicePort

## NodePort

If there is no LoadBalancer, you can choose to expose the service through NodePort. NodePort exposes services through the node's IP and static port. You can access a NodePort service from outside of the cluster by requesting `NodeIP + NodePort`.

To view the Node Port assigned by Service, run the following commands to obtain the Service object of TiDB:

{{< copyable "shell-regular" >}}

```bash
kubectl -n ${namespace} get svc ${cluster_name}-tidb -ojsonpath="{.spec.ports[?(@.name=='mysql-client')].nodePort}{'\n'}"
```

To check you can access TiDB services by using the IP of what nodes, see the following two cases:

- When `externalTrafficPolicy` is configured as `Cluster`, you can use the IP of any node to access TiDB services.
- When `externalTrafficPolicy` is configured as `Local`, use the following commands to get the nodes where the TiDB instance of a specified cluster is located:

    {{< copyable "shell-regular" >}}

    ```bash
    kubectl -n ${namespace} get pods -l "app.kubernetes.io/component=tidb,app.kubernetes.io/instance=${cluster_name}" -ojsonpath="{range .items[*]}{.spec.nodeName}{'\n'}{end}"
    ```

## LoadBalancer

If the TiDB cluster runs in an environment with LoadBalancer, such as on GCP or AWS, it is recommended to use the LoadBalancer feature of these cloud platforms by setting `tidb.service.type=LoadBalancer`.

To access TiDB Service through LoadBalancer, refer to [EKS](deploy-on-aws-eks.md#install-the-mysql-client-and-connect), [GKE](deploy-on-gcp-gke.md#install-the-mysql-client-and-connect) and [ACK](deploy-on-alibaba-cloud.md#access-the-database).

See [Kubernetes Service Documentation](https://kubernetes.io/docs/concepts/services-networking/service/) to know more about the features of Service and what LoadBalancer in the cloud platform supports.
