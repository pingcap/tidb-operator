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

package member

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func getComponentDesiredOrdinals(context *ComponentContext) sets.Int32 {
	tc := context.tc
	component := context.component

	switch component {
	case label.PDLabelVal:
		return tc.PDStsDesiredOrdinals(true)
	case label.TiKVLabelVal:
		return tc.TiKVStsDesiredOrdinals(true)
	case label.TiDBLabelVal:
		return tc.TiDBStsDesiredOrdinals(true)
	case label.TiFlashLabelVal:
		return tc.TiFlashStsDesiredOrdinals(true)
	}

	return sets.Int32{}
}

func getComponentLabel(context *ComponentContext, instanceName string) label.Label {
	component := context.component

	var componentLabel label.Label
	switch component {
	case label.PDLabelVal:
		componentLabel = label.New().Instance(instanceName).PD()
	case label.TiKVLabelVal:
		componentLabel = label.New().Instance(instanceName).TiKV()
	case label.TiFlashLabelVal:
		componentLabel = label.New().Instance(instanceName).TiFlash()
	case label.TiDBLabelVal:
		componentLabel = label.New().Instance(instanceName).TiDB()
	case label.TiCDCLabelVal:
		componentLabel = label.New().Instance(instanceName).TiCDC()
	case label.PumpLabelVal:
		componentLabel = label.New().Instance(instanceName).Pump()
	}
	return componentLabel
}

func getComponentUpdataRevision(context *ComponentContext) string {
	tc := context.tc
	component := context.component

	var componentUpdateRevision string
	switch component {
	case label.PDLabelVal:
		componentUpdateRevision = tc.Status.PD.StatefulSet.UpdateRevision
	case label.TiKVLabelVal:
		componentUpdateRevision = tc.Status.TiKV.StatefulSet.UpdateRevision
	case label.TiFlashLabelVal:
		componentUpdateRevision = tc.Status.TiFlash.StatefulSet.UpdateRevision
	case label.TiDBLabelVal:
		componentUpdateRevision = tc.Status.TiDB.StatefulSet.UpdateRevision
	case label.TiCDCLabelVal:
		componentUpdateRevision = tc.Status.TiCDC.StatefulSet.UpdateRevision
	case label.PumpLabelVal:
		componentUpdateRevision = tc.Status.Pump.StatefulSet.UpdateRevision
	}
	return componentUpdateRevision
}

func getComponentMemberName(context *ComponentContext) string {
	tc := context.tc
	component := context.component

	tcName := tc.GetName()
	var componentMemberName string
	switch component {
	case label.PDLabelVal:
		componentMemberName = controller.PDMemberName(tcName)
	case label.TiKVLabelVal:
		componentMemberName = controller.TiKVMemberName(tcName)
	case label.TiFlashLabelVal:
		componentMemberName = controller.TiFlashMemberName(tcName)
	case label.TiDBLabelVal:
		componentMemberName = controller.TiDBMemberName(tcName)
	case label.TiCDCLabelVal:
		componentMemberName = controller.TiCDCMemberName(tcName)
	case label.PumpLabelVal:
		componentMemberName = controller.PumpMemberName(tcName)
	}
	return componentMemberName
}

func syncNewComponentStatefulset(context *ComponentContext) error {
	tc := context.tc
	component := context.component

	switch component {
	case label.PDLabelVal:
		tc.Status.PD.StatefulSet = &apps.StatefulSetStatus{}
	case label.TiKVLabelVal:
		tc.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{}
	case label.TiDBLabelVal:
		tc.Status.TiDB.StatefulSet = &apps.StatefulSetStatus{}
	case label.TiFlashLabelVal:
		tc.Status.TiFlash.StatefulSet = &apps.StatefulSetStatus{}
	case label.TiCDCLabelVal:
		tc.Status.TiCDC.StatefulSet = &apps.StatefulSetStatus{}
	case label.PumpLabelVal:
		tc.Status.Pump.StatefulSet = &apps.StatefulSetStatus{}
	}
	return nil
}

func syncExistedComponentStatefulset(context *ComponentContext, set *apps.StatefulSet) error {
	tc := context.tc
	component := context.component
	switch component {
	case label.PDLabelVal:
		tc.Status.PD.StatefulSet = &set.Status
	case label.TiKVLabelVal:
		tc.Status.TiKV.StatefulSet = &set.Status
	case label.TiFlashLabelVal:
		tc.Status.TiFlash.StatefulSet = &set.Status
	case label.TiDBLabelVal:
		tc.Status.TiDB.StatefulSet = &set.Status
	case label.TiCDCLabelVal:
		tc.Status.TiCDC.StatefulSet = &set.Status
	case label.PumpLabelVal:
		tc.Status.Pump.StatefulSet = &set.Status
	}
	return nil
}

func syncComponentImage(context *ComponentContext, set *apps.StatefulSet) error {
	tc := context.tc
	component := context.component

	switch component {
	case label.PDLabelVal:
		tc.Status.PD.Image = ""
		c := filterContainer(set, "pd")
		if c != nil {
			tc.Status.PD.Image = c.Image
		}
	case label.TiKVLabelVal:
		tc.Status.TiKV.Image = ""
		c := filterContainer(set, "tikv")
		if c != nil {
			tc.Status.TiKV.Image = c.Image
		}
	case label.TiFlashLabelVal:
		tc.Status.TiFlash.Image = ""
		c := filterContainer(set, "tiflash")
		if c != nil {
			tc.Status.TiFlash.Image = c.Image
		}
	case label.TiDBLabelVal:
		tc.Status.TiDB.Image = ""
		c := filterContainer(set, "tidb")
		if c != nil {
			tc.Status.TiDB.Image = c.Image
		}
	}
	return nil
}

func getComponentMemberType(context *ComponentContext) v1alpha1.MemberType {
	component := context.component

	var memberType v1alpha1.MemberType
	switch component {
	case label.PDLabelVal:
		memberType = v1alpha1.PDMemberType
	case label.TiKVLabelVal:
		memberType = v1alpha1.TiKVMemberType
	case label.TiFlashLabelVal:
		memberType = v1alpha1.TiFlashMemberType
	case label.TiDBLabelVal:
		memberType = v1alpha1.TiDBMemberType
	}

	return memberType
}
