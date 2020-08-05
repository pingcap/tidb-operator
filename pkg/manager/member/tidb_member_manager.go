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
	"crypto/tls"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	v1 "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
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
	// tlsSecretRootCAKey is the key used in tls secret for the root CA.
	// When user use self-signed certificates, the root CA must be provided. We
	// following the same convention used in Kubernetes service token.
	tlsSecretRootCAKey = corev1.ServiceAccountRootCAKey
)

type tidbMemberManager struct {
	setControl                   controller.StatefulSetControlInterface
	svcControl                   controller.ServiceControlInterface
	tidbControl                  controller.TiDBControlInterface
	typedControl                 controller.TypedControlInterface
	setLister                    v1.StatefulSetLister
	svcLister                    corelisters.ServiceLister
	podLister                    corelisters.PodLister
	cmLister                     corelisters.ConfigMapLister
	secretLister                 corelisters.SecretLister
	tidbUpgrader                 Upgrader
	autoFailover                 bool
	tidbFailover                 Failover
	tidbStatefulSetIsUpgradingFn func(corelisters.PodLister, *apps.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
}

// NewTiDBMemberManager returns a *tidbMemberManager
func NewTiDBMemberManager(setControl controller.StatefulSetControlInterface,
	svcControl controller.ServiceControlInterface,
	tidbControl controller.TiDBControlInterface,
	typedControl controller.TypedControlInterface,
	setLister v1.StatefulSetLister,
	svcLister corelisters.ServiceLister,
	podLister corelisters.PodLister,
	secretLister corelisters.SecretLister,
	tidbUpgrader Upgrader,
	autoFailover bool,
	tidbFailover Failover) manager.Manager {
	return &tidbMemberManager{
		setControl:                   setControl,
		svcControl:                   svcControl,
		tidbControl:                  tidbControl,
		typedControl:                 typedControl,
		setLister:                    setLister,
		svcLister:                    svcLister,
		podLister:                    podLister,
		secretLister:                 secretLister,
		tidbUpgrader:                 tidbUpgrader,
		autoFailover:                 autoFailover,
		tidbFailover:                 tidbFailover,
		tidbStatefulSetIsUpgradingFn: tidbStatefulSetIsUpgrading,
	}
}

func (tmm *tidbMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	// If tikv is not specified return
	if tc.Spec.TiDB == nil {
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if !tc.TiKVIsAvailable() {
		return controller.RequeueErrorf("TidbCluster: [%s/%s], waiting for TiKV cluster running", ns, tcName)
	}
	if tc.Spec.Pump != nil {
		if !tc.PumpIsAvailable() {
			return controller.RequeueErrorf("TidbCluster: [%s/%s], waiting for Pump cluster running", ns, tcName)
		}
	}
	// Sync TiDB Headless Service
	if err := tmm.syncTiDBHeadlessServiceForTidbCluster(tc); err != nil {
		return err
	}

	// Sync TiDB Service before syncing TiDB StatefulSet
	if err := tmm.syncTiDBService(tc); err != nil {
		return err
	}

	if tc.Spec.TiDB.IsTLSClientEnabled() {
		if err := tmm.checkTLSClientCert(tc); err != nil {
			return err
		}
	}

	// Sync TiDB StatefulSet
	return tmm.syncTiDBStatefulSetForTidbCluster(tc)
}

func (tmm *tidbMemberManager) checkTLSClientCert(tc *v1alpha1.TidbCluster) error {
	ns := tc.Namespace
	secretName := tlsClientSecretName(tc)
	secret, err := tmm.secretLister.Secrets(ns).Get(secretName)
	if err != nil {
		return fmt.Errorf("unable to load certificates from secret %s/%s: %v", ns, secretName, err)
	}

	clientCert, certExists := secret.Data[corev1.TLSCertKey]
	clientKey, keyExists := secret.Data[corev1.TLSPrivateKeyKey]
	if !certExists || !keyExists {
		return fmt.Errorf("cert or key does not exist in secret %s/%s", ns, secretName)
	}

	_, err = tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return fmt.Errorf("unable to load certificates from secret %s/%s: %v", ns, secretName, err)
	}
	return nil
}

func (tmm *tidbMemberManager) syncTiDBHeadlessServiceForTidbCluster(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.Paused {
		klog.V(4).Infof("tidb cluster %s/%s is paused, skip syncing for tidb headless service", tc.GetNamespace(), tc.GetName())
		return nil
	}

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
		return fmt.Errorf("syncTiDBHeadlessServiceForTidbCluster: failed to get svc %s for cluster %s/%s, error: %s", controller.TiDBPeerMemberName(tcName), ns, tcName, err)
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
		return fmt.Errorf("syncTiDBStatefulSetForTidbCluster: failed to get sts %s for cluster %s/%s, error: %s", controller.TiDBMemberName(tcName), ns, tcName, err)
	}
	setNotExist := errors.IsNotFound(err)

	oldTiDBSet := oldTiDBSetTemp.DeepCopy()
	if err = tmm.syncTidbClusterStatus(tc, oldTiDBSet); err != nil {
		return err
	}

	if tc.Spec.Paused {
		klog.V(4).Infof("tidb cluster %s/%s is paused, skip syncing for tidb statefulset", tc.GetNamespace(), tc.GetName())
		return nil
	}

	cm, err := tmm.syncTiDBConfigMap(tc, oldTiDBSet)
	if err != nil {
		return err
	}

	newTiDBSet := getNewTiDBSetForTidbCluster(tc, cm)
	if setNotExist {
		err = SetStatefulSetLastAppliedConfigAnnotation(newTiDBSet)
		if err != nil {
			return err
		}
		err = tmm.setControl.CreateStatefulSet(tc, newTiDBSet)
		if err != nil {
			return err
		}
		tc.Status.TiDB.StatefulSet = &apps.StatefulSetStatus{}
		return nil
	}

	if !templateEqual(newTiDBSet, oldTiDBSet) || tc.Status.TiDB.Phase == v1alpha1.UpgradePhase {
		if err := tmm.tidbUpgrader.Upgrade(tc, oldTiDBSet, newTiDBSet); err != nil {
			return err
		}
	}

	if tmm.autoFailover {
		if tmm.shouldRecover(tc) {
			tmm.tidbFailover.Recover(tc)
		} else if tc.TiDBAllPodsStarted() && !tc.TiDBAllMembersReady() {
			if err := tmm.tidbFailover.Failover(tc); err != nil {
				return err
			}
		}
	}

	return updateStatefulSet(tmm.setControl, tc, newTiDBSet, oldTiDBSet)
}

