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

package readable

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/tkctl/alias"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func addPodStoreID(pod corev1.Pod, storeID string) corev1.Pod {
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[label.StoreIDLabelKey] = storeID
	return pod
}

func TestPrintFlags(t *testing.T) {
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "foons",
			Labels: map[string]string{
				"l1": "value",
			},
		},
	}

	testPV := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "foons",
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				"storage": resource.MustParse("10Gi"),
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				Local: &corev1.LocalVolumeSource{
					Path: "/mnt/disks/vol1",
				},
			},
			NodeAffinity: &corev1.VolumeNodeAffinity{
				Required: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "kubernetes.io/hostname",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"node"},
								},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name          string
		testObject    runtime.Object
		withKind      bool
		withNamespace bool
		// we do not test json/yaml format, because these are covered by package of JSONYamlPrintFlags
		outputFormat   string
		expectNoMatch  bool
		expectedOutput string
		expectedError  string
	}{
		{
			name:       "empty output format matches a humanreadable printer",
			testObject: testPod.DeepCopy(),
			expectedOutput: `NAME   READY   STATUS   MEMORY          CPU             RESTARTS   AGE
foo    0/0              <none>/<none>   <none>/<none>   0          <unknown>
`,
		},
		{
			name:          "withNamespace displays an additional NAMESPACE column",
			withNamespace: true,
			testObject:    testPod.DeepCopy(),
			expectedOutput: `NAMESPACE   NAME   READY   STATUS   MEMORY          CPU             RESTARTS   AGE
foons       foo    0/0              <none>/<none>   <none>/<none>   0          <unknown>
`,
		},
		{
			name:         "wide output format matches a humanreadable printer",
			testObject:   testPod.DeepCopy(),
			outputFormat: "wide",
			expectedOutput: `NAME   READY   STATUS   MEMORY          CPU             RESTARTS   AGE         IP       NODE
foo    0/0              <none>/<none>   <none>/<none>   0          <unknown>   <none>   <none>
`,
		},
		{
			name: "unsupported type",
			testObject: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			},
			expectedError: "unknown type",
		},
		{
			name: "pod list",
			testObject: &corev1.PodList{
				Items: []corev1.Pod{
					*testPod.DeepCopy(),
					*testPod.DeepCopy(),
				},
			},
			expectedOutput: "NAME   READY   STATUS   MEMORY          CPU             RESTARTS   AGE\nfoo    0/0              <none>/<none>   <none>/<none>   0          <unknown>\nfoo    0/0              <none>/<none>   <none>/<none>   0          <unknown>\n",
		},
		{
			name: "tidb cluster",
			testObject: &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Status: v1alpha1.TidbClusterStatus{
					PD: v1alpha1.PDStatus{
						StatefulSet: &apps.StatefulSetStatus{
							Replicas:      3,
							ReadyReplicas: 2,
						},
					},
					TiKV: v1alpha1.TiKVStatus{
						StatefulSet: &apps.StatefulSetStatus{
							Replicas:      3,
							ReadyReplicas: 2,
						},
					},
					TiDB: v1alpha1.TiDBStatus{
						StatefulSet: &apps.StatefulSetStatus{
							Replicas:      3,
							ReadyReplicas: 2,
						},
					},
				},
			},
			expectedOutput: "NAME      PD    TIKV   TIDB   AGE\ncluster   2/3   2/3    2/3    <unknown>\n",
		},
		{
			name: "tikv list",
			testObject: &alias.TikvList{
				PodList: &corev1.PodList{
					Items: []corev1.Pod{
						addPodStoreID(*testPod.DeepCopy(), "1"),
						addPodStoreID(*testPod.DeepCopy(), "2"),
					},
				},
				TikvStatus: &v1alpha1.TiKVStatus{
					Stores: map[string]v1alpha1.TiKVStore{
						"1": {
							State: v1alpha1.TiKVStateUp,
						},
						"2": {
							State: v1alpha1.TiKVStateDown,
						},
					},
				},
			},
			expectedOutput: `NAME   READY   STATUS   MEMORY          CPU             RESTARTS   AGE         STOREID   STORE STATE
foo    0/0              <none>/<none>   <none>/<none>   0          <unknown>   1         Up
foo    0/0              <none>/<none>   <none>/<none>   0          <unknown>   1         Up
`,
		},
		{
			name: "pv list",
			testObject: &corev1.PersistentVolumeList{
				Items: []corev1.PersistentVolume{
					*testPV.DeepCopy(),
				},
			},
			outputFormat: "wide",
			expectedOutput: `VOLUME   CLAIM    STATUS   CAPACITY   STORAGECLASS   NODE   LOCAL
foo      <none>            10Gi                      node   /mnt/disks/vol1
`,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %T", tt.name, tt.testObject), func(t *testing.T) {
			printFlags := PrintFlags{
				OutputFormat: tt.outputFormat,
			}

			p, err := printFlags.ToPrinter(tt.withKind, tt.withNamespace)
			if tt.expectNoMatch {
				if !genericclioptions.IsNoCompatiblePrinterError(err) {
					t.Fatalf("expected no printer matches for output format %q", tt.outputFormat)
				}
				return
			}

			if genericclioptions.IsNoCompatiblePrinterError(err) {
				t.Fatalf("expected to match template printer for output format %q", tt.outputFormat)
			}

			out := bytes.NewBuffer([]byte{})
			err = p.PrintObj(tt.testObject, out)
			if len(tt.expectedError) > 0 {
				if err == nil || !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expecting error %q, got %v", tt.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tt.expectedOutput, out.String()); diff != "" {
				t.Errorf("unexpected output (-want, +got): %s", diff)
			}
		})
	}
}
