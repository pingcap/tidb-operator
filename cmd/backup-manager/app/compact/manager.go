// Copyright 2019 PingCAP, Inc.
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

package compact

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/compact/options"
	backuputil "github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	pkgutil "github.com/pingcap/tidb-operator/pkg/backup/util"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	"k8s.io/klog/v2"
)

const (
	messageCompactionDone = "Finishing compaction."
	messageCompactAborted = "Compaction aborted."
)

// logLine is line of JSON log.
// It just extracted the message from the JSON and keeps the origin json bytes.
// So you may extract fields from it by `json.Unmarshal(l.Raw, ...)`.
type logLine struct {
	Message string          `json:"Message"`
	Raw     json.RawMessage `json:"-"`
}

// Manager mainly used to manage backup related work
type Manager struct {
	compact        *v1alpha1.CompactBackup
	resourceLister listers.CompactBackupLister
	statusUpdater  controller.CompactStatusUpdaterInterface
	options        options.CompactOpts
}

// NewManager return a Manager
func NewManager(
	lister listers.CompactBackupLister,
	statusUpdater controller.CompactStatusUpdaterInterface,
	compactOpts options.CompactOpts) *Manager {
	compact, err := lister.CompactBackups(compactOpts.Namespace).Get(compactOpts.ResourceName)
	if err != nil {
		klog.Errorf("can't find compact %s:%s CRD object, err: %v", compactOpts.Namespace, compactOpts.ResourceName, err)
		return nil
	}
	return &Manager{
		compact,
		lister,
		statusUpdater,
		compactOpts,
	}
}

func (cm *Manager) brBin() string {
	return filepath.Join(util.BRBinPath, "br")
}

func (cm *Manager) kvCtlBin() string {
	return filepath.Join(util.KVCTLBinPath, "tikv-ctl")
}

// ProcessBackup used to process the backup logic
func (cm *Manager) ProcessCompact() error {
	ctx, cancel := backuputil.GetContextForTerminationSignals(cm.options.ResourceName)
	defer cancel()

	compact, err := cm.resourceLister.CompactBackups(cm.options.Namespace).Get(cm.options.ResourceName)
	defer func() { cm.statusUpdater.OnFinish(ctx, cm.compact, err) }()
	if err != nil {
		return errors.New("backup not found")
	}
	if err := options.ParseCompactOptions(compact, &cm.options); err != nil {
		return errors.Annotate(err, "failed to parse compact options")
	}

	b64, err := cm.base64ifyStorage(ctx)
	if err != nil {
		return errors.Annotate(err, "failed to base64ify storage")
	}
	return cm.runCompaction(ctx, b64)
}

func (cm *Manager) base64ifyStorage(ctx context.Context) (string, error) {
	brCmd, err := cm.base64ifyCmd(ctx)
	if err != nil {
		return "", err
	}
	out, err := brCmd.Output()
	if err != nil {
		eerr, ok := err.(*exec.ExitError)
		if !ok {
			return "", errors.Annotatef(err, "failed to execute BR with args %v", brCmd.Args)
		}
		klog.Warningf("Failed to execute base64ify; stderr = %s", string(eerr.Stderr))
		return "", errors.Annotatef(err, "failed to execute BR with args %v", brCmd.Args)
	}
	out = bytes.Trim(out, "\r\n \t")
	return string(out), nil
}

func (cm *Manager) base64ifyCmd(ctx context.Context) (*exec.Cmd, error) {
	br := cm.brBin()
	args := []string{
		"operator",
		"base64ify",
	}
	StorageOpts, err := pkgutil.GenStorageArgsForFlag(cm.compact.Spec.StorageProvider, "storage")
	if err != nil {
		return nil, err
	}
	args = append(args, StorageOpts...)
	return exec.CommandContext(ctx, br, args...), nil
}

func (cm *Manager) runCompaction(ctx context.Context, base64Storage string) (err error) {
	cmd := cm.compactCmd(ctx, base64Storage)

	tikvLog, err := cmd.StderrPipe()
	if err != nil {
		return errors.Annotate(err, "failed to create stderr pipe for compact")
	}
	if err := cmd.Start(); err != nil {
		return errors.Annotate(err, "failed to start compact")
	}

	cm.statusUpdater.OnStart(ctx, cm.compact)
	err = cm.processCompactionLogs(ctx, io.TeeReader(tikvLog, os.Stdout))
	if err != nil {
		return err
	}

	return cmd.Wait()
}

func (cm *Manager) compactCmd(ctx context.Context, base64Storage string) *exec.Cmd {
	ctl := cm.kvCtlBin()
	// You should not change the log configuration here, it should sync with the upstream setting
	args := []string{
		"--log-level",
		"INFO",
		"--log-format",
		"json",
		"compact-log-backup",
		"--storage-base64",
		base64Storage,
		"--from",
		strconv.FormatUint(cm.options.FromTS, 10),
		"--until",
		strconv.FormatUint(cm.options.UntilTS, 10),
		"-N",
		strconv.FormatUint(cm.options.Concurrency, 10),
	}
	return exec.CommandContext(ctx, ctl, args...)
}

func (cm *Manager) processCompactionLogs(ctx context.Context, logStream io.Reader) error {
	dec := json.NewDecoder(logStream)

	for dec.More() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return errors.Annotate(err, "failed to decode raw log line")
		}

		var line logLine
		if err := json.Unmarshal(raw, &line); err != nil {
			return errors.Annotate(err, "failed to decode the line of log")
		}
		line.Raw = raw

		if err := cm.processLogLine(ctx, line); err != nil {
			return err
		}
	}

	return nil
}

func (cm *Manager) processLogLine(ctx context.Context, l logLine) error {
	switch l.Message {
	case messageCompactionDone:
		var prog controller.Progress
		if err := json.Unmarshal(l.Raw, &prog); err != nil {
			return errors.Annotatef(err, "failed to decode progress message: %s", string(l.Raw))
		}
		cm.statusUpdater.OnProgress(ctx, cm.compact, prog)
		return nil
	case messageCompactAborted:
		errContainer := struct {
			Err string `json:"err"`
		}{}
		if err := json.Unmarshal(l.Raw, &errContainer); err != nil {
			return errors.Annotatef(err, "failed to decode error message: %s", string(l.Raw))
		}
		return errors.New(errContainer.Err)
	default:
		return nil
	}
}
