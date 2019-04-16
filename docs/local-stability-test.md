# How to: run stability test in your local environment

Deploy & witness flow can be tedious when developing stability-test, this document introduce that how to run stability-test out of the cluster(your local machine, usually) while still operating the remote cluster.

### TL;DR: 
```shell
$ telepresence --new-deployment ${POD_NAME}
$ go build -o stability ./tests/cmd/stability/main.go
$ ./stability --operator-repo-dir=${ABITRARY_EMPTY_DIR_TO_CLONE_OPERATOR_REPO}
```

### Explained

Generally we have three problems to solve: 

1. **Out of cluster client**: Now we try to load configs in the following order:
    * if `kubeconfig` command line option provided, use it
    * if `KUBECONFIG` env variable set, use it
    * try loading `InClusterConfig()`
    * if no `InClusterConfig()` provided, try loading kubeconfig file from default location (`~/.kube/config`)
so typically you will get right client configs if you have proper `~/.kube/config` in your local environment. 
2. **Privilege issue**: If you don't want to or cannot run stability test with root privilege, change the working dir or create it in advance:
    * git repo dir can be overridden by option `--git-repo-dir=xxxx`, but helm dir must be created manually. 
```shell
# helm dir
$ mkdir /charts
$ chmod 777 /charts
# git repo dir if you don't set command line option
$ mkdir /tidb-operator
$ chmod 777 /tidb-operator
```
3. **DNS and network issue**: Two-way proxy using Telepresence. We cannot resolve cluster dns name and access cluster ip easily, `telepresence` helps with that, it creates a proxy pod in the cluster and open a vpn connection to kubernetes cluster via this pod. Just run ([full documentations](https://www.telepresence.io/reference/install)):
```shell
$ brew cask install osxfuse
$ brew install datawire/blackbird/telepresence
$ telepresence --new-deployment ${POD_NAME}
``` 
**PS**: If you cannot resolve cluster dns names after set up, try clear DNS cache.
**PSS**: Typically you can't use telepresence VPN mode with other VPNs (of course SSR is ok).