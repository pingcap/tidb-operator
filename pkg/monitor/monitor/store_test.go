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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStoreAddBasicAuth(t *testing.T) {
	g := NewGomegaWithT(t)
	tmm := newFakeTidbMonitorManager()
	store := &Store{
		secretLister:    tmm.deps.SecretLister,
		BasicAuthAssets: make(map[string]BasicAuthCredentials),
	}
	ns := "default"
	name := "test-secret"
	secret := &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: map[string][]byte{
			"password": []byte("password"),
			"username": []byte("username"),
		},
	}
	err := tmm.deps.SecretControl.Create(ns, secret)
	g.Expect(err).NotTo(HaveOccurred())
	key := fmt.Sprintf("remoteWrite/%d", 0)
	err = store.AddBasicAuth(ns, &v1alpha1.BasicAuth{
		Password: core.SecretKeySelector{
			LocalObjectReference: core.LocalObjectReference{Name: name},
			Key:                  "password",
		},
		Username: core.SecretKeySelector{
			LocalObjectReference: core.LocalObjectReference{Name: name},
			Key:                  "username",
		},
	}, key)
	g.Expect(err).NotTo(HaveOccurred())
	err = store.AddBasicAuth(ns, &v1alpha1.BasicAuth{
		Password: core.SecretKeySelector{
			LocalObjectReference: core.LocalObjectReference{Name: "test"},
			Key:                  "password",
		},
		Username: core.SecretKeySelector{
			LocalObjectReference: core.LocalObjectReference{Name: name},
			Key:                  "username",
		},
	}, key)
	g.Expect(err).To(HaveOccurred())
}

func TestStoreAddTLSAssets(t *testing.T) {
	g := NewGomegaWithT(t)
	tmm := newFakeTidbMonitorManager()
	store := &Store{
		secretLister: tmm.deps.SecretLister,
		TLSAssets:    make(map[TLSAssetKey]TLSAsset),
	}
	ns := "default"
	name := "test-secret"
	secret := &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: map[string][]byte{
			"password": []byte("password"),
			"username": []byte("username"),
		},
	}
	err := tmm.deps.SecretControl.Create(ns, secret)
	g.Expect(err).NotTo(HaveOccurred())
	err = store.addTLSAssets(ns, secret.Name)
	g.Expect(err).NotTo(HaveOccurred())
	m := make(map[TLSAssetKey]TLSAsset)
	m[TLSAssetKey{"secret", secret.Namespace, secret.Name, "password"}] = "password"
	m[TLSAssetKey{"secret", secret.Namespace, secret.Name, "username"}] = "username"
	g.Expect(store.TLSAssets).To(Equal(m))

}
