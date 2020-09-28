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

package member

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

type workerFailover struct {
	deps *controller.Dependencies
}

// NewWorkerFailover returns a worker Failover
func NewWorkerFailover(deps *controller.Dependencies) DMFailover {
	return &workerFailover{deps: deps}
}

func (wf *workerFailover) Failover(dc *v1alpha1.DMCluster) error {
	ns := dc.GetNamespace()
	dcName := dc.GetName()

	for podName, worker := range dc.Status.Worker.Members {
		if worker.LastTransitionTime.IsZero() {
			continue
		}
		if !isWorkerPodDesired(dc, podName) {
			// we should ignore the store record of deleted pod, otherwise the
			// record of deleted pod may be added back to failure stores
			// (before it enters into Offline/Tombstone state)
			continue
		}
		deadline := worker.LastTransitionTime.Add(wf.deps.CLIConfig.WorkerFailoverPeriod)
		exist := false
		for _, failureWorker := range dc.Status.Worker.FailureMembers {
			if failureWorker.PodName == podName {
				exist = true
				break
			}
		}
		if worker.Stage == v1alpha1.DMWorkerStateOffline && time.Now().After(deadline) && !exist {
			if dc.Status.Worker.FailureMembers == nil {
				dc.Status.Worker.FailureMembers = map[string]v1alpha1.WorkerFailureMember{}
			}
			if dc.Spec.Worker.MaxFailoverCount != nil && *dc.Spec.Worker.MaxFailoverCount > 0 {
				maxFailoverCount := *dc.Spec.Worker.MaxFailoverCount
				if len(dc.Status.Worker.FailureMembers) >= int(maxFailoverCount) {
					klog.Warningf("%s/%s failure workers count reached the limit: %d", ns, dcName, *dc.Spec.Worker.MaxFailoverCount)
					return nil
				}
				dc.Status.Worker.FailureMembers[podName] = v1alpha1.WorkerFailureMember{
					PodName:   podName,
					CreatedAt: metav1.Now(),
				}
				msg := fmt.Sprintf("worker[%s/%s] is Offline", ns, worker.Name)
				wf.deps.Recorder.Event(dc, corev1.EventTypeWarning, unHealthEventReason, fmt.Sprintf(unHealthEventMsgPattern, "worker", podName, msg))
			}
		}
	}

	return nil
}

func (wf *workerFailover) Recover(dc *v1alpha1.DMCluster) {
	dc.Status.Worker.FailureMembers = nil
	klog.Infof("dm-worker recover: clear FailureWorkers, %s/%s", dc.GetNamespace(), dc.GetName())
}

func (wf *workerFailover) RemoveUndesiredFailures(dc *v1alpha1.DMCluster) {
	for key, failureWorker := range dc.Status.Worker.FailureMembers {
		if !isWorkerPodDesired(dc, failureWorker.PodName) {
			// If we delete the pods, e.g. by using advanced statefulset delete
			// slots feature. We should remove the record of undesired pods,
			// otherwise an extra replacement pod will be created.
			delete(dc.Status.Worker.FailureMembers, key)
		}
	}
}
