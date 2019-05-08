# Tidb Cluster Control Image

`tidb-control` is a docker image containing the main control tools for tidb cluster: `pd-ctl` and `tidb-ctl`.

# Usage

```shell
$ docker run -it --rm pingcap/tidb-control:latest

# enter docker shell
→ / tidb-ctl
Usage:
  tidb-ctl [flags]
  tidb-ctl [command]

Available Commands:
  base64decode decode base64 value
  decoder      decode key
  etcd         control the info about etcd by grpc_gateway
  help         Help about any command
  mvcc         MVCC Information
  region       Region information
  schema       Schema Information
  table        Table information

Flags:
  -h, --help            help for tidb-ctl
      --host ip         TiDB server host (default 127.0.0.1)
      --pdhost ip       PD server host (default 127.0.0.1)
      --pdport uint16   PD server port (default 2379)
      --port uint16     TiDB server port (default 10080)

Use "tidb-ctl [command] --help" for more information about a command.
→ / pd-ctl --help
Usage of pd-ctl:
      --cacert string   path of file that contains list of trusted SSL CAs.
      --cert string     path of file that contains X509 certificate in PEM format.
  -d, --detach          Run pdctl without readline
      --key string      path of file that contains X509 key in PEM format.
  -u, --pd string       The pd address (default "http://127.0.0.1:2379")
  -V, --version         print version information and exit
pflag: help requested

# mount CA certificate files if necessary
$ docker run -v $(pwd)/ca.pem:$(pwd)/ca.pem -v $(pwd)/client.pem:$(pwd)/client.pem -it --rm pingcap/tidb-control:latest
```

PS: `library/bash` is a `busybox` based image, so some dynamic linked binaries like `tikv-ctl` cannot work in this minimal image. If you are looking for `tikv-ctl` or more tools, take a look at [tidb-debug](../tidb-debug/).
