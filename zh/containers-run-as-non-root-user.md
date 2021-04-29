---
title: 以非 root 用户运行 TiDB Operator 和 TiDB 集群
summary: 以非 root 用户运行所有 TiDB Operator 相关的容器
---

# 以非 root 用户运行 TiDB Operator 和 TiDB 集群

在某些 Kubernetes 环境中，无法用 root 用户运行容器。你可以通过配置 [`securityContext`](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod) 来以非 root 用户运行容器。

## 配置 TiDB Operator 相关的容器

对于 TiDB Operator 相关的容器，你可以在 helm 的 `values.yaml` 文件中配置安全上下文 (security context) 。所有 operator 的相关组件都支持该配置 (`<controllerManager/scheduler/advancedStatefulset/admissionWebhook>.securityContext`)。

以下是一个配置示例:

```yaml
controllerManager:
  securityContext:
    runAsUser: 1000
    runAsGroup: 2000
    fsGroup: 2000
```

## 配置按照 CR 生成的容器

对于按照 CR 生成的容器，你同样可以在任意一种 CR (TidbCluster/DMCluster/TiInitializer/TiMonitor/Backup/BackupSchedule/Restore) 中配置安全上下文 (security context) 。

`podSecurityContext` 可以配置在集群级别 (`spec.podSecurityContext`) 对所有组件生效或者配置在组件级别 (例如，配置 TidbCluster 的 `spec.tidb.podSecurityContext`，配置 DMCluster 的 `spec.master.podSecurityContext`) 仅对该组件生效。

以下是一个集群级别的配置示例：

```yaml
spec:
  podSecurityContext:
    runAsUser: 1000
    runAsGroup: 2000
    fsGroup: 2000
```

以下是一个组件级别的配置示例：

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

如果同时配置了集群级别和组件级别，则该组件以组件级别的配置为准。
