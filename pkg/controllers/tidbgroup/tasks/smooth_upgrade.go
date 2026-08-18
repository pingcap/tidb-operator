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

package tasks

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/Masterminds/semver/v3"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	operatorclient "github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/tidbapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

const (
	annSmoothUpgradeDDLPaused     = v1alpha1.AnnoKeyPrefix + "smooth-upgrade-ddl-paused"
	annSmoothUpgradeSourceVersion = v1alpha1.AnnoKeyPrefix + "smooth-upgrade-source-version"
	annSmoothUpgradeTargetVersion = v1alpha1.AnnoKeyPrefix + "smooth-upgrade-target-version"
	annSmoothUpgradeStartedAt     = v1alpha1.AnnoKeyPrefix + "smooth-upgrade-started-at"

	smoothUpgradeRetryAfter = 5 * time.Second
)

type smoothUpgradeSupport string

const (
	smoothUpgradeNotVersionUpgrade smoothUpgradeSupport = "NotVersionUpgrade"
	smoothUpgradeUnsupported       smoothUpgradeSupport = "Unsupported"
	smoothUpgradeAutoSupported     smoothUpgradeSupport = "AutoSupportedNoop"
	smoothUpgradeSwitchControlled  smoothUpgradeSupport = "SwitchControlled"
)

type tidbClientFactory func(context.Context, operatorclient.Client, *v1alpha1.Cluster, *v1alpha1.TiDB) (tidbapi.TiDBClient, error)

var newSmoothUpgradeTiDBClient tidbClientFactory = defaultSmoothUpgradeTiDBClient

func ensureSmoothUpgradeStarted(ctx context.Context, state *ReconcileContext, c operatorclient.Client) task.Result {
	dbg := state.TiDBGroup()
	updateRevision, _, _ := state.Revision()
	source, target, support := detectSmoothUpgrade(dbg, state.TiDBSlice(), updateRevision)
	if support == smoothUpgradeNotVersionUpgrade {
		return task.Complete().With("smooth upgrade is not needed")
	}

	ann := smoothUpgradeAnnotations(dbg)
	if ann.active {
		if ann.source == source && ann.target == target {
			return task.Complete().With("smooth upgrade already started")
		}
		if err := finishSmoothUpgrade(ctx, state, c); err != nil {
			return task.Fail().With("cannot finish stale smooth upgrade: %w", err)
		}
		clearSmoothUpgradeAnnotations(dbg)
		if err := patchSmoothUpgradeAnnotations(ctx, c, dbg, true); err != nil {
			return task.Fail().With("cannot clear stale smooth-upgrade annotations: %w", err)
		}
		return task.Retry(smoothUpgradeRetryAfter).With("stale smooth-upgrade annotations are cleared")
	}

	switch support {
	case smoothUpgradeAutoSupported:
		return task.Complete().With("smooth upgrade is auto-supported by TiDB")
	case smoothUpgradeUnsupported:
		return task.Complete().With("smooth upgrade is unsupported for %s -> %s", source, target)
	case smoothUpgradeSwitchControlled:
	default:
		return task.Complete().With("smooth upgrade is not needed")
	}

	if err := startSmoothUpgrade(ctx, state, c); err != nil {
		return task.Fail().With("cannot start smooth upgrade: %w", err)
	}
	setSmoothUpgradeAnnotations(dbg, source, target)
	if err := patchSmoothUpgradeAnnotations(ctx, c, dbg, false); err != nil {
		return task.Fail().With("cannot persist smooth-upgrade annotations: %w", err)
	}
	return task.Complete().With("smooth upgrade started")
}

func TaskFinishSmoothUpgrade(state *ReconcileContext, c operatorclient.Client) task.Task {
	return task.NameTaskFunc("FinishSmoothUpgrade", func(ctx context.Context) task.Result {
		dbg := state.TiDBGroup()
		ann := smoothUpgradeAnnotations(dbg)
		if !ann.active {
			return task.Complete().With("smooth upgrade is not active")
		}
		if !smoothUpgradeRolloutComplete(dbg, state.TiDBSlice(), ann.target) {
			return task.Wait().With("wait for tidb smooth-upgrade rollout to complete")
		}
		if err := finishSmoothUpgrade(ctx, state, c); err != nil {
			return task.Fail().With("cannot finish smooth upgrade: %w", err)
		}
		clearSmoothUpgradeAnnotations(dbg)
		if err := patchSmoothUpgradeAnnotations(ctx, c, dbg, true); err != nil {
			return task.Fail().With("cannot clear smooth-upgrade annotations: %w", err)
		}
		return task.Complete().With("smooth upgrade finished")
	})
}

