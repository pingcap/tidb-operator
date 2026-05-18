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

package member

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util/cmpver"

	apps "k8s.io/api/apps/v1"
	"k8s.io/klog/v2"
)

type smoothUpgradeSupport string

const (
	smoothUpgradeUnsupported      smoothUpgradeSupport = "Unsupported"
	smoothUpgradeAutoSupported    smoothUpgradeSupport = "AutoSupportedNoop"
	smoothUpgradeSwitchControlled smoothUpgradeSupport = "SwitchControlled"

	annSmoothUpgradeDDLPaused     = "tidb.pingcap.com/smooth-upgrade-ddl-paused"
	annSmoothUpgradeSourceVersion = "tidb.pingcap.com/smooth-upgrade-source-version"
	annSmoothUpgradeTargetVersion = "tidb.pingcap.com/smooth-upgrade-target-version"
	annSmoothUpgradeStartedAt     = "tidb.pingcap.com/smooth-upgrade-started-at"
)

func detectTiDBVersionUpgrade(oldSet, newSet *apps.StatefulSet) (string, string, bool) {
	oldImage, ok := mainContainerImage(oldSet, v1alpha1.TiDBMemberType.String())
	if !ok {
		return "", "", false
	}
	newImage, ok := mainContainerImage(newSet, v1alpha1.TiDBMemberType.String())
	if !ok {
		return "", "", false
	}
	source := imageTag(oldImage)
	target := imageTag(newImage)
	return source, target, source != "" && target != "" && source != target
}

func classifySmoothUpgrade(source, target string) smoothUpgradeSupport {
	if !validTiDBVersion(source) || !validTiDBVersion(target) {
		return smoothUpgradeUnsupported
	}

	sourceBefore710, _ := cmpver.Compare(source, cmpver.Less, "v7.1.0")
	if sourceBefore710 {
		return smoothUpgradeUnsupported
	}

	if isAutoSmoothUpgradePair(source, target) {
		return smoothUpgradeAutoSupported
	}
	if isSwitchControlledSmoothUpgradePair(source, target) {
		return smoothUpgradeSwitchControlled
	}
	return smoothUpgradeUnsupported
}

func isAutoSmoothUpgradePair(source, target string) bool {
	return (versionEqual(source, "v7.1.0") && (versionEqual(target, "v7.1.1") || versionEqual(target, "v7.2.0") || versionEqual(target, "v7.3.0"))) ||
		(versionEqual(source, "v7.1.1") && (versionEqual(target, "v7.2.0") || versionEqual(target, "v7.3.0"))) ||
		(versionEqual(source, "v7.2.0") && versionEqual(target, "v7.3.0"))
}

func isSwitchControlledSmoothUpgradePair(source, target string) bool {
	sourceIn712To720 := versionGTE(source, "v7.1.2") && versionLT(source, "v7.2.0")
	targetIn712To720 := versionGTE(target, "v7.1.2") && versionLT(target, "v7.2.0")
	sourceGTE740 := versionGTE(source, "v7.4.0")
	targetGTE740 := versionGTE(target, "v7.4.0")
	return (sourceIn712To720 && targetIn712To720) || ((sourceIn712To720 || sourceGTE740) && targetGTE740)
}

func versionEqual(version, target string) bool {
	gte, err := cmpver.Compare(version, cmpver.GreaterOrEqual, target)
	if err != nil || !gte {
		return false
	}
	lte, err := cmpver.Compare(version, cmpver.LessOrEqual, target)
	return err == nil && lte
}

func versionGTE(version, target string) bool {
	ok, err := cmpver.Compare(version, cmpver.GreaterOrEqual, target)
	return err == nil && ok
}

func versionLT(version, target string) bool {
	ok, err := cmpver.Compare(version, cmpver.Less, target)
	return err == nil && ok
}

func validTiDBVersion(version string) bool {
	if !strings.HasPrefix(version, "v") {
		return false
	}
	_, err := cmpver.Compare(version, cmpver.GreaterOrEqual, "v0.0.0")
	return err == nil
}

func mainContainerImage(set *apps.StatefulSet, name string) (string, bool) {
	if set == nil {
		return "", false
	}
	for _, c := range set.Spec.Template.Spec.Containers {
		if c.Name == name {
			return c.Image, true
		}
	}
	return "", false
}

func imageTag(image string) string {
	if image == "" || strings.Contains(image, "@") {
		return ""
	}
	idx := strings.LastIndex(image, ":")
	if idx < 0 || idx == len(image)-1 {
		return ""
	}
	return image[idx+1:]
}

func isSmoothUpgradePaused(tc *v1alpha1.TidbCluster) bool {
	return tc.Annotations != nil && tc.Annotations[annSmoothUpgradeDDLPaused] == "true"
}

func setSmoothUpgradeAnnotations(tc *v1alpha1.TidbCluster, source, target string) {
	if tc.Annotations == nil {
		tc.Annotations = map[string]string{}
	}
	tc.Annotations[annSmoothUpgradeDDLPaused] = "true"
	tc.Annotations[annSmoothUpgradeSourceVersion] = source
	tc.Annotations[annSmoothUpgradeTargetVersion] = target
	tc.Annotations[annSmoothUpgradeStartedAt] = time.Now().UTC().Format(time.RFC3339)
}

func smoothUpgradeAnnotationKeys() []string {
	return []string{
		annSmoothUpgradeDDLPaused,
		annSmoothUpgradeSourceVersion,
		annSmoothUpgradeTargetVersion,
		annSmoothUpgradeStartedAt,
	}
}

