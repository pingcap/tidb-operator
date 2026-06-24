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
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package coreutil

import (
	"strconv"
	"time"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

// MinRemainingToReviveBeforeDelete is the minimum remaining graceful shutdown time required
// before a draining TiProxy can be revived during scale-out.
const MinRemainingToReviveBeforeDelete = 5 * time.Minute

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

// GracefulShutdownRemainingFromSources returns how long to wait before graceful shutdown can
// complete, deriving both the configured delay and the begin time from the given annotation
// sources (for example TiProxy CR and Pod). enabled is false when no positive delay is configured.
func GracefulShutdownRemainingFromSources(now time.Time, sources ...map[string]string) (remaining time.Duration, enabled bool, err error) {
	var seconds int32
	for _, annotations := range sources {
		parsed, ok, perr := gracefulShutdownDeleteDelaySecondsFromAnnotations(annotations)
		if perr != nil {
			return 0, false, perr
		}
		if ok && parsed > 0 {
			seconds = parsed
			enabled = true
			break
		}
	}
	if !enabled {
		return 0, false, nil
	}

	startAt := GracefulShutdownBeginTimeFromSources(sources...)
	if startAt.IsZero() {
		return time.Duration(seconds) * time.Second, true, nil
	}

	remaining = startAt.Add(time.Duration(seconds) * time.Second).Sub(now)
	if remaining < 0 {
		remaining = 0
	}
	return remaining, true, nil
}

// GracefulShutdownConnectionsDrained reports whether scale-in drain has observed zero connections.
func GracefulShutdownConnectionsDrained(annotations map[string]string) bool {
	return annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownConnectionsDrained] == v1alpha1.AnnoValTrue
}

// HasGracefulDrainState reports whether graceful scale-in drain annotations are present.
func HasGracefulDrainState(annotations map[string]string) bool {
	if annotations == nil {
		return false
	}
	if annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime] != "" {
		return true
	}
	return GracefulShutdownConnectionsDrained(annotations)
}

// GracefulShutdownBeginTimeFromSources returns the earliest non-zero begin time across annotation sets.
// When Pod and CR annotations diverge, the earliest time reflects the most advanced drain progress.
func GracefulShutdownBeginTimeFromSources(sources ...map[string]string) time.Time {
	var earliest time.Time
	for _, annotations := range sources {
		startAt := GracefulShutdownBeginTime(annotations)
		if startAt.IsZero() {
			continue
		}
		if earliest.IsZero() || startAt.Before(earliest) {
			earliest = startAt
		}
	}
	return earliest
}

// RevivableForGracefulScaleOutFromSources reports whether a draining TiProxy instance can be revived,
// considering multiple annotation sources (for example TiProxy CR and Pod) to avoid incorrect revive
// decisions when annotations are temporarily out of sync.
//
// An instance is revivable when its drain has not started yet, or still has at least
// MinRemainingToReviveBeforeDelete left. Once zero connections were observed (connections-drained),
// scale-in commits to deletion and the instance is no longer revivable.
func RevivableForGracefulScaleOutFromSources(now time.Time, sources ...map[string]string) (bool, error) {
	for _, annotations := range sources {
		if GracefulShutdownConnectionsDrained(annotations) {
			// TiProxy controller already observed zero connections during scale-in drain. Scale-in
			// should delete the pod/CR instead of being cancelled by scale-out revival.
			return false, nil
		}
	}

	remaining, enabled, err := GracefulShutdownRemainingFromSources(now, sources...)
	if err != nil {
		return false, err
	}
	if !enabled {
		return true, nil
	}
	if GracefulShutdownBeginTimeFromSources(sources...).IsZero() {
		// Drain has not started yet; always revivable regardless of the configured delay.
		return true, nil
	}
	return remaining >= MinRemainingToReviveBeforeDelete, nil
}
