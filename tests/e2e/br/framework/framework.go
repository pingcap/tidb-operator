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

package framework

import (
	"fmt"
	"strings"

	"github.com/onsi/ginkgo"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	onceutil "github.com/pingcap/tidb-operator/tests/e2e/br/utils/once"
	"github.com/pingcap/tidb-operator/tests/e2e/br/utils/portforward"
	"github.com/pingcap/tidb-operator/tests/e2e/br/utils/s3"
	tlsutil "github.com/pingcap/tidb-operator/tests/e2e/br/utils/tls"
	yamlutil "github.com/pingcap/tidb-operator/tests/e2e/br/utils/yaml"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	apiregistration "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
)

type Framework struct {
	*framework.Framework

	once onceutil.Once

	// client for tidb-operator.
	ExtClient versioned.Interface
	// client for advanced statefulset.
	ASClient asclientset.Interface
	// client for apiregistration.
	ARClient apiregistration.Interface
	// client for apiextensions.
	AEClient apiextensions.Interface

	RESTMapper *restmapper.DeferredDiscoveryRESTMapper

	// client can create with yaml file.
	// It's useful to create some dynamic resource or custom resource without client
	YAMLClient yamlutil.Interface

	// TLSManager is prepared for generating tls.
	// TODO: manage cert-manager lifecycle.
	TLSManager tlsutil.Manager

	// PortForwarder is defined to visit pod in local.
	PortForwarder portforward.PortForwarder

	// Storage defines interface of s3 storage
	Storage s3.Interface

	cleanupHandle framework.CleanupActionHandle
}

func NewFramework(baseName string) *Framework {
	f := &Framework{}
	ginkgo.AfterEach(f.AfterEach)
	base := framework.NewDefaultFramework(baseName)
	f.Framework = base
	ginkgo.BeforeEach(f.BeforeEach)
	return f
}

func (f *Framework) BeforeEach() {
	f.cleanupHandle = framework.AddCleanupAction(f.AfterEach)
	if err := f.once.Do(func() error {
		ginkgo.By("Creating a tidb operator client")
		config, err := f.LoadConfig()
		if err != nil {
			return err
		}

		f.ExtClient, err = versioned.NewForConfig(config)
		if err != nil {
			return err
		}

		f.ASClient, err = asclientset.NewForConfig(config)
		if err != nil {
			return err
		}

		f.ARClient, err = apiregistration.NewForConfig(config)
		if err != nil {
			return err
		}

		f.AEClient, err = apiextensions.NewForConfig(config)
		if err != nil {
			return err
		}

		dc := memory.NewMemCacheClient(f.ClientSet.Discovery())
		f.RESTMapper = restmapper.NewDeferredDiscoveryRESTMapper(dc)
		f.RESTMapper.Reset()

		f.YAMLClient = yamlutil.New(f.DynamicClient, f.RESTMapper)
		f.TLSManager = tlsutil.New(f.YAMLClient)

		f.PortForwarder, err = portforward.NewPortForwarderForConfig(config)
		if err != nil {
			return err
		}

		provider := framework.TestContext.Provider
		f.Storage, err = s3.New(provider, f.ClientSet, f.PortForwarder)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		framework.ExpectNoError(err, "init client failed")
	}
	// always reset mapper cache
	f.RESTMapper.Reset()
}

func (f *Framework) AfterEach() {
	framework.RemoveCleanupAction(f.cleanupHandle)
	if !ginkgo.CurrentGinkgoTestDescription().Failed {
		ginkgo.By("Try to clean up all backups")
		f.ForceCleanBackups(f.Namespace.Name)
	} else {
		framework.Logf("Skip cleaning up backup")
	}
}

func (f *Framework) ForceCleanBackups(ns string) {
	bl, err := f.ExtClient.PingcapV1alpha1().Backups(ns).List(metav1.ListOptions{})
	if err != nil {
		framework.Logf("failed to list backups in namespace %s: %v", ns, err)
		return
	}
	for i := range bl.Items {
		name := bl.Items[i].Name
		if err := f.ExtClient.PingcapV1alpha1().Backups(ns).Delete(name, nil); err != nil {
			framework.Logf("failed to delete backup(%s) in namespace %s: %v", name, ns, err)
			return
		}
		// use patch to avoid update conflicts
		patch := []byte(`[{"op":"remove","path":"/metadata/finalizers"}]`)
		if _, err := f.ExtClient.PingcapV1alpha1().Backups(ns).Patch(name, types.JSONPatchType, patch); err != nil {
			framework.Logf("failed to clean backup(%s) finalizers in namespace %s: %v", name, ns, err)
			return
		}
	}
}

func (f *Framework) RecycleReleasedPV() {
	// tidb-operator may set persistentVolumeReclaimPolicy to Retain if
	// users request this. To reduce storage usage, we try to recycle them
	// if the PVC namespace does not exist anymore.
	c := f.ClientSet
	pvList, err := c.CoreV1().PersistentVolumes().List(metav1.ListOptions{})
	if err != nil {
		framework.Logf("failed to list pvs: %v", err)
		return
	}
	var (
		total          int = len(pvList.Items)
		retainReleased int
		skipped        int
		failed         int
		succeeded      int
	)
	defer func() {
		framework.Logf("recycling orphan PVs (total: %d, retainReleased: %d, skipped: %d, failed: %d, succeeded: %d)", total, retainReleased, skipped, failed, succeeded)
	}()
	for _, pv := range pvList.Items {
		if pv.Spec.PersistentVolumeReclaimPolicy != v1.PersistentVolumeReclaimRetain || pv.Status.Phase != v1.VolumeReleased {
			continue
		}
		retainReleased++
		pvcNamespaceName, ok := pv.Labels[label.NamespaceLabelKey]
		if !ok {
			framework.Logf("label %q does not exist in PV %q", label.NamespaceLabelKey, pv.Name)
			failed++
			continue
		}
		_, err := c.CoreV1().Namespaces().Get(pvcNamespaceName, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			framework.Logf("failed to get namespace %q: %v", pvcNamespaceName, err)
			failed++
			continue
		}
		if !apierrors.IsNotFound(err) {
			skipped++
			continue
		}
		// now we can safely recycle the PV
		pv.Spec.PersistentVolumeReclaimPolicy = v1.PersistentVolumeReclaimDelete
		_, err = c.CoreV1().PersistentVolumes().Update(&pv)
		if err != nil {
			failed++
			framework.Logf("failed to set PersistentVolumeReclaimPolicy of PV %q to Delete: %v", pv.Name, err)
		} else {
			succeeded++
			framework.Logf("successfully set PersistentVolumeReclaimPolicy of PV %q to Delete", pv.Name)
		}
	}
}

func (f *Framework) LoadConfig() (*rest.Config, error) {
	config, err := framework.LoadConfig()
	if err != nil {
		return nil, err
	}
	testDesc := ginkgo.CurrentGinkgoTestDescription()
	if len(testDesc.ComponentTexts) > 0 {
		componentTexts := strings.Join(testDesc.ComponentTexts, " ")
		config.UserAgent = fmt.Sprintf(
			"%v -- %v",
			rest.DefaultKubernetesUserAgent(),
			componentTexts)
	}

	config.QPS = f.Options.ClientQPS
	config.Burst = f.Options.ClientBurst
	return config, nil
}

// CreateYAML can be used for creating kubernetes object in yaml format.
// It is prepared for some CRD objects which have no client imported.
func (f *Framework) CreateYAML(yamlBytes []byte) error {
	return f.YAMLClient.Create(yamlBytes)
}
