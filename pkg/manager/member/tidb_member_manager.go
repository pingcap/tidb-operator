// Copyright 2018 PingCAP, Inc.
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
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	v1 "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/utils/pointer"
)

const (
	slowQueryLogVolumeName = "slowlog"
	slowQueryLogDir        = "/var/log/tidb"
	slowQueryLogFile       = slowQueryLogDir + "/slowlog"
	// clusterCertPath is where the cert for inter-cluster communication stored (if any)
	clusterCertPath = "/var/lib/tidb-tls"
	// serverCertPath is where the tidb-server cert stored (if any)
	serverCertPath = "/var/lib/tidb-server-tls"
	// serviceAccountCAPath is where is CABundle of serviceaccount locates
	serviceAccountCAPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

type tidbMemberManager struct {
	setControl                   controller.StatefulSetControlInterface
	svcControl                   controller.ServiceControlInterface
	tidbControl                  controller.TiDBControlInterface
	typedControl                 controller.TypedControlInterface
	certControl                  controller.CertControlInterface
	setLister                    v1.StatefulSetLister
	svcLister                    corelisters.ServiceLister
	podLister                    corelisters.PodLister
	cmLister                     corelisters.ConfigMapLister
	tidbUpgrader                 Upgrader
	autoFailover                 bool
	tidbFailover                 Failover
	tidbStatefulSetIsUpgradingFn func(corelisters.PodLister, *apps.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
}

// NewTiDBMemberManager returns a *tidbMemberManager
func NewTiDBMemberManager(setControl controller.StatefulSetControlInterface,
	svcControl controller.ServiceControlInterface,
	tidbControl controller.TiDBControlInterface,
	certControl controller.CertControlInterface,
	typedControl controller.TypedControlInterface,
	setLister v1.StatefulSetLister,
	svcLister corelisters.ServiceLister,
	podLister corelisters.PodLister,
	cmLister corelisters.ConfigMapLister,
	tidbUpgrader Upgrader,
	autoFailover bool,
	tidbFailover Failover) manager.Manager {
	return &tidbMemberManager{
		setControl:                   setControl,
		svcControl:                   svcControl,
		tidbControl:                  tidbControl,
		typedControl:                 typedControl,
		certControl:                  certControl,
		setLister:                    setLister,
		svcLister:                    svcLister,
		podLister:                    podLister,
		cmLister:                     cmLister,
		tidbUpgrader:                 tidbUpgrader,
		autoFailover:                 autoFailover,
		tidbFailover:                 tidbFailover,
		tidbStatefulSetIsUpgradingFn: tidbStatefulSetIsUpgrading,
	}
}

func (tmm *tidbMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if !tc.TiKVIsAvailable() {
		return controller.RequeueErrorf("TidbCluster: [%s/%s], waiting for TiKV cluster running", ns, tcName)
	}

	// Sync TiDB Headless Service
	if err := tmm.syncTiDBHeadlessServiceForTidbCluster(tc); err != nil {
		return err
	}

	// Sync Tidb StatefulSet
	if err := tmm.syncTiDBStatefulSetForTidbCluster(tc); err != nil {
		return err
	}

	return tmm.syncTiDBService(tc)
}

func (tmm *tidbMemberManager) syncTiDBHeadlessServiceForTidbCluster(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := getNewTiDBHeadlessServiceForTidbCluster(tc)
	oldSvcTmp, err := tmm.svcLister.Services(ns).Get(controller.TiDBPeerMemberName(tcName))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return tmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
	}

	oldSvc := oldSvcTmp.DeepCopy()

	equal, err := controller.ServiceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}
	if !equal {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		err = controller.SetServiceLastAppliedConfigAnnotation(&svc)
		if err != nil {
			return err
		}
		_, err = tmm.svcControl.UpdateService(tc, &svc)
		return err
	}

	return nil
}

