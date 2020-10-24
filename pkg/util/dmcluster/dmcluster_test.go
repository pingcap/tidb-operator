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

package dmcluster

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCondition(t *testing.T) {
	g := NewGomegaWithT(t)

	var conditions []v1alpha1.DMClusterCondition

	c := NewDMClusterCondition(v1alpha1.DMClusterReady, v1.ConditionTrue, StatfulSetNotUpToDate, "reason1")
	conditions = append(conditions, *c)

	status := v1alpha1.DMClusterStatus{
		Conditions: conditions,
	}

	// test GetDMClusterCondition
	getc := GetDMClusterCondition(status, v1alpha1.DMClusterReady)
	g.Expect(getc).Should(Equal(c))
	getc = GetDMClusterCondition(status, v1alpha1.DMClusterConditionType("not exist"))
	g.Expect(getc).Should(BeNil())

	// test SetDMClusterCondition

	//  we are about to add already exists and has the same status and reason then we are not going to update
	SetDMClusterCondition(&status, *c)
	g.Expect(len(status.Conditions)).Should(Equal(1))
	getc = GetDMClusterCondition(status, v1alpha1.DMClusterReady)
	g.Expect(getc).Should(Equal(c))

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	c2 := NewDMClusterCondition(v1alpha1.DMClusterReady, v1.ConditionTrue, StatfulSetNotUpToDate, "reason2")
	for c2.LastTransitionTime.Equal(&c.LastTransitionTime) {
		c2.LastTransitionTime = metav1.NewTime(time.Now())
	}
	SetDMClusterCondition(&status, *c2)
	getc = GetDMClusterCondition(status, v1alpha1.DMClusterReady)
	g.Expect(getc.LastTransitionTime).Should(Equal(c.LastTransitionTime))
	g.Expect(getc.LastTransitionTime).ShouldNot(Equal(c2.LastTransitionTime))
	g.Expect(getc.Reason).Should(Equal(c2.Reason))

	// status change from True -> False
	c3 := NewDMClusterCondition(v1alpha1.DMClusterReady, v1.ConditionFalse, StatfulSetNotUpToDate, "reason3")
	SetDMClusterCondition(&status, *c3)
	getc = GetDMClusterCondition(status, v1alpha1.DMClusterReady)
	g.Expect(getc).Should(Equal(c3))

	// test GetDMClusterReadyCondition
	getc = GetDMClusterReadyCondition(status)
	g.Expect(getc).Should(Equal(c3))
}
