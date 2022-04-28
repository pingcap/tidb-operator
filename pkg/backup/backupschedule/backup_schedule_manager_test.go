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
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/controller"
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
}

func TestManagerGC(t *testing.T) {
	g := NewGomegaWithT(t)

	type subcase struct {
		desc string
		// maxBackupsConf contains three entries, which are MaxBackups, MaxCompletedBackups, MaxFailedBackups respectively.
		maxBackupsConf  [3]*int32
		maxReservedTime *string
		// expectedBackups contains three entries, which are expectedBackups, expectedCompletedBackups, expectedFailedBackups respectively.
		expectedBackups               [3]int
		expectedBackupsByReservedTime int

		initFn func(hp *helper, m *backupScheduleManager, bs *v1alpha1.BackupSchedule)
	}

	init := func(helper *helper, m *backupScheduleManager, bs *v1alpha1.BackupSchedule) {
		now := time.Now()
		m.now = func() time.Time { return now.AddDate(0, 0, -101) }
		m.resetLastBackup(bs)
		// create first 5 CompletedBackups and last 5 FailedBackups, one per day
		for i := -9; i <= 0; i++ {
			t.Log("loop id ", i)
			m.now = func() time.Time { return now.AddDate(0, 0, i) }
			err := m.Sync(bs)
			g.Expect(err).Should(BeNil())
			bks := helper.checkBacklist(bs.Namespace, i+10, nil)
			for i := range bks.Items {
				cond := v1alpha1.BackupFailed
				if i < 5 {
					cond = v1alpha1.BackupComplete
				}
				bk := bks.Items[i]
				v1alpha1.UpdateBackupCondition(&bk.Status, &v1alpha1.BackupCondition{
					Type:   v1alpha1.BackupScheduled,
					Status: v1.ConditionTrue,
				})
				changed := v1alpha1.UpdateBackupCondition(&bk.Status, &v1alpha1.BackupCondition{
					Type:   cond,
					Status: v1.ConditionTrue,
				})
				if changed {
					bk.CreationTimestamp = metav1.Time{Time: m.now()}
					helper.updateBackup(&bk)
				}
			}
		}
	}

	cases := []subcase{
		{
			desc:                          "setting MaxBackups and MaxReserveTime",
			maxBackupsConf:                [3]*int32{pointer.Int32Ptr(5)},
			maxReservedTime:               pointer.StringPtr("23h"),
			initFn:                        init,
			expectedBackups:               [3]int{5, -1, -1},
			expectedBackupsByReservedTime: 1,
		},
		{
			desc:            "setting MaxBackups, MaxCompletedBackups, MaxFailedBackups",
			maxBackupsConf:  [3]*int32{pointer.Int32Ptr(2), pointer.Int32Ptr(3), pointer.Int32Ptr(1)},
			initFn:          init,
			expectedBackups: [3]int{2, 1, 1},
		},
		{
			desc:            "setting MaxCompletedBackups, MaxFailedBackups without MaxBackups",
			maxBackupsConf:  [3]*int32{nil, pointer.Int32Ptr(1), pointer.Int32Ptr(0)},
			initFn:          init,
			expectedBackups: [3]int{1, 1, 0},
		},
		{
			desc:                          "setting MaxFailedBackups with MaxReservedTime",
			maxBackupsConf:                [3]*int32{nil, nil, pointer.Int32Ptr(1)},
			maxReservedTime:               pointer.StringPtr("23h"),
			initFn:                        init,
			expectedBackups:               [3]int{6, 5, 1},
			expectedBackupsByReservedTime: 1,
		},
	}

	for i, c := range cases {
		t.Log(c.desc)
		helper := newHelper(t)
		defer helper.close()
		deps := helper.deps
		bs := &v1alpha1.BackupSchedule{}
		bs.Namespace = "ns" + strconv.Itoa(i)
		bs.Name = "bsname"
		bs.Spec.Schedule = "0 0 * * *" // Run at midnight every day
		m := NewBackupScheduleManager(deps).(*backupScheduleManager)
		c.initFn(helper, m, bs)
		m.now = time.Now
		bs.Spec.MaxBackups, bs.Spec.MaxCompletedBackups, bs.Spec.MaxFailedBackups = c.maxBackupsConf[0], c.maxBackupsConf[1], c.maxBackupsConf[2]
		// perform backupGC
		err := m.Sync(bs)
		g.Expect(err).Should(BeNil())

		helper.checkBacklist(bs.Namespace, c.expectedBackups[0], nil)
		if c.expectedBackups[1] != -1 {
			helper.checkBacklist(bs.Namespace, c.expectedBackups[1], v1alpha1.IsBackupComplete)
		}
		if c.expectedBackups[2] != -1 {
			helper.checkBacklist(bs.Namespace, c.expectedBackups[2], v1alpha1.IsBackupFailed)
		}

		if c.maxReservedTime != nil {
			bs.Spec.MaxReservedTime = c.maxReservedTime
			err = m.Sync(bs)
			g.Expect(err).Should(BeNil())
			helper.checkBacklist(bs.Namespace, c.expectedBackupsByReservedTime, nil)
		}
	}
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
func (h *helper) checkBacklist(ns string, num int, cond func(*v1alpha1.Backup) bool) (bks *v1alpha1.BackupList) {
	t := h.t
	deps := h.deps
	g := NewGomegaWithT(t)

	t.Helper()
	g.Eventually(func() error {
		var err error
		bks, err = deps.Clientset.PingcapV1alpha1().Backups(ns).List(context.TODO(), metav1.ListOptions{})
		g.Expect(err).Should(BeNil())
		if cond != nil {
			filtered := bks.Items[:0]
			for i := range bks.Items {
				if cond(&bks.Items[i]) {
					filtered = append(filtered, bks.Items[i])
				}
			}
			bks.Items = filtered
		}
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
		g.Expect(err).Should(BeNil())
		if cond != nil {
			filtered := bks[:0]
			for i := range bks {
				if cond(bks[i]) {
					filtered = append(filtered, bks[i])
				}
			}
			bks = filtered
		}
		if len(bks) == num {
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
	_, err := deps.Clientset.PingcapV1alpha1().Backups(bk.Namespace).Update(context.TODO(), bk, metav1.UpdateOptions{})
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
	_, err := deps.Clientset.PingcapV1alpha1().Backups(bk.Namespace).Create(context.TODO(), bk, metav1.CreateOptions{})
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
	err := deps.Clientset.PingcapV1alpha1().Backups(bk.Namespace).Delete(context.TODO(), bk.Name, metav1.DeleteOptions{})
	g.Expect(err).Should(BeNil())
	g.Eventually(func() error {
		_, err := deps.BackupLister.Backups(bk.Namespace).Get(bk.Name)
		return err
	}, time.Second*10).ShouldNot(BeNil())
}
