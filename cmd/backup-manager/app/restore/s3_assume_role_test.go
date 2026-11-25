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
// See the License for the specific language governing permissions and
// limitations under the License.

package restore

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	kvbackup "github.com/pingcap/kvproto/pkg/brpb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/tests/pkg/backup/s3auth"
	"github.com/stretchr/testify/require"
)

func TestRestoreManagerUsesS3RoleForMetadata(t *testing.T) {
	const startLog = `printf '%s\n' '[2026/11/23 10:00:00.000 +00:00] [INFO] [stream.go:1385] ["start point in time restore"] [restore-from=100] [restore-to=456789] [log-min-ts=100] [log-max-ts=999999]'`
	const registryLog = `printf '%s\n' '[2026/11/23 10:00:01.000 +00:00] [INFO] [stream.go:2456] ["registry resolved a different RestoreTS, updating config"] [original_restored_ts=456789] [resolved_restored_ts=345678] [restore_id=42]'`
	const malformedLog = `printf '%s\n' '[2026/11/23 10:00:02.000 +00:00] [INFO] [stream.go:1385] ["start point in time restore"] [restore-to=18446744073709551616]'`
	const jsonLog = `printf '%s\n' '{"level":"INFO","caller":"stream.go:1385","message":"start point in time restore","restore-from":100,"restore-to":456789,"log-min-ts":100,"log-max-ts":999999}'`
	for _, tc := range []struct {
		name           string
		mode           v1alpha1.RestoreMode
		log            string
		pitrRestoredTS string
		extraOptions   []string
		wantCommitTS   string
		useS3Fields    bool
	}{
		{name: "snapshot", mode: v1alpha1.RestoreModeSnapshot, wantCommitTS: "202"},
		{name: "PITR log", mode: v1alpha1.RestoreModePiTR, log: startLog, pitrRestoredTS: "123456", wantCommitTS: "456789"},
		{name: "PITR S3 fields", mode: v1alpha1.RestoreModePiTR, log: startLog, pitrRestoredTS: "123456", wantCommitTS: "456789", useS3Fields: true},
		{name: "PITR registry override", mode: v1alpha1.RestoreModePiTR, log: startLog + "\n" + registryLog + "\n" + malformedLog, wantCommitTS: "345678"},
		{name: "PITR JSON log", mode: v1alpha1.RestoreModePiTR, log: jsonLog, wantCommitTS: "456789"},
		{name: "PITR numeric fallback", mode: v1alpha1.RestoreModePiTR, pitrRestoredTS: "123456", wantCommitTS: "123456"},
		{name: "PITR datetime fallback", mode: v1alpha1.RestoreModePiTR, pitrRestoredTS: "2018-05-11 01:42:23+0800", wantCommitTS: strconv.FormatInt(time.Date(2018, 5, 10, 17, 42, 23, 0, time.UTC).UnixMilli()<<18, 10)},
		{name: "PITR restored timestamp override", mode: v1alpha1.RestoreModePiTR, pitrRestoredTS: "700", extraOptions: []string{"--restored-ts=900"}, wantCommitTS: "900"},
		{name: "PITR log takes precedence over restored timestamp override", mode: v1alpha1.RestoreModePiTR, log: startLog, pitrRestoredTS: "700", extraOptions: []string{"--restored-ts", "900"}, wantCommitTS: "456789"},
		{name: "PITR empty restored timestamp override", mode: v1alpha1.RestoreModePiTR, pitrRestoredTS: "700", extraOptions: []string{"--restored-ts="}, wantCommitTS: "0"},
		{name: "PITR legacy BR with no restore point", mode: v1alpha1.RestoreModePiTR, wantCommitTS: "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := s3auth.Install(t)
			meta := &kvbackup.BackupMeta{EndVersion: 202}
			if tc.mode == v1alpha1.RestoreModePiTR {
				meta = &kvbackup.BackupMeta{StartVersion: 100}
			}
			data, err := proto.Marshal(meta)
			require.NoError(t, err)
			f.Objects["primary/backupmeta"] = data
			dir := t.TempDir()
			previous := brBinPath
			brBinPath = dir
			t.Cleanup(func() { brBinPath = previous })
			argv := filepath.Join(dir, "argv")
			t.Setenv("BR_ARGV_FILE", argv)
			script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$BR_ARGV_FILE\"\n" + tc.log + "\nexit 0\n"
			require.NoError(t, os.WriteFile(filepath.Join(dir, "br"), []byte(script), 0755))
			restore := &v1alpha1.Restore{Spec: v1alpha1.RestoreSpec{
				BR: &v1alpha1.BRConfig{Cluster: "target"},
				StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
					Bucket: "bucket", Prefix: "primary", Endpoint: s3auth.Endpoint, Region: "us-east-1", Provider: "aws",
				}},
			}}
			if tc.mode == v1alpha1.RestoreModePiTR {
				restore.Spec.PitrFullBackupStorageProvider = v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{Bucket: "bucket", Prefix: "full"}}
			}
			if tc.useS3Fields {
				f.ExpectedExternalID = "log+backup"
				restore.Spec.S3.RoleARN = s3auth.RoleARN
				restore.Spec.S3.ExternalID = f.ExpectedExternalID
				restore.Spec.PitrFullBackupStorageProvider.S3.RoleARN = "arn:aws:iam::123456789012:role/full"
				restore.Spec.PitrFullBackupStorageProvider.S3.ExternalID = "full+backup"
			} else {
				restore.Spec.BR.Options = append([]string{"--s3.role-arn=" + s3auth.RoleARN}, tc.extraOptions...)
			}
			updater := &recordingRestoreStatusUpdater{}
			manager := &Manager{StatusUpdater: updater}
			manager.Mode = string(tc.mode)
			manager.PitrRestoredTs = tc.pitrRestoredTS
			require.NoError(t, manager.performRestore(context.Background(), restore, nil))
			require.Equal(t, []v1alpha1.RestoreConditionType{v1alpha1.RestoreRunning, v1alpha1.RestoreComplete}, updater.conditions)
			require.NotEmpty(t, updater.updates)
			last := updater.updates[len(updater.updates)-1]
			require.NotNil(t, last)
			require.NotNil(t, last.CommitTs)
			require.Equal(t, tc.wantCommitTS, *last.CommitTs)
			if tc.useS3Fields {
				wantArgs := []string{
					"--storage=s3://bucket/primary",
					"--s3.role-arn=" + s3auth.RoleARN,
					"--s3.external-id=" + f.ExpectedExternalID,
					"--full-backup-storage=s3://bucket/full?role-arn=arn%3Aaws%3Aiam%3A%3A123456789012%3Arole%2Ffull&external-id=full%2Bbackup",
				}
				args, err := os.ReadFile(argv)
				require.NoError(t, err)
				require.Subset(t, strings.Split(string(args), "\n"), wantArgs)
				abortArgs, err := manager.constructBRAbortArgs(restore)
				require.NoError(t, err)
				require.Subset(t, abortArgs, wantArgs)
				require.Equal(t, []s3auth.Request{
					{Service: "sts", Method: http.MethodPost, Host: "sts.us-east-1.amazonaws.com", Action: "AssumeRole", RoleARN: s3auth.RoleARN,
						ExternalID: f.ExpectedExternalID, AccessKey: s3auth.SourceKey, Region: "us-east-1"},
					{Service: "s3", Method: http.MethodHead, Host: "s3.unit.test", AccessKey: s3auth.RoleKey, Region: "us-east-1"},
					{Service: "s3", Method: http.MethodGet, Host: "s3.unit.test", AccessKey: s3auth.RoleKey, Region: "us-east-1"},
				}, f.Requests)
				return
			}
			s3Requests := 0
			for _, request := range f.Requests {
				if request.Service == "s3" {
					s3Requests++
					require.Equal(t, s3auth.RoleKey, request.AccessKey)
				}
			}
			require.Positive(t, s3Requests)
		})
	}
}