func clearSmoothUpgradeAnnotations(tc *v1alpha1.TidbCluster) {
	if tc.Annotations == nil {
		return
	}
	delete(tc.Annotations, annSmoothUpgradeDDLPaused)
	delete(tc.Annotations, annSmoothUpgradeSourceVersion)
	delete(tc.Annotations, annSmoothUpgradeTargetVersion)
	delete(tc.Annotations, annSmoothUpgradeStartedAt)
}

func smoothUpgradeAnnotationsMatch(tc *v1alpha1.TidbCluster, source, target string) bool {
	if !isSmoothUpgradePaused(tc) {
		return false
	}
	return tc.Annotations[annSmoothUpgradeSourceVersion] == source && tc.Annotations[annSmoothUpgradeTargetVersion] == target
}

func (u *tidbUpgrader) ensureSmoothUpgradeStarted(tc *v1alpha1.TidbCluster, oldSet, newSet *apps.StatefulSet) error {
	source, target, versionUpgrade := detectTiDBVersionUpgrade(oldSet, newSet)
	if !versionUpgrade {
		return nil
	}
	if isSmoothUpgradePaused(tc) {
		if smoothUpgradeAnnotationsMatch(tc, source, target) {
			return nil
		}
		ordinal, err := chooseHealthyTiDBOrdinal(tc)
		if err != nil {
			return err
		}
		if err := u.deps.TiDBControl.FinishUpgrade(tc, ordinal); err != nil {
			return fmt.Errorf("finish stale TiDB smooth upgrade for %s/%s failed: %v", tc.Namespace, tc.Name, err)
		}
		if err := clearSmoothUpgradeAnnotationsPersisted(u.deps, tc); err != nil {
			return err
		}
		return controller.RequeueErrorf("tidbcluster: [%s/%s] cleared stale TiDB smooth upgrade annotations, retry upgrade start", tc.Namespace, tc.Name)
	}
	if support := classifySmoothUpgrade(source, target); support != smoothUpgradeSwitchControlled {
		klog.Warningf("TidbCluster: [%s/%s] TiDB smooth upgrade is %s for version pair %s -> %s, skip DDL pause API", tc.Namespace, tc.Name, support, source, target)
		return nil
	}

	ordinal, err := chooseHealthyTiDBOrdinal(tc)
	if err != nil {
		return err
	}
	if err := u.deps.TiDBControl.StartUpgrade(tc, ordinal); err != nil {
		return fmt.Errorf("start TiDB smooth upgrade for %s/%s from %s to %s failed: %v", tc.Namespace, tc.Name, source, target, err)
	}
	setSmoothUpgradeAnnotations(tc, source, target)
	return patchTidbClusterAnnotations(u.deps, tc)
}

func (m *tidbMemberManager) maybeFinishSmoothUpgrade(tc *v1alpha1.TidbCluster) error {
	if !isSmoothUpgradePaused(tc) {
		return nil
	}
	if tc.Status.TiDB.StatefulSet == nil || tc.Status.TiDB.StatefulSet.UpdatedReplicas != tc.Status.TiDB.StatefulSet.Replicas || tc.Status.TiDB.Phase != v1alpha1.NormalPhase || !tc.TiDBAllMembersReady() {
		return nil
	}
	ordinal, err := chooseHealthyTiDBOrdinal(tc)
	if err != nil {
		return err
	}
	if err := m.deps.TiDBControl.FinishUpgrade(tc, ordinal); err != nil {
		return fmt.Errorf("finish TiDB smooth upgrade for %s/%s failed: %v", tc.Namespace, tc.Name, err)
	}
	return clearSmoothUpgradeAnnotationsPersisted(m.deps, tc)
}

func chooseHealthyTiDBOrdinal(tc *v1alpha1.TidbCluster) (int32, error) {
	if tc.Status.TiDB.StatefulSet == nil {
		return 0, controller.RequeueErrorf("tidbcluster: [%s/%s] has no TiDB StatefulSet status for smooth upgrade", tc.Namespace, tc.Name)
	}
	prefix := controller.TiDBMemberName(tc.Name) + "-"
	chosen := int32(-1)
	for name, member := range tc.Status.TiDB.Members {
		if !member.Health || !strings.HasPrefix(name, prefix) {
			continue
		}
		ord64, err := strconv.ParseInt(name[len(prefix):], 10, 32)
		if err != nil {
			continue
		}
		ord := int32(ord64)
		if chosen < 0 || ord < chosen {
			chosen = ord
		}
	}
	if chosen < 0 {
		return 0, controller.RequeueErrorf("tidbcluster: [%s/%s] has no healthy TiDB pod for smooth upgrade", tc.Namespace, tc.Name)
	}
	return chosen, nil
}

func patchTidbClusterAnnotations(deps *controller.Dependencies, tc *v1alpha1.TidbCluster) error {
	annotations := map[string]any{
		annSmoothUpgradeDDLPaused:     tc.Annotations[annSmoothUpgradeDDLPaused],
		annSmoothUpgradeSourceVersion: tc.Annotations[annSmoothUpgradeSourceVersion],
		annSmoothUpgradeTargetVersion: tc.Annotations[annSmoothUpgradeTargetVersion],
		annSmoothUpgradeStartedAt:     tc.Annotations[annSmoothUpgradeStartedAt],
	}
	patch, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"annotations": annotations,
		},
	})
	if err != nil {
		return err
	}
	_, err = deps.TiDBClusterControl.Patch(tc, patch)
	return err
}

func clearSmoothUpgradeAnnotationsPersisted(deps *controller.Dependencies, tc *v1alpha1.TidbCluster) error {
	clearSmoothUpgradeAnnotations(tc)
	annotations := map[string]any{}
	for _, key := range smoothUpgradeAnnotationKeys() {
		annotations[key] = nil
	}
	patch, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"annotations": annotations,
		},
	})
	if err != nil {
		return err
	}
	_, err = deps.TiDBClusterControl.Patch(tc, patch)
	return err
}
