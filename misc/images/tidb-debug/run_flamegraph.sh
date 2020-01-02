#!/usr/bin/env bash

# Copyright 2020 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

perf record -F 99 -p $1 -g -- sleep 60
perf script >out.perf
/opt/FlameGraph/stackcollapse-perf.pl out.perf >out.folded
/opt/FlameGraph/flamegraph.pl out.folded >kernel.svg
curl --upload-file ./kernel.svg https://transfer.sh/kernel.svg
