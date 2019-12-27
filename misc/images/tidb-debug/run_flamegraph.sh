#!/bin/bash

set -e

perf record -F 99 -p $1 -g -- sleep 60
perf script >out.perf
/opt/FlameGraph/stackcollapse-perf.pl out.perf >out.folded
/opt/FlameGraph/flamegraph.pl out.folded >kernel.svg
curl --upload-file ./kernel.svg https://transfer.sh/kernel.svg
