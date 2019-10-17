// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.package spec

package tests

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetSubValuesOrDie(t *testing.T) {
	tests := []struct {
		name           string
		clusterName    string
		namespace      string
		topologyKey    string
		pdConfig       []string
		tidbConfig     []string
		tikvConfig     []string
		pumpConfig     []string
		drainerConfig  []string
		expectedOutput string
	}{
		{
			name:        "without component configs",
			clusterName: "cluster1",
			namespace:   "ns1",
			topologyKey: "rack",
			expectedOutput: `pd:
  config: |

  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 50
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app.kubernetes.io/instance: cluster1
              app.kubernetes.io/component: pd
          topologyKey: rack
          namespaces:
          - ns1
tikv:
  config: |

  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchLabels:
            app.kubernetes.io/instance: cluster1
            app.kubernetes.io/component: tikv
        topologyKey: rack
        namespaces:
        - ns1
tidb:
  config: |

  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 50
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app.kubernetes.io/instance: cluster1
              app.kubernetes.io/component: tidb
          topologyKey: rack
          namespaces:
          - ns1
binlog:
  pump:
    tolerations:
    - key: node-role
      operator: Equal
      value: tidb
      effect: "NoSchedule"
    affinity:
      podAntiAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 50
          podAffinityTerm:
            topologyKey: rack
            namespaces:
            - ns1

  drainer:
    tolerations:
    - key: node-role
      operator: Equal
      value: tidb
      effect: "NoSchedule"
    affinity:
      podAntiAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 50
          podAffinityTerm:
            topologyKey: rack
            namespaces:
            - ns1

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOutput := GetSubValuesOrDie(tt.clusterName, tt.namespace, tt.topologyKey, tt.pdConfig, tt.tidbConfig, tt.tikvConfig, tt.pumpConfig, tt.drainerConfig)
			if diff := cmp.Diff(tt.expectedOutput, gotOutput); diff != "" {
				t.Errorf("unexpected output (-want, +got): %s", diff)
			}
		})
	}
}
