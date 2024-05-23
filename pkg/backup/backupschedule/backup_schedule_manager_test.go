// Copyright 2023 PingCAP, Inc.
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
		bks := helper.checkBacklist(bs.Namespace, i+10, false)
		// complete the backup created
		for i := range bks.Items {
			bk := bks.Items[i]
			changed := v1alpha1.UpdateBackupCondition(&bk.Status, &v1alpha1.BackupCondition{
				Type:   v1alpha1.BackupComplete,
				Status: v1.ConditionTrue,
			})
			if changed {
				bk.CreationTimestamp = metav1.Time{Time: m.now()}
				bk.Status.CommitTs = getTSOStr(m.now().Add(10 * time.Minute).Unix())
				t.Log("complete backup: ", bk.Name)
				helper.updateBackup(&bk)
			}
			g.Expect(err).Should(BeNil())
		}
	}

	t.Log("test setting MaxBackups")
	m.now = time.Now
	bs.Spec.MaxBackups = pointer.Int32Ptr(5)
	err = m.Sync(bs)
	g.Expect(err).Should(BeNil())
	helper.checkBacklist(bs.Namespace, 5, false)

	t.Log("test setting MaxReservedTime")
	bs.Spec.MaxBackups = nil
	bs.Spec.MaxReservedTime = pointer.StringPtr("71h")
	err = m.Sync(bs)
	g.Expect(err).Should(BeNil())
	helper.checkBacklist(bs.Namespace, 3, false)

	t.Log("test has log backup")
	bs.Spec.LogBackupTemplate = &v1alpha1.BackupSpec{Mode: v1alpha1.BackupModeLog}
	logBackup := buildLogBackup(bs, now.Add(-72*time.Hour))
	logBackup.Status.CommitTs = getTSOStr(now.Add(-72 * time.Hour).Unix())
	logBackup.Status.LogCheckpointTs = getTSOStr(now.Unix())
	helper.createBackup(logBackup)
	bs.Status.LogBackup = &logBackup.Name
	bs.Spec.MaxReservedTime = pointer.StringPtr("23h")
	err = m.Sync(bs)
	g.Expect(err).Should(BeNil())
	helper.checkBacklist(bs.Namespace, 2, true)
}
func TestMultiSchedules(t *testing.T) {
	g := NewGomegaWithT(t)
	helper := newHelper(t)
	defer helper.close()
	deps := helper.deps
	m := NewBackupScheduleManager(deps).(*backupScheduleManager)
	var err error
	bs1 := &v1alpha1.BackupSchedule{}
	bs1.Namespace = "ns"
	bs1.Name = "bsname1"
	bs1.Status.LastBackup = "bs1_backupname"

	bk1 := &v1alpha1.Backup{}
	bk1.Namespace = bs1.Namespace
	bk1.Name = bs1.Status.LastBackup
	bk1.Status.Conditions = append(bk1.Status.Conditions, v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupComplete,
		Status: v1.ConditionTrue,
	})
	helper.createBackup(bk1)
	helper.createBackupSchedule(bs1)

	err = m.canPerformNextBackup(bs1)
	g.Expect(err).Should(BeNil())

	// create another schedule, without the special label
	bs2 := &v1alpha1.BackupSchedule{}
	bs2.Namespace = "ns"
	bs2.Name = "bsname2"
	bs2.Status.LastBackup = "bs2_backupname"

	// test backup complete
	bk2 := &v1alpha1.Backup{}
	bk2.Namespace = bs2.Namespace
	bk2.Name = bs2.Status.LastBackup
	bk2.Status.Conditions = append(bk2.Status.Conditions, v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupComplete,
		Status: v1.ConditionTrue,
	})
	helper.createBackup(bk2)
	helper.createBackupSchedule(bs2)
	err = m.canPerformNextBackup(bs2)
	g.Expect(err).Should(BeNil())
	helper.deleteBackup(bk1)
	helper.deleteBackup(bk2)
	helper.deleteBackupSchedule(bs1)
	helper.deleteBackupSchedule(bs2)

	// make 2 schedules in the same group, but neither has active backup
	bs11 := &v1alpha1.BackupSchedule{}
	bs11.Namespace = "ns"
	bs11.Name = "bsname11"
	bs11.Labels = label.NewBackupScheduleGroup("group1")
	bs11.Status.LastBackup = "bs11_backupname"

	bk11 := &v1alpha1.Backup{}
	bk11.Namespace = bs11.Namespace
	bk11.Name = bs11.Status.LastBackup
	bk11.Status.Conditions = append(bk11.Status.Conditions, v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupComplete,
		Status: v1.ConditionTrue,
	})
	helper.createBackup(bk11)
	helper.createBackupSchedule(bs11)
	err = m.canPerformNextBackup(bs11)
	g.Expect(err).Should(BeNil())

	// create another schedule
	bs12 := &v1alpha1.BackupSchedule{}
	bs12.Namespace = "ns"
	bs12.Name = "bsname12"
	bs12.Labels = label.NewBackupScheduleGroup("group1")
	bs12.Status.LastBackup = "bs12_backupname"

	// test backup complete
	bk12 := &v1alpha1.Backup{}
	bk12.Namespace = bs12.Namespace
	bk12.Name = bs12.Status.LastBackup
	bk12.Status.Conditions = append(bk12.Status.Conditions, v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupComplete,
		Status: v1.ConditionTrue,
	})
	helper.createBackup(bk12)
	helper.createBackupSchedule(bs12)
	err = m.canPerformNextBackup(bs12)
	g.Expect(err).Should(BeNil())
	helper.deleteBackup(bk11)
	helper.deleteBackup(bk12)
	helper.deleteBackupSchedule(bs11)
	helper.deleteBackupSchedule(bs12)

	// make 2 schedules in the same group, has conflicting backup
	bs21 := &v1alpha1.BackupSchedule{}
	bs21.Namespace = "ns"
	bs21.Name = "bsname21"
	bs21.Labels = label.NewBackupScheduleGroup("group2")
	bs21.Status.LastBackup = "bs21_backupname"

	bk21 := &v1alpha1.Backup{}
	bk21.Namespace = bs21.Namespace
	bk21.Name = bs21.Status.LastBackup
	bk21.Status.Conditions = append(bk21.Status.Conditions, v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupRunning,
		Status: v1.ConditionTrue,
	})
	helper.createBackup(bk21)
	helper.createBackupSchedule(bs21)
	err = m.canPerformNextBackup(bs21)
	g.Expect(err.Error()).Should(MatchRegexp("backup schedule ns/bsname21, the last backup bs21_backupname is still running"))

	// create another schedule
	bs22 := &v1alpha1.BackupSchedule{}
	bs22.Namespace = "ns"
	bs22.Name = "bsname22"
	bs22.Labels = label.NewBackupScheduleGroup("group2")
	bs22.Status.LastBackup = "bs22_backupname"

	// test backup complete
	bk22 := &v1alpha1.Backup{}
	bk22.Namespace = bs22.Namespace
	bk22.Name = bs22.Status.LastBackup
	bk22.Status.Conditions = append(bk22.Status.Conditions, v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupComplete,
		Status: v1.ConditionTrue,
	})
	helper.createBackup(bk22)
	helper.createBackupSchedule(bs22)
	err = m.canPerformNextBackup(bs22)
	g.Expect(err.Error()).Should(MatchRegexp("backup schedule ns/bsname22, the last backup bs21_backupname is still running"))
	helper.deleteBackup(bk21)
	helper.deleteBackup(bk22)
	helper.deleteBackupSchedule(bs21)
	helper.deleteBackupSchedule(bs22)
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
	bs.Status.LastBackupTime.Time = now.AddDate(-10, 0, 0)
	getTime, err = getLastScheduledTime(bs, time.Now)
	g.Expect(err).Should(BeNil())
	g.Expect(getTime).Should(BeNil())
	// next reconcile should succeed
	getTime, err = getLastScheduledTime(bs, time.Now)
	g.Expect(err).Should(BeNil())
	g.Expect(getTime).ShouldNot(BeNil())
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

