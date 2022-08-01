// Copyright 2021 PingCAP, Inc.
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

package v1alpha1

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestComponentStatus(t *testing.T) {

	tc := &TidbCluster{
		Spec: TidbClusterSpec{
			PD:      &PDSpec{},
			TiDB:    &TiDBSpec{},
			TiKV:    &TiKVSpec{},
			TiFlash: &TiFlashSpec{},
			Pump:    &PumpSpec{},
			TiCDC:   &TiCDCSpec{},
		},
	}
	dc := &DMCluster{
		Spec: DMClusterSpec{
			Master: MasterSpec{},
			Worker: &WorkerSpec{},
		},
	}

	t.Run("MemberType", func(t *testing.T) {
		g := NewGomegaWithT(t)

		components := allComponentStatus(tc.DeepCopy(), dc.DeepCopy())
		for _, status := range components {
			switch status.MemberType() {
			case PDMemberType:
				_, ok := status.(*PDStatus)
				g.Expect(ok).To(BeTrue())
			case TiDBMemberType:
				_, ok := status.(*TiDBStatus)
				g.Expect(ok).To(BeTrue())
			case TiKVMemberType:
				_, ok := status.(*TiKVStatus)
				g.Expect(ok).To(BeTrue())
			case TiFlashMemberType:
				_, ok := status.(*TiFlashStatus)
				g.Expect(ok).To(BeTrue())
			case TiCDCMemberType:
				_, ok := status.(*TiCDCStatus)
				g.Expect(ok).To(BeTrue())
			case PumpMemberType:
				_, ok := status.(*PumpStatus)
				g.Expect(ok).To(BeTrue())
			case DMMasterMemberType:
				_, ok := status.(*MasterStatus)
				g.Expect(ok).To(BeTrue())
			case DMWorkerMemberType:
				_, ok := status.(*WorkerStatus)
				g.Expect(ok).To(BeTrue())
			}
		}
	})

	t.Run("Conditions", func(t *testing.T) {
		g := NewGomegaWithT(t)
		components := allComponentStatus(tc.DeepCopy(), dc.DeepCopy())
		for _, status := range components {
			conds := status.GetConditions()
			g.Expect(conds).To(BeNil())

			// test to add a condition
			condInput := metav1.Condition{
				Type:    "Test",
				Status:  metav1.ConditionTrue,
				Reason:  "Test True Reason",
				Message: "Test True Message",
			}
			status.SetCondition(condInput)
			conds = status.GetConditions()
			condOutput := meta.FindStatusCondition(conds, condInput.Type)
			condOutput.LastTransitionTime = condInput.LastTransitionTime // ignore the last transition time
			g.Expect(cmp.Diff(*condOutput, condInput)).To(BeEmpty())

			// test to update a condition
			condInput.Status = metav1.ConditionFalse
			condInput.Reason = "Test False Reason"
			condInput.Message = "Test False Message"
			status.SetCondition(condInput)
			conds = status.GetConditions()
			condOutput = meta.FindStatusCondition(conds, condInput.Type)
			condOutput.LastTransitionTime = condInput.LastTransitionTime // ignore the last transition time
			g.Expect(cmp.Diff(*condOutput, condInput)).To(BeEmpty())

			// test to remove a condition
			status.RemoveCondition(condInput.Type)
			conds = status.GetConditions()
			g.Expect(conds).To(BeEmpty())
			condOutput = meta.FindStatusCondition(conds, condInput.Type)
			g.Expect(condOutput).To(BeNil())
		}
	})

	t.Run("Synced", func(t *testing.T) {
		g := NewGomegaWithT(t)

		components := allComponentStatus(tc.DeepCopy(), dc.DeepCopy())
		for _, status := range components {
			switch status.MemberType() {
			case TiDBMemberType, PumpMemberType:
				g.Expect(status.GetSynced()).To(BeTrue())
			default:
				g.Expect(status.GetSynced()).To(BeFalse())
			}

			status.SetSynced(true)
			g.Expect(status.GetSynced()).To(BeTrue())
		}
	})

	t.Run("MemberPhase", func(t *testing.T) {
		g := NewGomegaWithT(t)

		components := allComponentStatus(tc.DeepCopy(), dc.DeepCopy())
		for _, status := range components {
			g.Expect(status.GetPhase()).To(BeEmpty())

			phases := []MemberPhase{NormalPhase, UpgradePhase, ScalePhase, SuspendPhase}
			for _, phase := range phases {
				status.SetPhase(phase)
				g.Expect(status.GetPhase()).To(Equal(phase))
			}
		}
	})

	t.Run("StatefulSetStatus", func(t *testing.T) {
		g := NewGomegaWithT(t)

		components := allComponentStatus(tc.DeepCopy(), dc.DeepCopy())
		for _, status := range components {
			g.Expect(status.GetStatefulSet()).To(BeNil())

			sts := &appsv1.StatefulSetStatus{
				ObservedGeneration: 1,
			}
			status.SetStatefulSet(sts)
			g.Expect(status.GetStatefulSet()).To(Equal(sts))
			status.SetStatefulSet(nil)
			g.Expect(status.GetStatefulSet()).To(BeNil())
		}
	})

}

func allComponentStatus(tc *TidbCluster, dc *DMCluster) []ComponentStatus {
	components := tc.AllComponentStatus()
	components = append(components, dc.AllComponentStatus()...)
	return components
}
