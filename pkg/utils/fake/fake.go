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

package fake

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

type Object[T any] interface {
	client.Object
	*T
}

type Pointer[T any] interface {
	*T
}

type ChangeFunc[T any, PT Pointer[T]] func(obj PT) PT

func Fake[T any, PT Pointer[T]](changes ...ChangeFunc[T, PT]) PT {
	var obj PT = new(T)
	for _, change := range changes {
		obj = change(obj)
	}

	return obj
}

func FakeObj[T any, PT Object[T]](name string, changes ...ChangeFunc[T, PT]) PT {
	obj := Fake(changes...)
	obj.SetName(name)

	return obj
}

func Label[T any, PT Object[T]](k, v string) ChangeFunc[T, PT] {
	return func(obj PT) PT {
		ls := obj.GetLabels()
		if ls == nil {
			ls = map[string]string{}
		}
		ls[k] = v
		obj.SetLabels(ls)

		return obj
	}
}

func Annotation[T any, PT Object[T]](k, v string) ChangeFunc[T, PT] {
	return func(obj PT) PT {
		a := obj.GetAnnotations()
		if a == nil {
			a = map[string]string{}
		}
		a[k] = v
		obj.SetAnnotations(a)
		return obj
	}
}

func SetDeleteTimestamp[T any, PT Object[T]]() ChangeFunc[T, PT] {
	return func(obj PT) PT {
		now := metav1.Now()
		obj.SetDeletionTimestamp(&now)
		return obj
	}
}

func AddFinalizer[T any, PT Object[T]]() ChangeFunc[T, PT] {
	return func(obj PT) PT {
		controllerutil.AddFinalizer(obj, v1alpha1.Finalizer)
		return obj
	}
}

func SetGeneration[T any, PT Object[T]](gen int64) ChangeFunc[T, PT] {
	return func(obj PT) PT {
		obj.SetGeneration(gen)
		return obj
	}
}

func SetNamespace[T any, PT Object[T]](ns string) ChangeFunc[T, PT] {
	return func(obj PT) PT {
		obj.SetNamespace(ns)
		return obj
	}
}

func GVK[T any, PT Object[T]](gv schema.GroupVersion) ChangeFunc[T, PT] {
	return func(obj PT) PT {
		t := reflect.TypeOf(obj).Elem()
		obj.GetObjectKind().SetGroupVersionKind(gv.WithKind(t.Name()))

		return obj
	}
}

func UID[T any, PT Object[T]](uid string) ChangeFunc[T, PT] {
	return func(obj PT) PT {
		obj.SetUID(types.UID(uid))

		return obj
	}
}
