# TiDB Operator

[![Build Status](https://internal.pingcap.net/jenkins/job/build_tidb_operator_master/badge/icon)](https://internal.pingcap.net/jenkins/job/build_tidb_operator_master)

TiDB Operator manages [TiDB](https://github.com/pingcap/tidb) clusters on [Kubernetes](https://kubernetes.io) and automates tasks related to operating a TiDB cluster. It makes TiDB a truly cloud native database.


## Features

- __Safely scaling TiDB cluster__

    TiDB Operator empowers horizontal scalability of TiDB on the cloud.

- __Rolling upgrade TiDB cluster__

    Graceful rolling upgrade TiDB cluster in order, making zero-downtime of TiDB cluster.

- __Multi-tenant support__

    Users can deploy and manage multiple TiDB cluster on a single Kubernetes easily.

- __Auto fail-over__ (WIP)

    TiDB Operator will automatically do fail-over for your TiDB cluster when node failures.

- __Support Kubernetes package manager__

    By embracing Kubernetes package manager [Helm](https://helm.sh), users can easily deploy TiDB clusters with only one command.

- __Automatically monitoring TiDB cluster at creating__

    Automatically deploy Prometheus, Grafana for TiDB cluster monitoring.

## Architecture

## Roadmap

Read the [Roadmap](./ROADMAP.md).

## Quick start

Read the [local-dind-tutorial](./docs/local-dind-tutorial.md).

## Contributing

Contributions are welcomed and greatly appreciated. See [CONTRIBUTING.md](./docs/CONTRIBUTING.md) for details on submitting patches and the contribution workflow.

## License
TiDB is under the Apache 2.0 license. See the [LICENSE](./LICENSE) file for details.
