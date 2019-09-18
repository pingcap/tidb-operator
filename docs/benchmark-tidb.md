# Benchmark TiDB

## Table Of Contents

- [Prerequisites](#prerequisites)
- [Sysbench](#sysbench)
  * [Customize with kustomize](#customize-with-kustomize)
    + [More customizations](#more-customizations)
  * [Prepare: Create test tables and insert test records](#prepare-create-test-tables-and-insert-test-records)
  * [Warmup TiDB](#warmup-tidb)
  * [Run benchmark](#run-benchmark)

## Prerequisites

You must run a TiDB cluster in your Kubernetes environment first. You can follow our
docs [here](https://pingcap.com/docs/v3.0/tidb-in-kubernetes/deploy/prerequisites/).

Please note that it's recommended to prepare a dedicated node to run your
benchmark tool, e.g. sysbench.

We also require [kustomize] to customize our manifest yaml files. You can
follow
[this](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md)
to install latest kustomize on your machine.

## Sysbench

### Customize with kustomize

We assume you are working in the root of tidb-operator repo. If you don't have
a copy, you can run `git clone --depth=1 https://github.com/pingcap/tidb-operator`
to clone it onto your local machine.

Configure environments:

```
TIDB_HOST="<your-tidb-host>" # this can be tidb service name, pod ip or load balancer ip, etc.
```

Create `kustomization.yaml`:

```
mkdir -p sysbench/output
cat <<EOF > sysbench/kustomization.yaml
resources:
- ../manifests/sysbench
commonAnnotations:
  tidb-host: $TIDB_HOST
EOF
```

Optionally, if you want to change tidb user, port, password or other
configurations you can customize them in `commonAnnotations` or `patches`
fields:

```
commonAnnotations:
  tidb-password: "root"
patches:
- patch: |-
    apiVersion: batch/v1
    kind: Job
    metadata:
      name: sysbench-bench
    spec:
      template:
        metadata:
          annotations:
            sysbench-threads: "100"
            sysbench-test-name: oltp_read_write
```

Label the node to run sysbench:

```
kubectl label nodes <sysbench-node> dedicated=sysbench
```

Run `kustomize`:

```
kustomize build sysbench/ -o sysbench/output/
```

#### More customizations

Change sysbench image, here is an example:

```
patches:
- patch: |-
    - op: replace
      path: /spec/template/spec/containers/0/image
      value: your-sysbench-image
  target:
    kind: Job
```

### Prepare: Create test tables and insert test records

```
kubectl apply -f sysbench/output/*config.yaml
kubectl apply -f sysbench/output/*prepare.yaml
```

You can run `kubectl logs -l job-name=sysbench-prepare` to view logs.

### Warmup TiDB

```
kubectl apply -f sysbench/output/*warmup.yaml
```

You can run `kubectl logs -l job-name=sysbench-warmup` to view logs.

### Run benchmark

```
kubectl apply -f sysbench/output/*bench.yaml
```

You can run `kubectl logs -l job-name=sysbench-bench` to view logs.
