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
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"k8s.io/utils/pointer"

	// for sql/driver
	_ "github.com/go-sql-driver/mysql"
)

const (
	defaultSlowLogVolume = "slowlog"
	defaultSlowLogDir    = "/var/log/tidb"
	defaultSlowLogFile   = defaultSlowLogDir + "/slowlog"
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
	deps                         *controller.Dependencies
	scaler                       Scaler
	tidbUpgrader                 Upgrader
	tidbFailover                 Failover
	tidbStatefulSetIsUpgradingFn func(corelisters.PodLister, *apps.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
}

// NewTiDBMemberManager returns a *tidbMemberManager
func NewTiDBMemberManager(deps *controller.Dependencies, scaler Scaler, tidbUpgrader Upgrader, tidbFailover Failover) manager.Manager {
	return &tidbMemberManager{
		deps:                         deps,
		scaler:                       scaler,
		tidbUpgrader:                 tidbUpgrader,
		tidbFailover:                 tidbFailover,
		tidbStatefulSetIsUpgradingFn: tidbStatefulSetIsUpgrading,
	}
}

func (m *tidbMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	// If tidb is not specified return
	if tc.Spec.TiDB == nil {
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Spec.TiKV != nil && !tc.TiKVIsAvailable() {
		return controller.RequeueErrorf("TidbCluster: [%s/%s], waiting for TiKV cluster running", ns, tcName)
	}

	if tc.Spec.Pump != nil && !tc.PumpIsAvailable() {
		return controller.RequeueErrorf("TidbCluster: [%s/%s], waiting for Pump cluster running", ns, tcName)
	}

	// Sync TiDB Headless Service
	if err := m.syncTiDBHeadlessServiceForTidbCluster(tc); err != nil {
		return err
	}

	// Sync TiDB Service before syncing TiDB StatefulSet
	if err := m.syncTiDBService(tc); err != nil {
		return err
	}

	if tc.Spec.TiDB.IsTLSClientEnabled() {
		if err := m.checkTLSClientCert(tc); err != nil {
			return err
		}
	}

	if tc.NeedToSyncTiDBInitializer() {
		m.syncInitializer(tc)
	}

	// Sync TiDB StatefulSet
	return m.syncTiDBStatefulSetForTidbCluster(tc)
}

func (m *tidbMemberManager) checkTLSClientCert(tc *v1alpha1.TidbCluster) error {
	ns := tc.Namespace
	secretName := tlsClientSecretName(tc)
	secret, err := m.deps.SecretLister.Secrets(ns).Get(secretName)
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

func (m *tidbMemberManager) syncTiDBHeadlessServiceForTidbCluster(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.Paused {
		klog.V(4).Infof("tidb cluster %s/%s is paused, skip syncing for tidb headless service", tc.GetNamespace(), tc.GetName())
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := getNewTiDBHeadlessServiceForTidbCluster(tc)
	oldSvcTmp, err := m.deps.ServiceLister.Services(ns).Get(controller.TiDBPeerMemberName(tcName))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(tc, newSvc)
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
		_, err = m.deps.ServiceControl.UpdateService(tc, &svc)
		return err
	}

	return nil
}

func (m *tidbMemberManager) syncTiDBStatefulSetForTidbCluster(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	oldTiDBSetTemp, err := m.deps.StatefulSetLister.StatefulSets(ns).Get(controller.TiDBMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncTiDBStatefulSetForTidbCluster: failed to get sts %s for cluster %s/%s, error: %s", controller.TiDBMemberName(tcName), ns, tcName, err)
	}
	setNotExist := errors.IsNotFound(err)

	oldTiDBSet := oldTiDBSetTemp.DeepCopy()
	if err = m.syncTidbClusterStatus(tc, oldTiDBSet); err != nil {
		return err
	}

	if tc.Spec.Paused {
		klog.V(4).Infof("tidb cluster %s/%s is paused, skip syncing for tidb statefulset", tc.GetNamespace(), tc.GetName())
		return nil
	}

	cm, err := m.syncTiDBConfigMap(tc, oldTiDBSet)
	if err != nil {
		return err
	}

	newTiDBSet, err := getNewTiDBSetForTidbCluster(tc, cm)
	if err != nil {
		return err
	}

	if setNotExist {
		err = mngerutils.SetStatefulSetLastAppliedConfigAnnotation(newTiDBSet)
		if err != nil {
			return err
		}
		err = m.deps.StatefulSetControl.CreateStatefulSet(tc, newTiDBSet)
		if err != nil {
			return err
		}
		tc.Status.TiDB.StatefulSet = &apps.StatefulSetStatus{}
		return nil
	}

	// Scaling takes precedence over upgrading because:
	// - if a pod fails in the upgrading, users may want to delete it or add
	//   new replicas
	// - it's ok to scale in the middle of upgrading (in statefulset controller
	//   scaling takes precedence over upgrading too)
	if err := m.scaler.Scale(tc, oldTiDBSet, newTiDBSet); err != nil {
		return err
	}

	if m.deps.CLIConfig.AutoFailover {
		if m.shouldRecover(tc) {
			m.tidbFailover.Recover(tc)
		} else if tc.TiDBAllPodsStarted() && !tc.TiDBAllMembersReady() {
			if err := m.tidbFailover.Failover(tc); err != nil {
				return err
			}
		}
	}

	if !templateEqual(newTiDBSet, oldTiDBSet) || tc.Status.TiDB.Phase == v1alpha1.UpgradePhase {
		if err := m.tidbUpgrader.Upgrade(tc, oldTiDBSet, newTiDBSet); err != nil {
			return err
		}
	}

	return mngerutils.UpdateStatefulSetWithPrecheck(m.deps, tc, "FailedUpdateTiDBSTS", newTiDBSet, oldTiDBSet)
}

func (m *tidbMemberManager) syncInitializer(tc *v1alpha1.TidbCluster) {
	// set random password
	ns := tc.Namespace
	tcName := tc.Name
	//check endpoints ready
	isTiDBReady := false
	eps, epErr := m.deps.EndpointLister.Endpoints(ns).Get(controller.TiDBMemberName(tcName))
	if epErr != nil {
		klog.Errorf("Failed to get endpoints %s for cluster %s/%s, err: %s", controller.TiDBMemberName(tcName), ns, tcName, epErr)
		return
	}
	// TiDB service has endpoints
	if eps != nil && len(eps.Subsets) > 0 && len(eps.Subsets[0].Addresses) > 0 {
		isTiDBReady = true
	}

	if !isTiDBReady {
		klog.Infof("Wait for TiDB ready for cluster %s/%s", ns, tcName)
		return
	}
	// sync password secret
	var password string
	secretName := controller.TiDBInitSecret(tcName)
	secret, err := m.deps.SecretLister.Secrets(ns).Get(secretName)
	passwordSecretExist := true
	if err != nil {
		if errors.IsNotFound(err) {
			passwordSecretExist = false
		} else {
			klog.Errorf("Failed to get secret %s for cluster %s/%s, err: %s", secretName, ns, tcName, epErr)
			return
		}
	}

	if !passwordSecretExist {
		klog.Infof("Create random password secret for cluster %s/%s", ns, tcName)
		var secret *corev1.Secret
		secret, password = m.BuildRandomPasswordSecret(tc)
		err := m.deps.TypedControl.Create(tc, secret)
		if err != nil {
			klog.Errorf("Failed to create secret %s for cluster %s:%s, err: %s", secretName, ns, tcName, err)
			return
		}
	} else {
		password = string(secret.Data[constants.TidbRootKey])
	}
	// init password
	var db *sql.DB
	dsn := util.GetDSN(tc, "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err = util.OpenDB(ctx, dsn)

	if err != nil {
		if strings.Contains(fmt.Sprint(err), "Access denied") {
			klog.Errorf("Can't connect to the TiDB service of the TiDB cluster [%s:%s], error: %s", ns, tcName, err)
			val := true
			tc.Status.TiDB.PasswordInitialized = &val
			return
		}
		if ctx.Err() != nil {
			klog.Errorf("Can't connect to the TiDB service of the TiDB cluster [%s:%s], error: %s, context error: %s", ns, tcName, err, ctx.Err())
		} else {
			klog.Errorf("Can't connect to the TiDB service of the TiDB cluster [%s:%s], error: %s", ns, tcName, err)
		}
		return
	} else {
		klog.Infof("Set random password for cluster %s/%s", ns, tcName)
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				klog.Errorf("Closed db connection for TiDB cluster %s/%s, err: %v", ns, tcName, err)
			}
		}(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = util.SetPassword(ctx, db, password)
		if err != nil {
			klog.Errorf("Fail to set TiDB password for TiDB cluster %s/%s, err: %s", ns, tcName, err)
			return
		}
		val := true
		tc.Status.TiDB.PasswordInitialized = &val
		klog.Infof("Set password successfully for TiDB cluster %s/%s", ns, tcName)
	}
}

func (m *tidbMemberManager) BuildRandomPasswordSecret(tc *v1alpha1.TidbCluster) (*corev1.Secret, string) {

	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            controller.TiDBInitSecret(tc.Name),
			Namespace:       tc.Namespace,
			Labels:          label.New().Instance(tc.Name).Labels(),
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
	}
	var password []byte
	for {
		password = util.FixedLengthRandomPasswordBytes()
		// check password not contain "\" character
		if !strings.Contains(string(password), "\\") {
			break
		}
	}
	s.Data = map[string][]byte{
		constants.TidbRootKey: password,
	}
	return s, string(password)
}

func (m *tidbMemberManager) shouldRecover(tc *v1alpha1.TidbCluster) bool {
	if tc.Status.TiDB.FailureMembers == nil {
		return false
	}
	// If all desired replicas (excluding failover pods) of tidb cluster are
	// healthy, we can perform our failover recovery operation.
	// Note that failover pods may fail (e.g. lack of resources) and we don't care
	// about them because we're going to delete them.
	for ordinal := range tc.TiDBStsDesiredOrdinals(true) {
		name := fmt.Sprintf("%s-%d", controller.TiDBMemberName(tc.GetName()), ordinal)
		pod, err := m.deps.PodLister.Pods(tc.Namespace).Get(name)
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

func (m *tidbMemberManager) syncTiDBService(tc *v1alpha1.TidbCluster) error {
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

	oldSvcTmp, err := m.deps.ServiceLister.Services(ns).Get(newSvc.Name)
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncTiDBService: failed to get svc %s for cluster %s/%s, error: %s", newSvc.Name, ns, tc.GetName(), err)
	}
	oldSvc := oldSvcTmp.DeepCopy()
	if newSvc.Annotations == nil {
		newSvc.Annotations = map[string]string{}
	}
	if oldSvc.Annotations == nil {
		oldSvc.Annotations = map[string]string{}
	}
	if newSvc.Labels == nil {
		newSvc.Labels = map[string]string{}
	}
	if oldSvc.Labels == nil {
		oldSvc.Labels = map[string]string{}
	}
	util.RetainManagedFields(newSvc, oldSvc)

	equal, err := controller.ServiceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}

	delete(oldSvc.Annotations, LastAppliedConfigAnnotation)
	annoEqual := equality.Semantic.DeepEqual(newSvc.Annotations, oldSvc.Annotations)
	labelEqual := equality.Semantic.DeepEqual(newSvc.Labels, oldSvc.Labels)
	isOrphan := metav1.GetControllerOf(oldSvc) == nil

	if equal && annoEqual && labelEqual && !isOrphan {
		return nil
	}

	klog.V(2).Infof("Sync TiDB service %s/%s, spec equal: %v, annotations equal: %v, label equal: %v", newSvc.Namespace, newSvc.Name, equal, annoEqual, labelEqual)

	svc := *oldSvc
	svc.Annotations = newSvc.Annotations
	svc.Labels = newSvc.Labels
	svc.Spec = newSvc.Spec
	err = controller.SetServiceLastAppliedConfigAnnotation(&svc)
	if err != nil {
		return err
	}
	svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
	// also override labels when adopt orphan
	if isOrphan {
		svc.OwnerReferences = newSvc.OwnerReferences
	}

	_, err = m.deps.ServiceControl.UpdateService(tc, &svc)
	return err
}

// syncTiDBConfigMap syncs the configmap of tidb
func (m *tidbMemberManager) syncTiDBConfigMap(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) (*corev1.ConfigMap, error) {

	// For backward compatibility, only sync tidb configmap when .tidb.config is non-nil
	if tc.Spec.TiDB.Config == nil {
		return nil, nil
	}
	newCm, err := getTiDBConfigMap(tc)
	if err != nil {
		return nil, err
	}

	var inUseName string
	if set != nil {
		inUseName = mngerutils.FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.TiDBMemberName(tc.Name))
		})
	}

	klog.V(3).Info("get tidb in use config map name: ", inUseName)

	err = mngerutils.UpdateConfigMapIfNeed(m.deps.ConfigMapLister, tc.BaseTiDBSpec().ConfigUpdateStrategy(), inUseName, newCm)
	if err != nil {
		return nil, err
	}
	return m.deps.TypedControl.CreateOrUpdateConfigMap(tc, newCm)
}

func getTiDBConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	if tc.Spec.TiDB.Config == nil {
		return nil, nil
	}
	config := tc.Spec.TiDB.Config.DeepCopy()

	// override CA if tls enabled
	if tc.IsTLSClusterEnabled() {
		config.Set("security.cluster-ssl-ca", path.Join(clusterCertPath, tlsSecretRootCAKey))
		config.Set("security.cluster-ssl-cert", path.Join(clusterCertPath, corev1.TLSCertKey))
		config.Set("security.cluster-ssl-key", path.Join(clusterCertPath, corev1.TLSPrivateKeyKey))
	}
	if tc.Spec.TiDB.IsTLSClientEnabled() {
		// No need to configure the ssl-ca parameter when client authentication is disabled.
		if !tc.Spec.TiDB.TLSClient.DisableClientAuthn {
			config.Set("security.ssl-ca", path.Join(serverCertPath, tlsSecretRootCAKey))
		}
		config.Set("security.ssl-cert", path.Join(serverCertPath, corev1.TLSCertKey))
		config.Set("security.ssl-key", path.Join(serverCertPath, corev1.TLSPrivateKeyKey))
	}
	confText, err := config.MarshalTOML()
	if err != nil {
		return nil, err
	}

	plugins := tc.Spec.TiDB.Plugins
	tidbStartScriptModel := &TidbStartScriptModel{
		CommonModel: CommonModel{
			AcrossK8s:     tc.AcrossK8s(),
			ClusterDomain: tc.Spec.ClusterDomain,
		},
		EnablePlugin:    len(plugins) > 0,
		PluginDirectory: "/plugins",
		PluginList:      strings.Join(plugins, ","),
	}

	tidbStartScriptModel.Path = "${CLUSTER_NAME}-pd:2379"
	if tc.AcrossK8s() {
		tidbStartScriptModel.Path = "${CLUSTER_NAME}-pd:2379" // get pd addr from discovery in startup script
	} else if tc.Heterogeneous() && tc.WithoutLocalPD() {
		tidbStartScriptModel.Path = controller.PDMemberName(tc.Spec.Cluster.Name) + ":2379" // use pd of reference cluster
	}

	startScript, err := RenderTiDBStartScript(tidbStartScriptModel)
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
	tidbLabels := util.CombineStringMap(tidbSelector.Copy().UsedByEndUser().Labels(), svcSpec.Labels)
	ports := []corev1.ServicePort{
		{
			Name:       svcSpec.GetPortName(),
			Port:       tc.Spec.TiDB.GetServicePort(),
			TargetPort: intstr.FromInt(4000),
			Protocol:   corev1.ProtocolTCP,
			NodePort:   svcSpec.GetMySQLNodePort(),
		},
	}
	ports = append(ports, tc.Spec.TiDB.Service.AdditionalPorts...)
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
			Annotations:     util.CopyStringMap(svcSpec.Annotations),
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

