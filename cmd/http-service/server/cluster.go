// Copyright 2023 PingCAP, Inc.
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

package server

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/log"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"

	"github.com/pingcap/tidb-operator/http-service/kube"
	"github.com/pingcap/tidb-operator/http-service/pbgen/api"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

type ClusterStatus string

const (
	// the cluster is still being created
	ClusterStatusCreating ClusterStatus = "creating"
	// all components are running
	ClusterStatusRunning ClusterStatus = "running"
	// some components are deleting
	ClusterStatusDeleting ClusterStatus = "deleting"
	// some components are scaling
	ClusterStatusScaling ClusterStatus = "scaling"
	// some components are upgrading
	ClusterStatusUpgrading ClusterStatus = "upgrading"
	// some components are unavailable
	ClusterStatusUnavailable ClusterStatus = "unavailable"

	helperImage = "busybox:1.36"

	// try `kanshiori/mysqlclient-arm64` for ARM64
	tidbInitializerImage          = "tnir/mysqlclient"
	tidbInitializerName           = "tidb-initializer"
	tidbInitializerPasswordSecret = "tidb-secret"

	tidbMonitorName           = "tidb-monitor"
	prometheusGrafanaLogLevel = "info"
	prometheusBaseImage       = "prom/prometheus"
	grafanaBaseImage          = "grafana/grafana"
	reloaderBaseImage         = "pingcap/tidb-monitor-reloader"
	reloaderVersion           = "v1.0.1"
	tmInitializerBase         = "pingcap/tidb-monitor-initializer"
)

type ClusterServer struct {
	api.UnimplementedClusterServer

	KubeClient *kube.KubeClient
}

