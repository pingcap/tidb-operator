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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"text/template"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
)

func startWebhook(f *framework.Framework, image, ns, svcName string, cert []byte, key []byte) (*v1.Pod, *v1.Service) {
	var err error
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      "webhook",
		},
		BinaryData: map[string][]byte{
			"webhook.cert": cert,
			"webhook.key":  key,
		},
	}
	cm, err = f.ClientSet.CoreV1().ConfigMaps(ns).Create(cm)
	framework.ExpectNoError(err, "failed to create ConfigMap")

	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      "webhook",
		},
	}
	sa, err = f.ClientSet.CoreV1().ServiceAccounts(ns).Create(sa)
	framework.ExpectNoError(err, "failed to create ServiceAccount")

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      "webhook",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{"pingcap.com"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}
	role, err = f.ClientSet.RbacV1().Roles(ns).Create(role)
	framework.ExpectNoError(err, "failed to create Role")

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "webhook",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Namespace: sa.Namespace,
				Name:      sa.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role.Name,
		},
	}
	_, err = f.ClientSet.RbacV1().RoleBindings(ns).Create(roleBinding)
	framework.ExpectNoError(err, "failed to create RoleBinding")

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      svcName,
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": "webhook",
			},
			Ports: []v1.ServicePort{
				{
					Port:       443,
					TargetPort: intstr.FromInt(443),
				},
			},
		},
	}
	svc, err = f.ClientSet.CoreV1().Services(ns).Create(svc)
	framework.ExpectNoError(err, "failed to create Service")

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      "webhook",
			Labels: map[string]string{
				"app": "webhook",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "webhook",
					Image:           image,
					ImagePullPolicy: v1.PullIfNotPresent,
					Command:         []string{"/usr/local/bin/webhook"},
					Args: []string{
						fmt.Sprintf("--cert=%s", "/etc/tls/webhook.cert"),
						fmt.Sprintf("--key=%s", "/etc/tls/webhook.key"),
						fmt.Sprintf("--watch-namespaces=%s", ns),
					},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "tls",
							MountPath: "/etc/tls",
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: "tls",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: cm.Name,
							},
						},
					},
				},
			},
			ServiceAccountName: sa.Name,
			RestartPolicy:      v1.RestartPolicyNever,
		},
	}
	pod, err = f.ClientSet.CoreV1().Pods(ns).Create(pod)
	framework.ExpectNoError(err, "failed to create Pod")

	err = e2epod.WaitForPodRunningInNamespace(f.ClientSet, pod)
	framework.ExpectNoError(err, "failed to wait for pod %s/%s to be running", pod.Namespace, pod.Name)
	return pod, svc
}

func mustToString(set sets.Int32) string {
	b, err := json.Marshal(set.List())
	if err != nil {
		panic(err)
	}
	return string(b)
}

func installTemplate(tmplStr string, tp interface{}) error {
	var buf bytes.Buffer
	tmpl, err := template.New("template").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("error when parsing template: %v", err)
	}
	err = tmpl.Execute(&buf, tp)
	if err != nil {
		return fmt.Errorf("error when executing template: %v", err)
	}

	tmpFile, err := ioutil.TempFile(os.TempDir(), "tls-")
	if err != nil {
		return err
	}
	_, err = tmpFile.Write(buf.Bytes())
	if err != nil {
		return err
	}
	if data, err := exec.Command("sh", "-c", fmt.Sprintf("kubectl apply -f %s", tmpFile.Name())).CombinedOutput(); err != nil {
		framework.Logf("failed to apply template: %s, %v", string(data), err)
		return err
	}
	return nil
}
