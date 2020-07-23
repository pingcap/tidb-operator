# Stability Test Cookbook

> Important notes: this guide is under heavy development and have complicated environment pre-requesites, things are ought to change in the future.

## Run stability test

The following commands assumes you are in the `tidb-operator` working directory:
```shell
# image will be tagged as YOUR_DOCKER_REGISTRY/pingcap/tidb-operator-stability-test:latest
$ export DOCKER_REGISTRY=${YOUR_DOCKER_REGISTRY} 
$ make stability-test-push
$ kubectl apply -f ./tests/manifests/stability/stability-configmap.yaml
# edit the stability.yaml and change .spec.template.spec.containers[].image to the pushed image
$ vi ./tests/manifests/stability/stability.yaml
# apply the stability test pod
$ kubectl apply -f ./tests/manifests/stability/stability.yaml
```

## Get test report

```shell
$ kubectl -n tidb-operator-stability logs tidb-operator-stability
```

## Inspect overall cluster stats under various operations

It is useful to inspect how the cluster performs under various kind of operations or faults, you can access such information from the Grafana dashboard of each cluster:

```shell
$ kubectl port-forward -n ${CLUSTER_NAMESPACE} svc/${CLUSTER_GRAFANA_SERVICE} 3000:3000
```

Navigate to [localhost:3000](http://localhost:3000) to view the dashboards.
 
Optionally, you can view the event annotations like `scale cluster`, `upgrade cluster`, `vm crash` by querying annotations in Grafana to get better understanding of the system, follow this step-by-step guide:

1. click "Dashboard Setting" in the navigate bar
2. click the big "Make Editable" button
3. click "Annotations" in the sidebar
4. click "Add Annotation Query"
5. enter a name you like
6. switch "Match Any" on
7. add "stability" tag
8. click "add"
9. go back to dashboard and you will see the annotations trigger and the cluster events


## Alternative: run stability test in your local environment

Deploy & witness flow can be tedious when developing stability-test, this document introduce that how to run stability-test out of the cluster(your local machine, usually) while still operating the remote cluster.

### TL;DR: 
```shell
$ telepresence --new-deployment ${POD_NAME}
$ go build -o stability ./tests/cmd/stability/main.go
$ ./stability --kubeconfig=${YOUR_KUBE_CONFIG_PATH}
```

### Explained

Generally we have three problems to solve: 

1. **Out of cluster client**: Now we try to load configs in the following order:
    * if `kubeconfig` command line option provided, use it
    * if `KUBECONFIG` env variable set, use it
    * try loading `InClusterConfig()`
so you have to specify the `kubeconfig` path by either command line option or  env variable if you want to test locally.
2. **DNS and network issue**: Two-way proxy using Telepresence. We cannot resolve cluster dns name and access cluster ip easily, `telepresence` helps with that, it creates a proxy pod in the cluster and open a vpn connection to kubernetes cluster via this pod. Just run ([full documentations](https://www.telepresence.io/reference/install)):
```shell
$ brew cask install osxfuse
$ brew install datawire/blackbird/telepresence
$ telepresence --new-deployment ${POD_NAME}
``` 
**PS**: If you cannot resolve cluster dns names after set up, try clear DNS cache.
**PSS**: Typically you can't use telepresence VPN mode with other VPNs (of course SSR is ok).
