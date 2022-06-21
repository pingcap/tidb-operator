// Copyright 2020 PingCAP, Inc.
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

package remotewrite

const ThanosQueryServiceYaml = `apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/component: query-layer
    app.kubernetes.io/instance: thanos-query
    app.kubernetes.io/name: thanos-query
    app.kubernetes.io/version: v0.22.0
  name: thanos-query
spec:
  ports:
    - name: grpc
      port: 10901
      targetPort: 10901
    - name: http
      port: 9090
      targetPort: 9090
  selector:
    app.kubernetes.io/component: query-layer
    app.kubernetes.io/instance: thanos-query
    app.kubernetes.io/name: thanos-query
`

const ThanosQueryYaml = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/component: query-layer
    app.kubernetes.io/instance: thanos-query
    app.kubernetes.io/name: thanos-query
    app.kubernetes.io/version: v0.22.0
  name: thanos-query
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/component: query-layer
      app.kubernetes.io/instance: thanos-query
      app.kubernetes.io/name: thanos-query
  template:
    metadata:
      labels:
        app.kubernetes.io/component: query-layer
        app.kubernetes.io/instance: thanos-query
        app.kubernetes.io/name: thanos-query
        app.kubernetes.io/version: v0.22.0
    spec:
      containers:
        - args:
            - query
            - --grpc-address=0.0.0.0:10901
            - --http-address=0.0.0.0:9090
            - --log.level=debug
            - --log.format=logfmt
            - --query.replica-label=prometheus_replica
            - --query.replica-label=rule_replica
            - --store=thanos-receiver:10901
            - --query.timeout=5m
            - --query.lookback-delta=15m
            - |-
              --tracing.config="config":
                "sampler_param": 2
                "sampler_type": "ratelimiting"
                "service_name": "thanos-query"
              "type": "JAEGER"
          image: quay.io/thanos/thanos:v0.22.0
          livenessProbe:
            failureThreshold: 4
            httpGet:
              path: /-/healthy
              port: 9090
              scheme: HTTP
            periodSeconds: 30
          name: thanos-query
          ports:
            - containerPort: 10901
              name: grpc
            - containerPort: 9090
              name: http
          readinessProbe:
            failureThreshold: 20
            httpGet:
              path: /-/ready
              port: 9090
              scheme: HTTP
            periodSeconds: 5
          resources: {}
          terminationMessagePolicy: FallbackToLogsOnError
      terminationGracePeriodSeconds: 120
`

const ThanosReceiverServiceYaml = `apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/component: database-write-hashring
    app.kubernetes.io/instance: thanos-receiver
    app.kubernetes.io/name: thanos-receiver
    app.kubernetes.io/version: v0.22.0
  name: thanos-receiver
spec:
  clusterIP: None
  ports:
    - name: grpc
      port: 10901
      targetPort: 10901
    - name: http
      port: 10902
      targetPort: 10902
    - name: remote-write
      port: 19291
      targetPort: 19291
  selector:
    app.kubernetes.io/component: database-write-hashring
    app.kubernetes.io/instance: thanos-receiver
    app.kubernetes.io/name: thanos-receiver
`

const ThanosReceiverYaml = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  labels:
    app.kubernetes.io/component: database-write-hashring
    app.kubernetes.io/instance: thanos-receiver
    app.kubernetes.io/name: thanos-receiver
    app.kubernetes.io/version: v0.22.0
    controller.receive.thanos.io: thanos-receiver-controller
    controller.receive.thanos.io/hashring: default
  name: thanos-receiver
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/component: database-write-hashring
      app.kubernetes.io/instance: thanos-receiver
      app.kubernetes.io/name: thanos-receiver
      controller.receive.thanos.io/hashring: default
  serviceName: thanos-receiver
  template:
    metadata:
      labels:
        app.kubernetes.io/component: database-write-hashring
        app.kubernetes.io/instance: thanos-receiver
        app.kubernetes.io/name: thanos-receiver
        app.kubernetes.io/version: v0.22.0
        controller.receive.thanos.io/hashring: default
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - podAffinityTerm:
                labelSelector:
                  matchExpressions:
                    - key: app.kubernetes.io/name
                      operator: In
                      values:
                        - thanos-receiver
                    - key: app.kubernetes.io/instance
                      operator: In
                      values:
                        - thanos-receiver
                namespaces:
                  - thanos
                topologyKey: kubernetes.io/hostname
              weight: 100
            - podAffinityTerm:
                labelSelector:
                  matchExpressions:
                    - key: app.kubernetes.io/name
                      operator: In
                      values:
                        - thanos-receiver
                    - key: app.kubernetes.io/instance
                      operator: In
                      values:
                        - thanos-receiver
                namespaces:
                  - thanos
                topologyKey: topology.kubernetes.io/zone
              weight: 100
      containers:
        - args:
            - receive
            - --log.level=debug
            - --log.format=logfmt
            - --grpc-address=0.0.0.0:10901
            - --http-address=0.0.0.0:10902
            - --remote-write.address=0.0.0.0:19291
            - --receive.replication-factor=1
            - --tsdb.path=/var/thanos/receive
            - --tsdb.retention=15d
            - --label=replica="$(NAME)"
            - --label=receive="true"
            - --receive.local-endpoint=$(NAME).thanos-receiver.$(NAMESPACE).svc.cluster.local:10901
            - |-
              --tracing.config="config":
                "sampler_param": 2
                "sampler_type": "ratelimiting"
                "service_name": "thanos-receiver"
              "type": "JAEGER"
          env:
            - name: NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: HOST_IP_ADDRESS
              valueFrom:
                fieldRef:
                  fieldPath: status.hostIP
          image: quay.io/thanos/thanos:v0.22.0
          livenessProbe:
            failureThreshold: 8
            httpGet:
              path: /-/healthy
              port: 10902
              scheme: HTTP
            periodSeconds: 30
          name: thanos-receiver
          ports:
            - containerPort: 10901
              name: grpc
            - containerPort: 10902
              name: http
            - containerPort: 19291
              name: remote-write
          readinessProbe:
            failureThreshold: 20
            httpGet:
              path: /-/ready
              port: 10902
              scheme: HTTP
            periodSeconds: 5
          resources:
            limits:
              cpu: 0.42
              memory: 420Mi
            requests:
              cpu: 0.123
              memory: 123Mi
          terminationMessagePolicy: FallbackToLogsOnError
          volumeMounts:
            - mountPath: /var/thanos/receive
              name: data
              readOnly: false
            - mountPath: /var/lib/thanos-receiver
              name: hashring-config
      nodeSelector:
        kubernetes.io/os: linux
      securityContext:
        fsGroup: 65534
        runAsUser: 65534
      terminationGracePeriodSeconds: 900
      volumes:
        - configMap:
            name: hashring
          name: hashring-config
  volumeClaimTemplates:
    - metadata:
        labels:
          app.kubernetes.io/component: database-write-hashring
          app.kubernetes.io/instance: thanos-receiver
          app.kubernetes.io/name: thanos-receiver
          controller.receive.thanos.io/hashring: default
        name: data
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 10Gi
`

const ThanosReceiverConfigmapYaml = `apiVersion: v1
kind: ConfigMap
metadata:
  name: hashring
  labels:
    app.kubernetes.io/name: thanos-receive
data:
  hashrings.json: |
    [
        {
            "hashring": "athena",
            "endpoints": ["thanos-receiver-0.thanos-receiver.%s.svc:10901"],
            "tenants": ["athena"]
        }
    ]
`
