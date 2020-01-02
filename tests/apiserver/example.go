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

package apiserver

import (
	"fmt"
	"sync"
	"time"

	g "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/tests/pkg/apimachinery"
	"github.com/pingcap/tidb-operator/tests/pkg/apiserver/apis/example/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/pkg/apiserver/apis/example/v1beta1"
	clientv1alpha1 "github.com/pingcap/tidb-operator/tests/pkg/apiserver/client/clientset/versioned/typed/example/v1alpha1"
	clientv1beta1 "github.com/pingcap/tidb-operator/tests/pkg/apiserver/client/clientset/versioned/typed/example/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	rbacv1helpers "k8s.io/kubernetes/pkg/apis/rbac/v1"
	"k8s.io/utils/pointer"

	apierrors "github.com/pingcap/errors"
	example "github.com/pingcap/tidb-operator/tests/pkg/apiserver/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/tests/slack"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	aggclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	"k8s.io/klog"
	e2edeploy "k8s.io/kubernetes/test/e2e/framework/deployment"
)

func init() {
	g.RegisterFailHandler(func(message string, callerSkip ...int) {
		notifyAndPanic(message, nil)
	})
}

// E2eContext is the e2e context for example apiserver
type E2eContext struct {
	Namespace  string
	KubeCli    *kubernetes.Clientset
	ExampleCli *example.Clientset
	AggCli     *aggclient.Clientset
	V1Alpha1   clientv1alpha1.PodInterface
	V1Beta1    clientv1beta1.PodInterface
	Image      string
}

func NewE2eContext(ns string, restConfig *rest.Config, image string) *E2eContext {
	exampleCli := example.NewForConfigOrDie(restConfig)
	return &E2eContext{
		Namespace:  ns,
		KubeCli:    kubernetes.NewForConfigOrDie(restConfig),
		AggCli:     aggclient.NewForConfigOrDie(restConfig),
		ExampleCli: exampleCli,
		V1Alpha1:   exampleCli.ExampleV1alpha1().Pods(ns),
		V1Beta1:    exampleCli.ExampleV1beta1().Pods(ns),
		Image:      image,
	}
}