func (tmm *tidbMemberManager) syncTiDBStatefulSetForTidbCluster(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	oldTiDBSetTemp, err := tmm.setLister.StatefulSets(ns).Get(controller.TiDBMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	setNotExist := errors.IsNotFound(err)

	oldTiDBSet := oldTiDBSetTemp.DeepCopy()
	cm, err := tmm.syncTiDBConfigMap(tc, oldTiDBSet)
	if err != nil {
		return err
	}

	newTiDBSet := getNewTiDBSetForTidbCluster(tc, cm)
	if setNotExist {
		err = SetLastAppliedConfigAnnotation(newTiDBSet)
		if err != nil {
			return err
		}
		if tc.Spec.EnableTLSCluster {
			err := tmm.syncTiDBClusterCerts(tc)
			if err != nil {
				return err
			}
		}
		if tc.Spec.TiDB.EnableTLSClient {
			err := tmm.syncTiDBServerCerts(tc)
			if err != nil {
				return err
			}
			err = tmm.syncTiDBClientCerts(tc)
			if err != nil {
				return err
			}
		}
		err = tmm.setControl.CreateStatefulSet(tc, newTiDBSet)
		if err != nil {
			return err
		}
		tc.Status.TiDB.StatefulSet = &apps.StatefulSetStatus{}
		return nil
	}

	if err = tmm.syncTidbClusterStatus(tc, oldTiDBSet); err != nil {
		return err
	}

	if !templateEqual(newTiDBSet.Spec.Template, oldTiDBSet.Spec.Template) || tc.Status.TiDB.Phase == v1alpha1.UpgradePhase {
		if err := tmm.tidbUpgrader.Upgrade(tc, oldTiDBSet, newTiDBSet); err != nil {
			return err
		}
	}

	if tmm.autoFailover {
		if tc.Spec.TiDB.Replicas == int32(0) && tc.Status.TiDB.FailureMembers != nil {
			tmm.tidbFailover.Recover(tc)
		}
		if tc.TiDBAllPodsStarted() && tc.TiDBAllMembersReady() && tc.Status.TiDB.FailureMembers != nil {
			tmm.tidbFailover.Recover(tc)
		} else if tc.TiDBAllPodsStarted() && !tc.TiDBAllMembersReady() {
			if err := tmm.tidbFailover.Failover(tc); err != nil {
				return err
			}
		}
	}

	if !statefulSetEqual(*newTiDBSet, *oldTiDBSet) {
		set := *oldTiDBSet
		set.Annotations = newTiDBSet.Annotations
		set.Spec.Template = newTiDBSet.Spec.Template
		*set.Spec.Replicas = *newTiDBSet.Spec.Replicas
		set.Spec.UpdateStrategy = newTiDBSet.Spec.UpdateStrategy
		err := SetLastAppliedConfigAnnotation(&set)
		if err != nil {
			return err
		}
		_, err = tmm.setControl.UpdateStatefulSet(tc, &set)
		return err
	}

	return nil
}

// syncTiDBClusterCerts creates the cert pair for TiDB if not exist, the cert
// pair is used to communicate with other TiDB components, like TiKVs and PDs
func (tmm *tidbMemberManager) syncTiDBClusterCerts(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	svcName := controller.TiDBMemberName(tcName)
	peerName := controller.TiDBPeerMemberName(tcName)

	if tmm.certControl.CheckSecret(ns, svcName) {
		return nil
	}

	hostList := []string{
		svcName,
		peerName,
		fmt.Sprintf("%s.%s", svcName, ns),
		fmt.Sprintf("%s.%s", peerName, ns),
		fmt.Sprintf("*.%s.%s", peerName, ns),
	}

	certOpts := &controller.TiDBClusterCertOptions{
		Namespace:  ns,
		Instance:   tcName,
		CommonName: svcName,
		HostList:   hostList,
		Component:  "tidb",
		Suffix:     "tidb",
	}

	return tmm.certControl.Create(controller.GetOwnerRef(tc), certOpts)
}

// syncTiDBServerCerts creates the cert pair for TiDB if not exist, the cert
// pair is used to communicate with DB clients with encrypted connections
func (tmm *tidbMemberManager) syncTiDBServerCerts(tc *v1alpha1.TidbCluster) error {
	suffix := "tidb-server"
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	svcName := fmt.Sprintf("%s-%s", tcName, suffix)

	if tmm.certControl.CheckSecret(ns, svcName) {
		return nil
	}

	hostList := []string{
		svcName,
		fmt.Sprintf("%s.%s", svcName, ns),
	}

	certOpts := &controller.TiDBClusterCertOptions{
		Namespace:  ns,
		Instance:   tcName,
		CommonName: svcName,
		HostList:   hostList,
		Component:  "tidb",
		Suffix:     suffix,
	}

	return tmm.certControl.Create(controller.GetOwnerRef(tc), certOpts)
}

// syncTiDBClientCerts creates the cert pair for TiDB if not exist, the cert
// pair is used for DB clients to connect to TiDB server with encrypted connections
func (tmm *tidbMemberManager) syncTiDBClientCerts(tc *v1alpha1.TidbCluster) error {
	suffix := "tidb-client"
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	commonName := fmt.Sprintf("%s-%s", tcName, suffix)

	if tmm.certControl.CheckSecret(ns, commonName) {
		return nil
	}

	hostList := []string{
		commonName,
	}

	certOpts := &controller.TiDBClusterCertOptions{
		Namespace:  ns,
		Instance:   tcName,
		CommonName: commonName,
		HostList:   hostList,
		Component:  "tidb",
		Suffix:     suffix,
	}

	return tmm.certControl.Create(controller.GetOwnerRef(tc), certOpts)
}

func (tmm *tidbMemberManager) syncTiDBService(tc *v1alpha1.TidbCluster) error {

	newSvc := getNewTiDBServiceOrNil(tc)
	// TODO: delete tidb service if user remove the service spec deliberately
	if newSvc == nil {
		return nil
	}

	ns := newSvc.Namespace

	oldSvcTmp, err := tmm.svcLister.Services(ns).Get(newSvc.Name)
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return tmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
	}
	oldSvc := oldSvcTmp.DeepCopy()

	equal, err := controller.ServiceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}
	annoEqual := util.IsSubMapOf(newSvc.Annotations, oldSvc.Annotations)
	isOrphan := metav1.GetControllerOf(oldSvc) == nil

	if !equal || !annoEqual || isOrphan {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
		err = controller.SetServiceLastAppliedConfigAnnotation(&svc)
		if err != nil {
			return err
		}
		// apply change of annotations if any
		for k, v := range newSvc.Annotations {
			svc.Annotations[k] = v
		}
		// also override labels when adopt orphan
		if isOrphan {
			svc.OwnerReferences = newSvc.OwnerReferences
			svc.Labels = newSvc.Labels
		}
		_, err = tmm.svcControl.UpdateService(tc, &svc)
		return err
	}

	return nil
}

