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

package util

import (
	"github.com/juju/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/tidwall/gjson"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

type Result interface {
	Root() gjson.Result
	Get(path string) gjson.Result
	Raw() ([]byte, error)
	Into(obj runtime.Object) error
	Error() error
}

type result struct {
	rest.Result
	root gjson.Result
}

func WrapResult(res rest.Result) Result {
	raw, _ := res.Raw()
	return result{Result: res, root: gjson.ParseBytes(raw)}
}

func (r result) Root() gjson.Result { return r.root }

func (r result) Get(path string) gjson.Result { return r.root.Get(path) }

type GenericClient struct {
	cli *rest.RESTClient
}

func NewClientFor(cfg *rest.Config) (*GenericClient, error) {
	cfgCopy := *cfg
	cfgCopy.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(runtime.NewScheme())}
	cli, err := rest.UnversionedRESTClientFor(&cfgCopy)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &GenericClient{cli: cli}, nil
}

func (c *GenericClient) Request(gv schema.GroupVersion, res string, ns string) *rest.Request {
	base := "api"
	if len(gv.Group) != 0 {
		base = "apis"
	}
	return c.cli.Get().AbsPath(base, gv.String()).Namespace(ns).Resource(res)
}

func (c *GenericClient) GetByName(gv schema.GroupVersion, res string, ns string, name string) Result {
	return WrapResult(c.Request(gv, res, ns).Name(name).Do())
}

func (c *GenericClient) GetTiDBByName(ns string, name string) Result {
	return c.GetByName(v1alpha1.SchemeGroupVersion, "tidbclusters", ns, name)
}

func (c *GenericClient) GetCoreByName(res string, ns string, name string) Result {
	return c.GetByName(corev1.SchemeGroupVersion, res, ns, name)
}

func (c *GenericClient) GetAppsByName(res string, ns string, name string) Result {
	return c.GetByName(appsv1.SchemeGroupVersion, res, ns, name)
}

func (c *GenericClient) ListByLabels(gv schema.GroupVersion, res string, ns string, ls map[string]string) Result {
	opts := metav1.ListOptions{LabelSelector: labels.SelectorFromSet(ls).String()}
	return WrapResult(c.Request(gv, res, ns).VersionedParams(&opts, metav1.ParameterCodec).Do())
}

func (c *GenericClient) ListTiDBByLabels(ns string, ls map[string]string) Result {
	return c.ListByLabels(v1alpha1.SchemeGroupVersion, "tidbclusters", ns, ls)
}

func (c *GenericClient) ListCoreByLabels(res string, ns string, ls map[string]string) Result {
	return c.ListByLabels(corev1.SchemeGroupVersion, res, ns, ls)
}

func (c *GenericClient) ListAppsByLabels(res string, ns string, ls map[string]string) Result {
	return c.ListByLabels(appsv1.SchemeGroupVersion, res, ns, ls)
}
