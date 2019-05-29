# The TiDB Kubernetes Control(tkctl) User Manual

> **Disclaimer**: The tkctl CLI tool is currently **Alpha**. The design and sub-commands may change in the future, use at your own risk.

The TiDB Kubernetes Control(tkctl) is a command line utility for TiDB operators to operate and diagnose their TiDB clusters in Kubernetes.

- [Installation](#installation)
    - [Build from Source](#build-from-source)
    - [Shell Completion](#shell-completion)
    - [Kubernetes Configuration](#kubernetes-configuration)
- [Commands](#commands)
    - [tkctl version](#tkctl-version)
    - [tkctl list](#tkctl-list)
    - [tkctl use](#tkctl-use)
    - [tkctl info](#tkctl-info)
    - [tkctl get](#tkctl-get-component)
    - [tkctl debug](#tkctl-debug-podname)
    - [tkctl ctop](#tkctl-ctop)
    - [tkctl help](#tkctl-help-command)
    - [tkctl options](#tkctl-options)

# Installation

You can download the pre-built binary or build `tkctl` from source:

### Download the Latest Pre-built Binary

- [MacOS](http://download.pingcap.org/tkctl-darwin-amd64-latest.tgz)
- [Linux](http://download.pingcap.org/tkctl-linux-amd64-latest.tgz)
- [Windows](http://download.pingcap.org/tkctl-windows-amd64-latest.tgz)

### Build from Source

```shell
$ git clone https://github.com/pingcap/tidb-operator.git
$ GOOS=${YOUR_GOOS} make cli
$ mv tkctl /usr/local/bin/tkctl
```

## Shell Completion

BASH
```shell
# setup autocomplete in bash into the current shell, bash-completion package should be installed first.
source <(tkctl completion bash) 

# add autocomplete permanently to your bash shell.
echo "if hash tkctl 2>/dev/null; then source <(tkctl completion bash); fi" >> ~/.bashrc 
```

ZSH
```shell
# setup autocomplete in zsh into the current shell
source <(tkctl completion zsh)

# add autocomplete permanently to your zsh shell
echo "if hash tkctl 2>/dev/null; then source <(tkctl completion zsh); fi" >> ~/.zshrc 
```

## Kubernetes Configuration

`tkctl` reuse the kubeconfig(default to `~/.kube/config`) file to talk with kubernetes cluster. You don't have to set up `kubectl` to use `tkctl`, but make sure you have `~/.kube/config` properly set. You can verify the configuration by executing:

```shell
$ tkctl version
```

If you see the version of tkctl tool and version of TiDB operator installed in target cluster or "No TiDB Controller Manager found, please install one first.", `tkctl` is correctly configured to access your cluster.

# Commands

## tkctl version

This command used to show the version of **tkctl** and **tidb-operator** installed in target cluster.

Example:
```
$ tkctl version
Client Version: v1.0.0-beta.1-p2-93-g6598b4d3e75705-dirty
TiDB Controller Manager Version: pingcap/tidb-operator:latest
TiDB Scheduler Version: pingcap/tidb-operator:latest
```

## tkctl list

This command used to list all tidb clusters installed.

| Flags | Shorthand | Description |
| ----- | --------- | ----------- |
| --all-namespaces | -A | search all namespaces |
| --output | -o | output format, one of [default,json,yaml], the default format is `default` |

Example:

```
$ tkctl list -A
NAMESPACE NAME           PD    TIKV   TIDB   AGE
foo       demo-cluster   3/3   3/3    2/2    11m
bar       demo-cluster   3/3   3/3    1/2    11m
```

## tkctl use

This command used to specify the current TiDB cluster to use, the other commands could omit `--tidbcluster` option and defaults to select current TiDB cluster if there is a current TiDB cluster set.

Example:

```
$ tkctl use --namespace=foo demo-cluster
Tidb cluster switched to foo/demo-cluster
```

## tkctl info

This command used to get the information of TiDB cluster, the current TiDB cluster will be used if exists.

| Flags | Shorthand | Description |
| ----- | --------- | ----------- |
| --tidb-cluster | -t | select the tidb cluster, default to current TiDB cluster |

Example:

```
$ tkctl info
Name:               demo-cluster
Namespace:          foo
CreationTimestamp:  2019-04-17 17:33:41 +0800 CST
Overview:
         Phase    Ready  Desired  CPU    Memory  Storage  Version
         -----    -----  -------  ---    ------  -------  -------
  PD:    Normal   3      3        200m   1Gi     1Gi      pingcap/pd:v2.1.4
  TiKV:  Normal   3      3        1000m  2Gi     10Gi     pingcap/tikv:v2.1.4
  TiDB   Upgrade  1      2        500m   1Gi              pingcap/tidb:v2.1.4
Endpoints(NodePort):
  - 172.16.4.158:31441
  - 172.16.4.155:31441
```

## tkctl get [component]

This is a group of commands used to get the details of TiDB cluster componentes, the current TiDB cluster will be used if exists.

Available components: `pd`, `tikv`, `tidb`, `volume`, `all`(query all components)

| Flags | Shorthand | Description |
| ----- | --------- | ----------- |
| --tidb-cluster | -t | select the tidb cluster, default to current TiDB cluster |
| --output | -o | output format, one of [default,json,yaml], the default format is `default` |

Example:

```
$ tkctl get tikv
NAME                  READY   STATUS    MEMORY          CPU   RESTARTS   AGE     NODE
demo-cluster-tikv-0   2/2     Running   2098Mi/4196Mi         0          3m19s   172.16.4.155
demo-cluster-tikv-1   2/2     Running   2098Mi/4196Mi         0          4m8s    172.16.4.160
demo-cluster-tikv-2   2/2     Running   2098Mi/4196Mi         0          4m45s   172.16.4.157
$ tkctl get volume
tkctl get volume
VOLUME              CLAIM                      STATUS   CAPACITY   NODE           LOCAL
local-pv-d5dad2cf   tikv-demo-cluster-tikv-0   Bound    1476Gi     172.16.4.155   /mnt/disks/local-pv56
local-pv-5ade8580   tikv-demo-cluster-tikv-1   Bound    1476Gi     172.16.4.160   /mnt/disks/local-pv33
local-pv-ed2ffe50   tikv-demo-cluster-tikv-2   Bound    1476Gi     172.16.4.157   /mnt/disks/local-pv13
local-pv-74ee0364   pd-demo-cluster-pd-0       Bound    1476Gi     172.16.4.155   /mnt/disks/local-pv46
local-pv-842034e6   pd-demo-cluster-pd-1       Bound    1476Gi     172.16.4.158   /mnt/disks/local-pv74
local-pv-e54c122a   pd-demo-cluster-pd-2       Bound    1476Gi     172.16.4.156   /mnt/disks/local-pv72
```

## tkctl debug [pod_name]

This command used to diagnose the Pods of TiDB cluster. It launches a debug container for you which has the nessary troubleshooting tools installed.

| Flags | Shorthand | Description |
| ----- | --------- | ----------- |
| --image |    | specify the docker image of debug container, default to `pingcap/tidb-debug:lastest` |
| --container | -c | select the container to diagnose, default to the first container of target Pod |
| --docker-socket |    | specify the docker socket of cluster node, default to `/var/run/docker.sock` |
| --privileged |    | whether launch container in privileged mode (full container capabilities) |


The default image of debug container contains almost all the related tools you may use then diagnosing, however, the image size can be kinda big. You may use `--image=pingcap/tidb-control:latest` if your just need a basic shell, `pd-ctl` and `tidb-ctl`.

For the guide of using the default debug image (`tidb-debug`), refer to [tidb-debug](/misc/images/tidb-debug/README.md).

Example:
```
$ tkctl debug demo-cluster-tikv-0
# you may have to wait a few seconds or minutes for the debug container running, then you will get the shell prompt
```

## tkctl ctop

`tkctl ctop [pod_name | node/node_name ]`

This command used to view the real-time stats of target pod or node. Compare to `kubectl top`, `tkctl ctop` provides network and disk stats, which are important for diagnosing TiDB cluster problem.

| Flags | Shorthand | Description |
| ----- | --------- | ----------- |
| --image |    | specify the docker image of ctop, default to `quay.io/vektorlab/ctop:0.7.2` |
| --docker-socket |    | specify the docker socket of cluster node, default to `/var/run/docker.sock` |

Example:

```
$ tkctl ctop demo-cluster-tikv-0
$ tkctl ctop node/172.16.4.155
```

If you don't see the prompt, please wait a few seconds or minutes.

## tkctl help [command]

This command used to print the help message of abitrary sub command.

```
$ tkctl help debug
```

## tkctl options

This command used to view the global flags of `tkctl`.

Example:
```
$ tkctl options
The following options can be passed to any command:

      --alsologtostderr=false: log to standard error as well as files
      --as='': Username to impersonate for the operation
      --as-group=[]: Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --cache-dir='/Users/alei/.kube/http-cache': Default HTTP cache directory
      --certificate-authority='': Path to a cert file for the certificate authority
      --client-certificate='': Path to a client certificate file for TLS
      --client-key='': Path to a client key file for TLS
      --cluster='': The name of the kubeconfig cluster to use
      --context='': The name of the kubeconfig context to use
      --insecure-skip-tls-verify=false: If true, the server's certificate will not be checked for validity. This will
make your HTTPS connections insecure
      --kubeconfig='': Path to the kubeconfig file to use for CLI requests.
      --log_backtrace_at=:0: when logging hits line file:N, emit a stack trace
      --log_dir='': If non-empty, write log files in this directory
      --logtostderr=true: log to standard error instead of files
  -n, --namespace='': If present, the namespace scope for this CLI request
      --request-timeout='0': The length of time to wait before giving up on a single server request. Non-zero values
should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests.
  -s, --server='': The address and port of the Kubernetes API server
      --stderrthreshold=2: logs at or above this threshold go to stderr
  -t, --tidbcluster='': Tidb cluster name
      --token='': Bearer token for authentication to the API server
      --user='': The name of the kubeconfig user to use
  -v, --v=0: log level for V logs
      --vmodule=: comma-separated list of pattern=N settings for file-filtered logging
```
These options are mainly used to talk with the kubernetes cluster, there are two options that used often:

- `--context`: choose the kubernetes cluster
- `--namespace`: choose the kubernetes namespace