func (s *ClusterServer) CreateCluster(ctx context.Context, req *api.CreateClusterReq) (*api.CreateClusterResp, error) {
	k8sID := getKubernetesID(ctx)
	opCli := s.KubeClient.GetOperatorClient(k8sID)
	kubeCli := s.KubeClient.GetKubeClient(k8sID)
	logger := log.L().With(zap.String("request", "CreateCluster"), zap.String("k8sID", k8sID), zap.String("clusterID", req.ClusterId))
	if opCli == nil || kubeCli == nil {
		logger.Error("K8s client not found")
		message := fmt.Sprintf("no %s is specified in the request header or the kubeconfig context not exists", HeaderKeyKubernetesID)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.CreateClusterResp{Success: false, Message: &message}, nil
	}

	// create namespace but ignore the error if it already exists
	if _, err := kubeCli.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.ClusterId,
		},
	}, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		logger.Error("Create namespace failed", zap.Error(err))
		message := fmt.Sprintf("create namespace failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.CreateClusterResp{Success: false, Message: &message}, nil
	}

	// TODO(http-service): add verification for the request body
	// TODO(http-service): customize image support

	tc, err := assembleTidbCluster(req)
	if err != nil {
		logger.Error("Assemble TidbCluster CR failed", zap.Error(err))
		message := fmt.Sprintf("assemble TidbCluster CR failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.CreateClusterResp{Success: false, Message: &message}, nil
	}

	ti, tiSecret, err := assembleTidbInitializer(req)
	if err != nil {
		logger.Error("Assemble TidbInitializer CR failed", zap.Error(err))
		message := fmt.Sprintf("assemble TidbInitializer CR failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.CreateClusterResp{Success: false, Message: &message}, nil
	}

	tm, err := assembleTidbMonitor(req)
	if err != nil {
		logger.Error("Assemble TidbMonitor CR failed", zap.Error(err))
		message := fmt.Sprintf("assemble TidbMonitor CR failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.CreateClusterResp{Success: false, Message: &message}, nil
	}

	if _, err = opCli.PingcapV1alpha1().TidbClusters(req.ClusterId).Create(ctx, tc, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if ti == nil && tm == nil {
				// if TidbCluster already exists but no TidbInitializer & TidbMonitor is specified, return conflict error
				logger.Error("TidbCluster already exists", zap.Error(err))
				setResponseStatusCodes(ctx, http.StatusConflict)
				message := fmt.Sprintf("TidbCluster %s already exists", req.ClusterId)
				return &api.CreateClusterResp{Success: false, Message: &message}, nil
			} else {
				// if TidbCluster already exists and TidbInitializer/TidbMonitor is specified
				// ignore the error so that the TidbInitializer/TidbMonitor can be created (when re-requesting)
				logger.Warn("TidbCluster already exists, but still need to create TidbInitializer/TidbMonitor, ignore the error", zap.Error(err))
			}
		} else {
			logger.Error("Create TidbCluster failed", zap.Error(err))
			message := fmt.Sprintf("create TidbCluster failed: %s", err.Error())
			setResponseStatusCodes(ctx, http.StatusInternalServerError)
			return &api.CreateClusterResp{Success: false, Message: &message}, nil
		}
	}

	if ti != nil {
		// create or update the secret before creating the TidbInitializer
		if _, err = kubeCli.CoreV1().Secrets(req.ClusterId).Create(ctx, tiSecret, metav1.CreateOptions{}); err != nil {
			if apierrors.IsAlreadyExists(err) {
				if _, err = kubeCli.CoreV1().Secrets(req.ClusterId).Update(ctx, tiSecret, metav1.UpdateOptions{}); err != nil {
					logger.Error("Update username and password secret failed", zap.Error(err))
					message := fmt.Sprintf("update username and password secret failed: %s", err.Error())
					setResponseStatusCodes(ctx, http.StatusInternalServerError)
					return &api.CreateClusterResp{Success: false, Message: &message}, nil
				}
			} else {
				logger.Error("Create username and password secret failed", zap.Error(err))
				message := fmt.Sprintf("create username and password secret failed: %s", err.Error())
				setResponseStatusCodes(ctx, http.StatusInternalServerError)
				return &api.CreateClusterResp{Success: false, Message: &message}, nil
			}
		}

		if _, err = opCli.PingcapV1alpha1().TidbInitializers(req.ClusterId).Create(ctx, ti, metav1.CreateOptions{}); err != nil {
			if apierrors.IsAlreadyExists(err) {
				if tm == nil {
					// if TidbInitializer already exists but no TidbMonitor is specified, return conflict error
					logger.Error("TidbInitializer already exists", zap.Error(err))
					setResponseStatusCodes(ctx, http.StatusConflict)
					message := fmt.Sprintf("TidbInitializer %s already exists", req.ClusterId)
					return &api.CreateClusterResp{Success: false, Message: &message}, nil
				} else {
					// if TidbInitializer already exists and TidbMonitor is specified
					// ignore the error so that the TidbMonitor can be created (when re-requesting)
					logger.Warn("TidbInitializer already exists, but still need to create TidbMonitor, ignore the error", zap.Error(err))
				}
			} else {
				logger.Error("Create TidbInitializer failed", zap.Error(err))
				message := fmt.Sprintf("create TidbInitializer failed: %s", err.Error())
				setResponseStatusCodes(ctx, http.StatusInternalServerError)
				return &api.CreateClusterResp{Success: false, Message: &message}, nil
			}
		}
	}

	if tm != nil {
		if _, err = opCli.PingcapV1alpha1().TidbMonitors(req.ClusterId).Create(ctx, tm, metav1.CreateOptions{}); err != nil {
			if apierrors.IsAlreadyExists(err) {
				logger.Error("TidbMonitor already exists", zap.Error(err))
				setResponseStatusCodes(ctx, http.StatusConflict)
				message := fmt.Sprintf("TidbMonitor %s already exists", req.ClusterId)
				return &api.CreateClusterResp{Success: false, Message: &message}, nil
			} else {
				logger.Error("Create TidbMonitor failed", zap.Error(err))
				message := fmt.Sprintf("create TidbMonitor failed: %s", err.Error())
				setResponseStatusCodes(ctx, http.StatusInternalServerError)
				return &api.CreateClusterResp{Success: false, Message: &message}, nil
			}
		}
	}

	return &api.CreateClusterResp{Success: true}, nil
}

func (s *ClusterServer) PauseCluster(ctx context.Context, req *api.PauseClusterReq) (*api.PauseClusterResp, error) {
	k8sID := getKubernetesID(ctx)
	opCli := s.KubeClient.GetOperatorClient(k8sID)
	kubeCli := s.KubeClient.GetKubeClient(k8sID)
	logger := log.L().With(zap.String("request", "PauseCluster"), zap.String("k8sID", k8sID), zap.String("clusterID", req.ClusterId))

	if opCli == nil || kubeCli == nil {
		logger.Error("K8s client not found")
		message := fmt.Sprintf("no %s is specified in the request header or the kubeconfig context not exists", HeaderKeyKubernetesID)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.PauseClusterResp{Success: false, Message: &message}, nil
	}

	ns := req.ClusterId
	name := tidbClusterName

	tc, err := opCli.PingcapV1alpha1().TidbClusters(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Error("TidbCluster not found", zap.Error(err))
			setResponseStatusCodes(ctx, http.StatusNotFound)
			message := fmt.Sprintf("TidbCluster %s not found", req.ClusterId)
			return &api.PauseClusterResp{Success: false, Message: &message}, nil
		} else {
			logger.Error("Get TidbCluster failed", zap.Error(err))
			message := fmt.Sprintf("get TidbCluster failed: %s", err.Error())
			setResponseStatusCodes(ctx, http.StatusInternalServerError)
			return &api.PauseClusterResp{Success: false, Message: &message}, nil
		}
	}

	tc.Spec.SuspendAction = &v1alpha1.SuspendAction{SuspendStatefulSet: true}

	_, err = opCli.PingcapV1alpha1().TidbClusters(ns).Update(ctx, tc, metav1.UpdateOptions{})
	if err != nil {
		logger.Error("Pause TidbCluster failed", zap.Error(err))
		message := fmt.Sprintf("update TidbCluster failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.PauseClusterResp{Success: false, Message: &message}, nil
	}

	return &api.PauseClusterResp{Success: true}, nil
}

func (s *ClusterServer) ResumeCluster(ctx context.Context, req *api.ResumeClusterReq) (*api.ResumeClusterResp, error) {
	k8sID := getKubernetesID(ctx)
	opCli := s.KubeClient.GetOperatorClient(k8sID)
	kubeCli := s.KubeClient.GetKubeClient(k8sID)
	logger := log.L().With(zap.String("request", "ResumeCluster"), zap.String("k8sID", k8sID), zap.String("clusterID", req.ClusterId))

	if opCli == nil || kubeCli == nil {
		logger.Error("K8s client not found")
		message := fmt.Sprintf("no %s is specified in the request header or the kubeconfig context not exists", HeaderKeyKubernetesID)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.ResumeClusterResp{Success: false, Message: &message}, nil
	}

	ns := req.ClusterId
	name := tidbClusterName

	tc, err := opCli.PingcapV1alpha1().TidbClusters(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Error("TidbCluster not found", zap.Error(err))
			setResponseStatusCodes(ctx, http.StatusNotFound)
			message := fmt.Sprintf("TidbCluster %s not found", req.ClusterId)
			return &api.ResumeClusterResp{Success: false, Message: &message}, nil
		} else {
			logger.Error("Get TidbCluster failed", zap.Error(err))
			message := fmt.Sprintf("get TidbCluster failed: %s", err.Error())
			setResponseStatusCodes(ctx, http.StatusInternalServerError)
			return &api.ResumeClusterResp{Success: false, Message: &message}, nil
		}
	}

	tc.Spec.SuspendAction = &v1alpha1.SuspendAction{SuspendStatefulSet: false}

	_, err = opCli.PingcapV1alpha1().TidbClusters(ns).Update(ctx, tc, metav1.UpdateOptions{})
	if err != nil {
		logger.Error("Resume TidbCluster failed", zap.Error(err))
		message := fmt.Sprintf("update TidbCluster failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.ResumeClusterResp{Success: false, Message: &message}, nil
	}

	return &api.ResumeClusterResp{Success: true}, nil
}

func assembleTidbCluster(req *api.CreateClusterReq) (*v1alpha1.TidbCluster, error) {
	pdRes, tikvRes, tidbRes, tiflashRes, err := convertClusterComponentsResources(req)
	if err != nil {
		return nil, errors.New("invalid resource requirements")
	}

	pdCfg, tikvCfg, tidbCfg, tiflashCfg := convertClusterComponentsConfig(req)
	tidbPort := int32(4000)
	if req.Tidb.Port != nil {
		tidbPort = int32(*req.Tidb.Port)
	}

	deletePVP := corev1.PersistentVolumeReclaimDelete
	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.ClusterId,
			Name:      tidbClusterName,
		},
		Spec: v1alpha1.TidbClusterSpec{
			Version:              req.Version,
			PVReclaimPolicy:      &deletePVP,
			ConfigUpdateStrategy: v1alpha1.ConfigUpdateStrategyRollingUpdate,
			Helper: &v1alpha1.HelperSpec{
				Image: pointer.StringPtr(helperImage),
			},
			TopologySpreadConstraints: []v1alpha1.TopologySpreadConstraint{{
				TopologyKey: corev1.LabelHostname, // NOTE: `kubernetes.io/hostname`, add more scheduler policies if needed
			}},
			PD: &v1alpha1.PDSpec{
				Replicas:             int32(*req.Pd.Replicas),
				MaxFailoverCount:     pointer.Int32Ptr(0),
				ResourceRequirements: pdRes,
				Config:               pdCfg,
			},
			TiKV: &v1alpha1.TiKVSpec{
				Replicas:             int32(*req.Tikv.Replicas),
				MaxFailoverCount:     pointer.Int32Ptr(0),
				ResourceRequirements: tikvRes,
				Config:               tikvCfg,
			},
			TiDB: &v1alpha1.TiDBSpec{
				Replicas:             int32(*req.Tidb.Replicas),
				MaxFailoverCount:     pointer.Int32Ptr(0),
				ResourceRequirements: tidbRes,
				Config:               tidbCfg,
				Service: &v1alpha1.TiDBServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort, // NOTE: use NodePort for now
						Port: pointer.Int32Ptr(tidbPort),
					},
				},
			},
		},
	}

	if req.Tiflash != nil && req.Tiflash.Replicas != nil && *req.Tiflash.Replicas > 0 && req.Tiflash.Resource != nil {
		tc.Spec.TiFlash = &v1alpha1.TiFlashSpec{
			Replicas:             int32(*req.Tiflash.Replicas),
			MaxFailoverCount:     pointer.Int32Ptr(0),
			ResourceRequirements: tiflashRes,
			Config:               tiflashCfg,
			StorageClaims: []v1alpha1.StorageClaim{
				{
					Resources: tiflashRes,
				},
			},
		}
	}

	return tc, nil
}