func TestRestoreCommitTSTimeFormats(t *testing.T) {
	dir := t.TempDir()
	meta, err := proto.Marshal(&kvbackup.BackupMeta{StartVersion: 100})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, constants.MetaFile), meta, 0600))
	restore := newRestoreForObserverTest()
	restore.Spec.Local.VolumeMount.MountPath = dir
	restore.Spec.Local.Prefix = ""
	manager := &Manager{}
	manager.Mode = string(v1alpha1.RestoreModePiTR)
	for _, tc := range []struct {
		ts    string
		nanos int
	}{
		{ts: "2020-10-21T08:05:10Z"},
		{ts: "2020-10-21T16:05:10.501234567+08:00", nanos: 501234567},
		{ts: "2020-10-21 16:05:10.501234567+0800", nanos: 501234567},
		{ts: "2020-10-21T16:05:10.501234567+0800", nanos: 501234567},
		{ts: "2020-10-21 16:05:10.50+08:00", nanos: 500000000},
		{ts: "2020-10-21T16:05:10.50+08", nanos: 500000000},
		{ts: "2020-10-21 03:05:10.50-05", nanos: 500000000},
		{ts: "2020-10-21T13:35:10.50+0530", nanos: 500000000},
	} {
		t.Run(tc.ts, func(t *testing.T) {
			manager.PitrRestoredTs = tc.ts
			ts, err := manager.restoreCommitTS(context.Background(), restore, 0)
			require.NoError(t, err)
			want := uint64(time.Date(2020, 10, 21, 8, 5, 10, tc.nanos, time.UTC).UnixMilli()) << 18
			require.Equal(t, want, ts)
		})
	}
}

