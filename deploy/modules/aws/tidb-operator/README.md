The `tidb-operator` module for AWS spins up a control plane for TiDB in Kubernetes. The following resources will be provisioned:

- An EKS cluster
- An auto scaling group to run the control pods listed below
- local-volume-provisioner
- Tiller for Helm
