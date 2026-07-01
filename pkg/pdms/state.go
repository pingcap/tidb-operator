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

package pdms

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

type State struct {
	PDGroups              []*v1alpha1.PDGroup
	PDs                   []*v1alpha1.PD
	TSOGroups             []*v1alpha1.TSOGroup
	SchedulingGroups      []*v1alpha1.SchedulingGroup
	ResourceManagerGroups []*v1alpha1.ResourceManagerGroup
}

func GetState(ctx context.Context, c client.Client, namespace, cluster string) (*State, error) {
	pdgs, err := apicall.ListGroups[scope.PDGroup](ctx, c, namespace, cluster)
	if err != nil {
		return nil, err
	}
	pds, err := apicall.ListClusterInstances[scope.PD](ctx, c, namespace, cluster)
	if err != nil {
		return nil, err
	}
	tgs, err := apicall.ListGroups[scope.TSOGroup](ctx, c, namespace, cluster)
	if err != nil {
		return nil, err
	}
	sgs, err := apicall.ListGroups[scope.SchedulingGroup](ctx, c, namespace, cluster)
	if err != nil {
		return nil, err
	}
	rmgs, err := apicall.ListGroups[scope.ResourceManagerGroup](ctx, c, namespace, cluster)
	if err != nil {
		return nil, err
	}

	return &State{
		PDGroups:              pdgs,
		PDs:                   pds,
		TSOGroups:             tgs,
		SchedulingGroups:      sgs,
		ResourceManagerGroups: rmgs,
	}, nil
}

func (s *State) InvolvesMS() bool {
	for _, pdg := range s.PDGroups {
		if PDGroupInvolvesMS(pdg) {
			return true
		}
	}
	for _, pd := range s.PDs {
		if pd.Spec.Mode == v1alpha1.PDModeMS {
			return true
		}
	}
	return false
}

func PDGroupInvolvesMS(pdg *v1alpha1.PDGroup) bool {
	return pdg.Spec.Template.Spec.Mode == v1alpha1.PDModeMS ||
		pdg.Status.Mode == v1alpha1.PDModeMS ||
		modeSwitchingConditionActive(pdg)
}

func modeSwitchingConditionActive(pdg *v1alpha1.PDGroup) bool {
	cond := meta.FindStatusCondition(pdg.Status.Conditions, v1alpha1.CondModeSwitching)
	return cond != nil && cond.Status == metav1.ConditionTrue
}

func TSOGroupProtected(ctx context.Context, c client.Client, tg *v1alpha1.TSOGroup) (bool, error) {
	s, err := GetState(ctx, c, tg.Namespace, tg.Spec.Cluster.Name)
	if err != nil {
		return false, err
	}
	return s.InvolvesMS(), nil
}
