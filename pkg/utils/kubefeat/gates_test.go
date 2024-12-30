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

package kubefeat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFeatureGates(t *testing.T) {
	metrics := []byte(`
# HELP kubernetes_feature_enabled [BETA] This metric records the data about the stage and enablement of a k8s feature.
# TYPE kubernetes_feature_enabled gauge
kubernetes_feature_enabled{name="ImageMaximumGCAge",stage="BETA"} 1
kubernetes_feature_enabled{name="ImageVolume",stage="ALPHA"} 0
kubernetes_feature_enabled{name="InPlacePodVerticalScaling",stage="ALPHA"} 0
kubernetes_feature_enabled{name="InTreePluginPortworxUnregister",stage="ALPHA"} 0
kubernetes_feature_enabled{name="InformerResourceVersion",stage="ALPHA"} 0
kubernetes_feature_enabled{name="ValidatingAdmissionPolicy",stage=""} 1
kubernetes_feature_enabled{name="VolumeAttributesClass",stage="BETA"} 1
kubernetes_feature_enabled{name="VolumeCapacityPriority",stage="ALPHA"} 0
`)

	gates, err := parseFeatureGates(metrics)
	require.NoError(t, err)
	assert.True(t, gates.Stage(VolumeAttributesClass).Enabled(BETA))
	assert.False(t, gates.Stage(VolumeAttributesClass).Enabled(STABLE))

	assert.False(t, gates.Stage("InPlacePodVerticalScaling").Enabled(ALPHA))
	assert.False(t, gates.Stage("InPlacePodVerticalScaling").Enabled(BETA))
	assert.False(t, gates.Stage("InPlacePodVerticalScaling").Enabled(STABLE))

	_, err = parseFeatureGates([]byte("invalid"))
	require.Error(t, err)

	// without "kubernetes_feature_enabled"
	_, err = parseFeatureGates([]byte(`
aggregator_unavailable_apiservice{name="v1."} 0
aggregator_unavailable_apiservice{name="v1.acme.cert-manager.io"} 0
aggregator_unavailable_apiservice{name="v1.admissionregistration.k8s.io"} 0
aggregator_unavailable_apiservice{name="v1.apiextensions.k8s.io"} 0
aggregator_unavailable_apiservice{name="v1.apps"} 0
aggregator_unavailable_apiservice{name="v1.authentication.k8s.io"} 0
aggregator_unavailable_apiservice{name="v1.authorization.k8s.io"} 0
`))
	require.Error(t, err)
}
