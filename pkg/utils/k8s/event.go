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

package k8s

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

const FailedCreatePodReason = "FailedCreate"

// RecordFailedCreatePod records a warning event on the owner object when a Pod
// create request fails, matching the reason/message style used by Kubernetes
// built-in workload controllers.
func RecordFailedCreatePod(recorders []record.EventRecorder, object runtime.Object, err error) {
	if len(recorders) == 0 || recorders[0] == nil || object == nil || err == nil {
		return
	}

	recorders[0].Eventf(object, corev1.EventTypeWarning, FailedCreatePodReason, "Error creating: %v", apiError(err))
}

func apiError(err error) error {
	var statusErr *apierrors.StatusError
	if errors.As(err, &statusErr) {
		return statusErr
	}
	return err
}
