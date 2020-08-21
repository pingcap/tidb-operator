// Copyright 2020 PingCAP, Inc.
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

package tikv

import (
	"fmt"
	"strconv"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetStoreIDByPodName(c versioned.Interface, ns, clusterName, podName string) (uint64, error) {
	tc, err := c.PingcapV1alpha1().TidbClusters(ns).Get(clusterName, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}
	for _, store := range tc.Status.TiKV.Stores {
		if store.PodName == podName {
			return strconv.ParseUint(store.ID, 10, 64)
		}
	}
	return 0, fmt.Errorf("tikv store of pod %q not found in cluster %s/%s", podName, ns, clusterName)
}