func detectSmoothUpgrade(
	dbg *v1alpha1.TiDBGroup,
	dbs []*v1alpha1.TiDB,
	updateRevision string,
) (source, target string, support smoothUpgradeSupport) {
	target = dbg.Spec.Template.Spec.Version
	source = dbg.Status.Version
	if source == "" {
		source = sourceVersionFromOutdatedTiDBs(dbs, updateRevision)
	}
	if source == "" || source == target {
		return source, target, smoothUpgradeNotVersionUpgrade
	}
	return source, target, classifySmoothUpgrade(source, target)
}

func sourceVersionFromOutdatedTiDBs(dbs []*v1alpha1.TiDB, updateRevision string) string {
	versions := map[string]struct{}{}
	for _, db := range dbs {
		if db.Status.CurrentRevision == updateRevision {
			continue
		}
		if db.Spec.Version == "" {
			return ""
		}
		versions[db.Spec.Version] = struct{}{}
	}
	if len(versions) != 1 {
		return ""
	}
	for v := range versions {
		return v
	}
	return ""
}

func classifySmoothUpgrade(source, target string) smoothUpgradeSupport {
	sourceVer, err := semver.NewVersion(source)
	if err != nil {
		return smoothUpgradeUnsupported
	}
	targetVer, err := semver.NewVersion(target)
	if err != nil {
		return smoothUpgradeUnsupported
	}
	if !targetVer.GreaterThan(sourceVer) {
		return smoothUpgradeUnsupported
	}
	if isAutoSmoothUpgradePair(sourceVer, targetVer) {
		return smoothUpgradeAutoSupported
	}
	if isSwitchControlledSmoothUpgradePair(sourceVer, targetVer) {
		return smoothUpgradeSwitchControlled
	}
	return smoothUpgradeUnsupported
}

func isAutoSmoothUpgradePair(source, target *semver.Version) bool {
	pairs := map[string][]string{
		"7.1.0": []string{"7.1.1", "7.2.0", "7.3.0"},
		"7.1.1": []string{"7.2.0", "7.3.0"},
		"7.2.0": []string{"7.3.0"},
	}
	for _, t := range pairs[source.String()] {
		v, err := semver.NewVersion(t)
		if err == nil && target.Equal(v) {
			return true
		}
	}
	return false
}

func isSwitchControlledSmoothUpgradePair(source, target *semver.Version) bool {
	sourceIn712To720 := inRange(source, ">= 7.1.2, < 7.2.0")
	targetIn712To720 := inRange(target, ">= 7.1.2, < 7.2.0")
	sourceGE740 := inRange(source, ">= 7.4.0")
	targetGE740 := inRange(target, ">= 7.4.0")
	return sourceIn712To720 && targetIn712To720 || (sourceIn712To720 || sourceGE740) && targetGE740
}

func inRange(v *semver.Version, constraint string) bool {
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false
	}
	c.IncludePrerelease = true
	return c.Check(v)
}

type smoothUpgradeAnnotationState struct {
	active bool
	source string
	target string
}

func smoothUpgradeAnnotations(dbg *v1alpha1.TiDBGroup) smoothUpgradeAnnotationState {
	ann := dbg.GetAnnotations()
	return smoothUpgradeAnnotationState{
		active: ann[annSmoothUpgradeDDLPaused] == v1alpha1.AnnoValTrue,
		source: ann[annSmoothUpgradeSourceVersion],
		target: ann[annSmoothUpgradeTargetVersion],
	}
}

func setSmoothUpgradeAnnotations(dbg *v1alpha1.TiDBGroup, source, target string) {
	ann := maps.Clone(dbg.GetAnnotations())
	if ann == nil {
		ann = map[string]string{}
	}
	ann[annSmoothUpgradeDDLPaused] = v1alpha1.AnnoValTrue
	ann[annSmoothUpgradeSourceVersion] = source
	ann[annSmoothUpgradeTargetVersion] = target
	ann[annSmoothUpgradeStartedAt] = time.Now().UTC().Format(time.RFC3339)
	dbg.SetAnnotations(ann)
}