func (tmm *tidbMemberManager) shouldRecover(tc *v1alpha1.TidbCluster) bool {
	if tc.Status.TiDB.FailureMembers == nil {
		return false
	}
	// If all desired replicas (excluding failover pods) of tidb cluster are
	// healthy, we can perform our failover recovery operation.
	// Note that failover pods may fail (e.g. lack of resources) and we don't care
	// about them because we're going to delete them.
	for ordinal := range tc.TiDBStsDesiredOrdinals(true) {
		name := fmt.Sprintf("%s-%d", controller.TiDBMemberName(tc.GetName()), ordinal)
		pod, err := tmm.podLister.Pods(tc.Namespace).Get(name)
		if err != nil {
			klog.Errorf("pod %s/%s does not exist: %v", tc.Namespace, name, err)
			return false
		}
		if !podutil.IsPodReady(pod) {
			return false
		}
		status, ok := tc.Status.TiDB.Members[pod.Name]
		if !ok || !status.Health {
			return false
		}
	}
	return true
}

func (tmm *tidbMemberManager) syncTiDBService(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.Paused {
		klog.V(4).Infof("tidb cluster %s/%s is paused, skip syncing for tidb service", tc.GetNamespace(), tc.GetName())
		return nil
	}

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
		return fmt.Errorf("syncTiDBService: failed to get svc %s for cluster %s/%s, error: %s", newSvc.Name, ns, tc.GetName(), err)
	}
	oldSvc := oldSvcTmp.DeepCopy()
	util.RetainManagedFields(newSvc, oldSvc)

	equal, err := controller.ServiceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}
	annoEqual := util.IsSubMapOf(newSvc.Annotations, oldSvc.Annotations)
	isOrphan := metav1.GetControllerOf(oldSvc) == nil

	if !equal || !annoEqual || isOrphan {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		err = controller.SetServiceLastAppliedConfigAnnotation(&svc)
		if err != nil {
			return err
		}
		svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
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
	if set != nil && tc.BaseTiDBSpec().ConfigUpdateStrategy() == v1alpha1.ConfigUpdateStrategyInPlace {
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
	if tc.IsTLSClusterEnabled() {
		if config.Security == nil {
			config.Security = &v1alpha1.Security{}
		}
		config.Security.ClusterSSLCA = pointer.StringPtr(path.Join(clusterCertPath, tlsSecretRootCAKey))
		config.Security.ClusterSSLCert = pointer.StringPtr(path.Join(clusterCertPath, corev1.TLSCertKey))
		config.Security.ClusterSSLKey = pointer.StringPtr(path.Join(clusterCertPath, corev1.TLSPrivateKeyKey))
	}
	if tc.Spec.TiDB.IsTLSClientEnabled() {
		if config.Security == nil {
			config.Security = &v1alpha1.Security{}
		}
		config.Security.SSLCA = pointer.StringPtr(path.Join(serverCertPath, tlsSecretRootCAKey))
		config.Security.SSLCert = pointer.StringPtr(path.Join(serverCertPath, corev1.TLSCertKey))
		config.Security.SSLKey = pointer.StringPtr(path.Join(serverCertPath, corev1.TLSPrivateKeyKey))
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
		ClusterDomain:   tc.Spec.ClusterDomain,
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

	if tc.BaseTiDBSpec().ConfigUpdateStrategy() == v1alpha1.ConfigUpdateStrategyRollingUpdate {
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
	tidbSelector := label.New().Instance(instanceName).TiDB()
	svcName := controller.TiDBMemberName(tcName)
	tidbLabels := tidbSelector.Copy().UsedByEndUser().Labels()
	portName := "mysql-client"
	if svcSpec.PortName != nil {
		portName = *svcSpec.PortName
	}
	ports := []corev1.ServicePort{
		{
			Name:       portName,
			Port:       4000,
			TargetPort: intstr.FromInt(4000),
			Protocol:   corev1.ProtocolTCP,
			NodePort:   svcSpec.GetMySQLNodePort(),
		},
	}
	if svcSpec.ShouldExposeStatus() {
		ports = append(ports, corev1.ServicePort{
			Name:       "status",
			Port:       10080,
			TargetPort: intstr.FromInt(10080),
			Protocol:   corev1.ProtocolTCP,
			NodePort:   svcSpec.GetStatusNodePort(),
		})
	}

	tidbSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          tidbLabels,
			Annotations:     copyAnnotations(svcSpec.Annotations),
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			Type:     svcSpec.Type,
			Ports:    ports,
			Selector: tidbSelector.Labels(),
		},
	}
	if svcSpec.Type == corev1.ServiceTypeLoadBalancer {
		if svcSpec.LoadBalancerIP != nil {
			tidbSvc.Spec.LoadBalancerIP = *svcSpec.LoadBalancerIP
		}
		if svcSpec.LoadBalancerSourceRanges != nil {
			tidbSvc.Spec.LoadBalancerSourceRanges = svcSpec.LoadBalancerSourceRanges
		}
	}
	if svcSpec.ExternalTrafficPolicy != nil {
		tidbSvc.Spec.ExternalTrafficPolicy = *svcSpec.ExternalTrafficPolicy
	}
	if svcSpec.ClusterIP != nil {
		tidbSvc.Spec.ClusterIP = *svcSpec.ClusterIP
	}
	return tidbSvc
}

func getNewTiDBHeadlessServiceForTidbCluster(tc *v1alpha1.TidbCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	instanceName := tc.GetInstanceName()
	svcName := controller.TiDBPeerMemberName(tcName)
	tidbSelector := label.New().Instance(instanceName).TiDB()
	tidbLabel := tidbSelector.Copy().UsedByPeer().Labels()

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
			Selector:                 tidbSelector.Labels(),
			PublishNotReadyAddresses: true,
		},
	}
}

