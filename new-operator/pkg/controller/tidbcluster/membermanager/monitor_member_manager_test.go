package membermanager

import (
	"testing"

	"fmt"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/new-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"

	apps "k8s.io/api/apps/v1beta1"
)

func TestMonitorMemberManagerCreate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                    string
		prepare                 func(tc *v1.TidbCluster)
		err                     bool
		errWhenCreateDeployment bool
		errWhenCreateService    bool
		deploymentCreated       bool
		serviceCreated          bool
		expectTidbClusterFn     func(g *GomegaWithT, tc *v1.TidbCluster)
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newMonitorTidbCluster()
		ns := tc.GetNamespace()
		tcName := tc.GetName()
		if test.prepare != nil {
			test.prepare(tc)
		}

		mmm, fakeDeploymentControl, fakeSvcControl := newFakeMonitorMemberManager()

		if test.errWhenCreateDeployment {
			fakeDeploymentControl.SetCreateDeploymentError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenCreateService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err := mmm.Sync(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		deployment, err := mmm.deploymentLister.Deployments(ns).Get(controller.MonitorDeploymentName(tcName))
		if test.deploymentCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(deployment).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
		}

		svc, err := mmm.serviceLister.Services(ns).Get(controller.MonitorSvcName(tcName))
		if test.serviceCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(svc).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
		}

		if test.expectTidbClusterFn != nil {
			test.expectTidbClusterFn(g, tc)
		}
	}

	tests := []testcase{
		{
			name:                    "normal",
			prepare:                 nil,
			errWhenCreateDeployment: false,
			errWhenCreateService:    false,
			err:                     false,
			deploymentCreated:       true,
			serviceCreated:          true,
		},
		{
			name:                    "error when create deployment",
			prepare:                 nil,
			errWhenCreateDeployment: true,
			errWhenCreateService:    false,
			err:                     true,
			deploymentCreated:       false,
			serviceCreated:          false,
		},
		{
			name:                    "error when create service",
			prepare:                 nil,
			errWhenCreateDeployment: false,
			errWhenCreateService:    true,
			err:                     true,
			deploymentCreated:       true,
			serviceCreated:          false,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestMonitorMemberManagerUpdate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                    string
		modify                  func(tc *v1.TidbCluster)
		errWhenUpdateDeployment bool
		errWhenUpdateService    bool
		err                     bool
		exceptDeploymentFn      func(g *GomegaWithT, deployment *apps.Deployment, err error)
		exceptServiceFn         func(g *GomegaWithT, service *corev1.Service, err error)
		expectTidbClusterFn     func(g *GomegaWithT, tc *v1.TidbCluster)
	}

	testFn := func(test testcase, t *testing.T) {
		mmm, fakeDeploymentControl, fakeServiceControl := newFakeMonitorMemberManager()

		tc := newMonitorTidbCluster()
		ns := tc.GetNamespace()
		clusterName := tc.GetName()

		err := mmm.Sync(tc)
		g.Expect(err).NotTo(HaveOccurred())

		_, err = mmm.serviceLister.Services(ns).Get(controller.MonitorSvcName(clusterName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = mmm.deploymentLister.Deployments(ns).Get(controller.MonitorDeploymentName(clusterName))
		g.Expect(err).NotTo(HaveOccurred())

		if test.modify != nil {
			test.modify(tc)
		}
		if test.errWhenUpdateService {
			fakeServiceControl.SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("Api Update error")), 0)
		}
		if test.errWhenUpdateDeployment {
			fakeDeploymentControl.SetUpdateDeploymentError(errors.NewInternalError(fmt.Errorf("Api Update error")), 0)
		}

		err = mmm.Sync(tc)

		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.exceptDeploymentFn != nil {
			deployment, err := mmm.deploymentLister.Deployments(ns).Get(controller.MonitorDeploymentName(clusterName))
			test.exceptDeploymentFn(g, deployment, err)
		}

		if test.exceptServiceFn != nil {
			service, err := mmm.serviceLister.Services(ns).Get(controller.MonitorSvcName(clusterName))
			test.exceptServiceFn(g, service, err)
		}

		if test.errWhenUpdateService {
			fakeServiceControl.SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("Api Update error")), 0)
		}
		if test.errWhenUpdateDeployment {
			fakeDeploymentControl.SetUpdateDeploymentError(errors.NewInternalError(fmt.Errorf("Api Update error")), 0)
		}
		mmm.Sync(tc)
		if test.expectTidbClusterFn != nil {
			test.expectTidbClusterFn(g, tc)
		}
	}

	tests := []testcase{
		{
			name: "normal",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.Monitor.Prometheus.Image = "prometheus:new"
				tc.Spec.Monitor.Grafana = nil
			},
			errWhenUpdateDeployment: false,
			errWhenUpdateService:    false,
			err:                     false,
			exceptDeploymentFn: func(g *GomegaWithT, deployment *apps.Deployment, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				for _, container := range deployment.Spec.Template.Spec.Containers {
					if container.Name == "prometheus" {
						g.Expect(container.Image).To(Equal("prometheus:new"))
					}
				}
			},
			exceptServiceFn: func(g *GomegaWithT, service *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(service.Spec.Ports)).To(Equal(1))
			},

			expectTidbClusterFn: func(g *GomegaWithT, tc *v1.TidbCluster) {
				g.Expect(tc.Status.Monitor.Deployment.ObservedGeneration).Should(BeNumerically(">", int64(1)))
			},
		},
		{
			name: "err when deployment update",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.Monitor.Prometheus.Image = "prometheus:new"
				tc.Spec.Monitor.Grafana = nil
			},
			errWhenUpdateDeployment: true,
			errWhenUpdateService:    false,
			err:                     true,
			exceptDeploymentFn: func(g *GomegaWithT, deployment *apps.Deployment, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				for _, container := range deployment.Spec.Template.Spec.Containers {
					if container.Name == "prometheus" {
						g.Expect(container.Image).NotTo(Equal("prometheus:new"))
					}
				}
			},
			exceptServiceFn: func(g *GomegaWithT, service *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(service.Spec.Ports)).To(Equal(2))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1.TidbCluster) {
				g.Expect(tc.Status.Monitor.Deployment.ObservedGeneration).To(Equal(int64(1)))
			},
		},
		{
			name: "err when service update",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.Monitor.Prometheus.Image = "prometheus:new"
				tc.Spec.Monitor.Grafana = nil
			},
			errWhenUpdateDeployment: false,
			errWhenUpdateService:    true,
			err:                     true,
			exceptDeploymentFn: func(g *GomegaWithT, deployment *apps.Deployment, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				for _, container := range deployment.Spec.Template.Spec.Containers {
					if container.Name == "prometheus" {
						g.Expect(container.Image).To(Equal("prometheus:new"))
					}
				}
			},
			exceptServiceFn: func(g *GomegaWithT, service *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(service.Spec.Ports)).To(Equal(2))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1.TidbCluster) {
				g.Expect(tc.Status.Monitor.Deployment.ObservedGeneration).To(Equal(int64(2)))
			},
		},
	}

	for _, test := range tests {
		testFn(test, t)
	}
}

