/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Copied from https://github.com/kubernetes/kubernetes/blob/00236ae0d73d2455a2470469ed1005674f8ed61f/pkg/controller/history/controller_history.go#L49
// and made some modifications.

package history

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/util/retry"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/util/hash"
)

// HashLabel is the label used to indicate the hash value of a ControllerRevision's Data.
const HashLabel = "controller.kubernetes.io/hash"

// ControllerRevisionName returns the ControllerRevisionName for a ControllerRevision in the form prefix-hash. If the length
// of prefix is greater than 223 bytes, it is truncated to allow for a name that is no larger than 253 bytes.
func ControllerRevisionName(prefix string, hash string) string {
	if len(prefix) > 223 {
		prefix = prefix[:223]
	}

	return fmt.Sprintf("%s-%s", prefix, hash)
}

// NewControllerRevision returns a ControllerRevision with a ControllerRef pointing to parent and indicating that
// parent is of parentKind. The ControllerRevision has labels matching template labels, contains Data equal to data, and
// has a Revision equal to revision. The collisionCount is used when creating the name of the ControllerRevision
// so the name is likely unique. If the returned error is nil, the returned ControllerRevision is valid. If the
// returned error is not nil, the returned ControllerRevision is invalid for use.
func NewControllerRevision(parent metav1.Object,
	templateLabels map[string]string,
	data runtime.RawExtension,
	revision int64,
	collisionCount *int32) (*appsv1.ControllerRevision, error) {
	labelMap := make(map[string]string)
	for k, v := range templateLabels {
		labelMap[k] = v
	}
	cr := &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labelMap,
			Namespace: parent.GetNamespace(),
		},
		Data:     data,
		Revision: revision,
	}
	hash := HashControllerRevision(cr, collisionCount)
	cr.Name = ControllerRevisionName(parent.GetName(), hash)
	cr.Labels[HashLabel] = hash
	return cr, nil
}

// HashControllerRevision hashes the contents of revision's Data using FNV hashing. If probe is not nil, the byte value
// of probe is added written to the hash as well. The returned hash will be a safe encoded string to avoid bad words.
func HashControllerRevision(revision *appsv1.ControllerRevision, probe *int32) string {
	hf := fnv.New32()
	if len(revision.Data.Raw) > 0 {
		hf.Write(revision.Data.Raw)
	}
	if revision.Data.Object != nil {
		hash.DeepHashObject(hf, revision.Data.Object)
	}
	if probe != nil {
		hf.Write([]byte(strconv.FormatInt(int64(*probe), 10)))
	}
	return rand.SafeEncodeString(fmt.Sprint(hf.Sum32()))
}

// SortControllerRevisions sorts revisions by their Revision.
func SortControllerRevisions(revisions []*appsv1.ControllerRevision) {
	sort.Stable(byRevision(revisions))
}

// EqualRevision returns true if lhs and rhs are either both nil, or both point to non-nil ControllerRevisions that
// contain semantically equivalent data. Otherwise, this method returns false.
func EqualRevision(lhs *appsv1.ControllerRevision, rhs *appsv1.ControllerRevision) bool {
	var lhsHash, rhsHash *uint32
	if lhs == nil || rhs == nil {
		return lhs == rhs
	}
	if hs, found := lhs.Labels[HashLabel]; found {
		hash, err := strconv.ParseInt(hs, 10, 32)
		if err == nil {
			lhsHash = new(uint32)
			*lhsHash = uint32(hash)
		}
	}
	if hs, found := rhs.Labels[HashLabel]; found {
		hash, err := strconv.ParseInt(hs, 10, 32)
		if err == nil {
			rhsHash = new(uint32)
			*rhsHash = uint32(hash)
		}
	}
	if lhsHash != nil && rhsHash != nil && *lhsHash != *rhsHash {
		return false
	}
	return bytes.Equal(lhs.Data.Raw, rhs.Data.Raw) && apiequality.Semantic.DeepEqual(lhs.Data.Object, rhs.Data.Object)
}

// FindEqualRevisions returns all ControllerRevisions in revisions that are equal to needle using EqualRevision as the
// equality test. The returned slice preserves the order of revisions.
func FindEqualRevisions(revisions []*appsv1.ControllerRevision, needle *appsv1.ControllerRevision) []*appsv1.ControllerRevision {
	var eq []*appsv1.ControllerRevision
	for i := range revisions {
		if EqualRevision(revisions[i], needle) {
			eq = append(eq, revisions[i])
		}
	}
	return eq
}

// byRevision implements sort.Interface to allow ControllerRevisions to be sorted by Revision.
type byRevision []*appsv1.ControllerRevision

func (br byRevision) Len() int {
	return len(br)
}

// Less breaks ties first by creation timestamp, then by name
func (br byRevision) Less(i, j int) bool {
	if br[i].Revision == br[j].Revision {
		if br[j].CreationTimestamp.Equal(&br[i].CreationTimestamp) {
			return br[i].Name < br[j].Name
		}
		return br[j].CreationTimestamp.After(br[i].CreationTimestamp.Time)
	}
	return br[i].Revision < br[j].Revision
}

