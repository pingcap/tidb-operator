// Copyright 2018 PingCAP, Inc.
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

package pdapi

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/util/crypto"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
)

const (
	DefaultTimeout       = 5 * time.Second
	evictSchedulerLeader = "evict-leader-scheduler"
)

// GetTLSConfig returns *tls.Config for given TiDB cluster.
func GetTLSConfig(secretLister corelisterv1.SecretLister, namespace Namespace, secretName string) (*tls.Config, error) {
	secret, err := secretLister.Secrets(string(namespace)).Get(secretName)
	if err != nil {
		return nil, fmt.Errorf("unable to load certificates from secret %s/%s: %v", namespace, secretName, err)
	}

	return crypto.LoadTlsConfigFromSecret(secret)
}

// TiKVNotBootstrappedError represents that TiKV cluster is not bootstrapped yet
type TiKVNotBootstrappedError struct {
	s string
}

func (e *TiKVNotBootstrappedError) Error() string {
	return e.s
}

// TiKVNotBootstrappedErrorf returns a TiKVNotBootstrappedError
func TiKVNotBootstrappedErrorf(format string, a ...interface{}) error {
	return &TiKVNotBootstrappedError{fmt.Sprintf(format, a...)}
}

// IsTiKVNotBootstrappedError returns whether err is a TiKVNotBootstrappedError
func IsTiKVNotBootstrappedError(err error) bool {
	_, ok := err.(*TiKVNotBootstrappedError)
	return ok
}