func getNewTiDBSetForTidbCluster(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) *apps.StatefulSet {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	headlessSvcName := controller.TiDBPeerMemberName(tcName)
	baseTiDBSpec := tc.BaseTiDBSpec()
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
	if tc.IsTLSClusterEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "tidb-tls", ReadOnly: true, MountPath: clusterCertPath,
		})
	}
	if tc.Spec.TiDB.IsTLSClientEnabled() {
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
	if tc.IsTLSClusterEnabled() {
		vols = append(vols, corev1.Volume{
			Name: "tidb-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tcName, label.TiDBLabelVal),
				},
			},
		})
	}
	if tc.Spec.TiDB.IsTLSClientEnabled() {
		secretName := tlsClientSecretName(tc)
		vols = append(vols, corev1.Volume{
			Name: "tidb-server-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		})
	}

	sysctls := "sysctl -w"
	var initContainers []corev1.Container
	if baseTiDBSpec.Annotations() != nil {
		init, ok := baseTiDBSpec.Annotations()[label.AnnSysctlInit]
		if ok && (init == label.AnnSysctlInitVal) {
			if baseTiDBSpec.PodSecurityContext() != nil && len(baseTiDBSpec.PodSecurityContext().Sysctls) > 0 {
				for _, sysctl := range baseTiDBSpec.PodSecurityContext().Sysctls {
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
	podSecurityContext := baseTiDBSpec.PodSecurityContext().DeepCopy()
	if len(initContainers) > 0 {
		podSecurityContext.Sysctls = []corev1.Sysctl{}
	}

	var containers []corev1.Container
	if tc.Spec.TiDB.ShouldSeparateSlowLog() {
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
			Resources:       controller.ContainerResource(tc.Spec.TiDB.GetSlowLogTailerSpec().ResourceRequirements),
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
	if tc.Spec.TiDB.ShouldSeparateSlowLog() {
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
			Value: strconv.FormatBool(tc.IsTiDBBinlogEnabled()),
		},
		{
			Name:  "SLOW_LOG_FILE",
			Value: slowLogFileEnvVal,
		},
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "HEADLESS_SERVICE_NAME",
			Value: headlessSvcName,
		},
	}

	c := corev1.Container{
		Name:            v1alpha1.TiDBMemberType.String(),
		Image:           tc.TiDBImage(),
		Command:         []string{"/bin/sh", "/usr/local/bin/tidb_start_script.sh"},
		ImagePullPolicy: baseTiDBSpec.ImagePullPolicy(),
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
		Env:          util.AppendEnv(envs, baseTiDBSpec.Env()),
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(4000),
				},
			},
			InitialDelaySeconds: int32(10),
		},
	}
	if tc.Spec.TiDB.Lifecycle != nil {
		c.Lifecycle = tc.Spec.TiDB.Lifecycle
	}

	containers = append(containers, c)

	podSpec := baseTiDBSpec.BuildPodSpec()
	podSpec.Containers = append(containers, baseTiDBSpec.AdditionalContainers()...)
	podSpec.Volumes = append(vols, baseTiDBSpec.AdditionalVolumes()...)
	podSpec.SecurityContext = podSecurityContext
	podSpec.InitContainers = initContainers

	if baseTiDBSpec.HostNetwork() {
		podSpec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
	}

	tidbLabel := label.New().Instance(instanceName).TiDB()
	podAnnotations := CombineAnnotations(controller.AnnProm(10080), baseTiDBSpec.Annotations())
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
			Replicas: pointer.Int32Ptr(tc.TiDBStsDesiredReplicas()),
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
				RollingUpdate: &apps.RollingUpdateStatefulSetStrategy{Partition: pointer.Int32Ptr(tc.TiDBStsDesiredReplicas())},
			},
		},
	}

	return tidbSet
}

