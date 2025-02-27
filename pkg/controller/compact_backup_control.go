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
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
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

func (c *realCompactControl) recordCompactEvent(verb string, compact *v1alpha1.CompactBackup, err error) {
	backupName := compact.GetName()
	ns := compact.GetNamespace()

	bsName := compact.GetLabels()[label.BackupScheduleLabelKey]
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s CompactBackup %s/%s for backupSchedule/%s successful",
			strings.ToLower(verb), ns, backupName, bsName)
		c.recorder.Event(compact, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s CompactBackup %s/%s for backupSchedule/%s failed error: %s",
			strings.ToLower(verb), ns, backupName, bsName, err)
		c.recorder.Event(compact, corev1.EventTypeWarning, reason, msg)
	}
}

func (c *realCompactControl) CreateCompactBackup(compact *v1alpha1.CompactBackup) (*v1alpha1.CompactBackup, error) {
	ns := compact.GetNamespace()
	compactName := compact.GetName()

	compact, err := c.cli.PingcapV1alpha1().CompactBackups(ns).Create(context.TODO(), compact, metav1.CreateOptions{})
	bsName := compact.GetLabels()[label.BackupScheduleLabelKey]
	if err != nil {
		klog.Errorf("failed to create CompactBackup: [%s/%s] for backupSchedule/%s, err: %v", ns, compactName, bsName, err)
	} else {
		klog.V(4).Infof("create CompactBackup: [%s/%s] for backupSchedule/%s successfully", ns, compactName, bsName)
	}
	c.recordCompactEvent("create", compact, err)
	return compact, err
}

func (c *realCompactControl) DeleteCompactBackup(compact *v1alpha1.CompactBackup) error {
	ns := compact.GetNamespace()
	compactName := compact.GetName()

	err := c.cli.PingcapV1alpha1().CompactBackups(ns).Delete(context.TODO(), compactName, metav1.DeleteOptions{})
	bsName := compact.GetLabels()[label.BackupScheduleLabelKey]
	if err != nil {
		klog.Errorf("failed to delete CompactBackup: [%s/%s] for backupSchedule/%s, err: %v", ns, compactName, bsName, err)
	} else {
		klog.V(4).Infof("delete CompactBackup: [%s/%s] for backupSchedule/%s successfully", ns, compactName, bsName)
	}
	c.recordCompactEvent("delete", compact, err)
	return err
}

type FakeCompactControl struct {
	compactLister        listers.CompactBackupLister
	compactIndexer       cache.Indexer
	createCompactTracker RequestTracker
	deletecompactTracker RequestTracker
}

// NewFakeCompactControl returns a FakeCompactControl
func NewFakeCompactControl(compactInformer informers.CompactBackupInformer) *FakeCompactControl {
	return &FakeCompactControl{
		compactInformer.Lister(),
		compactInformer.Informer().GetIndexer(),
		RequestTracker{},
		RequestTracker{},
	}
}

func (fc *FakeCompactControl) CreateCompactBackup(compact *v1alpha1.CompactBackup) (*v1alpha1.CompactBackup, error) {
	defer fc.createCompactTracker.Inc()
	if fc.createCompactTracker.ErrorReady() {
		defer fc.createCompactTracker.Reset()
		return compact, fc.createCompactTracker.GetError()
	}

	return compact, fc.compactIndexer.Add(compact)
}

func (fc *FakeCompactControl) DeleteCompactBackup(compact *v1alpha1.CompactBackup) error {
	defer fc.createCompactTracker.Inc()
	if fc.createCompactTracker.ErrorReady() {
		defer fc.createCompactTracker.Reset()
		return fc.createCompactTracker.GetError()
	}

	return fc.compactIndexer.Delete(compact)
}
