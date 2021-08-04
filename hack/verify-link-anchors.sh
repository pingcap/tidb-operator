#!/bin/bash
#
# In addition to verify-links.sh, this script additionally check anchors.
#
# See https://docs.npmjs.com/resolving-eacces-permissions-errors-when-installing-packages-globally if you meet permission problems when executing npm install.

set -euo pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

npm install remark-cli@9.0.0 remark-lint@8.0.0 breeswish/remark-lint-pingcap-docs-anchor

echo "info: checking links anchors under $ROOT directory..."

npx remark --ignore-path .gitignore -u lint -u remark-lint-pingcap-docs-anchor . --frail --quiet
