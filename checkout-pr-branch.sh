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

# This script is used to checkout a tidb-operator PR branch in a forked repo.
if [[ -z ${1:-} ]]; then
echo "\
This script is used to checkout a tidb-operator PR branch in a forked repo

Usage:
    checkout-pr-branch.sh [github-username]:[pr-branch]

The argument can be copied directly from github PR page.
The local branch name would be [github-username]/[pr-branch].\
" >&2
exit
fi

username=$(echo ${1} | cut -d':' -f1)
branch=$(echo ${1} | cut -d':' -f2)
local_branch=${username}/${branch}
fork="https://github.com/${username}/tidb-operator"

exists=`git show-ref refs/heads/${local_branch}`
if [[ -n ${exists} ]]; then
	git checkout ${local_branch}
	git pull ${fork} ${branch}:${local_branch}
else
	git fetch ${fork} ${branch}:${local_branch}
	git checkout ${local_branch}
fi
