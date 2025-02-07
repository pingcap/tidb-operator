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

package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

func TestTopologyPolicy(t *testing.T) {
	policy, err := NewTopologyPolicy[*runtime.PD]([]v1alpha1.ScheduleTopology{
		{
			Topology: v1alpha1.Topology{
				"zone": "us-west-1a",
			},
		},
		{
			Topology: v1alpha1.Topology{
				"zone": "us-west-1b",
			},
		},
		{
			Topology: v1alpha1.Topology{
				"zone": "us-west-1c",
			},
		},
	})
	require.NoError(t, err)

	pd0 := &v1alpha1.PD{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pd-0",
		},
		Spec: v1alpha1.PDSpec{
			Topology: v1alpha1.Topology{},
		},
	}
	pd0r := policy.Add(runtime.FromPD(pd0))
	require.Equal(t, pd0, runtime.ToPD(pd0r)) // in-place update
	assert.Equal(t, v1alpha1.Topology{"zone": "us-west-1a"}, pd0r.Spec.Topology)

	pd1 := &v1alpha1.PD{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pd-1",
		},
		Spec: v1alpha1.PDSpec{
			Topology: v1alpha1.Topology{},
		},
	}
	pd1r := policy.Add(runtime.FromPD(pd1))
	require.Equal(t, pd1, runtime.ToPD(pd1r)) // in-place update
	assert.Equal(t, v1alpha1.Topology{"zone": "us-west-1b"}, pd1r.Spec.Topology)

	perfer := policy.Prefer([]*runtime.PD{pd0r, pd1r})
	assert.Equal(t, "pd-0", perfer[0].Name)

	policy.Delete("pd-0")
	perfer = policy.Prefer([]*runtime.PD{pd0r, pd1r})
	assert.Equal(t, "pd-1", perfer[0].Name)
}
