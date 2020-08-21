// Copyright 2019 PingCAP, Inc.
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

package tidbinitializer

import (
	"k8s.io/client-go/tools/record"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
)

// ControlInterface reconciles TidbInitializer
type ControlInterface interface {
	// ReconcileTidbInitializer implements the reconcile logic of TidbInitializer
	ReconcileTidbInitializer(ti *v1alpha1.TidbInitializer) error
}

// NewDefaultTidbInitializerControl returns a new instance of the default TidbInitializer ControlInterface
func NewDefaultTidbInitializerControl(recorder record.EventRecorder, manager member.InitManager) ControlInterface {
	return &defaultTidbInitializerControl{recorder, manager}
}

type defaultTidbInitializerControl struct {
	recorder       record.EventRecorder
	tidbInitManger member.InitManager
}

func (tic *defaultTidbInitializerControl) ReconcileTidbInitializer(ti *v1alpha1.TidbInitializer) error {
	return tic.tidbInitManger.Sync(ti)
}

var _ ControlInterface = &defaultTidbInitializerControl{}

// FakeTidbInitializerControl is a fake TidbInitializer ControlInterface
type FakeTidbInitializerControl struct {
	err error
}

// NewFakeTidbInitializerControl returns a FakeTidbInitializerControl
func NewFakeTidbInitializerControl() *FakeTidbInitializerControl {
	return &FakeTidbInitializerControl{}
}

// SetReconcileTidbInitializerError sets error for TidbInitializerControl
func (tic *FakeTidbInitializerControl) SetReconcileTidbInitializerError(err error) {
	tic.err = err
}

// ReconcileTidbInitializer fake ReconcileTidbInitializer
func (tic *FakeTidbInitializerControl) ReconcileTidbInitializer(ti *v1alpha1.TidbInitializer) error {
	if tic.err != nil {
		return tic.err
	}
	return nil
}

var _ ControlInterface = &FakeTidbInitializerControl{}
