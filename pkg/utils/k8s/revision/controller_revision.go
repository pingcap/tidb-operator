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

package revision

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	serializerjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/history"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/controller/statefulset"
)

const (
	defaultRevisionHistoryLimit = 10
)

var encoderMap = map[schema.GroupVersion]kuberuntime.Encoder{}

// GetCurrentAndUpdate returns the current and update ControllerRevisions. It also
// returns a collision count that records the number of name collisions set saw when creating
// new ControllerRevisions. This count is incremented on every name collision and is used in
// building the ControllerRevision names for name collision avoidance. This method may create
// a new revision, or modify the Revision of an existing revision if an update to ComponentGroup is detected.
// This method expects that revisions is sorted when supplied.
func GetCurrentAndUpdate(
	_ context.Context,
	group client.Object,
	component string,
	labels map[string]string,
	revisions []*appsv1.ControllerRevision,
	cli history.Interface,
	currentRevisionName string,
	currentCollisionCount *int32,
) (
	currentRevision *appsv1.ControllerRevision,
	updateRevision *appsv1.ControllerRevision,
	collisionCount int32,
	err error,
) {
	if currentCollisionCount != nil {
		collisionCount = *currentCollisionCount
	}

	gvk, err := apiutil.GVKForObject(group, scheme.Scheme)
	if err != nil {
		return nil, nil, collisionCount, fmt.Errorf("cannot get gvk of group: %w", err)
	}
	// create a new revision from the current object
	updateRevision, err = newRevision(group, component, labels, gvk, statefulset.NextRevision(revisions), &collisionCount)
	if err != nil {
		return nil, nil, collisionCount, fmt.Errorf("failed to new a revision: %w", err)
	}

	// find any equivalent revisions
	equalRevisions := history.FindEqualRevisions(revisions, updateRevision)
	equalCount := len(equalRevisions)

	revisionCount := len(revisions)
	if equalCount > 0 && history.EqualRevision(revisions[revisionCount-1], equalRevisions[equalCount-1]) {
		// if the equivalent revision is immediately prior the update revision has not changed
		updateRevision = revisions[revisionCount-1]
	} else if equalCount > 0 {
		// if the equivalent revision is not immediately prior we will roll back by incrementing the
		// Revision of the equivalent revision
		updateRevision, err = cli.UpdateControllerRevision(equalRevisions[equalCount-1], updateRevision.Revision)
		if err != nil {
			return nil, nil, collisionCount, fmt.Errorf("failed to update a revision: %w", err)
		}
	} else {
		// if there is no equivalent revision we create a new one
		updateRevision, err = cli.CreateControllerRevision(group, updateRevision, &collisionCount)
		if err != nil {
			return nil, nil, collisionCount, fmt.Errorf("failed to create a revision: %w", err)
		}
	}

	// attempt to find the revision that corresponds to the current revision
	for i := range revisions {
		if revisions[i].Name == currentRevisionName {
			currentRevision = revisions[i]
			break
		}
	}

	// if the current revision is nil we initialize the history by setting it to the update revision
	if currentRevision == nil {
		currentRevision = updateRevision
	}

	return currentRevision, updateRevision, collisionCount, nil
}

// newRevision creates a new ControllerRevision containing a patch that reapplies the target state of CR.
func newRevision(obj client.Object, component string, labels map[string]string, gvk schema.GroupVersionKind,
	revision int64, collisionCount *int32,
) (*appsv1.ControllerRevision, error) {
	patch, err := getPatch(obj, gvk)
	if err != nil {
		return nil, fmt.Errorf("failed to get patch: %w", err)
	}
	cr, err := history.NewControllerRevision(obj, component, labels, kuberuntime.RawExtension{Raw: patch}, revision, collisionCount)
	if err != nil {
		return nil, fmt.Errorf("failed to create a revision: %w", err)
	}
	if err = controllerutil.SetControllerReference(obj, cr, scheme.Scheme); err != nil {
		return nil, fmt.Errorf("failed to set controller reference: %w", err)
	}
	return cr, nil
}

// getPatch returns a merge patch that can be applied to restore a CR to a previous version.
// The current state that we save is just the spec.
func getPatch(obj client.Object, gvk schema.GroupVersionKind) ([]byte, error) {
	encoder, ok := encoderMap[gvk.GroupVersion()]
	if !ok {
		encoder = scheme.Codecs.EncoderForVersion(
			serializerjson.NewSerializerWithOptions(
				serializerjson.DefaultMetaFactory,
				scheme.Scheme,
				scheme.Scheme,
				serializerjson.SerializerOptions{
					Yaml:   false,
					Pretty: false,
					Strict: true,
				}),
			gvk.GroupVersion(),
		)
		encoderMap[gvk.GroupVersion()] = encoder
	}

	buf := bytes.Buffer{}
	if err := encoder.Encode(obj, &buf); err != nil {
		return nil, fmt.Errorf("failed to encode patch: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	objCopy := make(map[string]any)
	objCopy["$patch"] = "replace"
	objCopy["spec"] = raw["spec"].(map[string]any)
	return json.Marshal(objCopy)
}

// TruncateHistory truncates any non-live ControllerRevisions in revisions from group's history.
// The UpdateRevision and CurrentRevision in group's Status are considered to be live.
// Any revisions associated with the Pods in pods are also considered to be live.
// Non-live revisions are deleted, starting with the revision with the lowest Revision,
// until only RevisionHistoryLimit revisions remain.
// This method expects that revisions is sorted when supplied.
func TruncateHistory[T runtime.Instance](
	cli history.Interface,
	instances []T,
	revisions []*appsv1.ControllerRevision,
	updateRevision string,
	currentRevision string,
	limit *int32,
) error {
	// mark all live revisions
	live := make(map[string]bool, len(revisions))
	if updateRevision != "" {
		live[updateRevision] = true
	}
	if currentRevision != "" {
		live[currentRevision] = true
	}
	for _, ins := range instances {
		if ins.CurrentRevision() != "" {
			live[ins.CurrentRevision()] = true
		}
		if ins.GetUpdateRevision() != "" {
			live[ins.GetUpdateRevision()] = true
		}
	}

	// collect live revisions and historic revisions
	hist := make([]*appsv1.ControllerRevision, 0, len(revisions))
	for i := range revisions {
		if !live[revisions[i].Name] {
			hist = append(hist, revisions[i])
		}
	}

	historyLimit := defaultRevisionHistoryLimit
	if limit != nil {
		historyLimit = int(*limit)
	}
	historyLen := len(hist)
	if historyLen <= historyLimit {
		return nil
	}

	// delete any non-live history to maintain the revision limit.
	hist = hist[:(historyLen - historyLimit)]
	for i := 0; i < len(hist); i++ {
		if err := cli.DeleteControllerRevision(hist[i]); err != nil {
			return fmt.Errorf("failed to delete controller revision %s: %w", hist[i].Name, err)
		}
	}
	return nil
}
