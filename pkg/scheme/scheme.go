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

package scheme

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	brv1alpha1 "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/kubefeat"
)

// Scheme is used by client to visit kubernetes API.
var (
	Scheme         = runtime.NewScheme()
	Codecs         = serializer.NewCodecFactory(Scheme)
	ParameterCodec = runtime.NewParameterCodec(Scheme)
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))
	utilruntime.Must(v1alpha1.Install(Scheme))
	utilruntime.Must(brv1alpha1.Install(Scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(Scheme))
}

func GroupVersions() []schema.GroupVersion {
	gvs := []schema.GroupVersion{
		corev1.SchemeGroupVersion,
		rbacv1.SchemeGroupVersion,
		storagev1.SchemeGroupVersion,
		v1alpha1.SchemeGroupVersion,
		batchv1.SchemeGroupVersion,
	}

	return gvs
}

func DynamicGroupVersions() []schema.GroupVersion {
	gvs := GroupVersions()
	if kubefeat.Stage(kubefeat.VolumeAttributesClass).Enabled(kubefeat.BETA) {
		gvs = append(gvs, storagev1beta1.SchemeGroupVersion)
	}

	return gvs
}

func CRDGroupVersions() []schema.GroupVersion {
	return []schema.GroupVersion{
		apiextensionsv1.SchemeGroupVersion,
	}
}
