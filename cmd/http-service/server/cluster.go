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
	"net/http"

	"github.com/pingcap/log"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/pingcap/tidb-operator/http-service/kube"
	"github.com/pingcap/tidb-operator/http-service/pbgen/api"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

const (
	helperImage = "busybox:1.36"

	// try `kanshiori/mysqlclient-arm64` for ARM64
	tidbInitializerImage          = "tnir/mysqlclient"
	tidbInitializerName           = "tidb-initializer"
	tidbInitializerPasswordSecret = "tidb-secret"
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

	// TODO: add verification for the request body
	// TODO: customize image support

	// TODO(csuzhangxc): json config support

	// TODO(csuzhangxc): create TidbMonitor (prometheus and grafana)

	tc, err := assembleTidbCluster(ctx, req)
	if err != nil {
		logger.Error("Assemble TidbCluster CR failed", zap.Error(err))
		message := fmt.Sprintf("assemble TidbCluster CR failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.CreateClusterResp{Success: false, Message: &message}, nil
	}

	ti, tiSecret, err := assembleTidbInitializer(ctx, req)
	if err != nil {
		logger.Error("Assemble TidbInitializer CR failed", zap.Error(err))
		message := fmt.Sprintf("assemble TidbInitializer CR failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.CreateClusterResp{Success: false, Message: &message}, nil
	}

	if _, err = opCli.PingcapV1alpha1().TidbClusters(req.ClusterId).Create(ctx, tc, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if ti == nil { // TODO(csuzhangxc): check tm exists
				// if TidbCluster already exists but no TidbInitializer is specified, return conflict error
				logger.Error("TidbCluster already exists", zap.Error(err))
				setResponseStatusCodes(ctx, http.StatusConflict)
				message := fmt.Sprintf("TidbCluster %s already exists", req.ClusterId)
				return &api.CreateClusterResp{Success: false, Message: &message}, nil
			} else {
				// if TidbCluster already exists and TidbInitializer is specified
				// ignore the error so that the TidbInitializer can be created (when re-requesting)
				logger.Warn("TidbCluster already exists, but still need to create TidbInitializer, ignore the error", zap.Error(err))
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
			logger.Error("Create TidbInitializer failed", zap.Error(err))
			message := fmt.Sprintf("create TidbInitializer failed: %s", err.Error())
			if apierrors.IsAlreadyExists(err) {
				setResponseStatusCodes(ctx, http.StatusConflict)
				message = fmt.Sprintf("TidbInitializer %s already exists", req.ClusterId)
				return &api.CreateClusterResp{Success: false, Message: &message}, nil
			} else {
				setResponseStatusCodes(ctx, http.StatusInternalServerError)
				return &api.CreateClusterResp{Success: false, Message: &message}, nil
			}
		}
	}

	return &api.CreateClusterResp{Success: true}, nil
}

func assembleTidbCluster(ctx context.Context, req *api.CreateClusterReq) (*v1alpha1.TidbCluster, error) {
	deletePVP := corev1.PersistentVolumeReclaimDelete
	pdRes, tikvRes, tidbRes, tiflashRes, err := convertClusterComponetsResources(req)
	if err != nil {
		return nil, errors.New("invalid resource requirements")
	}
	tidbPort := int32(4000)
	if req.Tidb.Port != nil {
		tidbPort = int32(*req.Tidb.Port)
	}

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
				Replicas:             int32(req.Pd.Replicas),
				MaxFailoverCount:     pointer.Int32Ptr(0),
				ResourceRequirements: pdRes,
				Config:               v1alpha1.NewPDConfig(),
			},
			TiKV: &v1alpha1.TiKVSpec{
				Replicas:             int32(req.Tikv.Replicas),
				MaxFailoverCount:     pointer.Int32Ptr(0),
				ResourceRequirements: tikvRes,
				Config:               v1alpha1.NewTiKVConfig(),
			},
			TiDB: &v1alpha1.TiDBSpec{
				Replicas:             int32(req.Tidb.Replicas),
				MaxFailoverCount:     pointer.Int32Ptr(0),
				ResourceRequirements: tidbRes,
				Config:               v1alpha1.NewTiDBConfig(),
				Service: &v1alpha1.TiDBServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type: corev1.ServiceTypeNodePort, // NOTE: use NodePort for now
						Port: pointer.Int32Ptr(tidbPort),
					},
				},
			},
		},
	}

	if req.Tiflash != nil && req.Tiflash.Replicas > 0 {
		tc.Spec.TiFlash = &v1alpha1.TiFlashSpec{
			Replicas:             int32(req.Tiflash.Replicas),
			MaxFailoverCount:     pointer.Int32Ptr(0),
			ResourceRequirements: tiflashRes,
			Config:               v1alpha1.NewTiFlashConfig(),
			StorageClaims: []v1alpha1.StorageClaim{
				{
					Resources: tiflashRes,
				},
			},
		}
	}

	return tc, nil
}

func assembleTidbInitializer(ctx context.Context, req *api.CreateClusterReq) (*v1alpha1.TidbInitializer, *corev1.Secret, error) {
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

func convertClusterComponetsResources(req *api.CreateClusterReq) (pdRes, tikvRes, tidbRes, tiflash corev1.ResourceRequirements, err error) {
	if req.Pd == nil || req.Tikv == nil || req.Tidb == nil || req.Pd.Replicas <= 0 || req.Tikv.Replicas <= 0 || req.Tidb.Replicas <= 0 {
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

	if req.Tiflash != nil && req.Tiflash.Replicas > 0 {
		tiflash, err = convertResourceRequirements(req.Tiflash.Resource)
		if err != nil {
			return
		}
	}

	return
}
