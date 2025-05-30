#!/usr/bin/env bash
# Copyright 2024 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

tag=${1:-master}
dst=${2:-.}
tmp="./tmp"

mkdir -p "$dst"
mkdir -p "$tmp"

git clone --depth 1 -b "$tag" https://github.com/pingcap/tidb.git "$tmp/tidb"
cp "$tmp/tidb/pkg/metrics/grafana/"*.json "$dst/"
cp "$tmp/tidb/metrics/grafana/"*.json "$dst/"
cp "$tmp/tidb/br/metrics/grafana/"*.json "$dst/"

git clone --depth 1 -b "$tag" https://github.com/tikv/pd.git "$tmp/pd"
cp "$tmp/pd/metrics/grafana/"*.json "$dst/"

git clone --depth 1 -b "$tag" https://github.com/tikv/tikv.git "$tmp/tikv"
cp "$tmp/tikv/metrics/grafana/"*.json "$dst/"

git clone --depth 1 -b "$tag" https://github.com/pingcap/tiflash.git "$tmp/tiflash"
cp "$tmp/tiflash/metrics/grafana/"*.json "$dst/"

git clone --depth 1 -b "$tag" https://github.com/pingcap/tiflow.git "$tmp/tiflow"
cp "$tmp/tiflow/metrics/grafana/"*.json "$dst/"

tiproxy_branch="$tag"
if [[ "$tag" == "master" ]]; then
    tiproxy_branch="main"
fi
git clone --depth 1 -b "$tiproxy_branch" https://github.com/pingcap/TiProxy.git "$tmp/TiProxy"
cp "$tmp/TiProxy/pkg/metrics/grafana/tiproxy_summary.json" "$dst/"

git clone --depth 1 -b "$tag" https://github.com/pingcap/ticdc.git "$tmp/ticdc"
cp "$tmp/ticdc/metrics/grafana/"*.json "$dst/"

rm -rf "$tmp"