func TestConstructBRAbortArgs(t *testing.T) {
	for _, tc := range []struct {
		name, wantType string
		mode           v1alpha1.RestoreMode
		withFullBackup bool
		startTS        string
		wantPITRArgs   []string
	}{
		{name: "snapshot", mode: v1alpha1.RestoreModeSnapshot, wantType: "full"},
		{name: "PITR with full backup", mode: v1alpha1.RestoreModePiTR, wantType: "point", withFullBackup: true, wantPITRArgs: []string{"--restored-ts=900", "--full-backup-storage=s3://bucket/full"}},
		{name: "PITR with start TS", mode: v1alpha1.RestoreModePiTR, wantType: "point", startTS: "100", wantPITRArgs: []string{"--restored-ts=900", "--start-ts=100"}},
		{name: "PITR missing both sources", mode: v1alpha1.RestoreModePiTR},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options := []string{"--s3.role-arn=" + s3auth.RoleARN, "--s3.external-id=external", "--s3.region=us-west-2"}
			restore := &v1alpha1.Restore{Spec: v1alpha1.RestoreSpec{
				BR: &v1alpha1.BRConfig{Cluster: "target", ClusterNamespace: "cluster-ns", Options: options},
				StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
					Bucket: "bucket", Prefix: "primary", Region: "us-east-1",
				}},
				LogRestoreStartTs: tc.startTS,
			}}
			if tc.withFullBackup {
				restore.Spec.PitrFullBackupStorageProvider = v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{Bucket: "bucket", Prefix: "full"}}
			}
			manager := &Manager{}
			manager.Mode = string(tc.mode)
			manager.PitrRestoredTs = "900"
			args, err := manager.constructBRAbortArgs(restore)
			if tc.wantType == "" {
				require.ErrorContains(t, err, "PITR requires pitrFullBackupStorageProvider or logRestoreStartTs")
				return
			}
			require.NoError(t, err)
			require.Equal(t, []string{"abort", "restore", tc.wantType}, args[:3])
			require.Contains(t, args, "--pd=target-pd.cluster-ns:2379")
			require.Contains(t, args, "--storage=s3://bucket/primary")
			require.Contains(t, args, "--s3.region=us-east-1")
			for _, arg := range tc.wantPITRArgs {
				require.Contains(t, args, arg)
			}
			require.Equal(t, options, args[len(args)-len(options):])
		})
	}
	t.Run("missing BR config", func(t *testing.T) {
		_, err := (&Manager{}).constructBRAbortArgs(&v1alpha1.Restore{})
		require.ErrorContains(t, err, "no config for br")
	})
}
