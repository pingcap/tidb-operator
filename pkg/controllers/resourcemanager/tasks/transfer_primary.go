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
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/v2/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/resourcemanagerapi"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	httputil "github.com/pingcap/tidb-operator/v2/pkg/utils/http"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pdMicroservicePrimaryPrefix = "pd/api/v2/ms/primary"
	resourceManagerServiceName  = "resource_manager"

	defaultAPITimeout     = 5 * time.Second
	defaultTransferWait   = 2 * time.Second
	defaultTransferLogger = "TransferPrimary"
)

var (
	newRMClient = resourcemanagerapi.NewClient
)

func transferPrimaryIfNeeded(
	ctx context.Context,
	logger logr.Logger,
	c client.Client,
	cluster *v1alpha1.Cluster,
	rm *v1alpha1.ResourceManager,
) (bool, error) {
	groupName := rm.Labels[v1alpha1.LabelKeyGroup]
	if groupName == "" {
		// Not managed by ResourceManagerGroup, skip.
		return false, nil
	}

	if cluster == nil || strings.TrimSpace(cluster.Status.PD) == "" {
		return false, nil
	}

	peers, err := listGroupResourceManagers(ctx, c, cluster, rm, groupName)
	if err != nil {
		return false, err
	}
	if len(peers) <= 1 {
		return false, nil
	}

	var tlsConfig *tls.Config
	if coreutil.IsTLSClusterEnabled(cluster) {
		cfg, err := apicall.GetClientTLSConfig(ctx, c, cluster)
		if err != nil {
			return false, err
		}
		tlsConfig = cfg
	}

	hc := &http.Client{Timeout: defaultAPITimeout, Transport: &http.Transport{TLSClientConfig: tlsConfig}}

	pdAddr := strings.TrimRight(strings.Split(cluster.Status.PD, ",")[0], "/")
	if pdAddr == "" {
		return false, nil
	}

	primaryAddr, err := getMicroservicePrimary(ctx, hc, pdAddr, resourceManagerServiceName)
	if err != nil {
		return false, err
	}
	if primaryAddr == "" {
		return false, nil
	}

	myAddr := coreutil.InstanceAdvertiseURL[scope.ResourceManager](cluster, rm, coreutil.ResourceManagerClientPort(rm))
	if myAddr == "" || primaryAddr != myAddr {
		return false, nil
	}

	transferee := coreutil.LongestReadyPeer[scope.ResourceManager](rm, peers)
	if transferee == nil {
		logger.Info("no healthy transferee available for resource manager primary transfer", "name", rm.Name)
		return false, nil
	}

	logger.Info("try to transfer resource manager primary", "from", rm.Name, "to", transferee.Name)

	rmClient := newRMClient(strings.TrimRight(primaryAddr, "/"), defaultAPITimeout, tlsConfig)
	if err := rmClient.TransferPrimary(ctx, transferee.Name); err != nil {
		return false, err
	}

	return true, nil
}

func listGroupResourceManagers(
	ctx context.Context,
	c client.Client,
	cluster *v1alpha1.Cluster,
	rm *v1alpha1.ResourceManager,
	groupName string,
) ([]*v1alpha1.ResourceManager, error) {
	var list v1alpha1.ResourceManagerList
	selector := map[string]string{
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentResourceManager,
		v1alpha1.LabelKeyCluster:   cluster.Name,
		v1alpha1.LabelKeyGroup:     groupName,
	}
	if err := c.List(ctx, &list, ctrlclient.InNamespace(rm.Namespace), ctrlclient.MatchingLabels(selector)); err != nil {
		return nil, err
	}
	peers := make([]*v1alpha1.ResourceManager, 0, len(list.Items))
	for i := range list.Items {
		peer := &list.Items[i]
		peers = append(peers, peer)
	}
	return peers, nil
}

func getMicroservicePrimary(ctx context.Context, hc *http.Client, pdAddr, service string) (string, error) {
	apiURL := fmt.Sprintf("%s/%s/%s", strings.TrimRight(pdAddr, "/"), pdMicroservicePrimaryPrefix, service)
	body, err := httputil.GetBodyOK(ctx, hc, apiURL)
	if err != nil {
		return "", err
	}
	var primary string
	if err := json.Unmarshal(body, &primary); err != nil {
		return "", err
	}
	return primary, nil
}
