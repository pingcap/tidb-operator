package framework

import (
	"context"

	"github.com/pingcap/tidb-operator/tests/e2e/utils/s3"
)

func (f *Framework) MustCreateS3(ctx context.Context) {
	s, err := s3.New("minio", f.restConfig)
	f.Must(err)
	f.Must(s.Init(ctx, f.Namespace.Name))
}