// syncTiDBConfigMap syncs the configmap of tidb
func (tmm *tidbMemberManager) syncTiDBConfigMap(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) (*corev1.ConfigMap, error) {

	// For backward compatibility, only sync tidb configmap when .tidb.config is non-nil
	if tc.Spec.TiDB.Config == nil {
		return nil, nil
	}
	newCm, err := getTiDBConfigMap(tc)
	if err != nil {
		return nil, err
	}
	if set != nil && tc.TiDBConfigUpdateStrategy() == v1alpha1.ConfigUpdateStrategyInPlace {
		inUseName := FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.TiDBMemberName(tc.Name))
		})
		if inUseName != "" {
			newCm.Name = inUseName
		}
	}

	return tmm.typedControl.CreateOrUpdateConfigMap(tc, newCm)
}

func getTiDBConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {

	config := tc.Spec.TiDB.Config
	if config == nil {
		return nil, nil
	}

	// override CA if tls enabled
	if tc.Spec.EnableTLSCluster {
		if config.Security == nil {
			config.Security = &v1alpha1.Security{}
		}
		config.Security.ClusterSSLCA = pointer.StringPtr(serviceAccountCAPath)
		config.Security.ClusterSSLCert = pointer.StringPtr(path.Join(clusterCertPath, "cert"))
		config.Security.ClusterSSLKey = pointer.StringPtr(path.Join(clusterCertPath, "key"))
	}
	if tc.Spec.TiDB.EnableTLSClient {
		if config.Security == nil {
			config.Security = &v1alpha1.Security{}
		}
		config.Security.SSLCA = pointer.StringPtr(serviceAccountCAPath)
		config.Security.SSLCert = pointer.StringPtr(path.Join(serverCertPath, "cert"))
		config.Security.SSLKey = pointer.StringPtr(path.Join(serverCertPath, "key"))
	}
	confText, err := MarshalTOML(config)
	if err != nil {
		return nil, err
	}

	plugins := tc.Spec.TiDB.Plugins
	startScript, err := RenderTiDBStartScript(&TidbStartScriptModel{
		ClusterName:     tc.Name,
		EnablePlugin:    len(plugins) > 0,
		PluginDirectory: "/plugins",
		PluginList:      strings.Join(plugins, ","),
	})
	if err != nil {
		return nil, err
	}
	data := map[string]string{
		"config-file":    string(confText),
		"startup-script": startScript,
	}
	name := controller.TiDBMemberName(tc.Name)
	instanceName := tc.GetInstanceName()
	tidbLabels := label.New().Instance(instanceName).TiDB().Labels()

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       tc.Namespace,
			Labels:          tidbLabels,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Data: data,
	}

	if tc.TiDBConfigUpdateStrategy() == v1alpha1.ConfigUpdateStrategyRollingUpdate {
		if err := AddConfigMapDigestSuffix(cm); err != nil {
			return nil, err
		}
	}

	return cm, nil
}

