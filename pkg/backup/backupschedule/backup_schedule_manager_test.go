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

package backupschedule

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
)

func TestManager(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.close()
	deps := helper.deps
	m := NewBackupScheduleManager(deps).(*backupScheduleManager)
	var err error
	bs := &v1alpha1.BackupSchedule{}
	bs.Namespace = "ns"
	bs.Name = "bsname"

	// test pause
	bs.Spec.Pause = true
	err = m.Sync(bs)
	g.Expect(err).Should(BeAssignableToTypeOf(&controller.IgnoreError{}))
	g.Expect(err.Error()).Should(MatchRegexp(".*has been paused.*"))

	// test canPerformNextBackup
	//
	// test not found last backup
	bs.Spec.Pause = false
	bs.Status.LastBackup = "backupname"
	err = m.canPerformNextBackup(bs)
	g.Expect(err).Should(BeNil())

	// test backup complete
	bk := &v1alpha1.Backup{}
	bk.Namespace = bs.Namespace
	bk.Name = bs.Status.LastBackup
	bk.Status.Conditions = append(bk.Status.Conditions, v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupComplete,
		Status: v1.ConditionTrue,
	})
	helper.createBackup(bk)
	err = m.canPerformNextBackup(bs)
	g.Expect(err).Should(BeNil())
	helper.deleteBackup(bk)

	// test last backup failed state and not scheduled yet
	bk.Status.Conditions = nil
	bk.Status.Conditions = append(bk.Status.Conditions, v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupFailed,
		Status: v1.ConditionTrue,
	})
	helper.createBackup(bk)
	err = m.canPerformNextBackup(bs)
	g.Expect(err).Should(BeAssignableToTypeOf(&controller.RequeueError{}))
	helper.deleteBackup(bk)

	t.Log("start test normal Sync")
	bk.Status.Conditions = nil
	bs.Spec.Schedule = "0 0 * * *" // Run at midnight every day

	now := time.Now()
	m.now = func() time.Time { return now.AddDate(0, 0, -101) }
	m.resetLastBackup(bs)
	// 10 backup, one per day
	for i := -9; i <= 0; i++ {
		t.Log("loop id ", i)
		m.now = func() time.Time { return now.AddDate(0, 0, i) }
		err = m.Sync(bs)
		g.Expect(err).Should(BeNil())
		bks := helper.checkBacklist(bs.Namespace, i+10)
		// complete the backup created
		for _, bk := range bks.Items {
			changed := v1alpha1.UpdateBackupCondition(&bk.Status, &v1alpha1.BackupCondition{
				Type:   v1alpha1.BackupComplete,
				Status: v1.ConditionTrue,
			})
			if changed {
				bk.CreationTimestamp = metav1.Time{Time: m.now()}
				t.Log("complete backup: ", bk.Name)
				helper.updateBackup(&bk)
			}
			g.Expect(err).Should(BeNil())
		}
	}

	t.Log("test setting MaxBackups")
	m.now = time.Now
	bs.Spec.MaxBackups = pointer.Int32Ptr(3)
	err = m.Sync(bs)
	g.Expect(err).Should(BeNil())
	helper.checkBacklist(bs.Namespace, 3)

	t.Log("test setting setting MaxReservedTime")
	bs.Spec.MaxBackups = nil
	bs.Spec.MaxReservedTime = pointer.StringPtr("23h")
	err = m.Sync(bs)
	g.Expect(err).Should(BeNil())
	helper.checkBacklist(bs.Namespace, 1)
}

func TestGetLastScheduledTime(t *testing.T) {
	g := NewGomegaWithT(t)

	bs := &v1alpha1.BackupSchedule{
		Spec: v1alpha1.BackupScheduleSpec{},
		Status: v1alpha1.BackupScheduleStatus{
			LastBackupTime: &metav1.Time{},
		},
	}
	var getTime *time.Time
	var err error

	// test invalid format schedule
	bs.Spec.Schedule = "#$#$#$@"
	_, err = getLastScheduledTime(bs, time.Now)
	g.Expect(err).ShouldNot(BeNil())

	bs.Spec.Schedule = "0 0 * * *" // Run once a day at midnight
	now := time.Now()

	// test last backup time after now
	bs.Status.LastBackupTime.Time = now.AddDate(0, 0, 1)
	getTime, err = getLastScheduledTime(bs, time.Now)
	g.Expect(err).Should(BeNil())
	g.Expect(getTime).Should(BeNil())

	// test scheduled
	for i := 0; i < 10; i++ {
		bs.Status.LastBackupTime.Time = now.AddDate(0, 0, -i-1)
		getTime, err = getLastScheduledTime(bs, time.Now)
		g.Expect(err).Should(BeNil())
		g.Expect(getTime).ShouldNot(BeNil())
		expectTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		g.Expect(*getTime).Should(Equal(expectTime))
	}

	// test too many miss
	bs.Status.LastBackupTime.Time = now.AddDate(-1000, 0, 0)
	getTime, err = getLastScheduledTime(bs, time.Now)
	g.Expect(err).Should(BeNil())
	g.Expect(getTime).Should(BeNil())
}

