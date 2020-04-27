// Copyright 2020 PingCAP, Inc.
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

package member

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	extensionslister "k8s.io/client-go/listers/extensions/v1beta1"
)

type DashboardManager struct {
	ingressLister extensionslister.IngressLister
	typedControl  controller.TypedControlInterface
}

func NewDashboardManager(ingressLister extensionslister.IngressLister, typedControl controller.TypedControlInterface) DashboardManager {
	return DashboardManager{
		ingressLister: ingressLister,
		typedControl:  typedControl,
	}
}

func (dm *DashboardManager) Sync(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.Dashboard == nil {
		return nil
	}
	return dm.syncDashboardIngress(tc)
}

func (dm *DashboardManager) syncDashboardIngress(tc *v1alpha1.TidbCluster) error {
	// If DashboardIngress is not defined, check whether the ingress existed. If it does, delete it.
	if tc.Spec.Dashboard.Ingress == nil {
		ingress, err := dm.ingressLister.Ingresses(tc.Namespace).Get(util.GetDashboardIngressName(tc))
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		return dm.typedControl.Delete(tc, ingress)
	}
	ingress := getDashboardIngress(tc)
	_, err := dm.typedControl.CreateOrUpdateIngress(tc, ingress)
	return err
}

func getDashboardIngress(tc *v1alpha1.TidbCluster) *extensionsv1beta1.Ingress {
	instanceName := tc.GetInstanceName()
	pdLabel := label.New().Instance(instanceName).PD().Labels()
	backend := extensionsv1beta1.IngressBackend{
		ServiceName: controller.PDMemberName(tc.Name),
		ServicePort: intstr.FromInt(2379),
	}
	ingress := &extensionsv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:            util.GetDashboardIngressName(tc),
			Namespace:       tc.Namespace,
			Labels:          pdLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
			Annotations:     tc.Spec.Dashboard.Ingress.Annotations,
		},
		Spec: extensionsv1beta1.IngressSpec{
			TLS: tc.Spec.Dashboard.Ingress.TLS,
		},
	}

	for _, host := range tc.Spec.Dashboard.Ingress.Hosts {
		rule := extensionsv1beta1.IngressRule{
			Host: host,
			IngressRuleValue: extensionsv1beta1.IngressRuleValue{
				HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
					Paths: []extensionsv1beta1.HTTPIngressPath{
						{
							Path:    "/dashboard",
							Backend: backend,
						},
					},
				},
			},
		}
		ingress.Spec.Rules = append(ingress.Spec.Rules, rule)
	}
	return ingress
}
