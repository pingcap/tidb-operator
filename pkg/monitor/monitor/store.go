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

package monitor

import (
	"fmt"

	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// Store is a store that fetches and caches TLS materials, bearer tokens
// and auth credentials from configmaps and secrets.
// Data can be referenced directly from a TiDBMonitor object.
// In practice a new store is created and used by
// each reconciliation loop.
//
// Store doesn't support concurrent access.
type Store struct {
	objStore     cache.Store
	cmLister     corelisterv1.ConfigMapLister
	secretLister corelisterv1.SecretLister
	TLSAssets    map[TLSAssetKey]TLSAsset
}

// NewStore returns an empty assetStore.
func NewStore(cmLister corelisterv1.ConfigMapLister, secretLister corelisterv1.SecretLister) *Store {
	return &Store{
		cmLister:     cmLister,
		secretLister: secretLister,
		TLSAssets:    make(map[TLSAssetKey]TLSAsset),
	}
}

// addTLSAssets processes the given Secret and adds the referenced CA, certificate and key to the store.
func (s *Store) addTLSAssets(ns string, secretName string) error {
	secret, err := s.secretLister.Secrets(ns).Get(secretName)
	if err != nil {
		rerr := fmt.Errorf("get secret [%s/%s] failed, err: %v", ns, secretName, err)
		return rerr
	}
	for key, value := range secret.Data {
		s.TLSAssets[TLSAssetKey{"secret", secret.Namespace, secret.Name, key}] = TLSAsset(value)
	}
	return nil
}

// TLSAssetKey is a key for a TLS asset.
type TLSAssetKey struct {
	from string
	ns   string
	name string
	key  string
}

// TLSAsset represents any TLS related opaque string, e.g. CA files, client
// certificates.
type TLSAsset string

// String implements the fmt.Stringer interface.
func (k TLSAssetKey) String() string {
	return fmt.Sprintf("%s_%s_%s_%s", k.from, k.ns, k.name, k.key)
}
