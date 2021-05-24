// Copyright 2021 PingCAP, Inc.
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

package s3

import (
	"context"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/e2e/br/utils/portforward"
	"k8s.io/client-go/kubernetes"
)

type Interface interface {
	Init(ctx context.Context, ns, accessKey, secretKey string) error
	IsDataCleaned(ctx context.Context, ns, prefix string) (bool, error)
	Clean(ctx context.Context, ns string) error

	Config(ns, prefix string) *v1alpha1.S3StorageProvider
}

func New(provider string, c kubernetes.Interface, fw portforward.PortForwarder) (Interface, error) {
	switch provider {
	case "kind":
		return NewMinio(c, fw), nil
	}
	return nil, fmt.Errorf("no storage supported for this provider: %s", provider)
}
