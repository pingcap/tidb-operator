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

package tasks

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TestTaskFinalizerDel(t *testing.T) {
	now := metav1.Now()
	cases := []struct {
		desc                  string
		state                 *ReconcileContext
		subresources          []client.Object
		needDelStore          bool
		unexpectedDelStoreErr bool
		unexpectedErr         bool

		expectedStatus task.Status
		expectedObj    *v1alpha1.TiFlash
	}{
		{
			desc: "cluster is deleting, no sub resources, no finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
				},
			},

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				return obj
			}),
		},
		{
			desc: "cluster is deleting, has sub resources, has finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
				},
			},
			subresources: fakeSubresources("Pod", "ConfigMap", "PersistentVolumeClaim"),

			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "cluster is deleting, has sub resources, has finalizer, failed to del subresources(pod)",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
				},
			},
			subresources: fakeSubresources("Pod"),

			unexpectedErr: true,

			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "cluster is deleting, has sub resources, has finalizer, failed to del subresources(cm)",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
				},
			},
			subresources: fakeSubresources("ConfigMap"),

			unexpectedErr: true,

			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "cluster is deleting, has sub resources, has finalizer, failed to del subresources(pvc)",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
				},
			},
			subresources: fakeSubresources("PersistentVolumeClaim"),

			unexpectedErr: true,

			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "cluster is deleting, no sub resources, has finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
				},
			},

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = []string{}
				return obj
			}),
		},
		{
			desc: "cluster is deleting, no sub resources, has finalizer, failed to remove finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				return obj
			}),
		},
		{
			desc: "cluster is not deleting, store is removing",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
					storeState: v1alpha1.StoreStateRemoving,
				},
			},

			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "cluster is not deleting, store is removed, no subresources, no finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
					storeState: v1alpha1.StoreStateRemoved,
				},
			},

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				return obj
			}),
		},
		{
			desc: "cluster is not deleting, store is removed, no subresources, has finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
					storeState: v1alpha1.StoreStateRemoved,
				},
			},

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = []string{}
				return obj
			}),
		},
		{
			desc: "cluster is not deleting, store is removed, no subresources, has finalizer, failed to remove finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
					storeState: v1alpha1.StoreStateRemoved,
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "cluster is not deleting, store is removed, has subresources",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
					storeState: v1alpha1.StoreStateRemoved,
				},
			},
			subresources: fakeSubresources("Pod"),

			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "cluster is not deleting, store is removed, has subresources, failed to del subresources",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
					storeState: v1alpha1.StoreStateRemoved,
				},
			},
			subresources:  fakeSubresources("Pod"),
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "cluster is not deleting, store does not exist, no subresources, has finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
				},
				StoreNotExists: true,
			},

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = []string{}
				return obj
			}),
		},
		{
			desc: "cluster is not deleting, store exists",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
				},
				StoreID: "xxx",
			},
			needDelStore: true,

			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "cluster is not deleting, store exists, failed to del store",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
				},
				StoreID: "xxx",
			},
			needDelStore:          true,
			unexpectedDelStoreErr: true,

			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		// Test cases for cluster deletion with Pod scenarios
		{
			desc: "cluster is deleting, has pod with finalizer, no sub resources",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
					pod: fake.FakeObj("pod-0", func(obj *corev1.Pod) *corev1.Pod {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
			},

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = []string{}
				return obj
			}),
		},
		{
			desc: "cluster is deleting, has pod with finalizer, failed to remove pod finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
					pod: fake.FakeObj("pod-0", func(obj *corev1.Pod) *corev1.Pod {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "cluster is deleting, has pod with finalizer and sub resources",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
					pod: fake.FakeObj("pod-0", func(obj *corev1.Pod) *corev1.Pod {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
			},
			subresources: fakeSubresources("ConfigMap"),

			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		// Test cases for store removed/not exists with Pod scenarios
		{
			desc: "cluster is not deleting, store is removed, has pod with finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
					pod: fake.FakeObj("pod-0", func(obj *corev1.Pod) *corev1.Pod {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					storeState: v1alpha1.StoreStateRemoved,
				},
			},

			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "cluster is not deleting, store is removed, has pod with finalizer, failed to remove pod finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
					pod: fake.FakeObj("pod-0", func(obj *corev1.Pod) *corev1.Pod {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					storeState: v1alpha1.StoreStateRemoved,
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "cluster is not deleting, store does not exist, has pod with finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
					pod: fake.FakeObj("pod-0", func(obj *corev1.Pod) *corev1.Pod {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
				StoreNotExists: true,
			},

			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "cluster is not deleting, store does not exist, has pod with finalizer, failed to remove pod finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
					pod: fake.FakeObj("pod-0", func(obj *corev1.Pod) *corev1.Pod {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
				StoreNotExists: true,
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		// Additional edge cases for pods without finalizers
		{
			desc: "cluster is deleting, has pod without finalizer, no sub resources",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
					pod: fake.FakeObj("pod-0", func(obj *corev1.Pod) *corev1.Pod {
						// Pod has no finalizer
						return obj
					}),
				},
			},

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = []string{}
				return obj
			}),
		},
		{
			desc: "cluster is not deleting, store is removed, has pod without finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
					pod: fake.FakeObj("pod-0", func(obj *corev1.Pod) *corev1.Pod {
						// Pod has no finalizer
						return obj
					}),
					storeState: v1alpha1.StoreStateRemoved,
				},
			},

			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "cluster is not deleting, store does not exist, has pod without finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
					pod: fake.FakeObj("pod-0", func(obj *corev1.Pod) *corev1.Pod {
						// Pod has no finalizer
						return obj
					}),
				},
				StoreNotExists: true,
			},

			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "cluster is not deleting, store is removed, has pod and subresources with finalizers",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
					pod: fake.FakeObj("pod-0", func(obj *corev1.Pod) *corev1.Pod {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					storeState: v1alpha1.StoreStateRemoved,
				},
			},
			subresources: fakeSubresources("ConfigMap"),

			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "cluster is not deleting, store does not exist, has pod and subresources with finalizers",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						return obj
					}),
					pod: fake.FakeObj("pod-0", func(obj *corev1.Pod) *corev1.Pod {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
				StoreNotExists: true,
			},
			subresources: fakeSubresources("ConfigMap"),

			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			objs := []client.Object{
				c.state.TiFlash(),
			}

			// Add pod if it exists in the state
			if pod := c.state.Pod(); pod != nil {
				objs = append(objs, pod)
			}

			objs = append(objs, c.subresources...)

			fc := client.NewFakeClient(objs...)
			if c.unexpectedErr {
				// cannot remove finalizer
				fc.WithError("update", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
				// cannot delete sub resources
				fc.WithError("delete", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var acts []action
			if c.needDelStore {
				var retErr error
				if c.unexpectedDelStoreErr {
					retErr = fmt.Errorf("fake err")
				}
				acts = append(acts, deleteStore(ctx, c.state.StoreID, retErr))
			}

			pdc := NewFakePDClient(tt, acts...)
			c.state.PDClient = pdc

			res, done := task.RunTask(ctx, TaskFinalizerDel(c.state, fc))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)

			// no need to check update result
			if c.unexpectedErr || c.unexpectedDelStoreErr {
				return
			}

			pd := &v1alpha1.TiFlash{}
			require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "aaa"}, pd), c.desc)
			assert.Equal(tt, c.expectedObj, pd, c.desc)
		})
	}
}

