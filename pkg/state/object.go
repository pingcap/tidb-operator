package state

import (
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
)

type IObject[T client.Object] interface {
	Object() T
}

type ICluster interface {
	Cluster() *v1alpha1.Cluster
}
