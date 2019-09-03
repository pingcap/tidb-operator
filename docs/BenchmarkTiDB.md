# Benchmark TiDB

## Table Of Contents

- [Prerequisites](#prerequisites)
- [Sysbench](#sysbench)
  * [Customize with kustomize](#customize-with-kustomize)
  * [Prepare: Create test tables and insert test records](#prepare-create-test-tables-and-insert-test-records)
  * [Warmup: Warmup TiDB](#warmup-warmup-tidb)
  * [Run: Run benchmarks](#run-run-benchmarks)

## Prerequisites

You must run a TiDB cluster in your Kubernetes environment first. You can follow our
docs [here](https://pingcap.com/docs/v3.0/tidb-in-kubernetes/deploy/prerequisites/).

Please note that it's recommended to prepare a dedicated node to run you
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

**Of course, you can edit the files later.**

Create `kustomization.yaml`:

```
mkdir sysbench
cp manifests/sysbench sysbench/manifests -r
cat <<EOF > sysbench/kustomization.yaml
resources:
- ../manifests/sysbench
commonAnnotations:
  tidb-host: $TIDB_HOST
patches:
- path: patch.yaml
  target:
    kind: Job
EOF
```

Create `patch.yaml`:

```
cat <<EOF > sysbench/patch.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: not-important
spec:
  template:
    spec:
      nodeSelector:
        dedicated: sysbench
EOF
```

Label the node to run sysbench:

```
kubectl label nodes <sysbench-node> dedicated=sysbench
```

Run `kustomize`:

```
kustomize build sysbench/ -o sysbench/output/
```

### Prepare: Create test tables and insert test records

```
kubectl apply -f sysbench/output/*config.yaml
kubectl apply -f sysbench/output/*prepare.yaml
```

You can run `kubectl logs -l job-name=sysbench-prepare` to view logs.

### Warmup: Warmup TiDB

```
kubectl apply -f sysbench/output/*warmup.yaml
```

You can run `kubectl logs -l job-name=sysbench-warmup` to view logs.

### Run: Run benchmarks

```
kubectl apply -f sysbench/output/*run.yaml
```

You can run `kubectl logs -l job-name=sysbench-run` to view logs.
