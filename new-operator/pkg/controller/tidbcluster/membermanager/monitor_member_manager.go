package membermanager

import (
	"fmt"
	"reflect"

	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util/label"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	appslisters "k8s.io/client-go/listers/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const (
	defaultPromRetentionDays = 15
	prometheusName           = "prometheus"
	prometheusPort           = 9090
	grafanaName              = "grafana"
	grafanaPort              = 3000
)

var services = map[string]int{}

type monitorMemberManager struct {
	deploymentControl controller.DeploymentControlInterface
	serviceControl    controller.ServiceControlInterface
	deploymentLister  appslisters.DeploymentLister
	serviceLister     corelisters.ServiceLister
}

// NewMonitorMemberManager returns monitorMemberManager
func NewMonitorMemberManager(deploymentControl controller.DeploymentControlInterface,
	serviceControl controller.ServiceControlInterface,
	deploymentLister appslisters.DeploymentLister,
	serviceLister corelisters.ServiceLister) MemberManager {
	return &monitorMemberManager{
		deploymentControl: deploymentControl,
		deploymentLister:  deploymentLister,
		serviceControl:    serviceControl,
		serviceLister:     serviceLister,
	}
}

func (mmm *monitorMemberManager) Sync(tc *v1.TidbCluster) error {
	if err := mmm.syncMonitorDeployment(tc); err != nil {
		return err
	}

	if err := mmm.syncMonitorServices(tc); err != nil {
		return err
	}
	return nil
}

func (mmm *monitorMemberManager) syncMonitorServices(tc *v1.TidbCluster) error {
	ns := tc.GetNamespace()
	clusterName := tc.GetName()

	svcName := controller.MonitorMemberName(clusterName)
	newSvc := mmm.getNewMonitorService(tc)
	oldSvc, err := mmm.serviceLister.Services(ns).Get(svcName)
	if apierrors.IsNotFound(err) {
		return mmm.serviceControl.CreateService(tc, newSvc)
	}

	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldSvc.Spec, newSvc.Spec) {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		return mmm.serviceControl.UpdateService(tc, &svc)
	}

	return nil
}

func (mmm *monitorMemberManager) getNewMonitorService(tc *v1.TidbCluster) *corev1.Service {
	ns := tc.GetNamespace()
	clusterName := tc.GetName()
	svcName := controller.MonitorMemberName(clusterName)
	svcLabel := label.New().Cluster(clusterName).Monitor().Labels()
	servicePorts := []corev1.ServicePort{
		{
			Name:       prometheusName,
			Port:       int32(prometheusPort),
			TargetPort: intstr.FromInt(prometheusPort),
			Protocol:   corev1.ProtocolTCP,
		},
	}
	if tc.Spec.Monitor.Grafana != nil {
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:       grafanaName,
			Port:       int32(grafanaPort),
			TargetPort: intstr.FromInt(grafanaPort),
			Protocol:   corev1.ProtocolTCP,
		})
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          svcLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			Type:     controller.GetServiceType(tc.Spec.Services, svcName),
			Ports:    servicePorts,
			Selector: svcLabel,
		},
	}

	return svc
}

func (mmm *monitorMemberManager) syncMonitorDeployment(tc *v1.TidbCluster) error {
	ns := tc.GetNamespace()
	clusterName := tc.GetName()

	newDeployment := mmm.getNewMonitorDeployment(tc)

	deploymentName := controller.MonitorMemberName(clusterName)
	oldDeployment, err := mmm.deploymentLister.Deployments(ns).Get(deploymentName)
	if apierrors.IsNotFound(err) {
		err = mmm.deploymentControl.CreateDeployment(tc, newDeployment)
		if err != nil {
			return err
		}
		tc.Status.Monitor.Deployment = &apps.DeploymentStatus{}
		return nil
	}

	if err != nil {
		return err
	}

	tc.Status.Monitor.Deployment = &oldDeployment.Status

	if !reflect.DeepEqual(oldDeployment.Spec, newDeployment.Spec) {
		deployment := *oldDeployment
		deployment.Spec = newDeployment.Spec
		return mmm.deploymentControl.UpdateDeployment(tc, &deployment)
	}

	return nil
}