func NewFakePDClient(t *testing.T, acts ...action) pdm.PDClient {
	ctrl := gomock.NewController(t)
	pdc := pdm.NewMockPDClient(ctrl)
	for _, act := range acts {
		act(ctrl, pdc)
	}

	return pdc
}

type action func(ctrl *gomock.Controller, pdc *pdm.MockPDClient)

func deleteStore(ctx context.Context, name string, err error) action {
	return func(ctrl *gomock.Controller, pdc *pdm.MockPDClient) {
		underlay := pdapi.NewMockPDClient(ctrl)
		pdc.EXPECT().Underlay().Return(underlay)
		underlay.EXPECT().DeleteStore(ctx, name).Return(err)
	}
}

func fakeSubresources(types ...string) []client.Object {
	var objs []client.Object
	for i, t := range types {
		var obj client.Object
		switch t {
		case "Pod":
			obj = fake.FakeObj(strconv.Itoa(i), func(obj *corev1.Pod) *corev1.Pod {
				obj.Labels = map[string]string{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyInstance:  "aaa",
					v1alpha1.LabelKeyCluster:   "",
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
				}
				return obj
			})
		case "ConfigMap":
			obj = fake.FakeObj(strconv.Itoa(i), func(obj *corev1.ConfigMap) *corev1.ConfigMap {
				obj.Labels = map[string]string{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyInstance:  "aaa",
					v1alpha1.LabelKeyCluster:   "",
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
				}
				return obj
			})
		case "PersistentVolumeClaim":
			obj = fake.FakeObj(strconv.Itoa(i), func(obj *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
				obj.Labels = map[string]string{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyInstance:  "aaa",
					v1alpha1.LabelKeyCluster:   "",
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
				}
				return obj
			})
		}
		if obj != nil {
			objs = append(objs, obj)
		}
	}

	return objs
}
