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

package whitelist

import (
	"context"

	"github.com/pingcap/tidb-operator/api/pkg/apis/wardle"
	"github.com/pingcap/tidb-operator/api/pkg/list"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
)

type WhitelistStorage struct {
}

func NewStorage() *WhitelistStorage {
	return &WhitelistStorage{}
}

func (s *WhitelistStorage) Kind() string {
	return "Whitelist"
}

func (m *WhitelistStorage) New() runtime.Object {
	obj := &wardle.Whitelist{}
	obj.Name = "ccc"
	return obj
}

func (m *WhitelistStorage) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	return obj, nil
}

//func (m *WhitelistStorage) Get(ctx context.Context, name string, opts *metav1.GetOptions) (runtime.Object, error) {
//	return &v1.Whitelist{}, nil
//}

func (m *WhitelistStorage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return &wardle.Whitelist{
		Name: name,
		Ips:  []string{"127.0.0.3", "127.0.0.4"},
		ID:   1,
	}, nil
}

//func (m *WhitelistStorage) NewGetOptions() (runtime.Object, bool, string) {
//	return &wardle.Whitelist{}, true, ""
//}

func (m *WhitelistStorage) List(ctx context.Context, options *metainternalversion.ListOptions, extraOptions *list.ListOptions) (runtime.Object, error) {
	return &wardle.WhitelistList{
		Items: []wardle.Whitelist{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "default",
					CreationTimestamp: metav1.Now(),
				},
				Name: "default",
				Ips:  []string{"127.0.0.1", "127.0.0.2"},
				ID:   1,
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "localhost",
					CreationTimestamp: metav1.Now(),
				},
				Name: "localhost",
				Ips:  []string{"127.0.0.3", "127.0.0.4"},
				ID:   2,
			},
		},
	}, nil
}

func (s *WhitelistStorage) NewList() runtime.Object {
	return &wardle.WhitelistList{}
}

func (m *WhitelistStorage) NamespaceScoped() bool {
	return false
}
