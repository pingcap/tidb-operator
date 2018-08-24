package member

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestPDUpgrader_Upgrade(t *testing.T) {
	g := NewGomegaWithT(t)
	upgrader := newPDUpgrader()
	tc := newTidbClusterForPDUpgrader()
	oldSet := newStatefulSetForPDUpgrader()
	newSet := oldSet.DeepCopy()
	newSet.Spec.Template.Spec.Containers[0].Image = "pd-test-images:v2"
	err := upgrader.Upgrade(tc, oldSet, newSet)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tc.Status.PD.Phase).To(Equal(v1alpha1.UpgradePhase))
}

func newPDUpgrader() Upgrader {
	return &pdUpgrader{}
}

func newStatefulSetForPDUpgrader() *apps.StatefulSet {
	return &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "upgrader-pd",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: apps.StatefulSetSpec{
			Replicas: int32Pointer(3),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "pd",
							Image: "pd-test-image",
						},
					},
				},
			},
		},
	}
}

func newTidbClusterForPDUpgrader() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "upgrader",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("upgrader"),
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: v1alpha1.PDSpec{
				ContainerSpec: v1alpha1.ContainerSpec{
					Image: "pd-test-image",
				},
				Replicas:         3,
				StorageClassName: "my-storage-class",
			},
		},
	}
}