func getNewTiDBSetForTidbCluster(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	setName := controller.TiDBMemberName(tcName)
	headlessSvcName := controller.TiDBPeerMemberName(tcName)
	baseTiDBSpec := tc.BaseTiDBSpec()
	instanceName := tc.GetInstanceName()
	tidbConfigMap := controller.MemberConfigMapName(tc, v1alpha1.TiDBMemberType)
	if cm != nil {
		tidbConfigMap = cm.Name
	}

	annoMount, annoVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annoMount,
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
		annoVolume,
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
					// Init container resourceRequirements should be equal to app container.
					// Scheduling is done based on effective requests/limits,
					// which means init containers can reserve resources for
					// initialization that are not used during the life of the Pod.
					// ref:https://kubernetes.io/docs/concepts/workloads/pods/init-containers/#resources
					Resources: controller.ContainerResource(tc.Spec.TiDB.ResourceRequirements),
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

	// handle StorageVolumes and AdditionalVolumeMounts in ComponentSpec
	storageVolMounts, additionalPVCs := util.BuildStorageVolumeAndVolumeMount(tc.Spec.TiDB.StorageVolumes, tc.Spec.TiDB.StorageClassName, v1alpha1.TiDBMemberType)
	volMounts = append(volMounts, storageVolMounts...)
	volMounts = append(volMounts, tc.Spec.TiDB.AdditionalVolumeMounts...)

	var containers []corev1.Container
	slowLogFileEnvVal := ""
	if tc.Spec.TiDB.ShouldSeparateSlowLog() {
		// mount a shared volume and tail the slow log to STDOUT using a sidecar.
		var slowQueryLogVolumeMount corev1.VolumeMount
		slowQueryLogVolumeName := tc.Spec.TiDB.SlowLogVolumeName
		if slowQueryLogVolumeName == "" {
			vols = append(vols, corev1.Volume{
				Name: defaultSlowLogVolume,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			})
			slowQueryLogVolumeMount = corev1.VolumeMount{Name: defaultSlowLogVolume, MountPath: defaultSlowLogDir}
			volMounts = append(volMounts, slowQueryLogVolumeMount)
			slowLogFileEnvVal = defaultSlowLogFile
		} else {
			existVolume := false
			for _, volMount := range storageVolMounts {
				volMountName := fmt.Sprintf("%s-%s", v1alpha1.TiDBMemberType.String(), slowQueryLogVolumeName)
				if volMount.Name == volMountName {
					slowQueryLogVolumeMount = volMount
					existVolume = true
					break
				}
			}
			if !existVolume {
				for _, volMount := range tc.Spec.TiDB.AdditionalVolumeMounts {
					if volMount.Name == slowQueryLogVolumeName {
						slowQueryLogVolumeMount = volMount
						existVolume = true
						break
					}
				}
			}
			if !existVolume {
				return nil, fmt.Errorf("failed to get slowLogVolume %s for cluster %s/%s", slowQueryLogVolumeName, ns, tcName)
			}
			slowLogFileEnvVal = path.Join(slowQueryLogVolumeMount.MountPath, slowQueryLogVolumeName)
		}
		containers = append(containers, corev1.Container{
			Name:            v1alpha1.SlowLogTailerMemberType.String(),
			Image:           tc.HelperImage(),
			ImagePullPolicy: tc.HelperImagePullPolicy(),
			Resources:       controller.ContainerResource(tc.Spec.TiDB.GetSlowLogTailerSpec().ResourceRequirements),
			VolumeMounts:    []corev1.VolumeMount{slowQueryLogVolumeMount},
			Command: []string{
				"sh",
				"-c",
				fmt.Sprintf("touch %s; tail -n0 -F %s;", slowLogFileEnvVal, slowLogFileEnvVal),
			},
		})
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
		EnvFrom:      baseTiDBSpec.EnvFrom(),
		ReadinessProbe: &corev1.Probe{
			Handler:             buildTiDBReadinessProbHandler(tc),
			InitialDelaySeconds: int32(10),
		},
	}
	if tc.Spec.TiDB.Lifecycle != nil {
		c.Lifecycle = tc.Spec.TiDB.Lifecycle
	}
	if tc.Spec.TiDB.ReadinessProbe != nil {
		if tc.Spec.TiDB.ReadinessProbe.InitialDelaySeconds != nil {
			c.ReadinessProbe.InitialDelaySeconds = *tc.Spec.TiDB.ReadinessProbe.InitialDelaySeconds
		}
		if tc.Spec.TiDB.ReadinessProbe.PeriodSeconds != nil {
			c.ReadinessProbe.PeriodSeconds = *tc.Spec.TiDB.ReadinessProbe.PeriodSeconds
		}
	}

	containers = append(containers, c)

	podSpec := baseTiDBSpec.BuildPodSpec()
	podSpec.Containers = append(containers, baseTiDBSpec.AdditionalContainers()...)
	podSpec.Volumes = append(vols, baseTiDBSpec.AdditionalVolumes()...)
	podSpec.SecurityContext = podSecurityContext
	podSpec.InitContainers = append(initContainers, baseTiDBSpec.InitContainers()...)
	podSpec.ServiceAccountName = tc.Spec.TiDB.ServiceAccount
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = tc.Spec.ServiceAccount
	}

	stsLabels := label.New().Instance(instanceName).TiDB()
	podLabels := util.CombineStringMap(stsLabels, baseTiDBSpec.Labels())
	podAnnotations := util.CombineStringMap(controller.AnnProm(10080), baseTiDBSpec.Annotations())
	stsAnnotations := getStsAnnotations(tc.Annotations, label.TiDBLabelVal)

	deleteSlotsNumber, err := util.GetDeleteSlotsNumber(stsAnnotations)
	if err != nil {
		return nil, fmt.Errorf("get delete slots number of statefulset %s/%s failed, err:%v", ns, setName, err)
	}

	updateStrategy := apps.StatefulSetUpdateStrategy{}
	if baseTiDBSpec.StatefulSetUpdateStrategy() == apps.OnDeleteStatefulSetStrategyType {
		updateStrategy.Type = apps.OnDeleteStatefulSetStrategyType
	} else {
		updateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType
		updateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{
			Partition: pointer.Int32Ptr(tc.TiDBStsDesiredReplicas() + deleteSlotsNumber),
		}
	}

	tidbSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          stsLabels.Labels(),
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(tc.TiDBStsDesiredReplicas()),
			Selector: stsLabels.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			ServiceName:         controller.TiDBPeerMemberName(tcName),
			PodManagementPolicy: baseTiDBSpec.PodManagementPolicy(),
			UpdateStrategy:      updateStrategy,
		},
	}

	tidbSet.Spec.VolumeClaimTemplates = append(tidbSet.Spec.VolumeClaimTemplates, additionalPVCs...)
	return tidbSet, nil
}