func TestBuildBackup(t *testing.T) {
	now := time.Now()
	var get *v1alpha1.Backup

	// build BackupSchedule template
	bs := &v1alpha1.BackupSchedule{
		Spec: v1alpha1.BackupScheduleSpec{},
		Status: v1alpha1.BackupScheduleStatus{
			LastBackupTime: &metav1.Time{},
		},
	}
	bs.Namespace = "ns"
	bs.Name = "bsname"

	// build Backup template
	bk := &v1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: bs.Namespace,
			Name:      bs.GetBackupCRDName(now),
			Labels:    label.NewBackupSchedule().Instance(bs.Name).BackupSchedule(bs.Name).Labels(),
			OwnerReferences: []metav1.OwnerReference{
				controller.GetBackupScheduleOwnerRef(bs),
			},
		},
		Spec: v1alpha1.BackupSpec{
			StorageSize: constants.DefaultStorageSize,
		},
	}

	// test BR == nil
	get = buildBackup(bs, now)
	if diff := cmp.Diff(bk, get); diff != "" {
		t.Errorf("unexpected (-want, +got): %s", diff)
	}
	// should keep StorageSize from BackupSchedule
	bs.Spec.StorageSize = "9527G"
	bk.Spec.StorageSize = bs.Spec.StorageSize
	get = buildBackup(bs, now)
	if diff := cmp.Diff(bk, get); diff != "" {
		t.Errorf("unexpected (-want, +got): %s", diff)
	}

	// test BR != nil
	bs.Spec.BackupTemplate.BR = &v1alpha1.BRConfig{}
	bk.Spec.BR = bs.Spec.BackupTemplate.BR.DeepCopy()
	bk.Spec.StorageSize = "" // no use for BR
	get = buildBackup(bs, now)
	if diff := cmp.Diff(bk, get); diff != "" {
		t.Errorf("unexpected (-want, +got): %s", diff)
	}
}

type helper struct {
	t    *testing.T
	deps *controller.Dependencies
	stop chan struct{}
}

func newHelper(t *testing.T) *helper {
	deps := controller.NewSimpleClientDependencies()
	stop := make(chan struct{})
	deps.InformerFactory.Start(stop)
	deps.KubeInformerFactory.Start(stop)
	deps.InformerFactory.WaitForCacheSync(stop)
	deps.KubeInformerFactory.WaitForCacheSync(stop)

	return &helper{
		t:    t,
		deps: deps,
		stop: stop,
	}
}

func (h *helper) close() {
	close(h.stop)
}

// check for exists num Backup and return the exists backups "BackupList".
func (h *helper) checkBacklist(ns string, num int) (bks *v1alpha1.BackupList) {
	t := h.t
	deps := h.deps
	g := NewGomegaWithT(t)

	t.Helper()
	g.Eventually(func() error {
		var err error
		bks, err = deps.Clientset.PingcapV1alpha1().Backups(ns).List(metav1.ListOptions{})
		g.Expect(err).Should(BeNil())
		if len(bks.Items) != num {
			var names []string
			for _, bk := range bks.Items {
				names = append(names, bk.Name)
			}
			return fmt.Errorf("there %d backup, but should be %d, cur backups: %v", len(bks.Items), num, names)
		}
		return nil
	}, time.Second*30).Should(BeNil())

	g.Eventually(func() error {
		var err error
		bks, err := deps.BackupLister.Backups(ns).List(labels.Everything())
		if err == nil && len(bks) == num {
			return nil
		}
		return fmt.Errorf("there %d backup, but should be %d", len(bks), num)
	}, time.Second*30).Should(BeNil())

	return
}

func (h *helper) updateBackup(bk *v1alpha1.Backup) {
	t := h.t
	deps := h.deps
	g := NewGomegaWithT(t)
	_, err := deps.Clientset.PingcapV1alpha1().Backups(bk.Namespace).Update(bk)
	g.Expect(err).Should(BeNil())

	g.Eventually(func() error {
		get, err := deps.BackupLister.Backups(bk.Namespace).Get(bk.Name)
		if err != nil {
			return err
		}

		diff := cmp.Diff(get, bk)
		if diff == "" {
			return nil
		}

		return fmt.Errorf("not synced yet: %s", diff)
	}, time.Second*10).Should(BeNil())
}

func (h *helper) createBackup(bk *v1alpha1.Backup) {
	t := h.t
	deps := h.deps
	g := NewGomegaWithT(t)
	_, err := deps.Clientset.PingcapV1alpha1().Backups(bk.Namespace).Create(bk)
	g.Expect(err).Should(BeNil())
	g.Eventually(func() error {
		_, err := deps.BackupLister.Backups(bk.Namespace).Get(bk.Name)
		return err
	}, time.Second*10).Should(BeNil())
}

func (h *helper) deleteBackup(bk *v1alpha1.Backup) {
	t := h.t
	deps := h.deps
	g := NewGomegaWithT(t)
	err := deps.Clientset.PingcapV1alpha1().Backups(bk.Namespace).Delete(bk.Name, nil)
	g.Expect(err).Should(BeNil())
	g.Eventually(func() error {
		_, err := deps.BackupLister.Backups(bk.Namespace).Get(bk.Name)
		return err
	}, time.Second*10).ShouldNot(BeNil())
}
