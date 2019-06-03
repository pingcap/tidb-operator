# TiDB Operator

- [**Stack Overflow**](https://stackoverflow.com/questions/tagged/tidb)
- [**Community Slack Channel**](https://join.slack.com/t/tidbcommunity/shared_invite/enQtNjIyNjA5Njk0NTAxLTVmZDkxOWY1ZGZhMDg3YzcwNGU0YmM4ZjIyODRhOTg4MWEwZjJmMGQzZTJlNjllMGY1YzdlNzIxZGE2NzRlMGY)
- [**Reddit**](https://www.reddit.com/r/TiDB/)
- **Mailing list**: [Google Group](https://groups.google.com/forum/#!forum/tidb-user)
- [**Blog**](https://www.pingcap.com/blog/)
- [**For support, please contact PingCAP**](http://bit.ly/contact_us_via_github)

[![Build Status](https://internal.pingcap.net/idc-jenkins/job/build_tidb_operator_master/badge/icon)](https://internal.pingcap.net/idc-jenkins/job/build_tidb_operator_master)
[![codecov](https://codecov.io/gh/pingcap/tidb-operator/branch/master/graph/badge.svg)](https://codecov.io/gh/pingcap/tidb-operator)

TiDB Operator manages [TiDB](https://github.com/pingcap/tidb) clusters on [Kubernetes](https://kubernetes.io) and automates tasks related to operating a TiDB cluster. It makes TiDB a truly cloud-native database.

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

Choose one of the following tutorials:

* [Deploy TiDB using Kubernetes on Your Laptop for deployment and testing](./docs/local-dind-tutorial.md)

* [Deploy TiDB by launching a Google Kubernetes Engine](./docs/google-kubernetes-tutorial.md):

  [![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.png)](https://console.cloud.google.com/cloudshell/open?git_repo=https://github.com/pingcap/tidb-operator&tutorial=docs/google-kubernetes-tutorial.md)

* [Deploy TiDB by launching an AWS EKS cluster](./docs/aws-eks-tutorial.md)

* [Deploy TiDB Operator and TiDB Cluster on Alibaba Cloud Kubernetes](/deploy/aliyun/README.md)

* [Deploy TiDB in the minikube cluster](./docs/minikube-tutorial.md)

## User guide

Read the [user guide](./docs/user-guide.md).

## Contributing

Contributions are welcome and greatly appreciated. See [CONTRIBUTING.md](./docs/CONTRIBUTING.md) for details on submitting patches and the contribution workflow.

## License

TiDB is under the Apache 2.0 license. See the [LICENSE](./LICENSE) file for details.