func (c *E2eContext) Do() {
	fooAlpha1 := &v1alpha1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo-1",
			Labels: map[string]string{
				"app": "foo",
			},
		},
		Spec: v1alpha1.PodSpec{
			Container: v1alpha1.ContainerSpec{
				Image: "foo",
			},
		},
	}
	fooAlpha2 := &v1alpha1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo-2",
			Labels: map[string]string{
				"app": "foo",
			},
		},
		Spec: v1alpha1.PodSpec{
			Container: v1alpha1.ContainerSpec{
				Image: "foo",
			},
		},
	}
	fooBeta3 := &v1beta1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo-3",
			Labels: map[string]string{
				"app": "foo",
			},
		},
		Spec: v1beta1.PodSpec{
			Containers: []v1beta1.ContainerSpec{
				{
					Image: "foo",
				},
			},
		},
	}
	barBeta1 := &v1beta1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bar-1",
			Labels: map[string]string{
				"app": "bar",
			},
		},
		Spec: v1beta1.PodSpec{
			Containers: []v1beta1.ContainerSpec{
				{
					Image: "bar",
				},
			},
		},
	}
	barBeta2 := &v1beta1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bar-2",
			Labels: map[string]string{
				"app": "bar",
			},
		},
		Spec: v1beta1.PodSpec{
			Containers: []v1beta1.ContainerSpec{
				{
					Image:           "bar",
					ImagePullPolicy: "IfNotPresent",
				},
			},
		},
	}
	alphaPods := []*v1alpha1.Pod{fooAlpha1, fooAlpha2}
	betaPods := []*v1beta1.Pod{fooBeta3, barBeta1, barBeta2}
	for _, pod := range alphaPods {
		if _, err := c.V1Alpha1.Create(pod); err != nil {
			notifyAndPanic("failed to create v1alpha Pod", err)
		}
	}
	for _, pod := range betaPods {
		if _, err := c.V1Beta1.Create(pod); err != nil {
			notifyAndPanic("failed to create v1alpha Pod", err)
		}
	}

	{
		// Get
		_, err := c.V1Alpha1.Get("not-existed", metav1.GetOptions{})
		g.Expect(err).Should(g.WithTransform(apierrors.IsNotFound, g.BeTrue()), "expected not found error when get not existed resource")
		foo, err := c.V1Alpha1.Get(fooAlpha1.Name, metav1.GetOptions{})
		g.Expect(err).ShouldNot(g.HaveOccurred())
		g.Expect(foo.Spec.Container.Image).Should(g.Equal(fooAlpha1.Spec.Container.Image))
		bar, err := c.V1Beta1.Get(barBeta1.Name, metav1.GetOptions{})
		g.Expect(err).ShouldNot(g.HaveOccurred())
		g.Expect(bar.Spec.Containers[0].Image).Should(g.Equal(barBeta1.Spec.Containers[0].Image))
	}

	{
		// List
		all, err := c.V1Beta1.List(metav1.ListOptions{})
		g.Expect(err).ShouldNot(g.HaveOccurred())
		g.Expect(all.Items).Should(g.HaveLen(5))
		foos, err := c.V1Alpha1.List(metav1.ListOptions{
			LabelSelector: "app=foo",
		})
		g.Expect(err).ShouldNot(g.HaveOccurred())
		g.Expect(foos.Items).Should(g.HaveLen(3))
		bars, err := c.V1Alpha1.List(metav1.ListOptions{
			LabelSelector: "app=bar",
		})
		g.Expect(err).ShouldNot(g.HaveOccurred())
		g.Expect(bars.Items).Should(g.HaveLen(2))
		empty, err := c.V1Alpha1.List(metav1.ListOptions{
			LabelSelector: "app=not-existed",
		})
		g.Expect(err).ShouldNot(g.HaveOccurred())
		g.Expect(empty.Items).Should(g.BeEmpty())
	}

	{
		// Update
		orig, err := c.V1Alpha1.Get(fooAlpha1.Name, metav1.GetOptions{})
		g.Expect(err).ShouldNot(g.HaveOccurred())

		updated := orig.DeepCopy()
		updatedImage := "foo:v2"
		updated.Spec.Container.Image = updatedImage
		updated, err = c.V1Alpha1.Update(updated)
		g.Expect(err).ShouldNot(g.HaveOccurred())
		g.Expect(updated.Spec.Container.Image).Should(g.Equal(updatedImage), "expected update to new version")

		_, err = c.V1Alpha1.Update(orig.DeepCopy())
		g.Expect(err).Should(g.HaveOccurred(), "Expected error when update on a stale object")

		rollbacked := updated.DeepCopy()
		rollbacked.Spec.Container.Image = "foo"
		rollbacked, err = c.V1Alpha1.Update(rollbacked)
		g.Expect(err).ShouldNot(g.HaveOccurred())
		g.Expect(rollbacked.Spec.Container.Image).Should(g.Equal("foo"), "expected rollback to old version")
	}

	{
		// Patch
		orig, err := c.V1Beta1.Get(barBeta1.Name, metav1.GetOptions{})
		g.Expect(err).ShouldNot(g.HaveOccurred())

		toPatch := orig.DeepCopy()
		toPatch.Spec.Tolerations = []string{"t1"}
		toPatch, err = c.V1Beta1.Update(toPatch)
		g.Expect(err).ShouldNot(g.HaveOccurred())
		g.Expect(toPatch.Spec.Tolerations).Should(g.ConsistOf("t1"))

		patched, err := c.V1Beta1.Patch(barBeta1.Name, types.StrategicMergePatchType, []byte(`{"spec":{"tolerations": ["t2", "t3"]}}`))
		g.Expect(err).ShouldNot(g.HaveOccurred())
		g.Expect(patched.Spec.Tolerations).Should(g.ConsistOf("t2", "t3"))

		rollbacked, err := c.V1Beta1.Patch(barBeta1.Name, types.StrategicMergePatchType, []byte(`{"spec":{"tolerations": []}}`))
		g.Expect(err).ShouldNot(g.HaveOccurred())
		g.Expect(rollbacked.Spec.Tolerations).Should(g.BeEmpty())
	}

	{
		// Watch
		w, err := c.V1Alpha1.Watch(metav1.ListOptions{
			LabelSelector: "app=watch",
		})
		g.Expect(err).ShouldNot(g.HaveOccurred())
		var eventWatched sync.WaitGroup
		done := make(chan struct{})
		go func() {
			eventWatched.Add(1)
			o, err := c.V1Alpha1.Create(&v1alpha1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "temp",
					Labels: map[string]string{
						"app": "watch",
					},
				},
				Spec: v1alpha1.PodSpec{
					Container: v1alpha1.ContainerSpec{
						Image: "temp",
					},
				},
			})
			g.Expect(err).ShouldNot(g.HaveOccurred())
			waitWithTimeout(&eventWatched, 30*time.Second)

			eventWatched.Add(1)
			o.Spec.NodeSelector = map[string]string{"k": "v"}
			o, err = c.V1Alpha1.Update(o)
			g.Expect(err).ShouldNot(g.HaveOccurred())
			waitWithTimeout(&eventWatched, 30*time.Second)

			// creation of Pod do not match the selector should not be watched
			_, err = c.V1Alpha1.Create(&v1alpha1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "no-one-cares"},
				Spec: v1alpha1.PodSpec{
					Container: v1alpha1.ContainerSpec{Image: "x"},
				},
			})
			g.Expect(err).ShouldNot(g.HaveOccurred())

			eventWatched.Add(1)
			o.Spec.NodeSelector = nil
			o, err = c.V1Alpha1.Update(o)
			g.Expect(err).ShouldNot(g.HaveOccurred())
			waitWithTimeout(&eventWatched, 30*time.Second)

			eventWatched.Add(1)
			err = c.V1Alpha1.Delete(o.Name, metav1.NewDeleteOptions(0))
			g.Expect(err).ShouldNot(g.HaveOccurred())
			waitWithTimeout(&eventWatched, 30*time.Second)

			close(done)
		}()
		events := w.ResultChan()
		eventIdx := 0
	OUT:
		for {
			select {
			case <-done:
				break OUT
			case event := <-events:
				switch eventIdx {
				case 0:
					g.Expect(event.Type).Should(g.Equal(watch.Added))
					g.Expect(event.Object.(*v1alpha1.Pod).Name).Should(g.Equal("temp"))
				case 1:
					g.Expect(event.Type).Should(g.Equal(watch.Modified))
					g.Expect(event.Object.(*v1alpha1.Pod).Spec.NodeSelector).ShouldNot(g.BeEmpty(),
						"fooAlpha1 watched should be updated and has non-empty node selectors")
				case 2:
					g.Expect(event.Type).Should(g.Equal(watch.Modified))
					g.Expect(event.Object.(*v1alpha1.Pod).Spec.NodeSelector).Should(g.BeEmpty(),
						"fooAlpha1 watched should be updated back and has empty node selectors")
				case 3:
					g.Expect(event.Type).Should(g.Equal(watch.Deleted))
					g.Expect(event.Object.(*v1alpha1.Pod).Name).Should(g.Equal("temp"))
				default:
					notifyAndPanic(fmt.Sprintf("too much events watched, %v", event), nil)
				}
				eventIdx += 1
				eventWatched.Done()
			}
		}
		w.Stop()
	}
	{
		// Validation
		_, err := c.V1Alpha1.Create(&v1alpha1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "empty-image"},
		})
		g.Expect(err).Should(g.HaveOccurred(), "expected error when creating Pod with empty image")
		_, err = c.V1Beta1.Create(&v1beta1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "empty-image"},
		})
		g.Expect(err).Should(g.HaveOccurred(), "expected error when creating Pod with empty containers")
		_, err = c.V1Beta1.Create(&v1beta1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "empty-image"},
			Spec: v1beta1.PodSpec{
				Containers: []v1beta1.ContainerSpec{
					{
						Image: "not-empty",
					},
					{
						Image: "",
					},
				},
			},
		})
		g.Expect(err).Should(g.HaveOccurred(), "expected error when creating Pod with containers with empty image")
		created, err := c.V1Alpha1.Create(&v1alpha1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "try-to-update-hostname"},
			Spec: v1alpha1.PodSpec{
				Container: v1alpha1.ContainerSpec{Image: "foo"},
				HostName:  "1.1.1.1",
			},
		})
		g.Expect(err).ShouldNot(g.HaveOccurred())
		created.Spec.HostName = "2.2.2.2"
		_, err = c.V1Alpha1.Update(created)
		g.Expect(err).Should(g.HaveOccurred(), "expected error when try to update .spec.hostName")
	}
	{
		var a *v1alpha1.Pod
		var b *v1beta1.Pod
		var err error
		// Conversion
		name := "conversion"
		image := "conversion"
		a, err = c.V1Alpha1.Create(&v1alpha1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: v1alpha1.PodSpec{
				Container: v1alpha1.ContainerSpec{
					Image: image,
				},
			},
		})
		g.Expect(err).ShouldNot(g.HaveOccurred(), "expected creation in v1alpha1 success")
		b, err = c.V1Beta1.Get(name, metav1.GetOptions{})
		g.Expect(err).ShouldNot(g.HaveOccurred(), "expected getting in v1beta1 success")
		g.Expect(b.Spec.Containers).Should(g.HaveLen(1),
			"expected .spec.container in v1alpha1 is properly conversed to .spec.containers in v1beta1")
		g.Expect(b.Spec.Containers[0].Image).Should(g.Equal(a.Spec.Container.Image))
		b.Spec.Tolerations = []string{"t"}
		b, err = c.V1Beta1.Update(b)
		g.Expect(err).ShouldNot(g.HaveOccurred(), "expected set tolerations in v1beta1 success")
		g.Expect(b.Spec.Tolerations).Should(g.ConsistOf("t"))
		a, err = c.V1Alpha1.Get(name, metav1.GetOptions{})
		g.Expect(err).ShouldNot(g.HaveOccurred(), "expected getting in v1alpha1 again success")
		// simply write back
		a, err = c.V1Alpha1.Update(a)
		g.Expect(err).ShouldNot(g.HaveOccurred())
		b, err = c.V1Beta1.Get(name, metav1.GetOptions{})
		g.Expect(err).ShouldNot(g.HaveOccurred())
		g.Expect(b.Spec.Tolerations).Should(g.ConsistOf("t"), "expected write pod in v1alpha1 again do not override the .spec.tolerations field")
	}
	{
		// Defaulting
		default1, err := c.V1Alpha1.Create(&v1alpha1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "default1"},
			Spec: v1alpha1.PodSpec{
				Container: v1alpha1.ContainerSpec{
					Image: "foo",
				},
			},
		})
		g.Expect(err).ShouldNot(g.HaveOccurred())
		g.Expect(default1.Spec.Container.ImagePullPolicy).Should(g.Equal("IfNotPresent"),
			"expected imagePullPolicy of pod created in v1alpha1 default to IfNotPresent")
		default1InBeta, err := c.V1Beta1.Get("default1", metav1.GetOptions{})
		g.Expect(err).ShouldNot(g.HaveOccurred())
		g.Expect(default1InBeta.Spec.Containers[0].ImagePullPolicy).Should(g.Equal("IfNotPresent"),
			"expected imagePullPolicy of pod created in v1alpha1 default to IfNotPresent when fetched by v1beta1 client after creation")
		default2, err := c.V1Beta1.Create(&v1beta1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "default2"},
			Spec: v1beta1.PodSpec{
				Containers: []v1beta1.ContainerSpec{
					{
						Image:           "foo",
						ImagePullPolicy: "IfNotPresent",
					},
					{
						Image: "bar",
					},
				},
			},
		})
		g.Expect(err).ShouldNot(g.HaveOccurred())
		g.Expect(default2.Spec.Containers).Should(g.ConsistOf([]v1beta1.ContainerSpec{
			{
				Image:           "foo",
				ImagePullPolicy: "IfNotPresent",
			},
			{
				Image:           "bar",
				ImagePullPolicy: "Always",
			},
		},
		))
	}
}