func clearSmoothUpgradeAnnotations(dbg *v1alpha1.TiDBGroup) {
	ann := maps.Clone(dbg.GetAnnotations())
	for _, key := range smoothUpgradeAnnotationKeys() {
		delete(ann, key)
	}
	dbg.SetAnnotations(ann)
}

func smoothUpgradeAnnotationKeys() []string {
	return []string{
		annSmoothUpgradeDDLPaused,
		annSmoothUpgradeSourceVersion,
		annSmoothUpgradeTargetVersion,
		annSmoothUpgradeStartedAt,
	}
}

func patchSmoothUpgradeAnnotations(ctx context.Context, c operatorclient.Client, dbg *v1alpha1.TiDBGroup, clear bool) error {
	annotations := map[string]any{}
	for _, key := range smoothUpgradeAnnotationKeys() {
		if clear {
			annotations[key] = nil
			continue
		}
		annotations[key] = dbg.Annotations[key]
	}
	patch, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"annotations": annotations,
		},
	})
	if err != nil {
		return err
	}
	return c.Patch(ctx, dbg, client.RawPatch(types.MergePatchType, patch))
}

func startSmoothUpgrade(ctx context.Context, state *ReconcileContext, c operatorclient.Client) error {
	tidb, err := chooseSmoothUpgradeTiDB(state.TiDBSlice())
	if err != nil {
		return err
	}
	cli, err := newSmoothUpgradeTiDBClient(ctx, c, state.Cluster(), tidb)
	if err != nil {
		return err
	}
	return cli.StartUpgrade(ctx)
}

func finishSmoothUpgrade(ctx context.Context, state *ReconcileContext, c operatorclient.Client) error {
	tidb, err := chooseSmoothUpgradeTiDB(state.TiDBSlice())
	if err != nil {
		return err
	}
	cli, err := newSmoothUpgradeTiDBClient(ctx, c, state.Cluster(), tidb)
	if err != nil {
		return err
	}
	return cli.FinishUpgrade(ctx)
}

func chooseSmoothUpgradeTiDB(dbs []*v1alpha1.TiDB) (*v1alpha1.TiDB, error) {
	candidates := slices.Clone(dbs)
	slices.SortFunc(candidates, func(a, b *v1alpha1.TiDB) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	for _, db := range candidates {
		if coreutil.IsReady[scope.TiDB](db) {
			return db, nil
		}
	}
	return nil, fmt.Errorf("no ready tidb instance for smooth upgrade")
}

func defaultSmoothUpgradeTiDBClient(ctx context.Context, c operatorclient.Client, cluster *v1alpha1.Cluster, tidb *v1alpha1.TiDB) (tidbapi.TiDBClient, error) {
	var tlsConfig *tls.Config
	if coreutil.IsTLSClusterEnabled(cluster) {
		var err error
		tlsConfig, err = apicall.GetClientTLSConfig(ctx, c, cluster)
		if err != nil {
			return nil, fmt.Errorf("cannot get tls config from secret: %w", err)
		}
	}
	return tidbapi.NewTiDBClient(
		coreutil.InstanceAdvertiseURL[scope.TiDB](cluster, tidb, coreutil.TiDBStatusPort(tidb)),
		10*time.Second,
		tlsConfig,
	), nil
}

func smoothUpgradeRolloutComplete(dbg *v1alpha1.TiDBGroup, dbs []*v1alpha1.TiDB, target string) bool {
	desired := coreutil.Replicas[scope.TiDBGroup](dbg)
	if dbg.Status.Version != target {
		return false
	}
	if dbg.Status.Replicas != desired ||
		dbg.Status.ReadyReplicas != desired ||
		dbg.Status.UpdatedReplicas != desired ||
		dbg.Status.CurrentReplicas != desired {
		return false
	}
	if dbg.Status.UpdateRevision == "" || dbg.Status.UpdateRevision != dbg.Status.CurrentRevision {
		return false
	}
	if int32(len(dbs)) != desired {
		return false
	}
	for _, db := range dbs {
		if db.Spec.Version != target || !coreutil.IsReady[scope.TiDB](db) {
			return false
		}
	}
	return true
}
