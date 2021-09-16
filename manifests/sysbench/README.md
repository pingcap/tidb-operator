
# sysbench

## configuration

Modify the configuration in `base/sysbench.conf` and `base/options.conf` 

## execute sysbench job

```shell
kubectl apply -k ${SYSBENCH_TYPE} -n ${NAMESPACE}
```

## delete sysbench job

```shell
kubectl delete -k ${SYSBENCH_TYPE} -n ${NAMESPACE}
```