func (c *E2eContext) Setup() {
	c.Clean()
	ns, serverName := c.Namespace, "example-apiserver"

	_, err := c.KubeCli.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		notifyAndPanic(fmt.Sprintf("failed to create namespace %s", c.Namespace), err)
	}

	certCtx, err := apimachinery.SetupServerCert(ns, serverName)
	g.Expect(err).ShouldNot(g.HaveOccurred(), "generating cert for example apiserver")

	secretName := fmt.Sprintf("%s-cert", serverName)
	secret := &v1.Secret{

		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{
			"tls.crt": certCtx.Cert,
			"tls.key": certCtx.Key,
		},
	}
	_, err = c.KubeCli.CoreV1().Secrets(ns).Create(secret)
	g.Expect(err).ShouldNot(g.HaveOccurred(), "crate cert secret for example apiserver")

	readerRoleName := fmt.Sprintf("%s-reader", serverName)
	// allow read for namespace, webhook and all operations for pingcap.com group
	_, err = c.KubeCli.RbacV1().ClusterRoles().Create(&rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: readerRoleName},
		Rules: []rbacv1.PolicyRule{
			rbacv1helpers.NewRule("list").Groups("").Resources("namespaces").RuleOrDie(),
			rbacv1helpers.NewRule("list").Groups("admissionregistration.k8s.io").Resources("*").RuleOrDie(),
			rbacv1helpers.NewRule("*").Groups("pingcap.com").Resources("*").RuleOrDie(),
		},
	})
	g.Expect(err).ShouldNot(g.HaveOccurred(), "create clusterrole fro example apiserver")

	_, err = c.KubeCli.CoreV1().ServiceAccounts(ns).Create(&v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: serverName},
	})
	g.Expect(err).ShouldNot(g.HaveOccurred(), "create SA for example apiserver")

	_, err = c.KubeCli.RbacV1().ClusterRoleBindings().Create(&rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.Namespace + ":example-apiserver-reader",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     readerRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup:  "",
				Kind:      "ServiceAccount",
				Name:      serverName,
				Namespace: ns,
			},
		},
	})
	g.Expect(err).ShouldNot(g.HaveOccurred(), "bind reader role to example-apiserver SA")

	_, err = c.KubeCli.RbacV1().ClusterRoleBindings().Create(&rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.Namespace + ":auth-delegator",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "system:auth-delegator",
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup:  "",
				Kind:      "ServiceAccount",
				Name:      serverName,
				Namespace: ns,
			},
		},
	})
	g.Expect(err).ShouldNot(g.HaveOccurred(), "bind auth-delegator role to example-apiserver SA")

	_, err = c.KubeCli.RbacV1().RoleBindings("kube-system").Create(&rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "example-apiserver-auth-reader",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "",
			Kind:     "Role",
			Name:     "extension-apiserver-authentication-reader",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serverName,
				Namespace: ns,
			},
		},
	})
	g.Expect(err).ShouldNot(g.HaveOccurred(), "bind auth-reader role to example-apiserver SA")

	one := int32(1)
	podLabels := map[string]string{"app": serverName}
	deployment, err := c.KubeCli.AppsV1().Deployments(ns).Create(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: serverName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &one,
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name: secretName,
						Command: []string{
							"/usr/local/bin/apiserver",
							"--tls-cert-file=/apiserver.local.config/certificates/tls.crt",
							"--tls-private-key-file=/apiserver.local.config/certificates/tls.key",
						},
						Image:           c.Image,
						ImagePullPolicy: v1.PullIfNotPresent,
						VolumeMounts: []v1.VolumeMount{{
							Name:      secretName,
							ReadOnly:  true,
							MountPath: "/apiserver.local.config/certificates",
						}},
					}},
					Volumes: []v1.Volume{{
						Name: secretName,
						VolumeSource: v1.VolumeSource{
							Secret: &v1.SecretVolumeSource{SecretName: secretName},
						},
					}},
					ServiceAccountName: serverName,
				},
			},
		},
	})
	g.Expect(err).ShouldNot(g.HaveOccurred(), "create example apiserver deployment")
	err = e2edeploy.WaitForDeploymentComplete(c.KubeCli, deployment)
	g.Expect(err).ShouldNot(g.HaveOccurred(), "wait for apiserver deployment complete")

	_, err = c.KubeCli.CoreV1().Services(ns).Create(&v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: serverName},
		Spec: v1.ServiceSpec{
			Selector: podLabels,
			Ports: []v1.ServicePort{{
				Protocol:   "TCP",
				Port:       443,
				TargetPort: intstr.FromInt(443),
			}},
		},
	})
	g.Expect(err).ShouldNot(g.HaveOccurred(), "create example apiserver service")

	for _, v := range []string{"v1alpha1", "v1beta1"} {
		_, err = c.AggCli.ApiregistrationV1().APIServices().Create(&apiregistrationv1.APIService{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s.example.pingcap.com", v)},
			Spec: apiregistrationv1.APIServiceSpec{
				Service: &apiregistrationv1.ServiceReference{
					Namespace: ns,
					Name:      serverName,
					Port:      pointer.Int32Ptr(443),
				},
				Group:                "example.pingcap.com",
				Version:              v,
				CABundle:             certCtx.SigningCert,
				GroupPriorityMinimum: 2000,
				VersionPriority:      200,
			},
		})
		g.Expect(err).ShouldNot(g.HaveOccurred(), "create apiservice for group: example.pingcap.com, version: %s", v)
	}

	for _, v := range []string{"v1alpha1", "v1beta1"} {
		err = wait.Poll(time.Second, 1*time.Minute, func() (bool, error) {
			apiservice, err := c.AggCli.ApiregistrationV1().APIServices().Get(fmt.Sprintf("%s.example.pingcap.com", v), metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			for _, condition := range apiservice.Status.Conditions {
				if condition.Type == apiregistrationv1.Available {
					return true, nil
				}
			}
			return false, nil
		})
		g.Expect(err).ShouldNot(g.HaveOccurred(), "wait for apiservice ready")
	}

	err = wait.Poll(30*time.Second, 3*time.Minute, func() (bool, error) {
		err := c.ExampleCli.ExampleV1alpha1().Pods(c.Namespace).DeleteCollection(
			metav1.NewDeleteOptions(0),
			metav1.ListOptions{},
		)
		if err != nil && !apierrors.IsNotFound(err) {
			klog.Errorf("Error deleting pods.example.pingcap.com, %v", err)
			return false, nil
		}
		return true, nil
	})
	g.Expect(err).ShouldNot(g.HaveOccurred(), "wait for cleaning pods.example.pingcap.com")
}

