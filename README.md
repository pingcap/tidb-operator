# TiDB Operator

- [**Stack Overflow**](https://stackoverflow.com/questions/tagged/tidb)
- [**Community Slack Channel**](https://pingcap.com/tidbslack/)
- [**Reddit**](https://www.reddit.com/r/TiDB/)
- **Mailing list**: [Google Group](https://groups.google.com/forum/#!forum/tidb-user)
- [**Blog**](https://www.pingcap.com/blog/)
- [**For support, please contact PingCAP**](http://bit.ly/contact_us_via_github)

[![Build Status](https://internal.pingcap.net/idc-jenkins/job/tidb-operator-master/badge/icon)](https://internal.pingcap.net/idc-jenkins/job/tidb-operator-master)
[![codecov](https://codecov.io/gh/pingcap/tidb-operator/branch/master/graph/badge.svg)](https://codecov.io/gh/pingcap/tidb-operator)
[![LICENSE](https://img.shields.io/github/license/pingcap/tidb-operator.svg)](https://github.com/pingcap/tidb-operator/blob/master/LICENSE)
[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/pingcap/tidb-operator)](https://goreportcard.com/report/github.com/pingcap/tidb-operator)
[![GitHub release](https://img.shields.io/github/tag/pingcap/tidb-operator.svg?label=release)](https://github.com/pingcap/tidb-operator/releases)
[![GoDoc](https://img.shields.io/badge/Godoc-reference-blue.svg)](https://godoc.org/github.com/pingcap/tidb-operator)

TiDB Operator manages [TiDB](https://github.com/pingcap/tidb) clusters on [Kubernetes](https://kubernetes.io) and automates tasks related to operating a TiDB cluster. It makes TiDB a truly cloud-native database.

![TiDB Operator Architecture](/static/tidb-operator-overview.png)

## Features

- __Safely scaling the TiDB cluster__

    TiDB Operator empowers TiDB with horizontal scalability on the cloud.

- __Rolling update of the TiDB cluster__

    Gracefully perform rolling updates for the TiDB cluster in order, achieving zero-downtime of the TiDB cluster.

- __Multi-tenant support__

    Users can deploy and manage multiple TiDB clusters on a single Kubernetes cluster easily.

- __Automatic failover__

    TiDB Operator automatically performs failover for your TiDB cluster when node failures occur.

- __Kubernetes package manager support__

    By embracing Kubernetes package manager [Helm](https://helm.sh), users can easily deploy TiDB clusters with only one command.

- __Automatically monitoring TiDB cluster at creating__

    Automatically deploy Prometheus, Grafana for TiDB cluster monitoring.

## Roadmap

Read the [Roadmap](./ROADMAP.md).

## Quick start

Read the [Quick Start Guide](https://pingcap.com/docs/v3.0/tidb-in-kubernetes/tidb-operator-overview/), which includes all the guides for managing TiDB clusters in Kubernetes.

## Documentation

All the TiDB Operator documentation is maintained in the [docs-tidb-operator repository](https://github.com/pingcap/docs-tidb-operator). You can also see the documentation at PingCAP website:

- [English](https://pingcap.com/docs/tidb-in-kubernetes/stable/tidb-operator-overview/)
- [简体中文](https://pingcap.com/docs-cn/tidb-in-kubernetes/stable/tidb-operator-overview/)

## Contributing

Contributions are welcome and greatly appreciated. See [CONTRIBUTING.md](./docs/CONTRIBUTING.md) for details on submitting patches and the contribution workflow.

## License

TiDB is under the Apache 2.0 license. See the [LICENSE](./LICENSE) file for details.
