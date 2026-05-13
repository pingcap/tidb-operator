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

package dm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/toml"
)

func TestOverlayEnablesOpenAPI(t *testing.T) {
	cluster := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "cluster-1",
		},
	}
	dm := &v1alpha1.DM{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "dm-0",
			Annotations: map[string]string{
				v1alpha1.AnnoKeyInitialClusterNum: "1",
			},
		},
		Spec: v1alpha1.DMSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: "cluster-1",
			},
			Subdomain: "dm-peer",
			DMTemplateSpec: v1alpha1.DMTemplateSpec{
				DataVolume: v1alpha1.Volume{
					Name: "data",
					Mounts: []v1alpha1.VolumeMount{
						{Type: v1alpha1.VolumeMountTypeDMData},
					},
				},
			},
		},
	}

	cfg := &Config{}
	err := cfg.Overlay(cluster, dm, []*v1alpha1.DM{dm}, "dm.default:8261")
	require.NoError(t, err)
	assert.True(t, cfg.OpenAPI)
}

func TestCodecOverridesOpenAPIDisabled(t *testing.T) {
	decoder, encoder := toml.Codec[Config]()
	cfg := &Config{}
	require.NoError(t, decoder.Decode([]byte("openapi = false\n"), cfg))

	cfg.OpenAPI = true
	data, err := encoder.Encode(cfg)
	require.NoError(t, err)
	assert.Contains(t, string(data), "openapi = true")
}
