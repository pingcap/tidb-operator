---
name: "\U0001F41B Bug Report"
about: Something isn't working as expected

---

## Bug Report

**What version of Kubernetes are you using?**
<!-- You can run `kubectl version` -->

**What storage classes exist in the Kubernetes cluster?**
<!-- You can run `kubectl get sc` -->

**What version of TiDB Operator are you using?**
<!-- You can run `kubectl exec -n tidb-admin {tidb-controller-manager-pod-name} -- tidb-controller-manager -V` -->

**What storage classes are used for PD and TiKV?**
<!-- You can run `kubectl get pvc -n {tidb-cluster-namespace}` -->

**What's the status of the TiDB cluster pods?**
<!-- You can run `kubectl get po -n {tidb-cluster-namespace} -o wide` -->

**What did you do?**
<!-- If possible, provide a recipe for reproducing the error. A complete runnable program is good. -->

**What did you expect to see?**

**What did you see instead?**
