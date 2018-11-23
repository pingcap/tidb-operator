# Troubleshooting

## Some pods are pending for a long time

When a pod is pending, it means the required resources are not satisfied. The most common cases are:

* CPU, memory or storage insufficient

  Check the detail info of the pod by:

  ```shell
  $ kubectl describe po -n <ns> <pod-name>
  ```

  When this happens, either reduce the resource requests of the TiDB cluster and then using `helm` to upgrade the cluster. If the storage request is larger than any of the available volumes, you have to delete the pod and corresponding pending PVC.

* Storage class not exist or no PV available

  You can check this by:

  ```shell
  $ kubectl get pvc -n <ns>
  $ kubectl get pv | grep <storage-class-name> | grep Available
  ```

  When this happens, you can change the `storageClassName` and then using `helm` to upgrade the cluster. After that, delete the pending pods and the corresponding pending PVC and waiting new pod and pvc to be created.
