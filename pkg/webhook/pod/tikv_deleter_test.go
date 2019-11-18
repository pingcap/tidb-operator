package pod

import (
	"fmt"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	memberUtils "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	"k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"testing"
)

var (
	deleteTiKVPodOrdinal = int32(2)
	tiKVStsName          = tcName + "-tikv"
	deletetiKVPodName    = memberUtils.TikvPodName(tcName, deleteTiKVPodOrdinal)
)

func TestTiKVDeleterDelete(t *testing.T) {

	g := NewGomegaWithT(t)

	type testcase struct {
		name           string
		isOutOfOrdinal bool
		isUpgrading    bool
		storeState     string
		UpdatePVCErr   bool
		expectFn       func(g *GomegaWithT, response *v1beta1.AdmissionResponse)
	}

	testFn := func(test *testcase) {

		t.Log(test.name)

		deleteTiKVPod := newTiKVPodForTiKVPodAdmissionControl()
		ownerStatefulSet := newOwnerStatefulSetForTiKVPodAdmissionControl()
		tc := newTidbClusterForPodAdmissionControl()
		pvc := newPVCForDeleteTiKVPod()
		kubeCli := kubefake.NewSimpleClientset()

		podAdmissionControl, fakePVCControl, pvcIndexer, _, _, _ := newPodAdmissionControl()
		pdControl := pdapi.NewFakePDControl(kubeCli)
		fakePDClient := controller.NewFakePDClient(pdControl, tc)

		if test.isOutOfOrdinal {
			deleteTiKVPod.Name = memberUtils.TikvPodName(tcName, 3)
			tc.Status.TiKV.Stores["3"] = v1alpha1.TiKVStore{
				PodName:     memberUtils.TikvPodName(tcName, 3),
				LeaderCount: 1,
				State:       v1alpha1.TiKVStateUp,
			}
			pvcIndexer.Add(pvc)
		}

		if test.isUpgrading {
			ownerStatefulSet.Status.CurrentRevision = "1"
			ownerStatefulSet.Status.UpdateRevision = "2"
		}

		fakePVCControl.SetUpdatePVCError(nil, 0)
		if test.UpdatePVCErr {
			fakePVCControl.SetUpdatePVCError(fmt.Errorf("update pvc error"), 0)
		}

		ordinal, _ := operatorUtils.GetOrdinalFromPodName(deleteTiKVPod.Name)
		ordinalStr := fmt.Sprintf("%d", ordinal)
		tc.Status.TiKV.Stores[ordinalStr] = v1alpha1.TiKVStore{
			PodName:     memberUtils.TikvPodName(tcName, ordinal),
			LeaderCount: 1,
			State:       test.storeState,
		}

		response := podAdmissionControl.admitDeleteTiKVPods(deleteTiKVPod, ownerStatefulSet, tc, fakePDClient)
		test.expectFn(g, response)
	}

	tests := []testcase{
		{
			name:           "first normal upgraded",
			isOutOfOrdinal: false,
			isUpgrading:    true,
			storeState:     v1alpha1.TiKVStateUp,
			UpdatePVCErr:   false,
			expectFn: func(g *GomegaWithT, response *v1beta1.AdmissionResponse) {
				g.Expect(response.Allowed, false)
			},
		},
	}

	for _, test := range tests {
		testFn(&test)
	}
}

func newTiKVPodForTiKVPodAdmissionControl() *core.Pod {

	pod := core.Pod{}
	pod.Name = deletetiKVPodName
	pod.Labels = map[string]string{
		label.ComponentLabelKey: label.TiKVLabelVal,
	}
	pod.Namespace = namespace
	return &pod
}

func newOwnerStatefulSetForTiKVPodAdmissionControl() *apps.StatefulSet {
	sts := apps.StatefulSet{}
	sts.Spec.Replicas = func() *int32 { a := int32(deleteTiKVPodOrdinal); return &a }()
	sts.Name = tiKVStsName
	sts.Namespace = namespace
	sts.Status.CurrentRevision = "1"
	sts.Status.UpdateRevision = "1"
	return &sts
}

func newPVCForDeleteTiKVPod() *core.PersistentVolumeClaim {
	return &core.PersistentVolumeClaim{
		ObjectMeta: meta.ObjectMeta{
			Name:      operatorUtils.OrdinalPVCName(v1alpha1.TiKVMemberType, tiKVStsName, deleteTiKVPodOrdinal),
			Namespace: namespace,
		},
	}
}
