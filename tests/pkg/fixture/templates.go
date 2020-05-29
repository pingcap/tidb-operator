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
// limitations under the License.

package fixture

var Tidbcluster3_1_1 = `apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  timezone: UTC
  pvReclaimPolicy: Delete
  imagePullPolicy: IfNotPresent
  pd:
    image: pingcap/pd:v3.1.1
    replicas: 1
    requests:
      storage: "1Gi"
    config: {}
  tikv:
    config:
      pessimistic-txn:
        wait-for-lock-timeout: 1000
        wake-up-delay-duration: 20
    image: pingcap/tikv:v3.1.1
    replicas: 1
    requests:
      storage: "1Gi"
  tidb:
    enableAdvertiseAddress: true
    image: pingcap/tidb:v3.1.1
    imagePullPolicy: IfNotPresent
    replicas: 1
    service:
      type: ClusterIP
    config: {}
    requests:
      cpu: 1
`

var TidbCluster4_0_0 = `
apiVersion: pingcap.com/v1alpha1
kind: TidbCluster
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  timezone: UTC
  pvReclaimPolicy: Delete
  imagePullPolicy: IfNotPresent
  pd:
    image: pingcap/pd:v4.0.0
    replicas: 1
    requests:
      storage: "1Gi"
    config: {}
  tikv:
    config:
      pessimistic-txn:
        wait-for-lock-timeout: "1s"
        wake-up-delay-duration: "20ms"
    image: pingcap/tikv:v4.0.0
    replicas: 1
    requests:
      storage: "1Gi"
  tidb:
    image: pingcap/tidb:v4.0.0
    imagePullPolicy: IfNotPresent
    replicas: 1
    service:
      type: ClusterIP
    config: {}
    requests:
      cpu: 1
`
