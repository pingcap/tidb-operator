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

package tidbcluster

import (
	"context"
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/test/e2e/framework"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WaitObjectToBeControlledBy wait desired owner become the controller of the object
func WaitObjectToBeControlledByOrDie(c client.Client, obj runtime.Object, owner runtime.Object, timeout time.Duration) {
	meta, ok := obj.(metav1.Object)
	if !ok {
		e2elog.Failf("object is not a metav1.Object, cannot call WaitObjectToBeControlledByOrDie")
	}
	objGVK, err := controller.InferObjectKind(obj)
	framework.ExpectNoError(err, "Object should have GVK")
	ownerGVK, err := controller.InferObjectKind(owner)
	framework.ExpectNoError(err, "Owner should have GVK")
	ownerMeta, ok := owner.(metav1.Object)
	if !ok {
		e2elog.Failf("owner is not a metav1.Object, cannot call WaitObjectToBeControlledByOrDie")
	}
	fetched := obj.DeepCopyObject()
	err = wait.PollImmediate(10*time.Second, timeout, func() (bool, error) {
		key, err := client.ObjectKeyFromObject(obj)
		if err != nil {
			return false, err
		}
		err = c.Get(context.TODO(), key, fetched)
		if err != nil && errors.IsNotFound(err) {
			return false, err
		}
		if err != nil {
			e2elog.Logf("error get object %s/%s: %v", objGVK.Kind, meta.GetName(), err)
			return false, nil
		}

		// checked at beginning, safe to cast here
		meta := fetched.(metav1.Object)
		if !metav1.IsControlledBy(meta, ownerMeta) {
			e2elog.Logf("wait object %s/%s to be controlled by %s/%s...", objGVK.Kind, meta.GetName(), ownerGVK.Kind, ownerMeta.GetName())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		e2elog.Failf("error object %s/%s to be controlled by %s/%s: %v", objGVK.Kind, meta.GetName(), ownerGVK.Kind, ownerMeta.GetName(), err)
	}
}
