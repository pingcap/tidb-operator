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

## Quick Start

You can follow our [Get Started](https://docs.pingcap.com/tidb-in-kubernetes/stable/get-started) guide to quickly start a testing Kubernetes cluster and play with TiDB Operator on your own machine.

## Documentation

You can see our documentation at PingCAP website for more in-depth installation and instructions for production:

- [English](https://docs.pingcap.com/tidb-in-kubernetes/stable)
- [简体中文](https://docs.pingcap.com/zh/tidb-in-kubernetes/stable)

All the TiDB Operator documentation is maintained in the [docs-tidb-operator repository](https://github.com/pingcap/docs-tidb-operator). 

## Community

Feel free to reach out if you have any questions. The maintainers of this project are reachable via:

- [TiDB Community Slack](https://pingcap.com/tidbslack/) in the [#sig-k8s](https://tidbcommunity.slack.com/archives/CHD0HA3LZ) channel
- [Filling an issue](https://github.com/pingcap/tidb-operator/issue) against this repo

Pull Requests are welcome! Check the [issue tracker](https://github.com/pingcap/tidb-operator/issue) for `status/help-wanted` issues if you're unsure where to start.

If you're planning a new feature, please file an issue or join [#sig-k8s](https://tidbcommunity.slack.com/archives/CHD0HA3LZ) channel to discuss first.

## Contributing

Contributions are welcome and greatly appreciated. See [CONTRIBUTING.md](./docs/CONTRIBUTING.md) for details on submitting patches and the contribution workflow.

## License

TiDB is under the Apache 2.0 license. See the [LICENSE](./LICENSE) file for details.
