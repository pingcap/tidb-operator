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

package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os/exec"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	manager "github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/backup"
)

type stopBeforeBackup struct{ err error }

func (s stopBeforeBackup) Update(context.Context, *br.Backup, *br.BackupCondition, *manager.BackupUpdateStatus) error {
	return s.err
}

// TestCredentialsNeverReachManagerLogs runs startup and command success/failure with sentinel credentials.
func TestCredentialsNeverReachManagerLogs(t *testing.T) {
	const sentinel = "sv=2025-01-01&sig=sentinel%2Bcredential%3D"
	normalized, err := url.ParseQuery(sentinel)
	require.NoError(t, err)
	normalizedURL := "https://account.blob.core.windows.net/data?" + normalized.Encode()
	var output bytes.Buffer
	state := klog.CaptureState()
	t.Cleanup(state.Restore)
	klog.LogToStderr(false)
	klog.SetOutput(&output)
	backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "tenant"}, Spec: br.BackupSpec{BR: &br.BRConfig{Cluster: "cluster"}, Mode: br.BackupModeSnapshot}, Status: br.BackupStatus{Phase: br.BackupScheduled}}
	backup.Spec.Azblob = &br.AzblobStorageProvider{Container: "data", StorageAccount: "account", SasToken: sentinel}
	annotation, err := json.Marshal(backup)
	require.NoError(t, err)
	backup.Annotations = map[string]string{"kubectl.kubernetes.io/last-applied-configuration": string(annotation), "stale-config": "old-sas-sentinel"}
	stop := errors.New("stop before external backup")
	m := &Manager{cli: client.NewFakeClient(backup), StatusUpdater: stopBeforeBackup{err: stop}, Options: Options{}}
	m.Namespace = "tenant"
	m.ResourceName = "backup"
	require.ErrorIs(t, m.ProcessBackup(), stop)
	for _, mode := range []string{"success", "start failure", "child failure"} {
		patches := gomonkey.ApplyFunc(exec.CommandContext, func(_ context.Context, _ string, args ...string) *exec.Cmd {
			require.Contains(t, strings.Join(args, " "), url.QueryEscape(sentinel))
			if mode == "child failure" {
				return exec.Command("/bin/sh", "-c", `printf '[ERROR] %s\n' "$1"; printf '%s\n' "$2" "$3" >&2; exit 1`, "br", sentinel, url.QueryEscape(sentinel), normalizedURL) //nolint:noctx
			}
			binary := "/usr/bin/true"
			if mode == "start failure" {
				binary = "/does-not-exist/br"
			}
			return exec.Command(binary) //nolint:noctx // Isolated factory replacement avoids recursively calling the patched function.
		})
		var rawOutput strings.Builder
		err := (&Options{}).brCommandRunWithLogCallback(context.Background(), []string{"backup", "full", "--storage=azure://data/?sas-token=" + url.QueryEscape(sentinel)}, func(line string) { rawOutput.WriteString(line) })
		if mode == "child failure" {
			require.Contains(t, rawOutput.String(), sentinel)
		}
		patches.Reset()
		if mode == "success" {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
			require.NotContains(t, err.Error(), sentinel)
			require.NotContains(t, err.Error(), url.QueryEscape(sentinel))
			require.NotContains(t, err.Error(), "sentinel%2Bcredential")
		}
	}
	klog.Flush()
	require.NotContains(t, output.String(), sentinel)
	require.NotContains(t, output.String(), url.QueryEscape(sentinel))
	require.Contains(t, output.String(), "[REDACTED]")
	require.NotContains(t, output.String(), "sentinel%2Bcredential")
	require.NotContains(t, output.String(), "old-sas-sentinel")
	require.Equal(t, sentinel, backup.Spec.Azblob.SasToken)
	require.Contains(t, output.String(), "start to process backup")
	require.Contains(t, output.String(), "successfully")
}