func (m *tidbMemberManager) syncTidbClusterStatus(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	if set == nil {
		// skip if not created yet
		return nil
	}

	tc.Status.TiDB.StatefulSet = &set.Status

	upgrading, err := m.tidbStatefulSetIsUpgradingFn(m.deps.PodLister, set, tc)
	if err != nil {
		return err
	}

	if tc.TiDBStsDesiredReplicas() != *set.Spec.Replicas {
		tc.Status.TiDB.Phase = v1alpha1.ScalePhase
	} else if upgrading && tc.Status.TiKV.Phase != v1alpha1.UpgradePhase &&
		tc.Status.PD.Phase != v1alpha1.UpgradePhase && tc.Status.Pump.Phase != v1alpha1.UpgradePhase {
		tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.TiDB.Phase = v1alpha1.NormalPhase
	}

	tidbStatus := map[string]v1alpha1.TiDBMember{}
	for id := range helper.GetPodOrdinals(tc.Status.TiDB.StatefulSet.Replicas, set) {
		name := fmt.Sprintf("%s-%d", controller.TiDBMemberName(tc.GetName()), id)
		health, err := m.deps.TiDBControl.GetHealth(tc, int32(id))
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
		pod, err := m.deps.PodLister.Pods(tc.GetNamespace()).Get(name)
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("syncTidbClusterStatus: failed to get pods %s for cluster %s/%s, error: %s", name, tc.GetNamespace(), tc.GetName(), err)
		}
		if pod != nil && pod.Spec.NodeName != "" {
			// Update assigned node if pod exists and is scheduled
			newTidbMember.NodeName = pod.Spec.NodeName
		}
		tidbStatus[name] = newTidbMember
	}

	tc.Status.TiDB.Members = tidbStatus
	tc.Status.TiDB.Image = ""
	c := findContainerByName(set, "tidb")
	if c != nil {
		tc.Status.TiDB.Image = c.Image
	}
	return nil
}

