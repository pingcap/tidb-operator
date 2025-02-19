package v1alpha1

import (
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

func TestGetBackupOwnerRef(t *testing.T) {
	g := NewGomegaWithT(t)

	b := newBackup()
	b.UID = types.UID("demo-uid")
	ref := GetBackupOwnerRef(b)
	g.Expect(ref.APIVersion).To(Equal(BackupControllerKind.GroupVersion().String()))
	g.Expect(ref.Kind).To(Equal(BackupControllerKind.Kind))
	g.Expect(ref.Name).To(Equal(b.GetName()))
	g.Expect(ref.UID).To(Equal(types.UID("demo-uid")))
	g.Expect(*ref.Controller).To(BeTrue())
	g.Expect(*ref.BlockOwnerDeletion).To(BeTrue())
}

func TestGetRestoreOwnerRef(t *testing.T) {
	g := NewGomegaWithT(t)

	r := newRestore()
	r.UID = types.UID("demo-uid")
	ref := GetRestoreOwnerRef(r)
	g.Expect(ref.APIVersion).To(Equal(RestoreControllerKind.GroupVersion().String()))
	g.Expect(ref.Kind).To(Equal(RestoreControllerKind.Kind))
	g.Expect(ref.Name).To(Equal(r.GetName()))
	g.Expect(ref.UID).To(Equal(types.UID("demo-uid")))
	g.Expect(*ref.Controller).To(BeTrue())
	g.Expect(*ref.BlockOwnerDeletion).To(BeTrue())
}

func TestGetBackupScheduleOwnerRef(t *testing.T) {
	g := NewGomegaWithT(t)

	b := newBackupSchedule()
	b.UID = types.UID("demo-uid")
	ref := GetBackupScheduleOwnerRef(b)
	g.Expect(ref.APIVersion).To(Equal(backupScheduleControllerKind.GroupVersion().String()))
	g.Expect(ref.Kind).To(Equal(backupScheduleControllerKind.Kind))
	g.Expect(ref.Name).To(Equal(b.GetName()))
	g.Expect(ref.UID).To(Equal(types.UID("demo-uid")))
	g.Expect(*ref.Controller).To(BeTrue())
	g.Expect(*ref.BlockOwnerDeletion).To(BeTrue())
}

func newBackup() *Backup {
	backup := &Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-backup",
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				metav1alpha1.NameLabelKey: metav1alpha1.BackupJobLabelVal,
			},
		},
		Spec: BackupSpec{},
	}
	return backup
}

func newRestore() *Restore {
	restore := &Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-backup",
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				metav1alpha1.NameLabelKey: metav1alpha1.RestoreJobLabelVal,
			},
		},
		Spec: RestoreSpec{},
	}
	return restore
}

func newBackupSchedule() *BackupSchedule {
	backup := &BackupSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-backup",
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				metav1alpha1.NameLabelKey: metav1alpha1.BackupScheduleJobLabelVal,
			},
		},
		Spec: BackupScheduleSpec{
			BackupTemplate: BackupSpec{},
		},
	}
	return backup
}