func assembleTidbInitializer(req *api.CreateClusterReq) (*v1alpha1.TidbInitializer, *corev1.Secret, error) {
	if req.User == nil || (req.User.Username == "" && req.User.Password == "") {
		return nil, nil, nil // no need to init user
	} else if req.User.Username == "" || req.User.Password == "" {
		return nil, nil, errors.New("username and password must be specified at the same time")
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.ClusterId,
			Name:      tidbInitializerPasswordSecret,
		},
		StringData: map[string]string{
			req.User.Username: req.User.Password,
		},
	}

	ti := &v1alpha1.TidbInitializer{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.ClusterId,
			Name:      tidbInitializerName,
		},
		Spec: v1alpha1.TidbInitializerSpec{
			Image: tidbInitializerImage,
			Clusters: v1alpha1.TidbClusterRef{
				Name: tidbClusterName,
			},
			PasswordSecret: pointer.StringPtr(tidbInitializerPasswordSecret),
		},
	}

	return ti, secret, nil
}

func assembleTidbMonitor(req *api.CreateClusterReq) (*v1alpha1.TidbMonitor, error) {
	if req.Prometheus == nil && req.Grafana == nil {
		return nil, nil // no need to create monitor
	} else if req.Prometheus == nil {
		return nil, errors.New("prometheus must be specified if grafana is specified")
	}

	promRes, grafanaRes, err := convertMonitorComponentsResources(req)
	if err != nil {
		return nil, errors.New("invalid resource requirements")
	}
	grafanaPort := int32(3000)
	if req.Grafana.Port != nil {
		grafanaPort = int32(*req.Grafana.Port)
	}

	// TODO(http-service): persistent support
	deletePVP := corev1.PersistentVolumeReclaimDelete
	tm := &v1alpha1.TidbMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.ClusterId,
			Name:      tidbMonitorName,
		},
		Spec: v1alpha1.TidbMonitorSpec{
			PVReclaimPolicy: &deletePVP,
			Clusters: []v1alpha1.TidbClusterRef{{
				Name: tidbClusterName,
			}},
			Prometheus: v1alpha1.PrometheusSpec{
				LogLevel: prometheusGrafanaLogLevel,
				Service: v1alpha1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP, // ClusterIP for Prometheus now
				},
				MonitorContainer: v1alpha1.MonitorContainer{
					Version:              req.Prometheus.Version,
					BaseImage:            prometheusBaseImage,
					ResourceRequirements: promRes,
				},
			},
			Reloader: v1alpha1.ReloaderSpec{
				Service: v1alpha1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
				},
				MonitorContainer: v1alpha1.MonitorContainer{
					Version:              reloaderVersion,
					BaseImage:            reloaderBaseImage,
					ResourceRequirements: corev1.ResourceRequirements{},
				},
			},
			Initializer: v1alpha1.InitializerSpec{
				MonitorContainer: v1alpha1.MonitorContainer{
					Version:              req.Version, // NOTE: use the same version as TidbCluster
					BaseImage:            tmInitializerBase,
					ResourceRequirements: corev1.ResourceRequirements{},
				},
			},
		},
	}

	if len(req.Prometheus.Config) > 0 {
		options := make([]string, 0, len(req.Prometheus.Config))
		for k, v := range req.Prometheus.Config {
			options = append(options, fmt.Sprintf("--%s=%s", k, v))
		}
		tm.Spec.Prometheus.Config = &v1alpha1.PrometheusConfiguration{
			CommandOptions: options,
		}
	}

	if req.Grafana != nil {
		tm.Spec.Grafana = &v1alpha1.GrafanaSpec{
			LogLevel: prometheusGrafanaLogLevel,
			Service: v1alpha1.ServiceSpec{
				Type: corev1.ServiceTypeNodePort, // NOTE: use NodePort for now
				Port: pointer.Int32Ptr(grafanaPort),
			},
			MonitorContainer: v1alpha1.MonitorContainer{
				Version:              req.Grafana.Version,
				BaseImage:            grafanaBaseImage,
				ResourceRequirements: grafanaRes,
			},
			Envs: req.Grafana.Config,
		}
	}

	return tm, nil
}

func convertClusterComponentsResources(req *api.CreateClusterReq) (pdRes, tikvRes, tidbRes, tiflash corev1.ResourceRequirements, err error) {
	if req.Pd == nil || req.Tikv == nil || req.Tidb == nil ||
		req.Pd.Replicas == nil || req.Tikv.Replicas == nil || req.Tidb.Replicas == nil ||
		*req.Pd.Replicas <= 0 || *req.Tikv.Replicas <= 0 || *req.Tidb.Replicas <= 0 ||
		req.Pd.Resource == nil || req.Tikv.Resource == nil || req.Tidb.Resource == nil {
		err = errors.New("resource requirements of PD/TiKV/TiDB must be specified and replicas must be greater than 0")
		return
	}

	pdRes, err = convertResourceRequirements(req.Pd.Resource)
	if err != nil {
		return
	}
	tikvRes, err = convertResourceRequirements(req.Tikv.Resource)
	if err != nil {
		return
	}
	tidbRes, err = convertResourceRequirements(req.Tidb.Resource)
	if err != nil {
		return
	}

	if req.Tiflash != nil && req.Tiflash.Replicas != nil && *req.Tiflash.Replicas > 0 {
		if req.Tiflash.Resource == nil {
			err = errors.New("resource requirements of TiFlash must be specified")
			return
		}
		tiflash, err = convertResourceRequirements(req.Tiflash.Resource)
		if err != nil {
			return
		}
	}

	return
}

func convertConfigValue(x *structpb.Value) interface{} {
	switch v := x.GetKind().(type) {
	case *structpb.Value_NumberValue:
		if v == nil {
			return nil
		}
		// convert float64 to int64 if possible
		val := x.GetNumberValue()
		if val == float64(int(val)) {
			return int64(val)
		}
		return val
	default:
		return x.AsInterface()
	}
}