func tidbStatefulSetIsUpgrading(podLister corelisters.PodLister, set *apps.StatefulSet, tc *v1alpha1.TidbCluster) (bool, error) {
	if mngerutils.StatefulSetIsUpgrading(set) {
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

func buildTiDBReadinessProbHandler(tc *v1alpha1.TidbCluster) corev1.Handler {
	if tc.Spec.TiDB.ReadinessProbe != nil {
		if tp := tc.Spec.TiDB.ReadinessProbe.Type; tp != nil {
			if *tp == v1alpha1.CommandProbeType {
				command := buildTiDBProbeCommand(tc)
				return corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: command,
					},
				}
			}
		}
	}

	// fall to default case v1alpha1.TCPProbeType
	return corev1.Handler{
		TCPSocket: &corev1.TCPSocketAction{
			Port: intstr.FromInt(4000),
		},
	}
}

func buildTiDBProbeCommand(tc *v1alpha1.TidbCluster) (command []string) {
	host := "127.0.0.1"

	readinessURL := fmt.Sprintf("%s://%s:10080/status", tc.Scheme(), host)
	command = append(command, "curl")
	command = append(command, readinessURL)

	// Fail silently (no output at all) on server errors
	// without this if the server return 500, the exist code will be 0
	// and probe is success.
	command = append(command, "--fail")
	// follow 301 or 302 redirect
	command = append(command, "--location")

	if tc.IsTLSClusterEnabled() {
		cacert := path.Join(clusterCertPath, tlsSecretRootCAKey)
		cert := path.Join(clusterCertPath, corev1.TLSCertKey)
		key := path.Join(clusterCertPath, corev1.TLSPrivateKeyKey)
		command = append(command, "--cacert", cacert)
		command = append(command, "--cert", cert)
		command = append(command, "--key", key)
	}
	return
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

func (m *FakeTiDBMemberManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeTiDBMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if m.err != nil {
		return m.err
	}
	if len(tc.Status.TiDB.Members) != 0 {
		// simulate status update
		tc.Status.ClusterID = string(uuid.NewUUID())
	}
	return nil
}