func (c *E2eContext) Clean() {
	ns, serverName := c.Namespace, "example-apiserver"

	for _, v := range []string{"v1alpha1", "v1beta1"} {
		err := c.AggCli.ApiregistrationV1().APIServices().Delete(fmt.Sprintf("%s.example.pingcap.com", v), nil)
		g.Expect(err).Should(g.Or(g.BeNil(), g.WithTransform(apierrors.IsNotFound, g.BeTrue())))
	}

	roleName := fmt.Sprintf("%s-reader", serverName)
	err := c.KubeCli.AppsV1().Deployments(ns).Delete(serverName, nil)
	g.Expect(err).Should(g.Or(g.BeNil(), g.WithTransform(apierrors.IsNotFound, g.BeTrue())))

	err = c.KubeCli.CoreV1().Services(ns).Delete(serverName, nil)
	g.Expect(err).Should(g.Or(g.BeNil(), g.WithTransform(apierrors.IsNotFound, g.BeTrue())))

	err = c.KubeCli.CoreV1().Secrets(ns).Delete(fmt.Sprintf("%s-cert", serverName), nil)
	g.Expect(err).Should(g.Or(g.BeNil(), g.WithTransform(apierrors.IsNotFound, g.BeTrue())))

	err = c.KubeCli.RbacV1().ClusterRoleBindings().Delete(ns+":auth-delegator", nil)
	g.Expect(err).Should(g.Or(g.BeNil(), g.WithTransform(apierrors.IsNotFound, g.BeTrue())))

	err = c.KubeCli.RbacV1().ClusterRoleBindings().Delete(fmt.Sprintf("%s:%s", ns, roleName), nil)
	g.Expect(err).Should(g.Or(g.BeNil(), g.WithTransform(apierrors.IsNotFound, g.BeTrue())))

	err = c.KubeCli.RbacV1().ClusterRoles().Delete(roleName, nil)
	g.Expect(err).Should(g.Or(g.BeNil(), g.WithTransform(apierrors.IsNotFound, g.BeTrue())))

	err = c.KubeCli.RbacV1().RoleBindings("kube-system").Delete("example-apiserver-auth-reader", nil)
	g.Expect(err).Should(g.Or(g.BeNil(), g.WithTransform(apierrors.IsNotFound, g.BeTrue())))

	err = c.KubeCli.CoreV1().ServiceAccounts(ns).Delete(serverName, nil)
	g.Expect(err).Should(g.Or(g.BeNil(), g.WithTransform(apierrors.IsNotFound, g.BeTrue())))
}

func notifyAndPanic(reason string, err error) {
	// Inject context
	slack.NotifyAndPanic(fmt.Errorf(fmt.Sprintf("Aggregated ApiServer test - %s, %v", reason, err)))
}

func waitWithTimeout(wg *sync.WaitGroup, timeout time.Duration) {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return
	case <-time.After(timeout):
		notifyAndPanic("wait event arrived timeout", nil)
	}
}
