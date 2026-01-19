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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	operatorclient "github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/resourcemanagerapi"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
)

type fakeRMClient struct {
	called     bool
	transferee string
	err        error
}

func (f *fakeRMClient) TransferPrimary(ctx context.Context, transferee string) error {
	f.called = true
	f.transferee = transferee
	return f.err
}

func TestTransferPrimaryIfNeeded_SkipWhenNotGroupManaged(t *testing.T) {
	ctx := context.Background()
	logger := logr.Discard()

	cluster := fake.FakeObj[v1alpha1.Cluster]("tc",
		fake.SetNamespace[v1alpha1.Cluster]("ns"),
	)
	cluster.Status.PD = "http://127.0.0.1:0" // would fail if contacted

	rm := fake.FakeObj[v1alpha1.ResourceManager]("tc-0",
		fake.SetNamespace[v1alpha1.ResourceManager]("ns"),
	)

	fc := operatorclient.NewFakeClient(cluster, rm)

	wait, err := transferPrimaryIfNeeded(ctx, logger, fc, cluster, rm)
	require.NoError(t, err)
	require.False(t, wait)
}

func TestTransferPrimaryIfNeeded_SkipWhenNotPrimary(t *testing.T) {
	ctx := context.Background()
	logger := logr.Discard()

	pd := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/pd/api/v2/ms/primary/resource_manager", r.URL.Path)
		_ = json.NewEncoder(w).Encode("http://other-primary:1234")
	}))
	defer pd.Close()

	cluster := fake.FakeObj[v1alpha1.Cluster]("tc",
		fake.SetNamespace[v1alpha1.Cluster]("ns"),
	)
	cluster.Status.PD = pd.URL

	rm0 := fake.FakeObj[v1alpha1.ResourceManager]("tc-0",
		fake.SetNamespace[v1alpha1.ResourceManager]("ns"),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentResourceManager),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyCluster, cluster.Name),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyGroup, "g0"),
	)
	rm0.Spec.Subdomain = "rm"

	rm1 := fake.FakeObj[v1alpha1.ResourceManager]("tc-1",
		fake.SetNamespace[v1alpha1.ResourceManager]("ns"),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentResourceManager),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyCluster, cluster.Name),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyGroup, "g0"),
	)
	rm1.Spec.Subdomain = "rm"
	rm1.Status.Conditions = []metav1.Condition{
		{
			Type:               v1alpha1.CondReady,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(time.Now().Add(-10 * time.Minute)),
		},
	}

	fc := operatorclient.NewFakeClient(cluster, rm0, rm1)

	orig := newRMClient
	newRMClient = func(url string, timeout time.Duration, tlsConfig *tls.Config) resourcemanagerapi.Client {
		t.Fatalf("unexpected RM client creation for url=%s", url)
		return nil
	}
	defer func() { newRMClient = orig }()

	wait, err := transferPrimaryIfNeeded(ctx, logger, fc, cluster, rm0)
	require.NoError(t, err)
	require.False(t, wait)
}

func TestTransferPrimaryIfNeeded_TransferWhenPrimary(t *testing.T) {
	ctx := context.Background()
	logger := logr.Discard()

	cluster := fake.FakeObj[v1alpha1.Cluster]("tc",
		fake.SetNamespace[v1alpha1.Cluster]("ns"),
	)

	rm0 := fake.FakeObj[v1alpha1.ResourceManager]("tc-0",
		fake.SetNamespace[v1alpha1.ResourceManager]("ns"),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentResourceManager),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyCluster, cluster.Name),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyGroup, "g0"),
	)
	rm0.Spec.Subdomain = "rm"

	rm1 := fake.FakeObj[v1alpha1.ResourceManager]("tc-1",
		fake.SetNamespace[v1alpha1.ResourceManager]("ns"),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentResourceManager),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyCluster, cluster.Name),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyGroup, "g0"),
	)
	rm1.Spec.Subdomain = "rm"
	rm1.Status.Conditions = []metav1.Condition{
		{
			Type:               v1alpha1.CondReady,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.NewTime(time.Now().Add(-10 * time.Minute)),
		},
	}

	myAddr := coreutil.InstanceAdvertiseURL[scope.ResourceManager](cluster, rm0, coreutil.ResourceManagerClientPort(rm0))

	pd := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/pd/api/v2/ms/primary/resource_manager", r.URL.Path)
		_ = json.NewEncoder(w).Encode(myAddr)
	}))
	defer pd.Close()
	cluster.Status.PD = pd.URL

	fc := operatorclient.NewFakeClient(cluster, rm0, rm1)

	recorder := &fakeRMClient{}
	orig := newRMClient
	newRMClient = func(url string, timeout time.Duration, tlsConfig *tls.Config) resourcemanagerapi.Client {
		require.Equal(t, strings.TrimRight(myAddr, "/"), url)
		return recorder
	}
	defer func() { newRMClient = orig }()

	wait, err := transferPrimaryIfNeeded(ctx, logger, fc, cluster, rm0)
	require.NoError(t, err)
	require.True(t, wait)
	require.True(t, recorder.called)
	require.Equal(t, rm1.Name, recorder.transferee)
}

func TestTransferPrimaryIfNeeded_SkipWhenNoHealthyTransferee(t *testing.T) {
	ctx := context.Background()
	logger := logr.Discard()

	cluster := fake.FakeObj[v1alpha1.Cluster]("tc",
		fake.SetNamespace[v1alpha1.Cluster]("ns"),
	)

	rm0 := fake.FakeObj[v1alpha1.ResourceManager]("tc-0",
		fake.SetNamespace[v1alpha1.ResourceManager]("ns"),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentResourceManager),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyCluster, cluster.Name),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyGroup, "g0"),
	)
	rm0.Spec.Subdomain = "rm"

	rm1 := fake.FakeObj[v1alpha1.ResourceManager]("tc-1",
		fake.SetNamespace[v1alpha1.ResourceManager]("ns"),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentResourceManager),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyCluster, cluster.Name),
		fake.Label[v1alpha1.ResourceManager](v1alpha1.LabelKeyGroup, "g0"),
	)
	rm1.Spec.Subdomain = "rm"
	// rm1 is not ready -> no transferee should be selected.

	myAddr := coreutil.InstanceAdvertiseURL[scope.ResourceManager](cluster, rm0, coreutil.ResourceManagerClientPort(rm0))

	pd := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/pd/api/v2/ms/primary/resource_manager", r.URL.Path)
		_ = json.NewEncoder(w).Encode(myAddr)
	}))
	defer pd.Close()
	cluster.Status.PD = pd.URL

	fc := operatorclient.NewFakeClient(cluster, rm0, rm1)

	orig := newRMClient
	newRMClient = func(url string, timeout time.Duration, tlsConfig *tls.Config) resourcemanagerapi.Client {
		t.Fatalf("unexpected RM client creation for url=%s", url)
		return nil
	}
	defer func() { newRMClient = orig }()

	wait, err := transferPrimaryIfNeeded(ctx, logger, fc, cluster, rm0)
	require.NoError(t, err)
	require.False(t, wait)
}
