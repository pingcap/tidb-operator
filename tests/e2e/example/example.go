package example

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/yaml"
)

var _ = ginkgo.Describe("Example", label.KindExample, label.P0, func() {
	f := framework.New()
	f.Setup(framework.WithSkipClusterCreation())

	ginkgo.DescribeTable("Example",
		func(ctx context.Context, dir string, cn string) {
			// NOTE(liubo02): ignore resources config to avoid resource limit in local
			f.Must(yaml.ApplyDir(ctx, f.Client, dir, yaml.Namespace(f.Namespace.Name), yaml.IgnoreResources{}))
			f.WaitForAllReady(ctx, cn)

			// now delete ns directly may be blocked by tikv prestop hook
			// TODO(liubo02): remove it after prestop hook is fixed
			c := v1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: f.Namespace.Name,
					Name:      cn,
				},
			}
			f.Must(f.Client.Delete(ctx, &c))
			f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, &c, waiter.LongTaskTimeout))
		},
		ginkgo.Entry("basic", "example/data/basic", "basic"),
	)
})