func (tmm *tidbMemberManager) syncTidbClusterStatus(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	if set == nil {
		// skip if not created yet
		return nil
	}

	tc.Status.TiDB.StatefulSet = &set.Status

	upgrading, err := tmm.tidbStatefulSetIsUpgradingFn(tmm.podLister, set, tc)
	if err != nil {
		return err
	}
	if upgrading && tc.Status.TiKV.Phase != v1alpha1.UpgradePhase &&
		tc.Status.PD.Phase != v1alpha1.UpgradePhase && tc.Status.Pump.Phase != v1alpha1.UpgradePhase {
		tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
	} else if tc.TiDBStsDesiredReplicas() != *set.Spec.Replicas {
		tc.Status.TiDB.Phase = v1alpha1.ScalePhase
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
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("syncTidbClusterStatus: failed to get pods %s for cluster %s/%s, error: %s", name, tc.GetNamespace(), tc.GetName(), err)
		}
		if pod != nil && pod.Spec.NodeName != "" {
			// Update assiged node if pod exists and is scheduled
			newTidbMember.NodeName = pod.Spec.NodeName
		}
		tidbStatus[name] = newTidbMember
	}
	tc.Status.TiDB.Members = tidbStatus
	tc.Status.TiDB.Image = ""
	c := filterContainer(set, "tidb")
	if c != nil {
		tc.Status.TiDB.Image = c.Image
	}
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
		return false, fmt.Errorf("tidbStatefulSetIsUpgrading: failed to get pods for cluster %s/%s, selector %s, error: %s", tc.GetNamespace(), tc.GetInstanceName(), selector, err)
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

func tlsClientSecretName(tc *v1alpha1.TidbCluster) string {
	return fmt.Sprintf("%s-server-secret", controller.TiDBMemberName(tc.Name))
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
