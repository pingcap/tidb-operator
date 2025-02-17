package coreutil

import (
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
)

func Cluster[
	S scope.Object[F, T],
	F client.Object,
	T runtime.Object,
](f F) string {
	return scope.From[S](f).Cluster()
}
