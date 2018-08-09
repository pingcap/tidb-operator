# TiDB Operator Roadmap

This document defines the roadmap for TiDB Operator development.

- [x] Synchronize cluster meta info to PV/PVC labels
- [x] Upgrade TiDB cluster in order: PD -> TiKV -> TiDB
- [x] Safely scale down TiDB cluster
- [ ] Graceful upgrade PD/TiKV/TiDB: evict raft leader or DDL owner before upgrade
- [ ] Auto fail-over
- [ ] Customize LoadBalancer service on public cloud
- [ ] Full Local Volume support
