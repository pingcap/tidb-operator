---
name: "\U0001F41B Bug Report"
about: Something isn't working as expected

---

## Bug Report

**What version of Kubernetes are you using?**
<!-- You can run `kubectl version` -->

**What version of TiDB Operator are you using?**
<!-- You can run `kubectl exec -n tidb-admin {tidb-controller-manager-pod-name} -- tidb-controller-manager -V` -->

**What storage classes exist in the Kubernetes cluster and what are used for PD/TiKV pods?**
<!-- You can run `kubectl get sc` and `kubectl get pvc -n {tidb-cluster-namespace}` -->

**What's the status of the TiDB cluster pods?**
<!-- You can run `kubectl get po -n {tidb-cluster-namespace} -o wide` -->

**What did you do?**
<!-- If possible, provide a recipe for reproducing the error. How you installed tidb-operator and tidb-cluster. -->

**What did you expect to see?**

**What did you see instead?**
