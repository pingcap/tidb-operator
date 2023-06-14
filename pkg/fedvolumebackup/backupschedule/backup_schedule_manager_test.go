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
	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
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
	bs := &v1alpha1.VolumeBackupSchedule{}
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
	bk := &v1alpha1.VolumeBackup{}
	bk.Namespace = bs.Namespace
	bk.Name = bs.Status.LastBackup
	bk.Status.Conditions = append(bk.Status.Conditions, v1alpha1.VolumeBackupCondition{
		Type:   v1alpha1.VolumeBackupComplete,
		Status: v1.ConditionTrue,
	})
	helper.createBackup(bk)
	err = m.canPerformNextBackup(bs)
	g.Expect(err).Should(BeNil())
	helper.deleteBackup(bk)

	// test last backup failed state
	bk.Status.Conditions = nil
	bk.Status.Conditions = append(bk.Status.Conditions, v1alpha1.VolumeBackupCondition{
		Type:   v1alpha1.VolumeBackupFailed,
		Status: v1.ConditionTrue,
	})
	helper.createBackup(bk)
	err = m.canPerformNextBackup(bs)
	g.Expect(err).Should(BeNil())
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
		for i := range bks.Items {
			bk := bks.Items[i].DeepCopy()
			changed := !v1alpha1.IsVolumeBackupComplete(bk)
			v1alpha1.UpdateVolumeBackupCondition(&bk.Status, &v1alpha1.VolumeBackupCondition{
				Type:   v1alpha1.VolumeBackupComplete,
				Status: v1.ConditionTrue,
			})

			if changed {
				bk.CreationTimestamp = metav1.Time{Time: m.now()}
				bk.Status.CommitTs = getTSOStr(m.now().Add(10 * time.Minute).Unix())
				t.Log("complete backup: ", bk.Name)
				helper.updateBackup(bk)
			}

			g.Expect(err).Should(BeNil())
		}
	}

	t.Log("test setting MaxBackups")
	m.now = time.Now
	bs.Spec.MaxBackups = pointer.Int32Ptr(5)
	err = m.Sync(bs)
	g.Expect(err).Should(BeNil())
	helper.checkBacklist(bs.Namespace, 5)

	t.Log("test setting MaxReservedTime")
	bs.Spec.MaxBackups = nil
	bs.Spec.MaxReservedTime = pointer.StringPtr("71h")
	err = m.Sync(bs)
	g.Expect(err).Should(BeNil())
	helper.checkBacklist(bs.Namespace, 3)
}

func TestGetLastScheduledTime(t *testing.T) {
	g := NewGomegaWithT(t)

	bs := &v1alpha1.VolumeBackupSchedule{
		Spec: v1alpha1.VolumeBackupScheduleSpec{},
		Status: v1alpha1.VolumeBackupScheduleStatus{
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
	var get *v1alpha1.VolumeBackup

	// build BackupSchedule template
	bs := &v1alpha1.VolumeBackupSchedule{
		Spec: v1alpha1.VolumeBackupScheduleSpec{},
		Status: v1alpha1.VolumeBackupScheduleStatus{
			LastBackupTime: &metav1.Time{},
		},
	}
	bs.Namespace = "ns"
	bs.Name = "bsname"

	// build Backup template
	bk := &v1alpha1.VolumeBackup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: bs.Namespace,
			Name:      bs.GetBackupCRDName(now),
			Labels:    label.NewBackupSchedule().Instance(bs.Name).BackupSchedule(bs.Name).Labels(),
			OwnerReferences: []metav1.OwnerReference{
				controller.GetFedVolumeBackupScheduleOwnerRef(bs),
			},
		},
		Spec: v1alpha1.VolumeBackupSpec{},
	}

	// test BR == nil
	get = buildBackup(bs, now)
	if diff := cmp.Diff(bk, get); diff != "" {
		t.Errorf("unexpected (-want, +got): %s", diff)
	}
	get = buildBackup(bs, now)
	if diff := cmp.Diff(bk, get); diff != "" {
		t.Errorf("unexpected (-want, +got): %s", diff)
	}

	// test BR != nil
	bs.Spec.BackupTemplate.Template.BR = &v1alpha1.BRConfig{}
	bk.Spec.Template.BR = bs.Spec.BackupTemplate.Template.BR.DeepCopy()
	get = buildBackup(bs, now)
	if diff := cmp.Diff(bk, get); diff != "" {
		t.Errorf("unexpected (-want, +got): %s", diff)
	}
}

