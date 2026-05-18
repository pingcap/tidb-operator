// Copyright 2026 PingCAP, Inc.
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

package member

import (
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"

	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSmoothUpgradeMatrix(t *testing.T) {
	g := NewGomegaWithT(t)
	cases := []struct {
		source string
		target string
		want   smoothUpgradeSupport
	}{
		{"v7.0.9", "v7.1.0", smoothUpgradeUnsupported},
		{"v7.1.0", "v7.1.1", smoothUpgradeAutoSupported},
		{"v7.1.0", "v7.2.0", smoothUpgradeAutoSupported},
		{"v7.1.1", "v7.3.0", smoothUpgradeAutoSupported},
		{"v7.2.0", "v7.3.0", smoothUpgradeAutoSupported},
		{"v7.1.2", "v7.1.3", smoothUpgradeSwitchControlled},
		{"v7.1.9", "v7.4.0", smoothUpgradeSwitchControlled},
		{"v7.4.0", "v7.5.0", smoothUpgradeSwitchControlled},
		{"v7.3.0", "v7.4.0", smoothUpgradeUnsupported},
		{"nightly", "v7.4.0", smoothUpgradeUnsupported},
	}
	for _, c := range cases {
		g.Expect(classifySmoothUpgrade(c.source, c.target)).To(Equal(c.want), "%s -> %s", c.source, c.target)
	}
}

func TestDetectTiDBVersionUpgrade(t *testing.T) {
	g := NewGomegaWithT(t)
	oldSet := smoothUpgradeStatefulSet("pingcap/tidb:v7.4.0")
	newSet := smoothUpgradeStatefulSet("pingcap/tidb:v7.5.0")
	source, target, ok := detectTiDBVersionUpgrade(oldSet, newSet)
	g.Expect(ok).To(BeTrue())
	g.Expect(source).To(Equal("v7.4.0"))
	g.Expect(target).To(Equal("v7.5.0"))

	newSet = smoothUpgradeStatefulSet("pingcap/tidb:v7.4.0")
	newSet.Spec.Template.Annotations = map[string]string{"config": "changed"}
	_, _, ok = detectTiDBVersionUpgrade(oldSet, newSet)
	g.Expect(ok).To(BeFalse())

	newSet = smoothUpgradeStatefulSet("custom")
	_, _, ok = detectTiDBVersionUpgrade(oldSet, newSet)
	g.Expect(ok).To(BeFalse())
}

func TestSmoothUpgradeAnnotations(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := &v1alpha1.TidbCluster{ObjectMeta: metav1.ObjectMeta{Name: "tc"}}
	g.Expect(isSmoothUpgradePaused(tc)).To(BeFalse())
	setSmoothUpgradeAnnotations(tc, "v7.4.0", "v7.5.0")
	g.Expect(isSmoothUpgradePaused(tc)).To(BeTrue())
	g.Expect(smoothUpgradeAnnotationsMatch(tc, "v7.4.0", "v7.5.0")).To(BeTrue())
	g.Expect(smoothUpgradeAnnotationsMatch(tc, "v7.4.0", "v7.6.0")).To(BeFalse())
	clearSmoothUpgradeAnnotations(tc)
	g.Expect(isSmoothUpgradePaused(tc)).To(BeFalse())
	g.Expect(tc.Annotations).NotTo(HaveKey(annSmoothUpgradeSourceVersion))
}

func smoothUpgradeStatefulSet(image string) *apps.StatefulSet {
	return &apps.StatefulSet{Spec: apps.StatefulSetSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: v1alpha1.TiDBMemberType.String(), Image: image}}}}}}
}
