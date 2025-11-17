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
	"fmt"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

type Framework struct {
	Namespace *corev1.Namespace
	Cluster   *v1alpha1.Cluster

	Client client.Client

	restConfig *rest.Config
	podClient  rest.Interface

	clusterPatches []data.ClusterPatch
}

func New() *Framework {
	return &Framework{}
}

type SetupOptions struct {
	SkipWaitForClusterDeleted    bool
	SkipWaitForNamespaceDeleted  bool
	SkipClusterCreation          bool
	SkipClusterDeletedWhenFailed bool
}

type SetupOption func(opts *SetupOptions)

func WithSkipWaitForClusterDeleted() SetupOption {
	return func(opts *SetupOptions) {
		opts.SkipWaitForClusterDeleted = true
	}
}

func WithSkipWaitForNamespaceDeleted() SetupOption {
	return func(opts *SetupOptions) {
		opts.SkipWaitForNamespaceDeleted = true
	}
}

func WithSkipClusterCreation() SetupOption {
	return func(opts *SetupOptions) {
		opts.SkipClusterCreation = true
	}
}

func WithSkipClusterDeletionWhenFailed() SetupOption {
	return func(opts *SetupOptions) {
		opts.SkipClusterDeletedWhenFailed = true
	}
}

func (f *Framework) Setup(opts ...SetupOption) {
	options := &SetupOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// TODO: get context and config path from options
	cfg, err := NewConfig("", "")
	gomega.Expect(err).To(gomega.Succeed())
	f.restConfig = cfg

	c, err := newClient(cfg)
	gomega.Expect(err).To(gomega.Succeed())
	f.Client = c

	podClient, err := newRESTClientForPod(cfg)
	gomega.Expect(err).To(gomega.Succeed())
	f.podClient = podClient

	ginkgo.BeforeEach(func(ctx context.Context) {
		ns := data.NewNamespace()

		f.Namespace = ns
		ginkgo.By(fmt.Sprintf("Creating a namespace %s", f.Namespace.Name))
		f.Must(f.Client.Create(ctx, f.Namespace))

		ginkgo.DeferCleanup(func(ctx context.Context) {
			if options.SkipClusterDeletedWhenFailed && ginkgo.CurrentSpecReport().Failed() {
				ginkgo.By(fmt.Sprintf("Case failed, skip cluster deletion: %s", f.Cluster.Name))
			} else {
				ginkgo.By(fmt.Sprintf("Delete the namespace %s", f.Namespace.Name))
				f.Must(f.Client.Delete(ctx, f.Namespace))

				if !options.SkipWaitForNamespaceDeleted {
					ginkgo.By(fmt.Sprintf("Ensure the namespace %s can be deleted", f.Namespace.Name))
					f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, f.Namespace, waiter.LongTaskTimeout))
				}
			}
		})
	})

	if !options.SkipClusterCreation {
		ginkgo.JustBeforeEach(func(ctx context.Context) {
			f.Cluster = data.NewCluster(f.Namespace.Name, f.clusterPatches...)
			ginkgo.By("Creating a cluster")
			f.Must(f.Client.Create(ctx, f.Cluster))

			ginkgo.DeferCleanup(func(ctx context.Context) {
				if options.SkipClusterDeletedWhenFailed && ginkgo.CurrentSpecReport().Failed() {
					ginkgo.By(fmt.Sprintf("Case failed, skip cluster deletion: %s", f.Cluster.Name))
				} else {
					ginkgo.By(fmt.Sprintf("Delete the cluster: %s", f.Cluster.Name))
					f.Must(f.Client.Delete(ctx, f.Cluster))

					if !options.SkipWaitForClusterDeleted {
						ginkgo.By(fmt.Sprintf("Ensure the cluster: %s can be deleted", f.Cluster.Name))
						f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, f.Cluster, waiter.LongTaskTimeout))
					}
				}
			})
		})
	}
}

func (f *Framework) SetupCluster(ps ...data.ClusterPatch) {
	ginkgo.BeforeEach(func(context.Context) {
		f.clusterPatches = ps
	})
	ginkgo.AfterEach(func(context.Context) {
		f.clusterPatches = nil
	})
}

func (f *Framework) MustCreateCluster(ctx context.Context, ps ...data.ClusterPatch) *v1alpha1.Cluster {
	tc := data.NewCluster(f.Namespace.Name, ps...)
	ginkgo.By("Creating a cluster")
	f.Must(f.Client.Create(ctx, tc))

	return tc
}

