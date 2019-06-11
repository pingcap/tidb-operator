# TiDB cluster debug toolkit

TiDB cluster debug toolkit is a docker image contains various troubleshooting tools for TiDB cluster.

# Usage

```shell
$ docker run -it --rm pingcap/tidb-debug:latest
```

## GDB and perf

This image includes useful troubleshooting tools like [GDB](https://www.gnu.org/software/gdb/) and [perf](https://en.wikipedia.org/wiki/Perf_(Linux)). However, using these tools in debug container is slightly different with the ordinary workflow due to the difference in root filesystems of the target container and the debug container.

### GDB

In order to use GDB properly, you must set the "program" argument to the binary in the target container and set sysroot to the target container's root dir by using gdb's `set sysroot` command. Moreover, if the target container is missing some dynamic libraries (e.g. `libthread_db-*.so`) required by GDB, you must set the corresponding search path to the debug container.

Taking TiKV as an example:

```shell
$ tkctl debug demo-tikv-0
$ gdb /proc/${pid:-1}/root/tikv-server 1

# .gdbinit in the debug container is configured to set sysroot to /proc/1/root/
# so if the target process pid is 1, you can omit this command
(gdb) set sysroot /proc/${pid:-1}/root/

# now you can start debugging
(gdb) thread apply all bt
(gdb) info threads
```

### perf (and flame graph)

To use `perf` and the `run_flamegraph.sh` script (which wraps the `perf` tool) properly, you must copy the program from the target container to the same location in the debug container:

Still taking TiKV as and example:

```shell
$ cp /proc/1/root/tikv-server /
$ ./run_flamegraph.sh 1
```

