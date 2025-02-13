package util

import (
	"context"
	"fmt"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/topology"
)

func FirstTikvGroup(cli client.Client, ns, cluster string) (*v1alpha1.TiKVGroup, error) {
	tikvGroups, err := topology.ListGroups[scope.TiKVGroup](context.TODO(), cli, ns, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to list tikv groups: %w", err)
	}
	if len(tikvGroups) == 0 {
		return nil, fmt.Errorf("no tikv groups found in cluster %w", cluster)
	}
	tikvGroup := tikvGroups[0]
	return tikvGroup, nil
}
