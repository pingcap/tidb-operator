package action

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
)

func MustUpdateCluster(
	ctx context.Context,
	f *framework.Framework,
	patches ...data.ClusterPatch,
) {
	ginkgo.By(fmt.Sprintf("update cluster %v", client.ObjectKeyFromObject(f.Cluster)))

	mp := client.MergeFrom(f.Cluster.DeepCopy())
	for _, p := range patches {
		p(f.Cluster)
	}
	err := f.Client.Patch(ctx, f.Cluster, mp)
	gomega.ExpectWithOffset(1, err).To(gomega.Succeed())
}
