# How to add basic auth to remotewrite example

> **Note:**
>
> this example describe how to add basic authorization to tidbMonitor remoteWrite configuration.
> we use prometheus as remote write receiver,the prometheus managed by [prometheus-operator](https://github.com/coreos/prometheus-operator).
> we will add authentication to the prometheus remote write receiver by ingress-nginx controller.

**Prerequisites**: 
- Has Prometheus operator installed. [Doc](https://github.com/coreos/kube-prometheus)
    
  This could by verified by the following command:
  
  ```bash
  >  kubectl get crd |grep coreos.com
  ```
  
  The output is similar to this:
  
  ```bash
        NAME                                       TIME
    alertmanagers.monitoring.coreos.com            2020-04-12T14:22:49Z
    podmonitors.monitoring.coreos.com              2020-04-12T14:22:50Z
    prometheuses.monitoring.coreos.com             2020-04-12T14:22:50Z
    prometheusrules.monitoring.coreos.com          2020-04-12T14:22:50Z
    servicemonitors.monitoring.coreos.com          2020-04-12T14:22:51Z
  ```

- Has Ingress nginx controller installed. [Doc](https://kubernetes.github.io/ingress-nginx/)

  This could by verified by the following command:

  ```bash
  >  kubectl get pods -n ingress-nginx 
  ```

  The output is similar to this:

  ```bash
        NAME                                       TIME
  ingress-nginx-admission-create-8ljj2       0/1     Completed   0          79m
  ingress-nginx-admission-patch-mcn8r        0/1     Completed   0          79m
  ingress-nginx-controller-cb87575f5-sd6kb   1/1     Running     0          79m

  ``` 
## Initialize

### Initialize remote write prometheus
```bash
> kubectl -n <namespace> apply -f ./prometheus.yaml
> kubectl -n <namespace> apply -f ./prometheus-service.yaml
```

Wait for Initialize remote write prometheus done:

```bash
$ kubectl get pods -n <namespace>| grep prometheus-basic
NAME         AGE
prometheus-basic-0                 2/2     Running   0          83m
```

### Create auth ingress
Ref [Doc](https://kubernetes.github.io/ingress-nginx/examples/auth/basic/).

First step: create username and password.
```bash
> htpasswd -c auth admin
New password: <admin>
New password:
Re-type new password:
Adding password for user admin
```

Second step: create ingress.
```bash
> kubectl apply -f ./ingress.yaml -n <namespace>
```

Third step: validate auth.
```bash
> kubectl port-forward svc/ingress-nginx-controller 8080:80 -n ingress-nginx
Use curl with the correct credentials to connect to the ingress
> curl -v  http://127.0.0.1:8080 -H 'Host: test.prometheus.com' -u 'admin:admin'
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to 127.0.0.1 (127.0.0.1) port 8080 (#0)
* Server auth using Basic with user 'admin'
> GET / HTTP/1.1
> Host: test.prometheus.com
> Authorization: Basic YWRtaW46YWRtaW4=
> User-Agent: curl/7.64.1
> Accept: */*
> 
< HTTP/1.1 302 Found
< Date: Sun, 13 Mar 2022 17:44:12 GMT
< Content-Type: text/html; charset=utf-8
< Content-Length: 29
< Connection: keep-alive
< Location: /graph
< 
<a href="/graph">Found</a>.

* Connection #0 to host 127.0.0.1 left intact
* Closing connection 0
```

### Create tidbMonitor and TiDB

Create TiDB:
```bash
> kubectl apply -f tidb-cluster.yaml
```

Create auth secret and TiDBMonitor:
```bash
> kubectl apply -f basic-auth.yaml
> kubectl apply -f tidb-monitor.yaml
```

Add hostAliases to tidbmonitor statefulset:
```bash
> kubectl edit sts basic-monitor -n <namespace>
Add dns `test.prometheus.com` and ingress nginx svc ip map to `hostAliases` field.
For exmaple:
      hostAliases:
      - hostnames:
        - test.prometheus.com
        ip: 10.96.223.84
```

Delete tidbmonitor pod and recreate manually.
```bash
> kubectl delete pods basic-monitor-0 -n <namespace>
```

### Validate remotewrite prometheus data

Port-Forward remotewrite prometheus svc port 9090 to local port 9090.
```bash
> kubectl port-forward svc/basic-prometheus-remote  9090:9090
```
The data is writing successfully.

## Destroy

```bash
> kubectl -n <namespace> delete -f ./
```
