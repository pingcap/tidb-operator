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

package tidbcluster

import (
	"context"
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
)

// MustWaitForComponentPhase wait a component to be in a specific phase
func MustWaitForComponentPhase(c versioned.Interface, tc *v1alpha1.TidbCluster, comp v1alpha1.MemberType, phase v1alpha1.MemberPhase,
	timeout, pollInterval time.Duration) {
	lastPhase, err := WaitForComponentPhase(c, tc, comp, phase, timeout, pollInterval)
	framework.ExpectNoError(err, "failed to wait for .Status.%s.Phase of tc %s/%s to be %s, last phase is %s",
		comp, tc.Namespace, tc.Name, phase, lastPhase)
}

func WaitForComponentPhase(c versioned.Interface, tc *v1alpha1.TidbCluster, comp v1alpha1.MemberType, phase v1alpha1.MemberPhase,
	timeout, pollInterval time.Duration) (v1alpha1.MemberPhase, error) {
	var lastPhase v1alpha1.MemberPhase
	cond := func(tc *v1alpha1.TidbCluster) (bool, error) {
		switch comp {
		case v1alpha1.PDMemberType:
			lastPhase = tc.Status.PD.Phase
		case v1alpha1.TiKVMemberType:
			lastPhase = tc.Status.TiKV.Phase
		case v1alpha1.TiDBMemberType:
			lastPhase = tc.Status.TiDB.Phase
		case v1alpha1.TiFlashMemberType:
			lastPhase = tc.Status.TiFlash.Phase
		case v1alpha1.TiCDCMemberType:
			lastPhase = tc.Status.TiCDC.Phase
		case v1alpha1.PumpMemberType:
			lastPhase = tc.Status.Pump.Phase
		}

		return lastPhase == phase, nil
	}

	return lastPhase, WaitForTCCondition(c, tc.Namespace, tc.Name, timeout, pollInterval, cond)
}

func CheckComponentStatusNotChanged(c versioned.Interface, oldTC *v1alpha1.TidbCluster) error {
	tcName := oldTC.GetName()
	tcNs := oldTC.GetNamespace()

	curTC, err := c.PingcapV1alpha1().TidbClusters(tcNs).Get(context.TODO(), tcName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get tc: %v", err)
	}

	// pd
	if oldTC.Spec.PD != nil {
		for pdName, oldMember := range oldTC.Status.PD.Members {
			curMember, exist := curTC.Status.PD.Members[pdName]
			if !exist {
				return fmt.Errorf("pd %q is not found in cur tc", pdName)
			}
			if oldMember.Name != curMember.Name {
				return fmt.Errorf("name of pd %q is changed from %q to %q", pdName, oldMember.Name, curMember.Name)
			}
			if oldMember.ClientURL != curMember.ClientURL {
				return fmt.Errorf("clientURL of pd %q changed from %q to %q", pdName, oldMember.ClientURL, curMember.ClientURL)
			}
			if oldMember.ID != curMember.ID {
				return fmt.Errorf("id of pd %q is changed from %q to %q", pdName, oldMember.ID, curMember.ID)
			}
		}
	}
	// tidb
	if oldTC.Spec.TiDB != nil {
		for tidbName := range oldTC.Status.TiDB.Members {
			_, exist := curTC.Status.TiDB.Members[tidbName]
			if !exist {
				return fmt.Errorf("tidb %q is not found in cur tc", tidbName)
			}
		}
	}
	// tikv
	if oldTC.Spec.TiKV != nil {
		for storeID, oldStore := range oldTC.Status.TiKV.Stores {
			curStore, exist := curTC.Status.TiKV.Stores[storeID]
			if !exist {
				return fmt.Errorf("tikv %q is not found in cur tc", storeID)
			}
			if oldStore.ID != curStore.ID {
				return fmt.Errorf("id of tikv %q is changed from %q to %q", storeID, oldStore.ID, curStore.ID)
			}
			if oldStore.IP != curStore.IP {
				return fmt.Errorf("ip of tikv %q is changed from %q to %q", storeID, oldStore.IP, curStore.IP)
			}
			if oldStore.PodName != curStore.PodName {
				return fmt.Errorf("podName of tikv %q is changed from %q to %q", storeID, oldStore.PodName, curStore.PodName)
			}
		}
	}
	// tiflash
	if oldTC.Spec.TiFlash != nil {
		for storeID, oldStore := range oldTC.Status.TiFlash.Stores {
			curStore, exist := curTC.Status.TiFlash.Stores[storeID]
			if !exist {
				return fmt.Errorf("tiflash %q is not found in cur tc", storeID)
			}
			if oldStore.ID != curStore.ID {
				return fmt.Errorf("id of tiflash %q is changed from %q to %q", storeID, oldStore.ID, curStore.ID)
			}
			if oldStore.IP != curStore.IP {
				return fmt.Errorf("ip of tiflash %q is changed from %q to %q", storeID, oldStore.IP, curStore.IP)
			}
			if oldStore.PodName != curStore.PodName {
				return fmt.Errorf("podName of tiflash %q is changed from %q to %q", storeID, oldStore.PodName, curStore.PodName)
			}
		}
	}
	// ticdc
	if oldTC.Spec.TiCDC != nil {
		for cdcName, oldCapture := range oldTC.Status.TiCDC.Captures {
			curCapture, exist := curTC.Status.TiCDC.Captures[cdcName]
			if !exist {
				return fmt.Errorf("ticdc %q is not found in cur tc", cdcName)
			}
			// Capture ID will be changed after ticdc restart.
			// if oldCapture.ID != curCapture.ID {
			// 	return fmt.Errorf("id of ticdc %q is changed from %q to %q", cdcName, oldCapture.ID, curCapture.ID)
			// }
			if oldCapture.PodName != curCapture.PodName {
				return fmt.Errorf("podName of ticdc %q is changed from %q to %q", cdcName, oldCapture.PodName, curCapture.PodName)
			}
		}
	}
	// pump
	if oldTC.Spec.Pump != nil {
		for _, oldMember := range oldTC.Status.Pump.Members {
			exist := false
			for _, curMember := range curTC.Status.Pump.Members {
				if oldMember.Host == curMember.Host {
					exist = true
					if oldMember.NodeID != curMember.NodeID {
						return fmt.Errorf("nodeID of pump %q is changed from %q to %q", oldMember.Host, oldMember.NodeID, curMember.NodeID)
					}
					break
				}
			}
			if !exist {
				return fmt.Errorf("pump %q is not found in cur tc", oldMember.Host)
			}
		}
	}

	return nil
}
