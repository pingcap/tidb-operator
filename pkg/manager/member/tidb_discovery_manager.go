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

package member

import (
	"encoding/json"
	"strconv"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

const (
	PdTlsCertPath = "/var/lib/pd-tls"
)

type TidbDiscoveryManager interface {
	Reconcile(tc *v1alpha1.TidbCluster) error
}

type realTidbDiscoveryManager struct {
	ctrl controller.TypedControlInterface
}

func NewTidbDiscoveryManager(typedControl controller.TypedControlInterface) TidbDiscoveryManager {
	return &realTidbDiscoveryManager{typedControl}
}

func (m *realTidbDiscoveryManager) Reconcile(tc *v1alpha1.TidbCluster) error {

	// If PD is not specified return
	if tc.Spec.PD == nil {
		return nil
	}
	meta, _ := getDiscoveryMeta(tc, controller.DiscoveryMemberName)

	// Ensure RBAC
	_, err := m.ctrl.CreateOrUpdateRole(tc, &rbacv1.Role{
		ObjectMeta: meta,
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{v1alpha1.GroupName},
				Resources:     []string{v1alpha1.TiDBClusterName},
				ResourceNames: []string{tc.Name},
				Verbs:         []string{"get"},
			},
			{
				APIGroups: []string{v1alpha1.GroupName},
				Resources: []string{v1alpha1.DMClusterName},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{corev1.GroupName},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list"},
			},
		},
	})
	if err != nil {
		return controller.RequeueErrorf("error creating or updating discovery role: %v", err)
	}
	_, err = m.ctrl.CreateOrUpdateServiceAccount(tc, &corev1.ServiceAccount{
		ObjectMeta: meta,
	})
	if err != nil {
		return controller.RequeueErrorf("error creating or updating discovery serviceaccount: %v", err)
	}
	_, err = m.ctrl.CreateOrUpdateRoleBinding(tc, &rbacv1.RoleBinding{
		ObjectMeta: meta,
		Subjects: []rbacv1.Subject{{
			Kind: rbacv1.ServiceAccountKind,
			Name: meta.Name,
		}},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     meta.Name,
			APIGroup: rbacv1.GroupName,
		},
	})
	if err != nil {
		return controller.RequeueErrorf("error creating or updating discovery rolebinding: %v", err)
	}
	d, err := getTidbDiscoveryDeployment(tc)
	if err != nil {
		return controller.RequeueErrorf("error generating discovery deployment: %v", err)
	}
	deploy, err := m.ctrl.CreateOrUpdateDeployment(tc, d)
	if err != nil {
		return controller.RequeueErrorf("error creating or updating discovery service: %v", err)
	}
	// RBAC ensured, reconcile
	_, err = m.ctrl.CreateOrUpdateService(tc, getTidbDiscoveryService(tc, deploy))
	if err != nil {
		return controller.RequeueErrorf("error creating or updating discovery service: %v", err)
	}
	return nil
}

func getTidbDiscoveryService(tc *v1alpha1.TidbCluster, deploy *appsv1.Deployment) *corev1.Service {
	meta, _ := getDiscoveryMeta(tc, controller.DiscoveryMemberName)
	return &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "discovery",
					Port:       10261,
					TargetPort: intstr.FromInt(10261),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "proxy",
					Port:       10262,
					TargetPort: intstr.FromInt(10262),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: deploy.Spec.Template.Labels,
		},
	}
}

func getTidbDiscoveryDeployment(tc *v1alpha1.TidbCluster) (*appsv1.Deployment, error) {
	meta, l := getDiscoveryMeta(tc, controller.DiscoveryMemberName)
	d := &appsv1.Deployment{
		ObjectMeta: meta,
		Spec: appsv1.DeploymentSpec{
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			Replicas: pointer.Int32Ptr(1),
			Selector: l.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: l.Labels(),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: meta.Name,
					Containers: []corev1.Container{{
						Name:      "discovery",
						Resources: controller.ContainerResource(tc.Spec.Discovery.ResourceRequirements),
						Command: []string{
							"/usr/local/bin/tidb-discovery",
						},
						Image:           controller.TidbDiscoveryImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							{
								Name: "MY_POD_NAMESPACE",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.namespace",
									},
								},
							},
							{
								Name:  "TZ",
								Value: tc.Timezone(),
							},
							{
								Name:  "TC_NAME",
								Value: tc.Name,
							},
						},
					}},
				},
			},
		},
	}
	if tc.IsTLSClusterEnabled() {
		d.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: "pd-tls",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: util.ClusterTLSSecretName(tc.Name, label.PDLabelVal),
					},
				},
			},
		}
		d.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "pd-tls",
				ReadOnly:  true,
				MountPath: PdTlsCertPath,
			},
		}
		d.Spec.Template.Spec.Containers[0].Env = append(d.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  "TC_TLS_ENABLED",
			Value: strconv.FormatBool(true),
		})
	}
	b, err := json.Marshal(d.Spec.Template.Spec)
	if err != nil {
		return nil, err
	}
	if d.Annotations == nil {
		d.Annotations = map[string]string{}
	}
	d.Annotations[controller.LastAppliedPodTemplate] = string(b)

	if tc.Spec.ImagePullSecrets != nil {
		d.Spec.Template.Spec.ImagePullSecrets = tc.Spec.ImagePullSecrets
	}
	return d, nil
}

func getDiscoveryMeta(tc *v1alpha1.TidbCluster, nameFunc func(string) string) (metav1.ObjectMeta, label.Label) {
	instanceName := tc.GetInstanceName()
	discoveryLabel := label.New().Instance(instanceName).Discovery()

	objMeta := metav1.ObjectMeta{
		Name:            nameFunc(tc.Name),
		Namespace:       tc.Namespace,
		Labels:          discoveryLabel,
		OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
	}
	return objMeta, discoveryLabel
}

type FakeDiscoveryManager struct {
	err error
}

func NewFakeDiscoveryManger() *FakeDiscoveryManager {
	return &FakeDiscoveryManager{}
}

func (fdm *FakeDiscoveryManager) SetReconcileError(err error) {
	fdm.err = err
}

func (fdm *FakeDiscoveryManager) Reconcile(_ *v1alpha1.TidbCluster) error {
	if fdm.err != nil {
		return fdm.err
	}
	return nil
}
