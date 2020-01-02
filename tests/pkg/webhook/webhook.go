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

package webhook

import (
	"net/http"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
)

type Interface interface {
	ServePods(w http.ResponseWriter, r *http.Request)
}

type webhook struct {
	namespaces sets.String
}

func NewWebhook(kubeCli kubernetes.Interface, versionCli versioned.Interface, namespaces []string) Interface {
	return &webhook{
		namespaces: sets.NewString(namespaces...),
	}
}