func convertClusterComponentsConfig(req *api.CreateClusterReq) (
	pdCfg *v1alpha1.PDConfigWraper, tikvCfg *v1alpha1.TiKVConfigWraper,
	tidbCfg *v1alpha1.TiDBConfigWraper, tiflashCfg *v1alpha1.TiFlashConfigWraper) {

	pdCfg = v1alpha1.NewPDConfig()
	for k, v := range req.Pd.Config {
		pdCfg.Set(k, convertConfigValue(v))
	}

	tidbCfg = v1alpha1.NewTiDBConfig()
	for k, v := range req.Tidb.Config {
		tidbCfg.Set(k, convertConfigValue(v))
	}

	tikvCfg = v1alpha1.NewTiKVConfig()
	for k, v := range req.Tikv.Config {
		tikvCfg.Set(k, convertConfigValue(v))
	}

	if req.Tiflash != nil && req.Tiflash.Replicas != nil && *req.Tiflash.Replicas > 0 {
		tiflashCfg = v1alpha1.NewTiFlashConfig()
		for k, v := range req.Tiflash.Config {
			tiflashCfg.Common.Set(k, convertConfigValue(v))
		}
		for k, v := range req.Tiflash.LearnerConfig {
			tiflashCfg.Proxy.Set(k, convertConfigValue(v))
		}
	}

	return
}

func convertMonitorComponentsResources(req *api.CreateClusterReq) (promRes, grafanaRes corev1.ResourceRequirements, err error) {
	if req.Prometheus != nil {
		promRes, err = convertResourceRequirements(req.Prometheus.Resource)
		if err != nil {
			return
		}
	}
	if req.Grafana != nil {
		grafanaRes, err = convertResourceRequirements(req.Grafana.Resource)
		if err != nil {
			return
		}
	}
	return
}

