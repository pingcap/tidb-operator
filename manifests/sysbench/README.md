
# sysbench

## configuration

Modify the configuration in `base/sysbench.conf` and `base/options.conf` 

## execute sysbench job

```shell
kubectl apply -k ${JOB_TYPE} -n ${NS}
```

## delete sysbench job

```shell
kubectl delete -k ${JOB_TYPE} -n ${NS}
```
