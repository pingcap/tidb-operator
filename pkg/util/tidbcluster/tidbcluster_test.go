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

package tidbcluster

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

	var conditions []v1alpha1.TidbClusterCondition

	c := NewTidbClusterCondition(v1alpha1.TidbClusterReady, v1.ConditionTrue, StatfulSetNotUpToDate, "reason1")

	conditions = append(conditions, *c)

	status := v1alpha1.TidbClusterStatus{
		Conditions: conditions,
	}

	// test GetTidbClusterCondition
	getc := GetTidbClusterCondition(status, v1alpha1.TidbClusterReady)
	g.Expect(getc).Should(Equal(c))
	getc = GetTidbClusterCondition(status, v1alpha1.TidbClusterConditionType("not exist"))
	g.Expect(getc).Should(BeNil())

	// test SetTidbClusterCondition

	//  we are about to add already exists and has the same status and reason then we are not going to update
	SetTidbClusterCondition(&status, *c)
	g.Expect(len(status.Conditions)).Should(Equal(1))
	getc = GetTidbClusterCondition(status, v1alpha1.TidbClusterReady)
	g.Expect(getc).Should(Equal(c))

	// Do not update lastTransitionTime if the status of the condition doesn't change.
	c2 := NewTidbClusterCondition(v1alpha1.TidbClusterReady, v1.ConditionTrue, StatfulSetNotUpToDate, "reason2")
	for c2.LastTransitionTime.Equal(&c.LastTransitionTime) {
		c2.LastTransitionTime = metav1.NewTime(time.Now())
	}
	SetTidbClusterCondition(&status, *c2)
	getc = GetTidbClusterCondition(status, v1alpha1.TidbClusterReady)
	g.Expect(getc.LastTransitionTime).Should(Equal(c.LastTransitionTime))
	g.Expect(getc.LastTransitionTime).ShouldNot(Equal(c2.LastTransitionTime))
	g.Expect(getc.Reason).Should(Equal(c2.Reason))

	// status change from True -> False
	c3 := NewTidbClusterCondition(v1alpha1.TidbClusterReady, v1.ConditionFalse, StatfulSetNotUpToDate, "reason3")
	SetTidbClusterCondition(&status, *c3)
	getc = GetTidbClusterCondition(status, v1alpha1.TidbClusterReady)
	g.Expect(getc).Should(Equal(c3))

	// test GetTidbClusterReadyCondition
	getc = GetTidbClusterReadyCondition(status)
	g.Expect(getc).Should(Equal(c3))
}
