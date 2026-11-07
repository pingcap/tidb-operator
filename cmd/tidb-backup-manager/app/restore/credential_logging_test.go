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
	manager "github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/restore"
)

type stopBeforeRestore struct{ err error }

func (s stopBeforeRestore) Update(context.Context, *br.Restore, *metav1.Condition, *manager.RestoreUpdateStatus) error {
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
	restore := &br.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: "tenant"}, Spec: br.RestoreSpec{BR: &br.BRConfig{Cluster: "cluster"}}}
	restore.Spec.Azblob = &br.AzblobStorageProvider{Container: "data", StorageAccount: "account", SasToken: sentinel}
	restore.Spec.PitrFullBackupStorageProvider.Azblob = &br.AzblobStorageProvider{Container: "full", StorageAccount: "account", SasToken: "pitr-only-secret"}
	annotation, err := json.Marshal(restore)
	require.NoError(t, err)
	restore.Annotations = map[string]string{"kubectl.kubernetes.io/last-applied-configuration": string(annotation), "stale-config": "old-sas-sentinel"}
	stop := errors.New("stop before external restore")
	m := &Manager{cli: client.NewFakeClient(restore), StatusUpdater: stopBeforeRestore{err: stop}}
	m.Namespace = "tenant"
	m.ResourceName = "restore"
	require.ErrorIs(t, m.ProcessRestore(), stop)

	for _, mode := range []string{"success", "start failure", "child failure"} {
		patches := gomonkey.ApplyFunc(exec.CommandContext, func(_ context.Context, _ string, args ...string) *exec.Cmd {
			require.Contains(t, strings.Join(args, " "), url.QueryEscape(sentinel))
			require.Contains(t, args, "--full-backup-storage=azure://full/?account-name=account&sas-token=pitr-only-secret")
			if mode == "child failure" {
				return exec.Command("/bin/sh", "-c", `printf '[ERROR] %s\n' "$1"; printf '%s\n' "$2" "$3" "$4" >&2; exit 1`, "br", sentinel, url.QueryEscape(sentinel), normalizedURL, "pitr-only-secret") //nolint:noctx
			}
			binary := "/usr/bin/true"
			if mode == "start failure" {
				binary = "/does-not-exist/br"
			}
			return exec.Command(binary) //nolint:noctx // Isolated factory replacement avoids recursively calling the patched function.
		})
		opts := &Options{}
		opts.Mode = string(br.RestoreModePiTR)
		err := opts.restoreData(context.Background(), restore, nil)
		patches.Reset()
		if mode == "success" {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
			require.NotContains(t, err.Error(), sentinel)
			require.NotContains(t, err.Error(), url.QueryEscape(sentinel))
			require.NotContains(t, err.Error(), "sentinel%2Bcredential")
			require.NotContains(t, err.Error(), "pitr-only-secret")
		}
	}
	klog.Flush()
	require.NotContains(t, output.String(), sentinel)
	require.NotContains(t, output.String(), url.QueryEscape(sentinel))
	require.Contains(t, output.String(), "[REDACTED]")
	require.NotContains(t, output.String(), "sentinel%2Bcredential")
	require.NotContains(t, output.String(), "old-sas-sentinel")
	require.Equal(t, sentinel, restore.Spec.Azblob.SasToken)
	require.NotContains(t, output.String(), "pitr-only-secret")
	require.Equal(t, "pitr-only-secret", restore.Spec.PitrFullBackupStorageProvider.Azblob.SasToken)
	require.Contains(t, output.String(), "start to process restore")
	require.Contains(t, output.String(), "Running br command")
}