func getNewTiDBServiceOrNil(tc *v1alpha1.TidbCluster) *corev1.Service {

	svcSpec := tc.Spec.TiDB.Service
	if svcSpec == nil {
		return nil
	}

	ns := tc.Namespace
	tcName := tc.Name
	instanceName := tc.GetInstanceName()
	tidbLabels := label.New().Instance(instanceName).TiDB().Labels()
	svcName := controller.TiDBMemberName(tcName)

	ports := []corev1.ServicePort{
		{
			Name:       "mysql-client",
			Port:       4000,
			TargetPort: intstr.FromInt(4000),
			Protocol:   corev1.ProtocolTCP,
		},
	}
	if svcSpec.ExposeStatus {
		ports = append(ports, corev1.ServicePort{
			Name:       "status",
			Port:       10080,
			TargetPort: intstr.FromInt(10080),
			Protocol:   corev1.ProtocolTCP,
		})
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          tidbLabels,
			Annotations:     svcSpec.Annotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			Type:                  svcSpec.Type,
			Ports:                 ports,
			ExternalTrafficPolicy: svcSpec.ExternalTrafficPolicy,
			ClusterIP:             svcSpec.ClusterIP,
			LoadBalancerIP:        svcSpec.LoadBalancerIP,
			Selector:              tidbLabels,
		},
	}
}

func getNewTiDBHeadlessServiceForTidbCluster(tc *v1alpha1.TidbCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	instanceName := tc.GetInstanceName()
	svcName := controller.TiDBPeerMemberName(tcName)
	tidbLabel := label.New().Instance(instanceName).TiDB().Labels()

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          tidbLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "status",
					Port:       10080,
					TargetPort: intstr.FromInt(10080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector:                 tidbLabel,
			PublishNotReadyAddresses: true,
		},
	}
}

