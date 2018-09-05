// Copyright 2018 PingCAP, Inc.
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

package controller

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

type fakeIndexer struct {
	cache.Indexer
	getError error
}

func (f *fakeIndexer) GetByKey(_ string) (interface{}, bool, error) {
	return nil, false, f.getError
}

func collectEvents(source <-chan string) []string {
	done := false
	events := make([]string, 0)
	for !done {
		select {
		case event := <-source:
			events = append(events, event)
		default:
			done = true
		}
	}
	return events
}

func newTidbCluster() *v1alpha1.TidbCluster {
	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: metav1.NamespaceDefault,
		},
	}
	return tc
}

func newService(tc *v1alpha1.TidbCluster, _ string) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetName(tc.Name, "pd"),
			Namespace: metav1.NamespaceDefault,
		},
	}
	return svc
}

func newStatefulSet(tc *v1alpha1.TidbCluster, _ string) *apps.StatefulSet {
	set := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetName(tc.Name, "pd"),
			Namespace: metav1.NamespaceDefault,
		},
	}
	return set
}

// GetName concatenate tidb cluster name and member name, used for controller managed resource name
func GetName(tcName string, name string) string {
	return fmt.Sprintf("%s-%s", tcName, name)
}