func (f *Framework) MustCreatePD(ctx context.Context, ps ...data.GroupPatch[*v1alpha1.PDGroup]) *v1alpha1.PDGroup {
	pdg := data.NewPDGroup(f.Namespace.Name, ps...)
	ginkgo.By("Creating a pd group")
	f.Must(f.Client.Create(ctx, pdg))

	return pdg
}

func (f *Framework) MustCreateTiDB(ctx context.Context, ps ...data.GroupPatch[*v1alpha1.TiDBGroup]) *v1alpha1.TiDBGroup {
	dbg := data.NewTiDBGroup(f.Namespace.Name, ps...)
	ginkgo.By("Creating a tidb group")
	f.Must(f.Client.Create(ctx, dbg))

	return dbg
}

func (f *Framework) MustCreateTSO(ctx context.Context, ps ...data.GroupPatch[*v1alpha1.TSOGroup]) *v1alpha1.TSOGroup {
	tg := data.NewTSOGroup(f.Namespace.Name, ps...)
	ginkgo.By("Creating a tso group")
	f.Must(f.Client.Create(ctx, tg))

	return tg
}

func (f *Framework) MustCreateScheduling(ctx context.Context, ps ...data.GroupPatch[*v1alpha1.SchedulingGroup]) *v1alpha1.SchedulingGroup {
	sg := data.NewSchedulingGroup(f.Namespace.Name, ps...)
	ginkgo.By("Creating a scheduler group")
	f.Must(f.Client.Create(ctx, sg))
	return sg
}

func (f *Framework) MustCreateTiProxy(ctx context.Context, ps ...data.GroupPatch[*v1alpha1.TiProxyGroup]) *v1alpha1.TiProxyGroup {
	tpg := data.NewTiProxyGroup(f.Namespace.Name, ps...)
	ginkgo.By("Creating a tiproxy group")
	f.Must(f.Client.Create(ctx, tpg))
	return tpg
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

func (*Framework) True(b bool, opts ...any) {
	gomega.ExpectWithOffset(1, b).To(gomega.BeTrue(), opts...)
}

// DescribeFeatureTable generates a case with different features
func (*Framework) DescribeFeatureTable(f func(fs ...metav1alpha1.Feature), fss ...[]metav1alpha1.Feature) {
	var entries []ginkgo.TableEntry
	for _, fs := range fss {
		var args []any
		args = append(args, label.Features(fs...))
		for _, f := range fs {
			args = append(args, f)
		}
		entries = append(entries, ginkgo.Entry(nil, args...))
	}
	var args []any
	args = append(args, f)
	args = append(args,
		func(fs ...metav1alpha1.Feature) string {
			sb := strings.Builder{}
			for _, f := range fs {
				sb.WriteString("[")
				sb.WriteString(string(f))
				sb.WriteString("]")
			}
			return sb.String()
		},
	)
	for _, entry := range entries {
		args = append(args, entry)
	}
	ginkgo.DescribeTableSubtree("FeatureTable", args...)
}

func (f *Framework) PortForwardPod(ctx context.Context, pod *corev1.Pod, ports []string) []portforward.ForwardedPort {
	readyCh := make(chan struct{})
	pf, err := newPodPortForwarder(ctx, f.restConfig, f.podClient, pod, ports, readyCh, ginkgo.GinkgoWriter)
	f.Must(err)
	go func() {
		defer ginkgo.GinkgoRecover()
		f.Must(pf.ForwardPorts())
	}()
	<-readyCh
	ps, err := pf.GetPorts()
	f.Must(err)

	return ps
}

func AsyncWaitPodsRollingUpdateOnce[
	S scope.Group[F, T],
	F client.Object,
	T runtime.Group,
](ctx context.Context, f *Framework, obj F, to int) chan struct{} {
	maxSurge := 0
	comp := scope.Component[S]()
	switch comp {
	case "tiproxy", "tidb", "ticdc":
		maxSurge = 1
	}
	ch := make(chan struct{})
	nobj := obj.DeepCopyObject().(F)
	go func() {
		defer close(ch)
		defer ginkgo.GinkgoRecover()
		f.Must(waiter.WaitPodsRollingUpdateOnce[S](ctx, f.Client, nobj, to, maxSurge, waiter.LongTaskTimeout))
	}()

	return ch
}
