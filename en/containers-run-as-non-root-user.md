---
title: Run TiDB Operator and TiDB Clusters as a Non-root User
summary: Make TiDB Operator related containers run as a non-root user
---

# Run TiDB Operator and TiDB Clusters as a Non-root User

In some Kubernetes environments, containers cannot be run as the root user. In this case, you can set [`securityContext`](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod) to run containers as a non-root user.

## Configure TiDB Operator containers

For TiDB Operator containers, you can configure security context in the helm `values.yaml` file. All TiDB Operator components (at `<controllerManager/scheduler/advancedStatefulset/admissionWebhook>.securityContext`) support this configuration.

The following is an example configuration:

```yaml
controllerManager:
  securityContext:
    runAsUser: 1000
    runAsGroup: 2000
    fsGroup: 2000
```

## Configure containers controlled by CR

For the containers controlled by CR, you can configure security context in any CRs (TidbCluster/DMCluster/TiInitializer/TiMonitor/Backup/BackupSchedule/Restore) to make the containers run as a non-root user.

You can either configure `podSecurityContext` at a cluster level (`spec.podSecurityContext`) for all components or at a component level (such as `spec.tidb.podSecurityContext` for TidbCluster and `spec.master.podSecurityContext` for DMCluster) for a specific component.

The following is an example configuration at a cluster level:

```yaml
spec:
  podSecurityContext:
    runAsUser: 1000
    runAsGroup: 2000
    fsGroup: 2000
```

The following is an example configuration at a component level:

```yaml
spec:
  pd:
    podSecurityContext:
      runAsUser: 1000
      runAsGroup: 2000
      fsGroup: 2000
  tidb:
    podSecurityContext:
      runAsUser: 1000
      runAsGroup: 2000
      fsGroup: 2000
```

For a component, if both the cluster level and the component level are configured, only the configuration of the component level takes effect.
