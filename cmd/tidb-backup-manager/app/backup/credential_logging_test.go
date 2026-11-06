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
	"errors"
	"io"
	"os/exec"
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
	const sentinel = "sentinel-sas-credential"
	var output bytes.Buffer
	klog.LogToStderr(false)
	klog.SetOutput(&output)
	t.Cleanup(func() { klog.Flush(); klog.SetOutput(io.Discard); klog.LogToStderr(true) })
	backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "tenant"}, Spec: br.BackupSpec{BR: &br.BRConfig{Cluster: "cluster"}, Mode: br.BackupModeSnapshot}, Status: br.BackupStatus{Phase: br.BackupScheduled}}
	backup.Spec.Azblob = &br.AzblobStorageProvider{Container: "data", StorageAccount: "account", SasToken: sentinel}
	stop := errors.New("stop before external backup")
	m := &Manager{cli: client.NewFakeClient(backup), StatusUpdater: stopBeforeBackup{err: stop}, Options: Options{}}
	m.Namespace = "tenant"
	m.ResourceName = "backup"
	require.ErrorIs(t, m.ProcessBackup(), stop)
	for _, ok := range []bool{true, false} {
		patches := gomonkey.ApplyFunc(exec.CommandContext, func(_ context.Context, _ string, _ ...string) *exec.Cmd {
			binary := "/usr/bin/true"
			if !ok {
				binary = "/does-not-exist/br"
			}
			return exec.Command(binary) //nolint:noctx // Isolated factory replacement avoids recursively calling the patched function.
		})
		err := (&Options{}).brCommandRun(context.Background(), []string{"backup", "full", "--storage=azure://data/?sas-token=" + sentinel})
		patches.Reset()
		if ok {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
			require.NotContains(t, err.Error(), sentinel)
		}
	}
	klog.Flush()
	require.NotContains(t, output.String(), sentinel)
	require.Contains(t, output.String(), "start to process backup")
	require.Contains(t, output.String(), "successfully")
}