// UpdateCluster includes:
// - version: upgrade/downgrade the cluster
// - [pd/tikv/tiflash/tidb].replicas: scale the cluster
// - [pd/tikv/tiflash/tidb].resource: update the resource requirements of the cluster
// - [pd/tikv/tiflash/tidb].config: update the config of the cluster
func (s *ClusterServer) UpdateCluster(ctx context.Context, req *api.UpdateClusterReq) (*api.UpdateClusterResp, error) {
	k8sID := getKubernetesID(ctx)
	opCli := s.KubeClient.GetOperatorClient(k8sID)
	logger := log.L().With(zap.String("request", "UpdateCluster"), zap.String("k8sID", k8sID), zap.String("clusterID", req.ClusterId))
	if opCli == nil {
		logger.Error("K8s client not found")
		message := fmt.Sprintf("no %s is specified in the request header or the kubeconfig context not exists", HeaderKeyKubernetesID)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.UpdateClusterResp{Success: false, Message: &message}, nil
	}

	// get the current TidbCluster
	tc, err := opCli.PingcapV1alpha1().TidbClusters(req.ClusterId).Get(ctx, tidbClusterName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Error("TidbCluster not found", zap.Error(err))
			setResponseStatusCodes(ctx, http.StatusNotFound)
			message := fmt.Sprintf("TidbCluster %s not found", req.ClusterId)
			return &api.UpdateClusterResp{Success: false, Message: &message}, nil
		} else {
			logger.Error("Get TidbCluster failed", zap.Error(err))
			message := fmt.Sprintf("get TidbCluster failed: %s", err.Error())
			setResponseStatusCodes(ctx, http.StatusInternalServerError)
			return &api.UpdateClusterResp{Success: false, Message: &message}, nil
		}
	}

	// update the TidbCluster
	if err = updateTidbCluster(tc, req); err != nil {
		logger.Error("Update TidbCluster failed", zap.Error(err))
		message := fmt.Sprintf("update TidbCluster failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusBadRequest) // bad request if the request body is invalid
		return &api.UpdateClusterResp{Success: false, Message: &message}, nil
	}
	if _, err = opCli.PingcapV1alpha1().TidbClusters(req.ClusterId).Update(ctx, tc, metav1.UpdateOptions{}); err != nil {
		logger.Error("Apply the update failed", zap.Error(err))
		message := fmt.Sprintf("apply the update failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.UpdateClusterResp{Success: false, Message: &message}, nil
	}

	return &api.UpdateClusterResp{Success: true}, nil
}

func updateTidbCluster(tc *v1alpha1.TidbCluster, req *api.UpdateClusterReq) error {
	// update version if specified
	if req.Version != nil {
		tc.Spec.Version = *req.Version
	}

	// update PD's replicas/resource/config if specified
	if req.Pd != nil {
		if req.Pd.Replicas != nil && *req.Pd.Replicas > 0 {
			tc.Spec.PD.Replicas = int32(*req.Pd.Replicas)
		}
		if req.Pd.Resource != nil {
			res, err := convertResourceRequirements(req.Pd.Resource)
			if err != nil {
				return errors.New("invalid resource requirements")
			}
			tc.Spec.PD.ResourceRequirements = res
		}
		if req.Pd.Config != nil {
			// drop old config and set new config
			// empty req.Pd.Config means reset to default
			tc.Spec.PD.Config = v1alpha1.NewPDConfig()
			for k, v := range req.Pd.Config {
				tc.Spec.PD.Config.Set(k, convertConfigValue(v))
			}
		}
	}

	if req.Tikv != nil {
		if req.Tikv.Replicas != nil && *req.Tikv.Replicas > 0 {
			tc.Spec.TiKV.Replicas = int32(*req.Tikv.Replicas)
		}
		if req.Tikv.Resource != nil {
			res, err := convertResourceRequirements(req.Tikv.Resource)
			if err != nil {
				return errors.New("invalid resource requirements")
			}
			tc.Spec.TiKV.ResourceRequirements = res
		}
		if req.Tikv.Config != nil {
			tc.Spec.TiKV.Config = v1alpha1.NewTiKVConfig()
			for k, v := range req.Tikv.Config {
				tc.Spec.TiKV.Config.Set(k, convertConfigValue(v))
			}
		}
	}

	// NOTE: DO NOT support scale-out TiFlash from 0 replicas to non-zero replicas for now
	if req.Tiflash != nil && tc.Spec.TiFlash != nil && tc.Spec.TiFlash.Replicas > 0 {
		if req.Tiflash.Replicas != nil && *req.Tiflash.Replicas > 0 {
			tc.Spec.TiFlash.Replicas = int32(*req.Tiflash.Replicas)
		}
		if req.Tiflash.Resource != nil {
			res, err := convertResourceRequirements(req.Tiflash.Resource)
			if err != nil {
				return errors.New("invalid resource requirements")
			}
			tc.Spec.TiFlash.ResourceRequirements = res
		}
		if tc.Spec.TiFlash.Config == nil {
			tc.Spec.TiFlash.Config = v1alpha1.NewTiFlashConfig()
		}
		if req.Tiflash.Config != nil {
			tc.Spec.TiFlash.Config.Common = v1alpha1.NewTiFlashCommonConfig()
			for k, v := range req.Tiflash.Config {
				tc.Spec.TiFlash.Config.Common.Set(k, convertConfigValue(v))
			}
		}
		if req.Tiflash.LearnerConfig != nil {
			tc.Spec.TiFlash.Config.Proxy = v1alpha1.NewTiFlashProxyConfig()
			for k, v := range req.Tiflash.LearnerConfig {
				tc.Spec.TiFlash.Config.Proxy.Set(k, convertConfigValue(v))
			}
		}
	}

	if req.Tidb != nil {
		if req.Tidb.Replicas != nil && *req.Tidb.Replicas > 0 {
			tc.Spec.TiDB.Replicas = int32(*req.Tidb.Replicas)
		}
		if req.Tidb.Resource != nil {
			res, err := convertResourceRequirements(req.Tidb.Resource)
			if err != nil {
				return errors.New("invalid resource requirements")
			}
			tc.Spec.TiDB.ResourceRequirements = res
		}
		if req.Tidb.Config != nil {
			tc.Spec.TiDB.Config = v1alpha1.NewTiDBConfig()
			for k, v := range req.Tidb.Config {
				tc.Spec.TiDB.Config.Set(k, convertConfigValue(v))
			}
		}
	}

	return nil
}

func (s *ClusterServer) GetCluster(ctx context.Context, req *api.GetClusterReq) (*api.GetClusterResp, error) {
	k8sID := getKubernetesID(ctx)
	opCli := s.KubeClient.GetOperatorClient(k8sID)
	kubeCli := s.KubeClient.GetKubeClient(k8sID)
	logger := log.L().With(zap.String("request", "GetCluster"), zap.String("k8sID", k8sID), zap.String("clusterID", req.ClusterId))
	if opCli == nil || kubeCli == nil {
		logger.Error("K8s client not found")
		message := fmt.Sprintf("no %s is specified in the request header or the kubeconfig context not exists", HeaderKeyKubernetesID)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.GetClusterResp{Success: false, Message: &message}, nil
	}

	tc, err := opCli.PingcapV1alpha1().TidbClusters(req.ClusterId).Get(ctx, tidbClusterName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// assume ti's secret wouldn't block the deletion of the TidbCluster
			existing, err := checkPodPVCExisting(ctx, kubeCli, req.ClusterId)
			if err != nil {
				logger.Error("Check pods and pvcs failed", zap.Error(err))
				message := fmt.Sprintf("check pods and pvcs failed: %s", err.Error())
				setResponseStatusCodes(ctx, http.StatusInternalServerError)
				return &api.GetClusterResp{Success: false, Message: &message}, nil
			}
			if existing {
				// if tc not found, but Pods or PVCs exist, return with Deleting status
				return &api.GetClusterResp{
					Success: true,
					Data: &api.ClusterInfo{
						ClusterId: req.ClusterId,
						Status:    string(ClusterStatusDeleting),
					}}, nil
			}

			logger.Warn("TidbCluster not found", zap.Error(err))
			message := fmt.Sprintf("TidbCluster %s not found", req.ClusterId)
			setResponseStatusCodes(ctx, http.StatusNotFound)
			return &api.GetClusterResp{Success: false, Message: &message}, nil
		}
		logger.Error("Get TidbCluster failed", zap.Error(err))
		message := fmt.Sprintf("get TidbCluster failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.GetClusterResp{Success: false, Message: &message}, nil
	}

	tm, err := opCli.PingcapV1alpha1().TidbMonitors(req.ClusterId).Get(ctx, tidbMonitorName, metav1.GetOptions{})
	if err != nil {
		// for TidbMonitor, we don't return error if previous TiDBCluster exists
		if apierrors.IsNotFound(err) {
			logger.Warn("TidbMonitor not found", zap.Error(err))
			tm = nil // set empty TidbMonitor to nil
		} else {
			logger.Error("Get TidbMonitor failed", zap.Error(err))
		}
	}

	info := convertToClusterInfo(logger, kubeCli, tc, tm)
	return &api.GetClusterResp{Success: true, Data: info}, nil
}

func convertToClusterInfo(logger *zap.Logger, kubeCli kubernetes.Interface, tc *v1alpha1.TidbCluster, tm *v1alpha1.TidbMonitor) *api.ClusterInfo {
	// get all pods for the TidbCluster
	podList, err := kubeCli.CoreV1().Pods(tc.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			label.InstanceLabelKey: tc.Name,
		}).String(),
	})
	if err != nil {
		logger.Error("List pods failed", zap.Error(err))
	}

	pdRes, tikvRes, TidbRes, TiflashRes := reConvertClusterComponetsResources(tc)
	pdCfg, tikvCfg, TidbCfg, TiflashCfg, TiflashLearnerCfg := reConvertClusterComponetsConfig(tc)
	info := &api.ClusterInfo{
		ClusterId: tc.Namespace,
		Version:   tc.Spec.Version,
		Status:    string(convertClusterStatus(tc)),
		Pd: &api.PDStatus{
			Phase:    string(tc.Status.PD.Phase),
			Replicas: uint32(tc.Spec.PD.Replicas),
			Resource: pdRes,
			Config:   pdCfg,
			Members:  make([]*api.PDMember, 0, len(tc.Status.PD.Members)),
		},
		Tikv: &api.TiKVStatus{
			Phase:    string(tc.Status.TiKV.Phase),
			Replicas: uint32(tc.Spec.TiKV.Replicas),
			Resource: tikvRes,
			Config:   tikvCfg,
			Members:  make([]*api.TiKVMember, 0, len(tc.Status.TiKV.Stores)),
		},
		Tidb: &api.TiDBStatus{
			Phase:    string(tc.Status.TiDB.Phase),
			Replicas: uint32(tc.Spec.TiDB.Replicas),
			Resource: TidbRes,
			Config:   TidbCfg,
			Members:  make([]*api.TiDBMember, 0, len(tc.Status.TiDB.Members)),
		},
	}

	namePrefix := fmt.Sprintf("%s-", tc.Name)
	for name, member := range tc.Status.PD.Members {
		id, err := strconv.ParseUint(member.ID, 10, 64)
		if err != nil {
			logger.Error("Parse PD member ID failed", zap.String("id", member.ID), zap.Error(err))
		}
		info.Pd.Members = append(info.Pd.Members, &api.PDMember{
			Name:      strings.TrimPrefix(name, namePrefix),
			Id:        id,
			StartTime: getPodStartTime(podList, name),
			Health:    member.Health,
		})
	}

	for _, member := range tc.Status.TiKV.Stores {
		id, err := strconv.ParseUint(member.ID, 10, 64)
		if err != nil {
			logger.Error("Parse TiKV store ID failed", zap.String("id", member.ID), zap.Error(err))
		}
		info.Tikv.Members = append(info.Tikv.Members, &api.TiKVMember{
			Name:      strings.TrimPrefix(member.PodName, namePrefix),
			Id:        id,
			StartTime: getPodStartTime(podList, member.PodName),
			State:     string(member.State),
		})
	}

	tidbHealthCount := 0
	for name, member := range tc.Status.TiDB.Members {
		info.Tidb.Members = append(info.Tidb.Members, &api.TiDBMember{
			Name:      strings.TrimPrefix(name, namePrefix),
			StartTime: getPodStartTime(podList, name),
			Health:    member.Health,
		})
		if member.Health {
			tidbHealthCount++
		}
	}

	if !tc.PDIsAvailable() || !tc.TiKVIsAvailable() || tidbHealthCount == 0 {
		// for TiDB, only mark as unavailable when all TiDB members are unavailable
		info.Status = string(ClusterStatusUnavailable)
	}

	if tc.Spec.TiFlash != nil && tc.Spec.TiFlash.Replicas > 0 {
		info.Tiflash = &api.TiFlashStatus{
			Phase:         string(tc.Status.TiFlash.Phase),
			Replicas:      uint32(tc.Spec.TiFlash.Replicas),
			Resource:      TiflashRes,
			Config:        TiflashCfg,
			LearnerConfig: TiflashLearnerCfg,
			Members:       make([]*api.TiFlashMember, 0, len(tc.Status.TiFlash.Stores)),
		}
		tiflashReadyCount := 0
		for _, member := range tc.Status.TiFlash.Stores {
			id, err := strconv.ParseUint(member.ID, 10, 64)
			if err != nil {
				logger.Error("Parse TiFlash store ID failed", zap.String("id", member.ID), zap.Error(err))
			}
			info.Tiflash.Members = append(info.Tiflash.Members, &api.TiFlashMember{
				Name:      strings.TrimPrefix(member.PodName, namePrefix),
				Id:        id,
				StartTime: getPodStartTime(podList, member.PodName),
				State:     string(member.State),
			})
			if member.State == v1alpha1.TiKVStateUp {
				tiflashReadyCount++
			}
		}
		if tiflashReadyCount == 0 {
			info.Status = string(ClusterStatusUnavailable)
		}
	}

	// hack for `creating` status
	if tc.Status.PD.Phase == "" || tc.Status.TiKV.Phase == "" || tc.Status.TiDB.Phase == "" ||
		(tc.Spec.TiFlash != nil && tc.Spec.TiFlash.Replicas > 0 && tc.Status.TiFlash.Phase == "") {
		info.Status = string(ClusterStatusCreating)
	}

	if info.Status != string(ClusterStatusCreating) {
		host, port, err := getTiDBHostPort(kubeCli, tc)
		if err != nil {
			logger.Error("Get TiDB host and port failed", zap.Error(err))
		} else {
			info.Tidb.Host = host
			info.Tidb.NodePort = uint32(port)
		}
	}

	if tm != nil {
		info.Prometheus = &api.PrometheusStatus{
			Version:  tm.Spec.Prometheus.MonitorContainer.Version,
			Resource: reConvertResourceRequirements(tm.Spec.Prometheus.MonitorContainer.ResourceRequirements),
		}
		if tm.Spec.Prometheus.Config != nil {
			info.Prometheus.Config = make(map[string]string, len(tm.Spec.Prometheus.Config.CommandOptions))
			for _, opt := range tm.Spec.Prometheus.Config.CommandOptions {
				kv := strings.SplitN(strings.TrimPrefix(opt, "--"), "=", 2)
				if len(kv) == 2 {
					info.Prometheus.Config[kv[0]] = kv[1]
				}
			}
		}
		if tm.Spec.Grafana != nil {
			info.Grafana = &api.GrafanaStatus{
				Version:  tm.Spec.Grafana.MonitorContainer.Version,
				Resource: reConvertResourceRequirements(tm.Spec.Grafana.MonitorContainer.ResourceRequirements),
				Config:   tm.Spec.Grafana.Envs,
			}
			host, port, err := getGrafanaHostPort(kubeCli, tm)
			if err != nil {
				logger.Error("Get Grafana host and port failed", zap.Error(err))
			} else {
				info.Grafana.Host = host
				info.Grafana.NodePort = uint32(port)
			}
		}
	}

	return info
}

