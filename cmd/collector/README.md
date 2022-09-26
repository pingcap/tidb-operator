# Collector - a tool for collecting tidb related information in a kubernetes cluster.

Collector is a tool for collecting tidb related kubernetes information in your own cluster.
To run the tool, you need a kubeconfig to access the cluster and a config file.

## Configuration

The configuration file for collector specifies the resources to be collected.
The collector will collect following information:
1. Pod information
2. Event information
3. Advanced statefulset information
4. Persistent volume information
5. Persistent volume claim information.
6. Service information
7. TidbCluster information
8. StatefulSet information
9. ConfigMap information

There're global configuration and per-resource configuration.
User can specify a namespace in global config, and override that in per-resource config.
For each resource, user can specify whether skip the collection for the resource (disable), the namespace for the resource and the label selector for the resource.

For instance, user can use the following config to collect information in `demo` namespace, enforce label selector for pods and skip the advanced stateful set.
```yaml
namespace: demo
asts:
  disable: true
pod:
  labels:
    env: dev
```

## Execution

To execute the tool, user need to specify the following flags:
1. `--kubeconfig`: the kubeconfig for the kubernetes cluster to be collected
2. `--config`: the config file for info collection
3. `--output`: the output path for the collection

The tool will collect the infornation and genrate a zip file.
User can review the contents for the collection before sharing it.
