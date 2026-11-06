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

package monitor

import (
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	core "k8s.io/api/core/v1"
)

const defaultMonitorDataGroup int64 = 2000

// Keep ownership preparation in the existing initializer image. Paths and group
// are positional arguments, not interpolated shell programs.
const prepareMonitorData = `root=$1
group=$2
shift 2
for dir do
    case "$dir" in "$root"/*) ;; *) echo "Invalid monitor data path: $dir" >&2; exit 1;; esac
    checked=$dir
    while [ "$checked" != "$root" ]; do
        if [ -L "$checked" ]; then
            echo "Monitor data path must not contain symlinks: $dir" >&2
            exit 1
        fi
        checked=${checked%/*}
    done
    mkdir -p -- "$dir"
    (
        cd -P -- "$dir"
        [ "$(pwd -P)" = "$dir" ] || exit 1
        if [ "$(stat -c %g .)" != "$group" ]; then
            chgrp -- "$group" . || {
                echo "Cannot prepare $dir: migrate its group with an authorized initializer or storage administrator" >&2
                exit 1
            }
        fi
        mode=$(stat -c %a .)
        if [ "$((0$mode & 02770))" != "$((02770))" ] || [ "$((0$mode & 0002))" != 0 ]; then
            chmod u+rwx,g+rwx,g+s,o-w . || {
                echo "Cannot prepare $dir: migrate its permissions with an authorized initializer or storage administrator" >&2
                exit 1
            }
        fi
    )
done
exec /usr/bin/init.sh`

func monitorDataGroup(monitor *v1alpha1.TidbMonitor) int64 {
	if context := monitor.Spec.PodSecurityContext; context != nil && context.FSGroup != nil {
		return *context.FSGroup
	}
	return defaultMonitorDataGroup
}

// Include explicitly configured writable Grafana paths on the shared data PVC.
// Other volumes and explicit additional mounts retain their existing behavior.
func grafanaDataPaths(monitor *v1alpha1.TidbMonitor) []string {
	if monitor.Spec.Grafana == nil {
		return nil
	}
	envs := monitor.Spec.Grafana.Envs
	data := "/data/grafana"
	if value, ok := envs["GF_PATHS_DATA"]; ok {
		data = value
	}
	var paths []string
	for _, value := range []string{data, envs["GF_PATHS_PLUGINS"], envs["GF_PATHS_LOGS"]} {
		clean := path.Clean(value)
		if monitorDataPathCovered(clean, monitor.Spec.Grafana.AdditionalVolumeMounts) {
			continue
		}
		if clean == "/data" || strings.HasPrefix(clean, "/data/") {
			if !slices.Contains(paths, clean) {
				paths = append(paths, clean)
			}
		}
	}
	return paths
}

func validateMonitorDataPaths(monitor *v1alpha1.TidbMonitor) error {
	if monitor.Spec.Grafana != nil {
		for _, key := range []string{"GF_PATHS_DATA", "GF_PATHS_PLUGINS", "GF_PATHS_LOGS"} {
			value := monitor.Spec.Grafana.Envs[key]
			if value != "" && (!path.IsAbs(value) || (strings.Contains(value, "$(") || strings.Contains(value, "$$"))) {
				return fmt.Errorf("grafana %s must be a literal absolute path, got %q; configure the resolved path explicitly", key, value)
			}
			if monitorDataPathCovered(value, monitor.Spec.Grafana.AdditionalVolumeMounts) {
				continue
			}
			if strings.Contains(value+"/", "/../") {
				return fmt.Errorf("grafana %s must not contain parent traversal: %q", key, value)
			}
		}
	}
	for _, dir := range grafanaDataPaths(monitor) {
		if !strings.HasPrefix(dir, "/data/") || strings.Contains(dir, "$(") {
			return fmt.Errorf("grafana data paths on the monitor PVC must be literal directories below /data, got %q; configure a dedicated path or an additional volume mount", dir)
		}
	}
	return nil
}

func monitorDataMount(dir string) core.VolumeMount {
	return core.VolumeMount{
		Name:      v1alpha1.TidbMonitorMemberType.String(),
		MountPath: dir,
		SubPath:   strings.TrimPrefix(dir, "/data/"),
	}
}

// Explicit data mounts also cover descendants; do not hide their storage with
// a default PVC subPath mount.
func monitorDataPathCovered(dir string, additional []core.VolumeMount) bool {
	dir = path.Clean(dir)
	for _, explicit := range additional {
		parent := path.Clean(explicit.MountPath)
		if dir == parent || strings.HasPrefix(dir, strings.TrimSuffix(parent, "/")+"/") {
			return true
		}
	}
	return false
}

func appendMonitorDataMounts(defaults, additional []core.VolumeMount) []core.VolumeMount {
	var mounts []core.VolumeMount
	for _, mount := range defaults {
		covered := mount.Name == v1alpha1.TidbMonitorMemberType.String() && mount.SubPath != "" && monitorDataPathCovered(mount.MountPath, additional)
		if !covered {
			mounts = append(mounts, mount)
		}
	}
	return append(mounts, additional...)
}

func getInitCommand(monitor *v1alpha1.TidbMonitor) []string {
	command := []string{"/bin/sh", "-ec", prepareMonitorData, "prepare-monitor-data", "/data", strconv.FormatInt(monitorDataGroup(monitor), 10), "/data/prometheus", "/data/latest-rules"}
	return append(command, grafanaDataPaths(monitor)...)
}

// Newer initializers can prepare rule ownership for an explicit serving UID;
// older images ignore this optional environment variable. Never override a
// value the deployer already supplied.
func initializerReloaderUID(monitor *v1alpha1.TidbMonitor, envs []core.EnvVar) []core.EnvVar {
	var uid *int64
	if context := monitor.Spec.PodSecurityContext; context != nil {
		uid = context.RunAsUser
	}
	if context := monitor.Spec.Reloader.SecurityContext; context != nil && context.RunAsUser != nil {
		uid = context.RunAsUser
	}
	if uid == nil || *uid <= 0 {
		return envs
	}
	for _, env := range envs {
		if env.Name == "RELOADER_UID" {
			return envs
		}
	}
	return append(envs, core.EnvVar{Name: "RELOADER_UID", Value: strconv.FormatInt(*uid, 10)})
}