func TestCalculateExpiredBackups(t *testing.T) {
	g := NewGomegaWithT(t)
	type testCase struct {
		backups                   []*v1alpha1.VolumeBackup
		reservedTime              time.Duration
		expectedDeleteBackupCount int
	}

	var (
		now       = time.Now()
		last10Min = now.Add(-time.Minute * 10).Unix()
		last1Day  = now.Add(-time.Hour * 24 * 1).Unix()
		last2Day  = now.Add(-time.Hour * 24 * 2).Unix()
		last3Day  = now.Add(-time.Hour * 24 * 3).Unix()
	)

	testCases := []*testCase{
		// no backup should be deleted
		{
			backups: []*v1alpha1.VolumeBackup{
				fakeBackup(&last10Min),
			},
			reservedTime:              24 * time.Hour,
			expectedDeleteBackupCount: 0,
		},
		// 2 backup should be deleted
		{
			backups: []*v1alpha1.VolumeBackup{
				fakeBackup(&last3Day),
				fakeBackup(&last2Day),
				fakeBackup(&last1Day),
				fakeBackup(&last10Min),
			},
			reservedTime:              24 * time.Hour,
			expectedDeleteBackupCount: 2,
		},
	}

	for _, tc := range testCases {
		deletedBackups, err := calculateExpiredBackups(tc.backups, tc.reservedTime)
		g.Expect(err).Should(BeNil())
		g.Expect(len(deletedBackups)).Should(Equal(tc.expectedDeleteBackupCount))
	}
}

type helper struct {
	t    *testing.T
	deps *controller.BrFedDependencies
	stop chan struct{}
}

func newHelper(t *testing.T) *helper {
	deps := controller.NewSimpleFedClientDependencies()
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
func (h *helper) checkBacklist(ns string, num int) (bks *v1alpha1.VolumeBackupList) {
	t := h.t
	deps := h.deps
	g := NewGomegaWithT(t)

	check := func(backups []*v1alpha1.VolumeBackup) error {
		snapshotBackups := sortSnapshotBackups(backups)
		// check snapshot backup num
		if len(snapshotBackups) != num {
			var names []string
			for _, bk := range snapshotBackups {
				names = append(names, bk.Name)
			}
			return fmt.Errorf("there %d backup, but should be %d, cur backups: %v", len(snapshotBackups), num, names)
		}
		// check truncateTSO, it should equal the earliest snapshot backup after gc
		if len(snapshotBackups) == 0 {
			return fmt.Errorf("there should have snapshot backup if need check log backup truncate tso")
		}
		return nil
	}

	t.Helper()
	g.Eventually(func() error {
		var err error
		bks, err = deps.Clientset.FederationV1alpha1().VolumeBackups(ns).List(context.TODO(), metav1.ListOptions{})
		g.Expect(err).Should(BeNil())
		backups := convertToBackupPtrList(bks.Items)
		return check(backups)
	}, time.Second*30).Should(BeNil())

	g.Eventually(func() error {
		var err error
		backups, err := deps.VolumeBackupLister.VolumeBackups(ns).List(labels.Everything())
		g.Expect(err).Should(BeNil())
		return check(backups)
	}, time.Second*30).Should(BeNil())

	return
}

func convertToBackupPtrList(backups []v1alpha1.VolumeBackup) []*v1alpha1.VolumeBackup {
	backupPtrs := make([]*v1alpha1.VolumeBackup, 0)
	for i := 0; i < len(backups); i++ {
		backupPtrs = append(backupPtrs, &backups[i])
	}
	return backupPtrs
}

func (h *helper) createBackup(bk *v1alpha1.VolumeBackup) {
	t := h.t
	deps := h.deps
	g := NewGomegaWithT(t)
	_, err := deps.Clientset.FederationV1alpha1().VolumeBackups(bk.Namespace).Create(context.TODO(), bk, metav1.CreateOptions{})
	g.Expect(err).Should(BeNil())
	g.Eventually(func() error {
		_, err := deps.VolumeBackupLister.VolumeBackups(bk.Namespace).Get(bk.Name)
		return err
	}, time.Second*10).Should(BeNil())
}

func (h *helper) deleteBackup(bk *v1alpha1.VolumeBackup) {
	t := h.t
	deps := h.deps
	g := NewGomegaWithT(t)
	err := deps.Clientset.FederationV1alpha1().VolumeBackups(bk.Namespace).Delete(context.TODO(), bk.Name, metav1.DeleteOptions{})
	g.Expect(err).Should(BeNil())
	g.Eventually(func() error {
		_, err := deps.VolumeBackupLister.VolumeBackups(bk.Namespace).Get(bk.Name)
		return err
	}, time.Second*10).ShouldNot(BeNil())
}

func (h *helper) updateBackup(bk *v1alpha1.VolumeBackup) {
	t := h.t
	deps := h.deps
	g := NewGomegaWithT(t)
	_, err := deps.Clientset.FederationV1alpha1().VolumeBackups(bk.Namespace).Update(context.TODO(), bk, metav1.UpdateOptions{})
	g.Expect(err).Should(BeNil())

	g.Eventually(func() error {
		get, err := deps.VolumeBackupLister.VolumeBackups(bk.Namespace).Get(bk.Name)
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

func fakeBackup(ts *int64) *v1alpha1.VolumeBackup {
	backup := &v1alpha1.VolumeBackup{}
	if ts == nil {
		return backup
	}
	backup.Status.CommitTs = getTSOStr(*ts)
	return backup
}

func getTSOStr(ts int64) string {
	tso := getTSO(ts)
	return strconv.FormatUint(tso, 10)
}

func getTSO(ts int64) uint64 {
	return uint64((ts << 18) * 1000)
}
