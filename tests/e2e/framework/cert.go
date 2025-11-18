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

package framework

import (
	"context"

	"github.com/onsi/ginkgo/v2"

	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/cert"
)

type CertManager struct {
	f *Framework

	cf cert.Factory
}

func (f *Framework) SetupCertManager() *CertManager {
	w := &CertManager{
		f:  f,
		cf: cert.NewFactory(f.Client),
	}
	w.deferCleanup()

	return w
}

// Install is called to install all certs of the whole cluster
// NOTE: Install must be called after all groups have been created
func (cm *CertManager) Install(ctx context.Context, ns, cluster string) {
	cm.f.Must(cm.cf.Install(ctx, ns, cluster))
}

func (cm *CertManager) deferCleanup() {
	ginkgo.BeforeEach(func(ctx context.Context) {
		ginkgo.DeferCleanup(func(ctx context.Context) {
			cm.f.Must(cm.cf.Cleanup(ctx))
		})
	})
}
