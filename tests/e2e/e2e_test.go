// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"io"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	_ "github.com/pingcap/tidb-operator/tests/e2e/br"
	_ "github.com/pingcap/tidb-operator/tests/e2e/cluster"
	_ "github.com/pingcap/tidb-operator/tests/e2e/example"
	_ "github.com/pingcap/tidb-operator/tests/e2e/pd"
	_ "github.com/pingcap/tidb-operator/tests/e2e/ticdc"
	_ "github.com/pingcap/tidb-operator/tests/e2e/tidb"
	_ "github.com/pingcap/tidb-operator/tests/e2e/tikv"
)

func TestE2E(t *testing.T) {
	ctrl.SetLogger(zap.New(zap.WriteTo(io.Discard)))

	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}