func TestCalculateExpiredBackupsWithLogBackup(t *testing.T) {
	g := NewGomegaWithT(t)
	type testCase struct {
		backups                   []*v1alpha1.Backup
		logBackup                 *v1alpha1.Backup
		reservedTime              time.Duration
		expectedDeleteBackupCount int
		expectedTruncateTS        uint64
	}

	var (
		now       = time.Now()
		last10Min = now.Add(-time.Minute * 10).Unix()
		last1Day  = now.Add(-time.Hour * 24 * 1).Unix()
		last2Day  = now.Add(-time.Hour * 24 * 2).Unix()
		last3Day  = now.Add(-time.Hour * 24 * 3).Unix()
		last4Day  = now.Add(-time.Hour * 24 * 4).Unix()
	)

	testCases := []*testCase{
		// no backup should be deleted and log backup just start, no commit ts/checkpoint ts
		{
			backups: []*v1alpha1.Backup{
				fakeBackup(&last10Min),
			},
			logBackup:                 fakeLogBackup(nil, nil),
			reservedTime:              24 * time.Hour,
			expectedDeleteBackupCount: 0,
			expectedTruncateTS:        0,
		},
		// backup should be delete and log backup just start, no commit ts/checkpoint ts
		{
			backups: []*v1alpha1.Backup{
				fakeBackup(&last3Day),
				fakeBackup(&last2Day),
				fakeBackup(&last1Day),
				fakeBackup(&last10Min),
			},
			logBackup:                 fakeLogBackup(&last10Min, nil),
			reservedTime:              24 * time.Hour,
			expectedDeleteBackupCount: 1,
			expectedTruncateTS:        0,
		},
		// no backup should be deleted, has log backup
		{
			backups: []*v1alpha1.Backup{
				fakeBackup(&last3Day),
				fakeBackup(&last1Day),
			},
			logBackup:                 fakeLogBackup(&last4Day, &last10Min),
			reservedTime:              24 * time.Hour,
			expectedDeleteBackupCount: 0,
			expectedTruncateTS:        0,
		},
		// 1 backup should be deleted, no log backup should be truncated
		{
			backups: []*v1alpha1.Backup{
				fakeBackup(&last3Day),
				fakeBackup(&last2Day),
				fakeBackup(&last1Day),
			},
			logBackup:                 fakeLogBackup(&last1Day, &last10Min),
			reservedTime:              24 * time.Hour,
			expectedDeleteBackupCount: 1,
			expectedTruncateTS:        0,
		},
		// 2 backup should be deleted, no log backup should be truncated
		{
			backups: []*v1alpha1.Backup{
				fakeBackup(&last4Day),
				fakeBackup(&last3Day),
				fakeBackup(&last2Day),
				fakeBackup(&last1Day),
			},
			logBackup:                 fakeLogBackup(&last1Day, &last10Min),
			reservedTime:              24 * time.Hour,
			expectedDeleteBackupCount: 2,
			expectedTruncateTS:        0,
		},
		// 2 backup should be deleted, has log backup should be truncated
		{
			backups: []*v1alpha1.Backup{
				fakeBackup(&last4Day),
				fakeBackup(&last3Day),
				fakeBackup(&last2Day),
				fakeBackup(&last1Day),
			},
			logBackup:                 fakeLogBackup(&last3Day, &last10Min),
			reservedTime:              24 * time.Hour,
			expectedDeleteBackupCount: 2,
			expectedTruncateTS:        getTSO(last2Day),
		},
	}

	for _, tc := range testCases {
		deletedBackups, truncateTS, err := calExpiredBackupsAndLogBackup(tc.backups, tc.logBackup, tc.reservedTime)
		g.Expect(err).Should(BeNil())
		g.Expect(len(deletedBackups)).Should(Equal(tc.expectedDeleteBackupCount))
		g.Expect(truncateTS).Should(Equal(tc.expectedTruncateTS))
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
func (h *helper) checkBacklist(ns string, num int, checkLogBackupTruncate bool) (bks *v1alpha1.BackupList) {
	t := h.t
	deps := h.deps
	g := NewGomegaWithT(t)

	check := func(backups []*v1alpha1.Backup) error {
		snapshotBackups, logBackup := separateAllSnapshotBackupsAndLogBackup(backups)
		// check snapshot backup num
		if len(snapshotBackups) != num {
			var names []string
			for _, bk := range snapshotBackups {
				names = append(names, bk.Name)
			}
			return fmt.Errorf("there %d backup, but should be %d, cur backups: %v", len(snapshotBackups), num, names)
		}
		if !checkLogBackupTruncate {
			return nil
		}
		// check has log backup
		if logBackup == nil {
			return fmt.Errorf("there is no log backup, but should have")
		}
		// check truncateTSO, it should equal the earliest snapshot backup after gc
		if len(snapshotBackups) == 0 {
			return fmt.Errorf("there should have snapshot backup if need check log backup truncate tso")
		}
		if logBackup.Status.LogSuccessTruncateUntil != snapshotBackups[0].Spec.CommitTs {
			return fmt.Errorf("log backup truncate tso should be %s, but cur is %s", snapshotBackups[0].Spec.CommitTs, logBackup.Status.LogSuccessTruncateUntil)
		}
		return nil
	}

	t.Helper()
	g.Eventually(func() error {
		var err error
		bks, err = deps.Clientset.PingcapV1alpha1().Backups(ns).List(context.TODO(), metav1.ListOptions{})
		g.Expect(err).Should(BeNil())
		backups := convertToBackupPtrList(bks.Items)
		return check(backups)
	}, time.Second*30).Should(BeNil())

	g.Eventually(func() error {
		var err error
		backups, err := deps.BackupLister.Backups(ns).List(labels.Everything())
		g.Expect(err).Should(BeNil())
		return check(backups)
	}, time.Second*30).Should(BeNil())

	return
}

func convertToBackupPtrList(backups []v1alpha1.Backup) []*v1alpha1.Backup {
	backupPtrs := make([]*v1alpha1.Backup, 0)
	for i := 0; i < len(backups); i++ {
		backupPtrs = append(backupPtrs, &backups[i])
	}
	return backupPtrs
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

func (h *helper) createBackupSchedule(bk *v1alpha1.BackupSchedule) {
	t := h.t
	deps := h.deps
	g := NewGomegaWithT(t)
	_, err := deps.Clientset.PingcapV1alpha1().BackupSchedules(bk.Namespace).Create(context.TODO(), bk, metav1.CreateOptions{})
	g.Expect(err).Should(BeNil())
	g.Eventually(func() error {
		_, err := deps.BackupScheduleLister.BackupSchedules(bk.Namespace).Get(bk.Name)
		return err
	}, time.Second*10).Should(BeNil())
}

func (h *helper) deleteBackupSchedule(bk *v1alpha1.BackupSchedule) {
	t := h.t
	deps := h.deps
	g := NewGomegaWithT(t)
	err := deps.Clientset.PingcapV1alpha1().BackupSchedules(bk.Namespace).Delete(context.TODO(), bk.Name, metav1.DeleteOptions{})
	g.Expect(err).Should(BeNil())
	g.Eventually(func() error {
		_, err := deps.BackupScheduleLister.BackupSchedules(bk.Namespace).Get(bk.Name)
		return err
	}, time.Second*10).ShouldNot(BeNil())
}

func fakeBackup(ts *int64) *v1alpha1.Backup {
	backup := &v1alpha1.Backup{}
	if ts == nil {
		return backup
	}
	backup.Status.CommitTs = getTSOStr(*ts)
	return backup
}

func fakeLogBackup(startTS, checkPointTS *int64) *v1alpha1.Backup {
	logBackup := &v1alpha1.Backup{}
	if startTS == nil {
		return logBackup
	}
	logBackup.Status.CommitTs = getTSOStr(*startTS)
	if checkPointTS == nil {
		return logBackup
	}
	logBackup.Status.LogCheckpointTs = getTSOStr(*checkPointTS)
	return logBackup
}

func getTSOStr(ts int64) string {
	tso := getTSO(ts)
	return strconv.FormatUint(tso, 10)
}

func getTSO(ts int64) uint64 {
	return uint64((ts << 18) * 1000)
}
