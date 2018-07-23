package membermanager

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/new-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestPrivilegedTiDBMemberManagerCreate(t *testing.T) {
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
		tc := newPrivilegedTidbCluster()
		ns := tc.GetNamespace()
		tcName := tc.GetName()
		if test.prepare != nil {
			test.prepare(tc)
		}

		ptmm, fakeDeployControl, fakeSvcControl := newFakePrivilegedTidbMemberManager()

		if test.errWhenCreateDeployment {
			fakeDeployControl.SetCreateDeploymentError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}
		if test.errWhenCreateService {
			fakeSvcControl.SetCreateServiceError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err := ptmm.Sync(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		deployment, err := ptmm.deployLister.Deployments(ns).Get(controller.PriTiDBMemberName(tcName))
		if test.deploymentCreated {
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(deployment).NotTo(Equal(nil))
		} else {
			expectErrIsNotFound(g, err)
		}

		svc, err := ptmm.svcLister.Services(ns).Get(controller.PriTiDBMemberName(tcName))
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
			name:                    "error when create privileged tidb deployment",
			prepare:                 nil,
			errWhenCreateDeployment: true,
			errWhenCreateService:    false,
			err:                     true,
			deploymentCreated:       false,
			serviceCreated:          true,
		},
		{
			name:                    "error when create privileged tidb service",
			prepare:                 nil,
			errWhenCreateDeployment: false,
			errWhenCreateService:    true,
			err:                     true,
			deploymentCreated:       false,
			serviceCreated:          false,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestPrivilegedTiDBMemberManagerUpdate(t *testing.T) {
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
		ptmm, fakeDeployControl, fakeSvcControl := newFakePrivilegedTidbMemberManager()

		tc := newPrivilegedTidbCluster()
		ns := tc.GetNamespace()
		clusterName := tc.GetName()

		err := ptmm.Sync(tc)
		g.Expect(err).NotTo(HaveOccurred())

		_, err = ptmm.svcLister.Services(ns).Get(controller.PriTiDBMemberName(clusterName))
		g.Expect(err).NotTo(HaveOccurred())
		_, err = ptmm.deployLister.Deployments(ns).Get(controller.PriTiDBMemberName(clusterName))
		g.Expect(err).NotTo(HaveOccurred())

		if test.modify != nil {
			test.modify(tc)
		}
		if test.errWhenUpdateService {
			fakeSvcControl.SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("Api Update error")), 0)
		}
		if test.errWhenUpdateDeployment {
			fakeDeployControl.SetUpdateDeploymentError(errors.NewInternalError(fmt.Errorf("Api Update error")), 0)
		}

		err = ptmm.Sync(tc)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.exceptDeploymentFn != nil {
			deployment, err := ptmm.deployLister.Deployments(ns).Get(controller.PriTiDBMemberName(clusterName))
			test.exceptDeploymentFn(g, deployment, err)
		}

		if test.exceptServiceFn != nil {
			service, err := ptmm.svcLister.Services(ns).Get(controller.PriTiDBMemberName(clusterName))
			test.exceptServiceFn(g, service, err)
		}

		if test.errWhenUpdateService {
			fakeSvcControl.SetUpdateServiceError(errors.NewInternalError(fmt.Errorf("Api Update error")), 0)
		}
		if test.errWhenUpdateDeployment {
			fakeDeployControl.SetUpdateDeploymentError(errors.NewInternalError(fmt.Errorf("Api Update error")), 0)
		}

		ptmm.Sync(tc)
		if test.expectTidbClusterFn != nil {
			test.expectTidbClusterFn(g, tc)
		}
	}

	tests := []testcase{
		{
			name: "normal",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.PrivilegedTiDB.Image = "privileged-tidb:new"
				tc.Spec.Services = []v1.Service{
					{Name: v1.PriTiDBMemberType.String(), Type: string(corev1.ServiceTypeNodePort)},
				}
			},
			errWhenUpdateDeployment: false,
			errWhenUpdateService:    false,
			err:                     false,
			exceptDeploymentFn: func(g *GomegaWithT, deployment *apps.Deployment, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				for _, container := range deployment.Spec.Template.Spec.Containers {
					if container.Name == v1.PriTiDBMemberType.String() {
						g.Expect(container.Image).To(Equal("privileged-tidb:new"))
					}
				}
			},
			exceptServiceFn: func(g *GomegaWithT, service *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1.TidbCluster) {
				g.Expect(tc.Status.PrivilegedTiDB.Deployment.ObservedGeneration).To(Equal(int64(2)))
			},
		},
		{
			name: "err when privileged tidb deployment update",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.PrivilegedTiDB.Image = "privileged-tidb:new"
			},
			errWhenUpdateDeployment: true,
			errWhenUpdateService:    false,
			err:                     true,
			exceptDeploymentFn: func(g *GomegaWithT, deployment *apps.Deployment, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				for _, container := range deployment.Spec.Template.Spec.Containers {
					if container.Name == v1.PriTiDBMemberType.String() {
						g.Expect(container.Image).NotTo(Equal("privileged-tidb:new"))
					}
				}
			},
			exceptServiceFn: nil,
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1.TidbCluster) {
				g.Expect(tc.Status.PrivilegedTiDB.Deployment.ObservedGeneration).To(Equal(int64(1)))
			},
		},
		{
			name: "err when privileged tidb service update",
			modify: func(tc *v1.TidbCluster) {
				tc.Spec.Services = []v1.Service{
					{Name: v1.PriTiDBMemberType.String(), Type: string(corev1.ServiceTypeNodePort)},
				}
			},
			errWhenUpdateDeployment: false,
			errWhenUpdateService:    true,
			err:                     true,
			exceptDeploymentFn:      nil,
			exceptServiceFn: func(g *GomegaWithT, service *corev1.Service, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
			},
			expectTidbClusterFn: func(g *GomegaWithT, tc *v1.TidbCluster) {
				g.Expect(tc.Status.PrivilegedTiDB.Deployment.ObservedGeneration).To(Equal(int64(0)))
			},
		},
	}

	for _, test := range tests {
		testFn(test, t)
	}
}

func newFakePrivilegedTidbMemberManager() (*priTidbMemberManager, *controller.FakeDeploymentControl, *controller.FakeServiceControl) {
	fakeCli := fake.NewSimpleClientset()
	fakeKubeCli := kubefake.NewSimpleClientset()

	deployInformer := kubeinformers.NewSharedInformerFactory(fakeKubeCli, 0).Apps().V1beta1().Deployments()
	svcInformer := kubeinformers.NewSharedInformerFactory(fakeKubeCli, 0).Core().V1().Services()
	tcInformer := informers.NewSharedInformerFactory(fakeCli, 0).Pingcap().V1().TidbClusters()
	deployControl := controller.NewFakeDeploymentControl(deployInformer, tcInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, tcInformer)

	return &priTidbMemberManager{
		deployControl: deployControl,
		svcControl:    svcControl,
		deployLister:  deployInformer.Lister(),
		svcLister:     svcInformer.Lister(),
	}, deployControl, svcControl
}
func newPrivilegedTidbCluster() *v1.TidbCluster {
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
			PrivilegedTiDB: &v1.PrivilegedTiDBSpec{
				ContainerSpec: v1.ContainerSpec{
					Image: v1.PriTiDBMemberType.String(),
					Requests: &v1.ResourceRequirement{
						CPU:    "100m",
						Memory: "2Gi",
					},
				},
				Replicas: 1,
			},
		},
	}
}