func (br byRevision) Swap(i, j int) {
	br[i], br[j] = br[j], br[i]
}

// Interface provides an interface allowing for management of a Controller's history as realized by recorded
// ControllerRevisions. An instance of Interface can be retrieved from NewHistory. Implementations must treat all
// pointer parameters as "in" parameter, and they must not be mutated.
type Interface interface {
	// ListControllerRevisions lists all ControllerRevisions matching selector and owned by parent or no other
	// controller. If the returned error is nil the returned slice of ControllerRevisions is valid. If the
	// returned error is not nil, the returned slice is not valid.
	ListControllerRevisions(parent client.Object, selector labels.Selector) ([]*appsv1.ControllerRevision, error)
	// CreateControllerRevision attempts to create the revision as owned by parent via a ControllerRef. If name
	// collision occurs, collisionCount (incremented each time collision occurs except for the first time) is
	// added to the hash of the revision, and it is renamed using ControllerRevisionName. Implementations may
	// cease to attempt to retry creation after some number of attempts and return an error. If the returned
	// error is not nil, creation failed. If the returned error is nil, the returned ControllerRevision has been
	// created.
	// Callers must make sure that collisionCount is not nil. An error is returned if it is.
	CreateControllerRevision(parent client.Object, revision *appsv1.ControllerRevision, collisionCount *int32) (*appsv1.ControllerRevision, error)
	// DeleteControllerRevision attempts to delete revision. If the returned error is not nil, deletion has failed.
	DeleteControllerRevision(revision *appsv1.ControllerRevision) error
	// UpdateControllerRevision updates revision such that its Revision is equal to newRevision. Implementations
	// may retry on conflict. If the returned error is nil, the update was successful and returned ControllerRevision
	// is valid. If the returned error is not nil, the update failed and the returned ControllerRevision is invalid.
	UpdateControllerRevision(revision *appsv1.ControllerRevision, newRevision int64) (*appsv1.ControllerRevision, error)
}

// NewClient returns an instance of Interface that uses client to communicate with the API Server and lister to list
// ControllerRevisions. This method should be used to create an Interface for all scenarios other than testing.
func NewClient(cli client.Client) Interface {
	return &realHistory{cli: cli}
}

type realHistory struct {
	cli client.Client
}

func (rh *realHistory) ListControllerRevisions(parent client.Object, selector labels.Selector) ([]*appsv1.ControllerRevision, error) {
	// List all revisions in the namespace that match the selector
	var list appsv1.ControllerRevisionList
	if err := rh.cli.List(context.TODO(), &list, &client.ListOptions{
		Namespace:     parent.GetNamespace(),
		LabelSelector: selector,
	}); err != nil {
		return nil, err
	}
	var owned []*appsv1.ControllerRevision
	for i := range list.Items {
		ref := metav1.GetControllerOfNoCopy(&list.Items[i])
		if ref == nil || ref.UID == parent.GetUID() {
			owned = append(owned, &list.Items[i])
		}

	}
	return owned, nil
}

func (rh *realHistory) CreateControllerRevision(parent client.Object, revision *appsv1.ControllerRevision, collisionCount *int32) (*appsv1.ControllerRevision, error) {
	if collisionCount == nil {
		return nil, fmt.Errorf("collisionCount should not be nil")
	}

	// Clone the input
	clone := revision.DeepCopy()

	// Continue to attempt to create the revision updating the name with a new hash on each iteration
	for {
		hash := HashControllerRevision(revision, collisionCount)
		// Update the revisions name
		clone.Name = ControllerRevisionName(parent.GetName(), hash)
		clone.Namespace = parent.GetNamespace()
		err := rh.cli.Create(context.TODO(), clone)
		if errors.IsAlreadyExists(err) {
			var exists appsv1.ControllerRevision
			if err := rh.cli.Get(context.TODO(), client.ObjectKey{
				Namespace: clone.Namespace,
				Name:      clone.Name,
			}, &exists); err != nil {
				return nil, err
			}
			if bytes.Equal(exists.Data.Raw, clone.Data.Raw) {
				return &exists, nil
			}
			*collisionCount++
			continue
		}
		return clone, err
	}
}

func (rh *realHistory) UpdateControllerRevision(revision *appsv1.ControllerRevision, newRevision int64) (*appsv1.ControllerRevision, error) {
	clone := revision.DeepCopy()
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if clone.Revision == newRevision {
			return nil
		}
		clone.Revision = newRevision
		updateErr := rh.cli.Update(context.TODO(), clone)
		if updateErr == nil {
			return nil
		}
		var updated appsv1.ControllerRevision
		if err := rh.cli.Get(context.TODO(), client.ObjectKey{
			Namespace: clone.Namespace,
			Name:      clone.Name,
		}, &updated); err == nil {
			// make a copy so we don't mutate the shared cache
			clone = updated.DeepCopy()
		}
		return updateErr
	})
	return clone, err
}

func (rh *realHistory) DeleteControllerRevision(revision *appsv1.ControllerRevision) error {
	return rh.cli.Delete(context.TODO(), revision)
}
