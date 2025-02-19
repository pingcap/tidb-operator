// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package restore

import (
	"context"
	"fmt"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/klog/v2"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/manager/restore"
	restoreMgr "github.com/pingcap/tidb-operator/pkg/controllers/br/manager/restore"
)

type Manager struct {
	cli           client.Client
	StatusUpdater restore.RestoreConditionUpdaterInterface
	Options
}

// NewManager return a RestoreManager
func NewManager(
	cli client.Client,
	statusUpdater restore.RestoreConditionUpdaterInterface,
	restoreOpts Options) *Manager {
	return &Manager{
		cli,
		statusUpdater,
		restoreOpts,
	}
}

// ProcessRestore used to process the restore logic
func (rm *Manager) ProcessRestore() error {
	ctx, cancel := util.GetContextForTerminationSignals(rm.ResourceName)
	defer cancel()

	var errs []error
	restore := &v1alpha1.Restore{}
	err := rm.cli.Get(ctx, types.NamespacedName{Namespace: rm.Namespace, Name: rm.ResourceName}, restore)
	if err != nil {
		errs = append(errs, err)
		klog.Errorf("can't find cluster %s restore %s CRD object, err: %v", rm, rm.ResourceName, err)
		uerr := rm.StatusUpdater.Update(restore, &metav1.Condition{
			Type:    string(v1alpha1.RestoreFailed),
			Status:  metav1.ConditionTrue,
			Reason:  "GetRestoreCRFailed",
			Message: err.Error(),
		}, nil)
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	if restore.Spec.BR == nil {
		return fmt.Errorf("no br config in %s", rm)
	}

	crData, err := json.Marshal(restore)
	if err != nil {
		klog.Errorf("failed to marshal restore %v to json, err: %s", restore, err)
	} else {
		klog.Infof("start to process restore: %s", string(crData))
	}

	if restore.Spec.To == nil {
		return rm.performRestore(ctx, restore.DeepCopy())
	}

	return fmt.Errorf("set .spec.to field is not supported in v2")
}

func (rm *Manager) performRestore(ctx context.Context, restore *v1alpha1.Restore) error {
	started := time.Now()

	err := rm.StatusUpdater.Update(restore, &metav1.Condition{
		Type:   string(v1alpha1.RestoreRunning),
		Status: metav1.ConditionTrue,
	}, nil)
	if err != nil {
		return err
	}

	var errs []error
	restoreErr := rm.restoreData(ctx, restore, rm.StatusUpdater)
	if restoreErr != nil {
		errs = append(errs, restoreErr)
		klog.Errorf("restore cluster %s from %s failed, err: %s", rm, restore.Spec.Type, restoreErr)
		uerr := rm.StatusUpdater.Update(restore, &metav1.Condition{
			Type:    string(v1alpha1.RestoreFailed),
			Status:  metav1.ConditionTrue,
			Reason:  "RestoreDataFromRemoteFailed",
			Message: restoreErr.Error(),
		}, nil)
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	klog.Infof("restore cluster %s from %s succeed", rm, restore.Spec.Type)

	var (
		commitTS    *string
		restoreType v1alpha1.RestoreConditionType
		allFinished bool
	)
	switch rm.Mode {
	default:
		ts, err := util.GetCommitTsFromBRMetaData(ctx, restore.Spec.StorageProvider)
		if err != nil {
			errs = append(errs, err)
			klog.Errorf("get cluster %s commitTs failed, err: %s", rm, err)
			uerr := rm.StatusUpdater.Update(restore, &metav1.Condition{
				Type:    string(v1alpha1.RestoreFailed),
				Status:  metav1.ConditionTrue,
				Reason:  "GetCommitTsFailed",
				Message: err.Error(),
			}, nil)
			errs = append(errs, uerr)
			return errorutils.NewAggregate(errs)
		}
		restoreType = v1alpha1.RestoreComplete
		tsStr := strconv.FormatUint(ts, 10)
		commitTS = &tsStr
		allFinished = true
	}

	updateStatus := &restoreMgr.RestoreUpdateStatus{
		CommitTs: commitTS,
	}
	if restore.Status.TimeStarted.Unix() <= 0 {
		updateStatus.TimeStarted = &metav1.Time{Time: started}
	}
	if allFinished {
		updateStatus.TimeCompleted = &metav1.Time{Time: time.Now()}
	}
	return rm.StatusUpdater.Update(restore, &metav1.Condition{
		Type:   string(restoreType),
		Status: metav1.ConditionTrue,
	}, updateStatus)
}
