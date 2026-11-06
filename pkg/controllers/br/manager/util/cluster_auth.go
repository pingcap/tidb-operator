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

package util

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	core "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

// AllowedBackupNamespaces is set on the target Cluster by its administrator.
const AllowedBackupNamespaces = "br.pingcap.com/allowed-backup-namespaces"

// GetBackupCluster requires target-side consent for a cross-namespace reference.
func GetBackupCluster(ctx context.Context, cli client.Reader, namespace string, config *br.BRConfig) (*core.Cluster, error) {
	if config == nil || config.Cluster == "" {
		return nil, fmt.Errorf("spec.br.cluster is required")
	}
	if err := ValidateBROptions(config); err != nil {
		return nil, err
	}
	target := namespace
	if config.ClusterNamespace != "" {
		target = config.ClusterNamespace
	}
	cluster := &core.Cluster{}
	if err := cli.Get(ctx, client.ObjectKey{Namespace: target, Name: config.Cluster}, cluster); err != nil {
		return nil, err
	}
	if target == namespace {
		return cluster, nil
	}
	for _, allowed := range strings.Split(cluster.Annotations[AllowedBackupNamespaces], ",") {
		if strings.TrimSpace(allowed) == namespace {
			return cluster, nil
		}
	}
	return nil, apierrors.NewForbidden(
		schema.GroupResource{Group: "core.pingcap.com", Resource: "clusters"}, config.Cluster,
		fmt.Errorf("cross-namespace backup/restore from %s requires target Cluster annotation %s", namespace, AllowedBackupNamespaces),
	)
}

// ValidateBROptions keeps raw flags from replacing authorized and recorded targets.
func ValidateBROptions(config *br.BRConfig) error {
	if config == nil {
		return nil
	}
	for _, option := range config.Options {
		// Short flags can be bundled (for example -cu=...), so require long names.
		if strings.HasPrefix(option, "-") && !strings.HasPrefix(option, "--") {
			return fmt.Errorf("spec.br.options must use long flag names; put negative values after '='")
		}
		flag, _, _ := strings.Cut(option, "=")
		switch flag {
		case "--pd", "--storage", "--full-backup-storage", "--s3.endpoint", "--s3.region", "--s3.provider",
			"--gcs.endpoint", "--azblob.endpoint", "--azblob.account-name":
			return fmt.Errorf("spec.br.options cannot override managed target flag %s; configure the Cluster or storage provider", flag)
		}
	}
	return nil
}

// LogBackupClusterIdentity binds the effective reference and Cluster incarnation.
func LogBackupClusterIdentity(cluster *core.Cluster) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(cluster.Namespace+"/"+cluster.Name+"/"+string(cluster.UID))))
}

// GetBackupClusterForBackup rejects redirecting an already bound log task.
func GetBackupClusterForBackup(ctx context.Context, cli client.Reader, backup *br.Backup) (*core.Cluster, error) {
	cluster, err := GetBackupCluster(ctx, cli, backup.Namespace, backup.Spec.BR)
	if err != nil {
		return nil, err
	}
	var bindingErr error
	switch {
	case backup.Status.LogClusterIdentity != "" && backup.Spec.Mode != br.BackupModeLog:
		bindingErr = fmt.Errorf("a bound log backup cannot change mode")
	case backup.Status.LogClusterIdentity != "" && backup.Status.LogClusterIdentity != LogBackupClusterIdentity(cluster):
		bindingErr = fmt.Errorf("log backup cluster has changed; restore its original reference")
	case HasLogBackupHistory(backup) && backup.Status.LogClusterIdentity == "":
		bindingErr = fmt.Errorf("original log backup cluster identity is unavailable; administrator verification is required")
	}
	if bindingErr != nil {
		return nil, apierrors.NewForbidden(schema.GroupResource{Group: "core.pingcap.com", Resource: "clusters"}, cluster.Name, bindingErr)
	}
	return cluster, nil
}

// HasLogBackupHistory includes the interval after BR starts but before CommitTs is recorded.
func HasLogBackupHistory(backup *br.Backup) bool {
	if backup.Status.LogClusterIdentity != "" ||
		(backup.Spec.Mode == br.BackupModeLog && (backup.Status.CommitTs != "" || backup.Status.BackupPath != "")) {
		return true
	}
	switch backup.Status.LogSubCommandStatuses[br.LogStartCommand].Phase {
	case br.BackupScheduled, br.BackupPrepare, br.BackupRunning, br.BackupComplete, br.BackupFailed:
		return true
	}
	return false
}
