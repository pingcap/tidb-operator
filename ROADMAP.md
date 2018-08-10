# TiDB Operator Roadmap

This document defines the roadmap for TiDB Operator development.

- [x] Synchronize cluster meta info to PV/PVC labels
- [x] Upgrade TiDB cluster in order: PD -> TiKV -> TiDB
- [x] Safely scale down the TiDB cluster
- [ ] Gracefully upgrade PD/TiKV/TiDB: evict the Raft leader or the DDL owner before upgrade
- [ ] Automatic failover
- [ ] Customize the LoadBalancer service on public cloud
- [ ] Full Local Volume support
