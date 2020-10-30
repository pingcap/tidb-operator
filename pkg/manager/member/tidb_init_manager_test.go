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

package member

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestTiDBInitManagerSync(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name    string
		tc      *v1alpha1.TidbCluster
		tls     bool
		mockErr bool
		err     bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		var err error
		tim, tmm, indexers := newFakeTiDBInitManager()
		if test.mockErr {
			tmm.deps.StatefulSetControl.(*controller.FakeStatefulSetControl).
				SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		ti := newTidbInitializerForTiDB()
		oldSpec := ti.Spec

		if test.tc != nil {
			tc := test.tc
			if test.tls {
				tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			}
			tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
				"tikv-0": {PodName: "tikv-0", State: v1alpha1.TiKVStateUp},
			}
			tc.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 1}
			// sync configmap
			_, err = tmm.deps.Controls.TiDBClusterControl.UpdateTidbCluster(tc, nil, nil)
			g.Expect(err).NotTo(HaveOccurred())

			err = indexers.ti.Add(ti)
			g.Expect(err).NotTo(HaveOccurred())

			job, err := tim.makeTiDBInitJob(ti)
			g.Expect(err).NotTo(HaveOccurred())
			err = indexers.job.Add(job)
			g.Expect(err).NotTo(HaveOccurred())
		}

		err = func() error {
			err = tim.syncTiDBInitConfigMap(ti)
			if err != nil {
				return err
			}
			err = tim.syncTiDBInitJob(ti)
			/* The test for genericClient is not fully working yet
			if err != nil {
				return err
			}
			err = tim.updateStatus(ti.DeepCopy())
			*/
			return err
		}()
		//err = tim.Sync(ti)

		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		g.Expect(ti.Spec).To(Equal(oldSpec))
	}

	tests := []testcase{
		{
			name:    "normal",
			tc:      newTidbClusterForTiDB(),
			mockErr: false,
			err:     false,
		},
		{
			name:    "normal with tls",
			tc:      newTidbClusterForTiDB(),
			tls:     true,
			mockErr: false,
			err:     false,
		},
		{
			name:    "no tidbcluster",
			mockErr: false,
			err:     true,
		},
		{
			name:    "error when sync init manager",
			mockErr: true,
			err:     true,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakeTiDBInitManager() (*tidbInitManager, *tidbMemberManager, *fakeIndexers) {
	tmm, _, _, indexers := newFakeTiDBMemberManager()
	indexers.job = tmm.deps.KubeInformerFactory.Batch().V1().Jobs().Informer().GetIndexer()
	indexers.ti = tmm.deps.InformerFactory.Pingcap().V1alpha1().TidbInitializers().Informer().GetIndexer()
	return &tidbInitManager{deps: tmm.deps}, tmm, indexers
}

func newTidbInitializerForTiDB() *v1alpha1.TidbInitializer {
	return &v1alpha1.TidbInitializer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: v1alpha1.TidbInitializerSpec{
			Clusters: v1alpha1.TidbClusterRef{
				Name:      "test",
				Namespace: corev1.NamespaceDefault,
			},
		},
	}
}
