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

package tidb

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"

	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/jwt"
)

var _ = ginkgo.Describe("TiDB", label.TiDB, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("Bootstrap SQL", label.P1, label.FeatureBootstrapSQL, func() {
		sql := "SET PASSWORD FOR 'root'@'%' = 'pingcap';"

		f.SetupBootstrapSQL(sql)
		f.SetupCluster(data.WithBootstrapSQL())
		workload := f.SetupWorkload()

		ginkgo.It("support init a cluster with bootstrap SQL specified", func(ctx context.Context) {
			ginkgo.By("Creating components")
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			workload.MustPing(ctx, data.DefaultTiDBServiceName, "root", "pingcap")
		})
	})

	ginkgo.Context("Auth token", label.P1, label.FeatureAuthToken, func() {
		const (
			kid   = "the-key-id-0"
			sub   = "user@pingcap.com"
			email = "user@pingcap.com"
			iss   = "issuer-abc"
		)
		sql := fmt.Sprintf(
			`CREATE USER '%s' IDENTIFIED WITH 'tidb_auth_token' REQUIRE TOKEN_ISSUER '%s' ATTRIBUTE '{"email": "%s"}';
GRANT ALL PRIVILEGES ON *.* TO '%s'@'%s';`, sub, iss, email, sub, "%")

		f.SetupBootstrapSQL(sql)
		f.SetupCluster(data.WithBootstrapSQL())
		workload := f.SetupWorkload()

		ginkgo.It("should connect to the TiDB cluster with JWT authentication", func(ctx context.Context) {
			token, err := jwt.GenerateJWT(kid, sub, email, iss)
			if err != nil {
				// ??
				ginkgo.Skip(fmt.Sprintf("failed to generate JWT token: %v", err))
			}
			jwksSecret := jwt.GenerateJWKSSecret(f.Namespace.Name, data.JWKsSecretName)
			f.Must(f.Client.Create(ctx, &jwksSecret))

			ginkgo.By("Creating components")
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx, data.WithAuthToken())

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			workload.MustPing(ctx, data.DefaultTiDBServiceName, sub, token)
		})
	})
})
