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

package coreutil

import (
	"strconv"
	"time"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

// GracefulShutdownDeleteDelaySeconds returns the configured graceful shutdown delay for a TiProxy instance.
func GracefulShutdownDeleteDelaySeconds(tiproxy *v1alpha1.TiProxy) (seconds int32, ok bool, err error) {
	if tiproxy == nil {
		return 0, false, nil
	}
	return gracefulShutdownDeleteDelaySecondsFromAnnotations(tiproxy.Annotations)
}

func gracefulShutdownDeleteDelaySecondsFromAnnotations(annotations map[string]string) (seconds int32, ok bool, err error) {
	raw := annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds]
	if raw == "" {
		return 0, false, nil
	}

	parsed, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, false, err
	}
	return int32(parsed), true, nil
}

// GracefulScaleInEnabled reports whether scale-in should mark the instance offline instead of deleting immediately.
func GracefulScaleInEnabled(tiproxy *v1alpha1.TiProxy) bool {
	seconds, ok, err := GracefulShutdownDeleteDelaySeconds(tiproxy)
	return err == nil && ok && seconds > 0
}

// GracefulShutdownBeginTime parses the graceful shutdown begin time from annotations.
// A zero value means drain has not started.
func GracefulShutdownBeginTime(annotations map[string]string) time.Time {
	raw := annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime]
	if raw == "" {
		return time.Time{}
	}

	startAt, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return startAt
}

// HasGracefulDrainState reports whether graceful scale-in drain annotations are present.
func HasGracefulDrainState(annotations map[string]string) bool {
	if annotations == nil {
		return false
	}
	return annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime] != ""
}
