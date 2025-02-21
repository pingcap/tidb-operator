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
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

type Framework struct {
	Namespace *corev1.Namespace
	Cluster   *v1alpha1.Cluster

	Client client.Client

	podLogClient rest.Interface

	clusterPatches []data.ClusterPatch
}

func New() *Framework {
	return &Framework{}
}

func (f *Framework) Setup() {
	// TODO: get context and config path from options
	cfg, err := newConfig("", "")
	gomega.Expect(err).To(gomega.Succeed())

	c, err := newClient(cfg)
	gomega.Expect(err).To(gomega.Succeed())
	f.Client = c

	podLogClient, err := newRESTClientForPod(cfg)
	gomega.Expect(err).To(gomega.Succeed())
	f.podLogClient = podLogClient

	ginkgo.BeforeEach(func(ctx context.Context) {
		ns := data.NewNamespace()

		f.Namespace = ns
		ginkgo.By("Creating a namespace")
		f.Must(f.Client.Create(ctx, f.Namespace))

		ginkgo.DeferCleanup(func(ctx context.Context) {
			ginkgo.By("Delete the namespace")
			f.Must(f.Client.Delete(ctx, f.Namespace))

			ginkgo.By("Ensure the namespace can be deleted")
			f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, f.Namespace, waiter.ShortTaskTimeout))
		})
	})

	ginkgo.JustBeforeEach(func(ctx context.Context) {
		f.Cluster = data.NewCluster(f.Namespace.Name, f.clusterPatches...)
		ginkgo.By("Creating a cluster")
		f.Must(f.Client.Create(ctx, f.Cluster))

		ginkgo.DeferCleanup(func(ctx context.Context) {
			ginkgo.By("Delete the cluster")
			f.Must(f.Client.Delete(ctx, f.Cluster))

			ginkgo.By("Ensure the cluster can be deleted")
			f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, f.Cluster, waiter.ShortTaskTimeout))
		})
	})
}

func (f *Framework) SetupCluster(ps ...data.ClusterPatch) {
	ginkgo.BeforeEach(func(context.Context) {
		f.clusterPatches = ps
	})
}

func (f *Framework) MustCreatePD(ctx context.Context, ps ...data.GroupPatch[*runtime.PDGroup]) *v1alpha1.PDGroup {
	pdg := data.NewPDGroup(f.Namespace.Name, ps...)
	ginkgo.By("Creating a pd group")
	f.Must(f.Client.Create(ctx, pdg))

	return pdg
}

func (f *Framework) MustCreateTiKV(ctx context.Context, ps ...data.GroupPatch[*runtime.TiKVGroup]) *v1alpha1.TiKVGroup {
	kvg := data.NewTiKVGroup(f.Namespace.Name, ps...)
	ginkgo.By("Creating a tikv group")
	f.Must(f.Client.Create(ctx, kvg))

	return kvg
}

func (f *Framework) MustCreateTiDB(ctx context.Context, ps ...data.GroupPatch[*runtime.TiDBGroup]) *v1alpha1.TiDBGroup {
	dbg := data.NewTiDBGroup(f.Namespace.Name, ps...)
	ginkgo.By("Creating a tidb group")
	f.Must(f.Client.Create(ctx, dbg))

	return dbg
}

func (f *Framework) SetupBootstrapSQL(sql string) {
	ginkgo.BeforeEach(func(ctx context.Context) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      data.BootstrapSQLName,
				Namespace: f.Namespace.Name,
			},
			Data: map[string]string{
				v1alpha1.ConfigMapKeyBootstrapSQL: sql,
			},
		}
		ginkgo.By("Creating a bootstrap sql configmap")
		f.Must(f.Client.Create(ctx, cm))
		ginkgo.DeferCleanup(func(ctx context.Context) {
			ginkgo.By("Delete the bootstrap sql configmap")
			f.Must(f.Client.Delete(ctx, cm))
		})
	})
}

func (*Framework) Must(err error) {
	gomega.ExpectWithOffset(1, err).To(gomega.Succeed())
}

func (*Framework) True(b bool) {
	gomega.ExpectWithOffset(1, b).To(gomega.BeTrue())
}
