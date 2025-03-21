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
// limitations under the License.i

package controller

import (
	"context"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

// BackupControlInterface manages Backups used in BackupSchedule
type CompactBackupControlInterface interface {
	CreateCompactBackup(compact *v1alpha1.CompactBackup) (*v1alpha1.CompactBackup, error)
	DeleteCompactBackup(compact *v1alpha1.CompactBackup) error
}

type realCompactControl struct {
	cli      versioned.Interface
	recorder record.EventRecorder
}

func NewRealCompactControl(cli versioned.Interface, recorder record.EventRecorder) CompactBackupControlInterface {
	return &realCompactControl{
		cli:      cli,
		recorder: recorder,
	}
}

func (c *realCompactControl) CreateCompactBackup(compact *v1alpha1.CompactBackup) (*v1alpha1.CompactBackup, error) {
	ns := compact.GetNamespace()

	return c.cli.PingcapV1alpha1().CompactBackups(ns).Create(context.TODO(), compact, metav1.CreateOptions{})
}

func (c *realCompactControl) DeleteCompactBackup(compact *v1alpha1.CompactBackup) error {
	ns := compact.GetNamespace()
	compactName := compact.GetName()

	return c.cli.PingcapV1alpha1().CompactBackups(ns).Delete(context.TODO(), compactName, metav1.DeleteOptions{})
}