func newFakeMonitorMemberManager() (*monitorMemberManager, *controller.FakeDeploymentControl, *controller.FakeServiceControl) {
	fakeCli := fake.NewSimpleClientset()
	fakeKubeCli := kubefake.NewSimpleClientset()

	deploymentInformer := kubeinformers.NewSharedInformerFactory(fakeKubeCli, 0).Apps().V1beta1().Deployments()
	svcInformer := kubeinformers.NewSharedInformerFactory(fakeKubeCli, 0).Core().V1().Services()
	tcInformer := informers.NewSharedInformerFactory(fakeCli, 0).Pingcap().V1().TidbClusters()
	deploymentControl := controller.NewFakeDeploymentControl(deploymentInformer, tcInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, tcInformer)

	return &monitorMemberManager{
		deploymentControl: deploymentControl,
		serviceControl:    svcControl,
		deploymentLister:  deploymentInformer.Lister(),
		serviceLister:     svcInformer.Lister(),
	}, deploymentControl, svcControl
}

func newMonitorTidbCluster() *v1.TidbCluster {
	return &v1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: v1.TidbClusterSpec{
			Monitor: &v1.MonitorSpec{
				Prometheus: v1.ContainerSpec{
					Image: "prometheus:v1.0",
					Requests: &v1.ResourceRequirement{
						CPU:    "100m",
						Memory: "2Gi",
					},
				},
				Grafana: &v1.ContainerSpec{
					Image: "grafana:v1.0",
				},
			},
		},
	}
}
