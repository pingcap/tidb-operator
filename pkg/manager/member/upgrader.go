package member

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	apps "k8s.io/api/apps/v1beta1"
)

// Upgrader implements the logic for upgrade the tidb cluster.
type Upgrader interface {
	// Upgrade upgrade the cluster
	Upgrade(*v1alpha1.TidbCluster, *apps.StatefulSet, *apps.StatefulSet) error
}
