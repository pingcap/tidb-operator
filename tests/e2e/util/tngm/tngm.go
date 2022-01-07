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

package tngm

import (
	"context"
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/test/e2e/framework"
)

func MustWaitForNGMPhase(c versioned.Interface, tngm *v1alpha1.TidbNGMonitoring, phase v1alpha1.MemberPhase, timeout, pollInterval time.Duration) {
	var err error
	locator := fmt.Sprintf("%s/%s", tngm.Namespace, tngm.Name)

	wait.Poll(pollInterval, timeout, func() (bool, error) {
		tc, err := c.PingcapV1alpha1().TidbNGMonitorings(tngm.Namespace).Get(context.TODO(), tngm.Name, metav1.GetOptions{})
		framework.ExpectNoError(err, "TidbNGMonitoring %q: failed to get: %v", locator, err)
		if tc.Status.NGMonitoring.Phase != phase {
			return false, nil
		}
		return true, nil
	})
	framework.ExpectNoError(err, "TidbNGMonitoring %q: failed to wait for phase of ngm to be %q", locator, phase)
}
