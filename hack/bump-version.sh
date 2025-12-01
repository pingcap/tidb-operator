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

SED_BIN=sed
if [[ "$OSTYPE" == "darwin"* ]]; then
    # Mac OSX, use gsed
    SED_BIN=gsed
fi

# parameters
OPERATOR_OLD="v1.6.3"
OPERATOR_NEW="v1.6.4"
TIDB_OLD="v8.5.3"
TIDB_NEW="v8.5.4"

find ./deploy -name "*\.tf"| xargs $SED_BIN -i "s/$OPERATOR_OLD/$OPERATOR_NEW/g"
find ./charts -name "*\.yaml"| xargs $SED_BIN -i "s/$OPERATOR_OLD/$OPERATOR_NEW/g"

find ./deploy -name "*\.yaml.example"| xargs $SED_BIN -i "s/$TIDB_OLD/$TIDB_NEW/g"
find ./examples -name "*\.yaml"| xargs $SED_BIN -i "s/$TIDB_OLD/$TIDB_NEW/g"
find ./deploy -name "*\.tf"| xargs $SED_BIN -i "s/$TIDB_OLD/$TIDB_NEW/g"
find ./charts -name "*\.yaml"| xargs $SED_BIN -i "s/$TIDB_OLD/$TIDB_NEW/g"
$SED_BIN -i "s/$TIDB_OLD/$TIDB_NEW/g" images/tidb-backup-manager/Dockerfile

find ./ -name "*\.mod" | xargs $SED_BIN -i -E "s/(github.com\/pingcap\/tidb-operator\/.+) ($OPERATOR_OLD)/\1 $OPERATOR_NEW/g"
find ./examples -name "*\.md"| xargs $SED_BIN -i "s/$OPERATOR_OLD/$OPERATOR_NEW/g"