func convertClusterStatus(tc *v1alpha1.TidbCluster) ClusterStatus {
	if tc.DeletionTimestamp != nil {
		return ClusterStatusDeleting
	}

	if tc.Status.PD.Phase == v1alpha1.UpgradePhase || tc.Status.TiKV.Phase == v1alpha1.UpgradePhase || tc.Status.TiDB.Phase == v1alpha1.UpgradePhase ||
		(tc.Spec.TiFlash != nil && tc.Spec.TiFlash.Replicas > 0 && tc.Status.TiFlash.Phase == v1alpha1.UpgradePhase) {
		return ClusterStatusUpgrading
	}

	if tc.Status.PD.Phase == v1alpha1.ScalePhase || tc.Status.TiKV.Phase == v1alpha1.ScalePhase || tc.Status.TiDB.Phase == v1alpha1.ScalePhase ||
		(tc.Spec.TiFlash != nil && tc.Spec.TiFlash.Replicas > 0 && tc.Status.TiFlash.Phase == v1alpha1.ScalePhase) {
		return ClusterStatusScaling
	}

	return ClusterStatusRunning
}

func reConvertClusterComponetsResources(tc *v1alpha1.TidbCluster) (pd, tikv, tidb, tiflash *api.Resource) {
	pd = reConvertResourceRequirements(tc.Spec.PD.ResourceRequirements)
	tikv = reConvertResourceRequirements(tc.Spec.TiKV.ResourceRequirements)
	tidb = reConvertResourceRequirements(tc.Spec.TiDB.ResourceRequirements)
	if tc.Spec.TiFlash != nil && tc.Spec.TiFlash.Replicas > 0 {
		tiflash = reConvertResourceRequirements(tc.Spec.TiFlash.ResourceRequirements)
	}
	return
}

func reConvertClusterComponetsConfig(tc *v1alpha1.TidbCluster) (
	pdCfg, tikvCfg, tidbCfg, tiflashCfg, tiflashLearnerCfg map[string]*structpb.Value) {
	// NOTE: we keep the internal default config for now
	pdCfg = make(map[string]*structpb.Value)
	for k, v := range flattenMap(tc.Spec.PD.Config.MP) {
		val, _ := structpb.NewValue(v)
		pdCfg[k] = val
	}

	tikvCfg = make(map[string]*structpb.Value)
	for k, v := range flattenMap(tc.Spec.TiKV.Config.MP) {
		val, _ := structpb.NewValue(v)
		tikvCfg[k] = val
	}

	tidbCfg = make(map[string]*structpb.Value)
	for k, v := range flattenMap(tc.Spec.TiDB.Config.MP) {
		val, _ := structpb.NewValue(v)
		tidbCfg[k] = val
	}

	if tc.Spec.TiFlash != nil && tc.Spec.TiFlash.Replicas > 0 {
		tiflashCfg = make(map[string]*structpb.Value)
		tiflashLearnerCfg = make(map[string]*structpb.Value)
		for k, v := range flattenMap(tc.Spec.TiFlash.Config.Common.MP) {
			val, _ := structpb.NewValue(v)
			tiflashCfg[k] = val
		}
		for k, v := range flattenMap(tc.Spec.TiFlash.Config.Proxy.MP) {
			val, _ := structpb.NewValue(v)
			tiflashLearnerCfg[k] = val
		}
	}

	return
}