func getNewTiDBSetForTidbCluster(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) *apps.StatefulSet {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	instanceName := tc.GetInstanceName()
	tidbConfigMap := controller.MemberConfigMapName(tc, v1alpha1.TiDBMemberType)
	if cm != nil {
		tidbConfigMap = cm.Name
	}

	annMount, annVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annMount,
		{Name: "config", ReadOnly: true, MountPath: "/etc/tidb"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
	}
	if tc.Spec.EnableTLSCluster {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "tidb-tls", ReadOnly: true, MountPath: clusterCertPath,
		})
	}
	if tc.Spec.TiDB.EnableTLSClient {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "tidb-server-tls", ReadOnly: true, MountPath: serverCertPath,
		})
	}

	vols := []corev1.Volume{
		annVolume,
		{Name: "config", VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: tidbConfigMap,
				},
				Items: []corev1.KeyToPath{{Key: "config-file", Path: "tidb.toml"}},
			}},
		},
		{Name: "startup-script", VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: tidbConfigMap,
				},
				Items: []corev1.KeyToPath{{Key: "startup-script", Path: "tidb_start_script.sh"}},
			}},
		},
	}
	if tc.Spec.EnableTLSCluster {
		vols = append(vols, corev1.Volume{
			Name: "tidb-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: controller.TiDBMemberName(tcName),
				},
			},
		})
	}
	if tc.Spec.TiDB.EnableTLSClient {
		vols = append(vols, corev1.Volume{
			Name: "tidb-server-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: fmt.Sprintf("%s-%s", controller.TiDBMemberName(tcName), "server"),
				},
			},
		})
	}

	sysctls := "sysctl -w"
	var initContainers []corev1.Container
	if tc.BaseTiDBSpec() != nil {
		init, ok := tc.BaseTiDBSpec().Annotations()[label.AnnSysctlInit]
		if ok && (init == label.AnnSysctlInitVal) {
			if tc.BaseTiDBSpec().PodSecurityContext() != nil && len(tc.BaseTiDBSpec().PodSecurityContext().Sysctls) > 0 {
				for _, sysctl := range tc.BaseTiDBSpec().PodSecurityContext().Sysctls {
					sysctls = sysctls + fmt.Sprintf(" %s=%s", sysctl.Name, sysctl.Value)
				}
				privileged := true
				initContainers = append(initContainers, corev1.Container{
					Name:  "init",
					Image: tc.HelperImage(),
					Command: []string{
						"sh",
						"-c",
						sysctls,
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
					},
				})
			}
		}
	}
	// Init container is only used for the case where allowed-unsafe-sysctls
	// cannot be enabled for kubelet, so clean the sysctl in statefulset
	// SecurityContext if init container is enabled
	podSecurityContext := tc.BaseTiDBSpec().PodSecurityContext().DeepCopy()
	if len(initContainers) > 0 {
		podSecurityContext.Sysctls = []corev1.Sysctl{}
	}

	var containers []corev1.Container
	if tc.Spec.TiDB.SeparateSlowLog {
		// mount a shared volume and tail the slow log to STDOUT using a sidecar.
		vols = append(vols, corev1.Volume{
			Name: slowQueryLogVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
		volMounts = append(volMounts, corev1.VolumeMount{Name: slowQueryLogVolumeName, MountPath: slowQueryLogDir})
		containers = append(containers, corev1.Container{
			Name:            v1alpha1.SlowLogTailerMemberType.String(),
			Image:           tc.HelperImage(),
			ImagePullPolicy: tc.HelperImagePullPolicy(),
			Resources:       controller.ContainerResource(tc.Spec.TiDB.SlowLogTailer.ResourceRequirements),
			VolumeMounts: []corev1.VolumeMount{
				{Name: slowQueryLogVolumeName, MountPath: slowQueryLogDir},
			},
			Command: []string{
				"sh",
				"-c",
				fmt.Sprintf("touch %s; tail -n0 -F %s;", slowQueryLogFile, slowQueryLogFile),
			},
		})
	}

	slowLogFileEnvVal := ""
	if tc.Spec.TiDB.SeparateSlowLog {
		slowLogFileEnvVal = slowQueryLogFile
	}
	envs := []corev1.EnvVar{
		{
			Name:  "CLUSTER_NAME",
			Value: tc.GetName(),
		},
		{
			Name:  "TZ",
			Value: tc.Spec.Timezone,
		},
		{
			Name:  "BINLOG_ENABLED",
			Value: strconv.FormatBool(tc.Spec.TiDB.BinlogEnabled),
		},
		{
			Name:  "SLOW_LOG_FILE",
			Value: slowLogFileEnvVal,
		},
	}

	scheme := corev1.URISchemeHTTP
	if tc.Spec.EnableTLSCluster {
		scheme = corev1.URISchemeHTTPS
	}
	containers = append(containers, corev1.Container{
		Name:            v1alpha1.TiDBMemberType.String(),
		Image:           tc.BaseTiDBSpec().Image(),
		Command:         []string{"/bin/sh", "/usr/local/bin/tidb_start_script.sh"},
		ImagePullPolicy: tc.BaseTiDBSpec().ImagePullPolicy(),
		Ports: []corev1.ContainerPort{
			{
				Name:          "server",
				ContainerPort: int32(4000),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "status", // pprof, status, metrics
				ContainerPort: int32(10080),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    controller.ContainerResource(tc.Spec.TiDB.ResourceRequirements),
		Env:          envs,
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/status",
					Port:   intstr.FromInt(10080),
					Scheme: scheme,
				},
			},
			InitialDelaySeconds: int32(10),
		},
	})

	podSpec := corev1.PodSpec{
		SchedulerName:     tc.BaseTiDBSpec().SchedulerName(),
		Affinity:          tc.BaseTiDBSpec().Affinity(),
		NodeSelector:      tc.BaseTiDBSpec().NodeSelector(),
		HostNetwork:       tc.BaseTiDBSpec().HostNetwork(),
		Containers:        containers,
		RestartPolicy:     corev1.RestartPolicyAlways,
		Tolerations:       tc.BaseTiDBSpec().Tolerations(),
		Volumes:           vols,
		SecurityContext:   podSecurityContext,
		PriorityClassName: tc.BaseTiDBSpec().PriorityClassName(),
		InitContainers:    initContainers,
	}

	if tc.BaseTiDBSpec().HostNetwork() {
		podSpec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
	}

	tidbLabel := label.New().Instance(instanceName).TiDB()
	podAnnotations := CombineAnnotations(controller.AnnProm(10080), tc.BaseTiDBSpec().Annotations())
	stsAnnotations := getStsAnnotations(tc, label.TiDBLabelVal)
	tidbSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            controller.TiDBMemberName(tcName),
			Namespace:       ns,
			Labels:          tidbLabel.Labels(),
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: controller.Int32Ptr(tc.TiDBStsDesiredReplicas()),
			Selector: tidbLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      tidbLabel.Labels(),
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			ServiceName:         controller.TiDBPeerMemberName(tcName),
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &apps.RollingUpdateStatefulSetStrategy{Partition: controller.Int32Ptr(tc.TiDBStsDesiredReplicas())},
			},
		},
	}
	return tidbSet
}

