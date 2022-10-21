// Copyright 2022 PingCAP, Inc.
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
	"errors"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

func TestRestoreControlUpdateRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	rs := newRestore()
	var testVal v1alpha1.BackupType = "test"
	rs.Spec.Type = testVal
	fakeClient := &fake.Clientset{}
	control := NewRealRestoreControl(fakeClient, nil, recorder)
	fakeClient.AddReactor("update", "restores", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		return true, update.GetObject(), nil
	})
	updateRs, err := control.UpdateRestore(rs)
	g.Expect(err).To(Succeed())
	g.Expect(updateRs.Spec.Type).To(Equal(testVal))
}

func TestRestoreControlUpdateRestoreConflictSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	rs := newRestore()
	fakeClient := &fake.Clientset{}
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	rsLister := listers.NewRestoreLister(indexer)
	control := NewRealRestoreControl(fakeClient, rsLister, recorder)
	conflict := false
	fakeClient.AddReactor("update", "restores", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		if !conflict {
			conflict = true
			return true, update.GetObject(), apierrors.NewConflict(action.GetResource().GroupResource(), rs.Name, errors.New("conflict"))
		}
		return true, update.GetObject(), nil
	})
	_, err := control.UpdateRestore(rs)
	g.Expect(err).To(Succeed())
}