// flattenMap convert mutil-layer map to single layer
// ref: https://github.com/pingcap/tiup/blob/9e47d78b5518999efc3763168e891d6910f26099/pkg/cluster/spec/server_config.go#L119
func flattenMap(ms map[string]any) map[string]any {
	result := map[string]any{}
	for k, v := range ms {
		var sub map[string]any

		if m, ok := v.(map[string]any); ok {
			sub = flattenMap(m)
		} else if m, ok := v.(map[any]any); ok {
			fixM := map[string]any{}
			for k, v := range m {
				if sk, ok := k.(string); ok {
					fixM[sk] = v
				}
			}
			sub = flattenMap(fixM)
		} else {
			result[k] = v
			continue
		}

		for sk, sv := range sub {
			result[k+"."+sk] = sv
		}
	}
	return result
}

func getTiDBHostPort(kubeCli kubernetes.Interface, tc *v1alpha1.TidbCluster) (host string, port int32, err error) {
	svc, err := kubeCli.CoreV1().Services(tc.Namespace).Get(context.Background(), fmt.Sprintf("%s-tidb", tc.Name), metav1.GetOptions{})
	if err != nil {
		return "", 0, err
	}
	podList, err := kubeCli.CoreV1().Pods(tc.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(label.New().Instance(tc.Name).TiDB().Labels()).String(),
	})
	if err != nil {
		return "", 0, err
	}
	if len(podList.Items) == 0 {
		return "", 0, fmt.Errorf("no TiDB Pod found")
	}

	for _, svcPort := range svc.Spec.Ports {
		if tc.Spec.TiDB.Service != nil && svcPort.Name == tc.Spec.TiDB.Service.GetPortName() {
			port = svcPort.NodePort // NOTE: use NodePort for now
		}
	}
	// get the Node IP for a random Pod
	pod := podList.Items[rand.Intn(len(podList.Items))]
	host = pod.Status.HostIP
	return
}

func getGrafanaHostPort(kubeCli kubernetes.Interface, tm *v1alpha1.TidbMonitor) (host string, port int32, err error) {
	svc, err := kubeCli.CoreV1().Services(tm.Namespace).Get(context.Background(), fmt.Sprintf("%s-grafana", tm.Name), metav1.GetOptions{})
	if err != nil {
		return "", 0, err
	}
	podList, err := kubeCli.CoreV1().Pods(tm.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(label.New().Instance(tm.Name).Monitor().Labels()).String(),
	})
	if err != nil {
		return "", 0, err
	}
	if len(podList.Items) == 0 {
		return "", 0, fmt.Errorf("no Grafana Pod found")
	}

	for _, svcPort := range svc.Spec.Ports {
		if tm.Spec.Grafana != nil && svcPort.Name == "http-grafana" {
			port = svcPort.NodePort // NOTE: use NodePort for now
		}
	}
	// get the Node IP for a random Pod
	pod := podList.Items[rand.Intn(len(podList.Items))]
	host = pod.Status.HostIP
	return
}

func getPodStartTime(podList *corev1.PodList, name string) string {
	if podList == nil {
		return ""
	}

	for _, pod := range podList.Items {
		if pod.Name == name {
			return pod.CreationTimestamp.Format(time.RFC3339)
		}
	}
	return ""
}

func checkPodPVCExisting(ctx context.Context, kubeCli kubernetes.Interface, namespace string) (bool, error) {
	ls := metav1.FormatLabelSelector(&metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      label.InstanceLabelKey,
				Operator: metav1.LabelSelectorOpIn,
				Values: []string{
					tidbClusterName,
					tidbMonitorName,
					tidbInitializerName,
				},
			},
		},
	})

	podList, err := kubeCli.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: ls,
	})
	if err != nil {
		return false, err
	}
	if len(podList.Items) > 0 {
		return true, nil
	}

	pvcList, err := kubeCli.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: ls,
	})
	if err != nil {
		return false, err
	}
	if len(pvcList.Items) > 0 {
		return true, nil
	}
	return false, nil
}

func (s *ClusterServer) DeleteCluster(ctx context.Context, req *api.DeleteClusterReq) (*api.DeleteClusterResp, error) {
	k8sID := getKubernetesID(ctx)
	opCli := s.KubeClient.GetOperatorClient(k8sID)
	kubeCli := s.KubeClient.GetKubeClient(k8sID)
	logger := log.L().With(zap.String("request", "DeleteCluster"), zap.String("k8sID", k8sID), zap.String("clusterID", req.ClusterId))
	if opCli == nil || kubeCli == nil {
		logger.Error("K8s client not found")
		message := fmt.Sprintf("no %s is specified in the request header or the kubeconfig context not exists", HeaderKeyKubernetesID)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.DeleteClusterResp{Success: false, Message: &message}, nil
	}
	// delete tm
	if err := opCli.PingcapV1alpha1().TidbMonitors(req.ClusterId).Delete(ctx, tidbMonitorName, metav1.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("TidbMonitor not found", zap.Error(err))
		} else {
			logger.Error("Delete TidbMonitor failed", zap.Error(err))
			setResponseStatusCodes(ctx, http.StatusInternalServerError)
			message := fmt.Sprintf("delete TidbMonitor failed: %s", err.Error())
			return &api.DeleteClusterResp{Success: false, Message: &message}, nil
		}

	}
	// delete init and secret
	if err := opCli.PingcapV1alpha1().TidbInitializers(req.ClusterId).Delete(ctx, tidbInitializerName, metav1.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("TidbInitializer not found", zap.Error(err))
		} else {
			logger.Error("Delete TidbInitializer failed", zap.Error(err))
			setResponseStatusCodes(ctx, http.StatusInternalServerError)
			message := fmt.Sprintf("delete TidbInitializer failed: %s", err.Error())
			return &api.DeleteClusterResp{Success: false, Message: &message}, nil
		}
	}
	if err := kubeCli.CoreV1().Secrets(req.ClusterId).Delete(ctx, tidbInitializerPasswordSecret, metav1.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("TidbInitializer password secret not found", zap.Error(err))
		} else {
			logger.Error("Delete TidbInitializer password secret failed", zap.Error(err))
			setResponseStatusCodes(ctx, http.StatusInternalServerError)
			message := fmt.Sprintf("delete TidbInitializer password secret failed: %s", err.Error())
			return &api.DeleteClusterResp{Success: false, Message: &message}, nil
		}
	}

	// get restores for this cluster
	// NOTE: we list all restores in the cluster's ns now as we only have one cluster in one ns
	restoreList, err := opCli.PingcapV1alpha1().Restores(req.ClusterId).List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error("List TidbRestore failed", zap.Error(err))
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		message := fmt.Sprintf("list TidbRestore failed: %s", err.Error())
		return &api.DeleteClusterResp{Success: false, Message: &message}, nil
	}
	for _, restore := range restoreList.Items {
		// delete restore for this cluster
		if err := opCli.PingcapV1alpha1().Restores(req.ClusterId).Delete(ctx, restore.Name, metav1.DeleteOptions{}); err != nil {
			if apierrors.IsNotFound(err) {
				logger.Info("TidbRestore not found", zap.String("restore", restore.Name), zap.Error(err))
			} else {
				logger.Error("Delete TidbRestore failed", zap.String("restore", restore.Name), zap.Error(err))
				setResponseStatusCodes(ctx, http.StatusInternalServerError)
				message := fmt.Sprintf("delete TidbRestore %s failed: %s", restore.Name, err.Error())
				return &api.DeleteClusterResp{Success: false, Message: &message}, nil
			}
		}
	}

	// delete tc
	if err := opCli.PingcapV1alpha1().TidbClusters(req.ClusterId).Delete(ctx, tidbClusterName, metav1.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("TidbCluster not found", zap.Error(err))
		} else {
			logger.Error("Delete TidbCluster failed", zap.Error(err))
			setResponseStatusCodes(ctx, http.StatusInternalServerError)
			message := fmt.Sprintf("delete TidbCluster failed: %s", err.Error())
			return &api.DeleteClusterResp{Success: false, Message: &message}, nil
		}

	}
	// we don't delete the ns for now, since some backup may still stored in it
	// TODO(http-service): we delete all pvc in ns for now, but we may only delete tidbcluster pvc in future
	if err := kubeCli.CoreV1().PersistentVolumeClaims(req.ClusterId).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("PersistentVolumeClaims not found", zap.Error(err))
		} else {
			logger.Error("Delete PersistentVolumeClaims failed", zap.Error(err))
			setResponseStatusCodes(ctx, http.StatusInternalServerError)
			message := fmt.Sprintf("delete PersistentVolumeClaims failed: %s", err.Error())
			return &api.DeleteClusterResp{Success: false, Message: &message}, nil
		}
	}
	return &api.DeleteClusterResp{Success: true}, nil
}

