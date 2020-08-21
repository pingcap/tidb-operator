The `tidb-cluster` module for AWS spins up a TiDB cluster in the specified `EKS` cluster. The following resources will be provisioned:

- An auto scaling group for PD
- An auto scaling group for TiKV
- An auto scaling group for TiDB
- An auto scaling group for Monitoring
- A `TidbCluster` custom resource