func (mmm *monitorMemberManager) getNewMonitorDeployment(tc *v1.TidbCluster) *apps.Deployment {
	ns := tc.GetNamespace()
	clusterName := tc.GetName()
	deploymentName := controller.MonitorMemberName(clusterName)
	deploymentLabel := label.New().Cluster(clusterName).Monitor().Labels()

	var reserveDays int32 = defaultPromRetentionDays
	if tc.Spec.Monitor.RetentionDays > 0 {
		reserveDays = tc.Spec.Monitor.RetentionDays
	}

	configName := tc.Spec.ConfigMap

	prometheusVolumeMounts := []corev1.VolumeMount{
		{Name: "timezone", ReadOnly: true, MountPath: "/etc/localtime"},
		{Name: "prometheus-config", ReadOnly: true, MountPath: "/etc/prometheus"},
		{Name: "prometheus-data", MountPath: "/prometheus"},
	}

	grafanaVolumeMounts := []corev1.VolumeMount{
		{Name: "timezone", ReadOnly: true, MountPath: "/etc/localtime"},
		{Name: "grafana-data", MountPath: "/data"},
		{Name: "grafana-config", MountPath: "/etc/grafana"},
	}

	volumes := []corev1.Volume{
		{Name: "timezone", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/etc/localtime"}}},
		{Name: "prometheus-data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: "grafana-data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: "prometheus-config", VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: configName},
				Items: []corev1.KeyToPath{
					{Key: "prometheus-config", Path: "prometheus.yml"},
					{Key: "alert-rules-config", Path: "alert.rules"},
				},
			},
		}},
		{Name: "grafana-config", VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: configName},
				Items: []corev1.KeyToPath{
					{Key: "grafana-config", Path: "grafana.ini"},
				},
			},
		}},
	}

	containers := []corev1.Container{
		{
			Name:  "prometheus",
			Image: tc.Spec.Monitor.Prometheus.Image,
			Command: []string{
				"/bin/prometheus",
			},
			Args: []string{
				"--config.file=/etc/prometheus/prometheus.yml",
				fmt.Sprintf("--storage.tsdb.retention=%dd", reserveDays),
				"--storage.tsdb.path=/prometheus",
			},
			Ports: []corev1.ContainerPort{
				{
					Name:          "prometheus",
					ContainerPort: int32(9090),
					Protocol:      corev1.ProtocolTCP,
				},
			},
			VolumeMounts: prometheusVolumeMounts,
			Resources:    util.ResourceRequirement(tc.Spec.Monitor.Prometheus),
		},
	}
	if tc.Spec.Monitor.Grafana != nil {
		containers = append(containers, corev1.Container{
			Name:  "grafana",
			Image: tc.Spec.Monitor.Grafana.Image,
			Ports: []corev1.ContainerPort{
				{
					Name:          "grafana",
					ContainerPort: int32(3000),
					Protocol:      corev1.ProtocolTCP,
				},
			},
			VolumeMounts: grafanaVolumeMounts,
			Env: []corev1.EnvVar{
				{Name: "GF_PATHS_DATA", Value: "/data"},
			},
			Resources: util.ResourceRequirement(*tc.Spec.Monitor.Grafana),
		})
	}

	deploy := &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            deploymentName,
			Namespace:       ns,
			Labels:          deploymentLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   deploymentName,
					Labels: deploymentLabel,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: tc.Spec.Monitor.ServiceAccount,
					Affinity: util.AffinityForNodeSelector(
						ns,
						tc.Spec.Monitor.NodeSelectorRequired,
						label.New().Cluster(clusterName).Monitor(),
						tc.Spec.Monitor.NodeSelector,
					),
					Containers: containers,
					Volumes:    volumes,
				},
			},
		},
	}

	return deploy
}