// RestartCluster
// graceful rolling restart to all Pods in specified components
// ref https://docs.pingcap.com/tidb-in-kubernetes/stable/restart-a-tidb-cluster
func (s *ClusterServer) RestartCluster(ctx context.Context, req *api.RestartClusterReq) (*api.RestartClusterResp, error) {
	k8sID := getKubernetesID(ctx)
	opCli := s.KubeClient.GetOperatorClient(k8sID)
	kubeCli := s.KubeClient.GetKubeClient(k8sID)
	logger := log.L().With(zap.String("request", "RestartCluster"), zap.String("k8sID", k8sID), zap.String("clusterID", req.ClusterId))
	if opCli == nil || kubeCli == nil {
		logger.Error("K8s client not found")
		message := fmt.Sprintf("no %s is specified in the request header or the kubeconfig context not exists", HeaderKeyKubernetesID)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.RestartClusterResp{Success: false, Message: &message}, nil
	}

	tc, err := opCli.PingcapV1alpha1().TidbClusters(req.ClusterId).Get(ctx, tidbClusterName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("TidbCluster not found", zap.Error(err))
			message := fmt.Sprintf("TidbCluster %s not found", req.ClusterId)
			setResponseStatusCodes(ctx, http.StatusNotFound)
			return &api.RestartClusterResp{Success: false, Message: &message}, nil
		}
		logger.Error("Get TidbCluster failed", zap.Error(err))
		message := fmt.Sprintf("get TidbCluster failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.RestartClusterResp{Success: false, Message: &message}, nil
	}

	// Add tidb.pingcap.com/restartedAt in the annotation of the spec of the TiDB component
	// to trigger the rolling restart of the TiDB component
	// set its value to be the current time.
	tc, err = setRestartAnnotation(tc, req.Component)
	if err != nil {
		logger.Error("Restart TidbCluster failed", zap.Error(err))
		message := fmt.Sprintf("restart TidbCluster failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.RestartClusterResp{Success: false, Message: &message}, nil
	}

	// update tc
	_, err = opCli.PingcapV1alpha1().TidbClusters(req.ClusterId).Update(ctx, tc, metav1.UpdateOptions{})
	if err != nil {
		logger.Error("Update TidbCluster failed", zap.Error(err))
		message := fmt.Sprintf("update TidbCluster failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.RestartClusterResp{Success: false, Message: &message}, nil
	}

	return &api.RestartClusterResp{Success: true}, nil
}

func setRestartAnnotation(tc *v1alpha1.TidbCluster, components []string) (*v1alpha1.TidbCluster, error) {
	// check component
	if len(components) == 0 {
		return nil, fmt.Errorf("no components are specified that need to be restarted")
	}

	restartAt := time.Now().Format(time.RFC3339)
	for _, comp := range components {
		switch strings.ToLower(comp) {
		case "pd":
			if tc.Spec.PD.Annotations == nil {
				tc.Spec.PD.Annotations = make(map[string]string)
			}
			tc.Spec.PD.Annotations["tidb.pingcap.com/restartedAt"] = restartAt
		case "tikv":
			if tc.Spec.TiKV.Annotations == nil {
				tc.Spec.TiKV.Annotations = make(map[string]string)
			}
			tc.Spec.TiKV.Annotations["tidb.pingcap.com/restartedAt"] = restartAt
		case "tidb":
			if tc.Spec.TiDB.Annotations == nil {
				tc.Spec.TiDB.Annotations = make(map[string]string)
			}
			tc.Spec.TiDB.Annotations["tidb.pingcap.com/restartedAt"] = restartAt
		case "tiflash":
			if tc.Spec.TiFlash.Annotations == nil {
				tc.Spec.TiFlash.Annotations = make(map[string]string)
			}
			tc.Spec.TiFlash.Annotations["tidb.pingcap.com/restartedAt"] = restartAt
		case "ticdc":
			if tc.Spec.TiCDC.Annotations == nil {
				tc.Spec.TiCDC.Annotations = make(map[string]string)
			}
			tc.Spec.TiCDC.Annotations["tidb.pingcap.com/restartedAt"] = restartAt
		case "tiproxy":
			if tc.Spec.TiDB.Annotations == nil {
				tc.Spec.TiDB.Annotations = make(map[string]string)
			}
			tc.Spec.TiDB.Annotations["tidb.pingcap.com/restartedAt"] = restartAt
		case "pump":
			if tc.Spec.Pump.Annotations == nil {
				tc.Spec.Pump.Annotations = make(map[string]string)
			}
			tc.Spec.Pump.Annotations["tidb.pingcap.com/restartedAt"] = restartAt
		default:
			return nil, fmt.Errorf("invalid component: %s", comp)
		}
	}

	return tc, nil
}
