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

package restore

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/backup/restore"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"
)

func TestRestoreControlUpdateRestore(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		name                  string
		syncRestoreManagerErr bool
		errExpectFn           func(*GomegaWithT, error)
	}{
		{
			name:                  "restore manager sync failed",
			syncRestoreManagerErr: true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "restore manager sync error")).To(Equal(true))
			},
		},
		{
			name:                  "update newly create restore normally",
			syncRestoreManagerErr: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := newRestore()
			control, restoreIndexer, restoreManager := newFakeRestoreControl()

			restoreIndexer.Add(restore)

			if tt.syncRestoreManagerErr {
				restoreManager.SetSyncError(fmt.Errorf("restore manager sync error"))
			}

			err := control.UpdateRestore(restore)
			if tt.errExpectFn != nil {
				tt.errExpectFn(g, err)
			}
		})
	}
}

func newFakeRestoreControl() (ControlInterface, cache.Indexer, *restore.FakeRestoreManager) {
	cli := &fake.Clientset{}

	restoreInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().Restores()
	restoreManager := restore.NewFakeRestoreManager()
	control := NewDefaultRestoreControl(restoreManager)

	return control, restoreInformer.Informer().GetIndexer(), restoreManager
}
