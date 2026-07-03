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

package runtime

import (
	"strconv"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

// GracefulOfflineScaleInEnabled reports whether the instance should use graceful
// offline-before-delete during pure scale-in (TiProxy).
func GracefulOfflineScaleInEnabled(annotations map[string]string) bool {
	if annotations == nil {
		return false
	}
	raw := annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds]
	if raw == "" {
		return false
	}
	seconds, err := strconv.ParseInt(raw, 10, 32)
	return err == nil && seconds > 0
}
