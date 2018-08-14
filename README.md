# TiDB Operator

[![Build Status](https://internal.pingcap.net/jenkins/job/build_tidb_operator_master/badge/icon)](https://internal.pingcap.net/jenkins/job/build_tidb_operator_master)

TiDB Operator manages [TiDB](https://github.com/pingcap/tidb) clusters on [Kubernetes](https://kubernetes.io) and automates tasks related to operating a TiDB cluster. It makes TiDB a truly cloud native database.

## Features

- __Safely scaling the TiDB cluster__

    TiDB Operator empowers TiDB with horizontal scalability on the cloud.

- __Rolling update of the TiDB cluster__

    Gracefully perform rolling updates for the TiDB cluster in order, achieving zero-downtime of the TiDB cluster.

- __Multi-tenant support__

    Users can deploy and manage multiple TiDB clusters on a single Kubernetes cluster easily.

- __Automatic failover__ (WIP)

    TiDB Operator automatically performs failover for your TiDB cluster when node failures occur.

- __Kubernetes package manager support__

    By embracing Kubernetes package manager [Helm](https://helm.sh), users can easily deploy TiDB clusters with only one command.

- __Automatically monitoring TiDB cluster at creating__

    Automatically deploy Prometheus, Grafana for TiDB cluster monitoring.

## Roadmap

Read the [Roadmap](./ROADMAP.md).

## Quick start

Read the [local-dind-tutorial](./docs/local-dind-tutorial.md).

## Contributing

Contributions are welcomed and greatly appreciated. See [CONTRIBUTING.md](./docs/CONTRIBUTING.md) for details on submitting patches and the contribution workflow.

## License

TiDB is under the Apache 2.0 license. See the [LICENSE](./LICENSE) file for details.
