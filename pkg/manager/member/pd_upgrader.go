package member

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	apps "k8s.io/api/apps/v1beta1"
)

type pdUpgrader struct{}

// NewPDUpgrader return a pd Upgrader
func NewPDUpgrader() Upgrader {
	return &pdUpgrader{}
}

func (pu *pdUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	tc.Status.PD.Phase = v1alpha1.UpgradePhase
	return nil
}

type fakePDUpgrader struct{}

// NewFakePDUpgrader return a fake pd Upgrader
func NewFakePDUpgrader() Upgrader {
	return &fakePDUpgrader{}
}

func (fpu *fakePDUpgrader) Upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	tc.Status.PD.Phase = v1alpha1.UpgradePhase
	return nil
}
