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


set -o errexit
set -o nounset
set -o pipefail

ROOT=$(cd $(dirname "${BASH_SOURCE[0]}")/..; pwd -P)
source $ROOT/hack/lib/vars.sh

OUTPUT=$ROOT/_output/manifests
HELM=$ROOT/_output/bin/helm
CRDS=$OUTPUT/tidb-operator.crds.yaml
OPERATOR=$OUTPUT/tidb-operator.yaml
BOILERPLATE=$ROOT/hack/boilerplate/boilerplate.yaml.txt

mkdir -p $OUTPUT

echo "Combine CRDs into tidb-operator.crds.yaml"
cat $BOILERPLATE > $CRDS
for f in $ROOT/manifests/crd/*.yaml; do
    echo "append $f"
    cat $f >> $CRDS
done


echo "Generate tidb-operator.yaml"
cat << EOF > $OUTPUT/helm-values.yaml
operator:
  image: pingcap/tidb-operator:${V_RELEASE}
backupManager:
  image: pingcap/tidb-backup-manager:${V_RELEASE}
EOF

cat $BOILERPLATE > $OPERATOR
cat << EOF >> $OPERATOR
---
apiVersion: v1
kind: Namespace
metadata:
  name: tidb-admin
EOF

${HELM} template tidb-operator $ROOT/charts/tidb-operator \
    -n ${V_DEPLOY_NAMESPACE} \
    -f $OUTPUT/helm-values.yaml \
    --kube-version ${V_KUBE_VERSION} \
    >> $OPERATOR
