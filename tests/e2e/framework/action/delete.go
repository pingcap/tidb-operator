package action

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
)

func MustDelete(ctx context.Context, f *framework.Framework, obj client.Object) {
	ginkgo.By(fmt.Sprintf("delete %v", client.ObjectKeyFromObject(obj)))

	err := f.Client.Delete(ctx, obj)
	gomega.ExpectWithOffset(1, err).To(gomega.Succeed())
}