func (tmm *tidbMemberManager) syncTidbClusterStatus(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	tc.Status.TiDB.StatefulSet = &set.Status

	upgrading, err := tmm.tidbStatefulSetIsUpgradingFn(tmm.podLister, set, tc)
	if err != nil {
		return err
	}
	if upgrading && tc.Status.TiKV.Phase != v1alpha1.UpgradePhase && tc.Status.PD.Phase != v1alpha1.UpgradePhase {
		tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.TiDB.Phase = v1alpha1.NormalPhase
	}

	tidbStatus := map[string]v1alpha1.TiDBMember{}
	for id := range helper.GetPodOrdinals(tc.Status.TiDB.StatefulSet.Replicas, set) {
		name := fmt.Sprintf("%s-%d", controller.TiDBMemberName(tc.GetName()), id)
		health, err := tmm.tidbControl.GetHealth(tc, int32(id))
		if err != nil {
			return err
		}
		newTidbMember := v1alpha1.TiDBMember{
			Name:   name,
			Health: health,
		}
		oldTidbMember, exist := tc.Status.TiDB.Members[name]

		newTidbMember.LastTransitionTime = metav1.Now()
		if exist {
			newTidbMember.NodeName = oldTidbMember.NodeName
			if oldTidbMember.Health == newTidbMember.Health {
				newTidbMember.LastTransitionTime = oldTidbMember.LastTransitionTime
			}
		}
		pod, err := tmm.podLister.Pods(tc.GetNamespace()).Get(name)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		if pod != nil && pod.Spec.NodeName != "" {
			// Update assiged node if pod exists and is scheduled
			newTidbMember.NodeName = pod.Spec.NodeName
		}
		tidbStatus[name] = newTidbMember
	}
	tc.Status.TiDB.Members = tidbStatus

	return nil
}

func tidbStatefulSetIsUpgrading(podLister corelisters.PodLister, set *apps.StatefulSet, tc *v1alpha1.TidbCluster) (bool, error) {
	if statefulSetIsUpgrading(set) {
		return true, nil
	}
	selector, err := label.New().
		Instance(tc.GetInstanceName()).
		TiDB().
		Selector()
	if err != nil {
		return false, err
	}
	tidbPods, err := podLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return false, err
	}
	for _, pod := range tidbPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != tc.Status.TiDB.StatefulSet.UpdateRevision {
			return true, nil
		}
	}
	return false, nil
}

type FakeTiDBMemberManager struct {
	err error
}

func NewFakeTiDBMemberManager() *FakeTiDBMemberManager {
	return &FakeTiDBMemberManager{}
}

func (ftmm *FakeTiDBMemberManager) SetSyncError(err error) {
	ftmm.err = err
}

func (ftmm *FakeTiDBMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if ftmm.err != nil {
		return ftmm.err
	}
	if len(tc.Status.TiDB.Members) != 0 {
		// simulate status update
		tc.Status.ClusterID = string(uuid.NewUUID())
	}
	return nil
}
