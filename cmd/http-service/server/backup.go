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
	"time"

	"github.com/pingcap/log"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/pingcap/tidb-operator/http-service/pbgen/api"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

const (
	backupManager  = "tidb-backup-manager"
	s3StorageClass = "STANDARD"
)

func (s *ClusterServer) CreateBackup(ctx context.Context, req *api.CreateBackupReq) (*api.CreateBackupResp, error) {
	k8sID := getKubernetesID(ctx)
	opCli := s.KubeClient.GetOperatorClient(k8sID)
	kubeCli := s.KubeClient.GetKubeClient(k8sID)
	logger := log.L().With(zap.String("request", "CreateBackup"), zap.String("k8sID", k8sID),
		zap.String("clusterID", req.ClusterId), zap.String("backupID", req.BackupId))
	if opCli == nil || kubeCli == nil {
		logger.Error("K8s client not found")
		message := fmt.Sprintf("no %s is specified in the request header or the kubeconfig context not exists", HeaderKeyKubernetesID)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.CreateBackupResp{Success: false, Message: &message}, nil
	}

	// check whether the cluster exists first
	_, err := opCli.PingcapV1alpha1().TidbClusters(req.ClusterId).Get(ctx, tidbClusterName, metav1.GetOptions{})
	if err != nil {
		logger.Error("TidbCluster not found", zap.Error(err))
		message := fmt.Sprintf("TidbCluster %s not found", req.ClusterId)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.CreateBackupResp{Success: false, Message: &message}, nil
	}

	secret, err := assembleBackupSecret(req)
	if err != nil {
		logger.Error("Assemble backup secret failed", zap.Error(err))
		message := fmt.Sprintf("assemble backup secret failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.CreateBackupResp{Success: false, Message: &message}, nil
	}

	backup, err := assembleBackup(req)
	if err != nil {
		logger.Error("Assemble backup CR failed", zap.Error(err))
		message := fmt.Sprintf("assemble backup CR failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.CreateBackupResp{Success: false, Message: &message}, nil
	}

	// ensure RBAC
	if err = ensureBackupRestoreRBAC(ctx, kubeCli, req.ClusterId); err != nil {
		logger.Error("Ensure backup RBAC failed", zap.Error(err))
		message := fmt.Sprintf("ensure backup RBAC failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.CreateBackupResp{Success: false, Message: &message}, nil
	}

	// create secret
	_, err = kubeCli.CoreV1().Secrets(req.ClusterId).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		logger.Error("Create backup secret failed", zap.Error(err))
		message := fmt.Sprintf("create backup secret failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.CreateBackupResp{Success: false, Message: &message}, nil
	}

	// create backup
	_, err = opCli.PingcapV1alpha1().Backups(req.ClusterId).Create(ctx, backup, metav1.CreateOptions{})
	if err != nil {
		logger.Error("Create backup failed", zap.Error(err))
		message := fmt.Sprintf("create backup failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.CreateBackupResp{Success: false, Message: &message}, nil
	}

	return &api.CreateBackupResp{Success: true}, nil
}

func assembleBackupSecret(req *api.CreateBackupReq) (*corev1.Secret, error) {
	if req.AccessKey == "" || req.SecretKey == "" {
		return nil, errors.New("access_key and secret_key must be specified")
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.ClusterId,
			Name:      req.BackupId, // same as backup name
		},
		Data: map[string][]byte{
			"access_key": []byte(req.AccessKey),
			"secret_key": []byte(req.SecretKey),
		},
	}, nil
}

func assembleBackup(req *api.CreateBackupReq) (*v1alpha1.Backup, error) {
	if req.Bucket == "" || req.Prefix == "" {
		return nil, errors.New("bucket and prefix must be specified")
	}

	var endpoint string
	if req.Endpoint != nil {
		endpoint = *req.Endpoint
	}

	return &v1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.ClusterId,
			Name:      req.BackupId,
		},
		Spec: v1alpha1.BackupSpec{
			CleanPolicy: v1alpha1.CleanPolicyTypeDelete, // always delete data when deleting backup
			BR: &v1alpha1.BRConfig{
				ClusterNamespace: req.ClusterId,
				Cluster:          tidbClusterName,
			},
			StorageProvider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					// NOTE: update these fields if necessary
					Provider:     v1alpha1.S3StorageProviderTypeAWS,
					StorageClass: s3StorageClass,
					SecretName:   req.BackupId, // same as backup name
					Endpoint:     endpoint,
					Bucket:       req.Bucket,
					Prefix:       req.Prefix,
				},
			},
		},
	}, nil
}

func ensureBackupRestoreRBAC(ctx context.Context, cli kubernetes.Interface, namespace string) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      backupManager,
		},
	}
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      backupManager,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{"pingcap.com"},
				Resources: []string{"backups", "restores"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
		},
	}
	roleBinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      backupManager,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      backupManager,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     backupManager,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	// create RBAC resources if not exists
	if _, err := cli.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	if _, err := cli.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	if _, err := cli.RbacV1().RoleBindings(namespace).Create(ctx, &roleBinding, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (s *ClusterServer) CreateRestore(ctx context.Context, req *api.CreateRestoreReq) (*api.CreateRestoreResp, error) {
	k8sID := getKubernetesID(ctx)
	opCli := s.KubeClient.GetOperatorClient(k8sID)
	kubeCli := s.KubeClient.GetKubeClient(k8sID)
	logger := log.L().With(zap.String("request", "CreateRestore"), zap.String("k8sID", k8sID),
		zap.String("clusterID", req.ClusterId), zap.String("restoreID", req.RestoreId))
	if opCli == nil || kubeCli == nil {
		logger.Error("K8s client not found")
		message := fmt.Sprintf("no %s is specified in the request header or the kubeconfig context not exists", HeaderKeyKubernetesID)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.CreateRestoreResp{Success: false, Message: &message}, nil
	}

	// check whether the cluster exists first
	_, err := opCli.PingcapV1alpha1().TidbClusters(req.ClusterId).Get(ctx, tidbClusterName, metav1.GetOptions{})
	if err != nil {
		logger.Error("TidbCluster not found", zap.Error(err))
		message := fmt.Sprintf("TidbCluster %s not found", req.ClusterId)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.CreateRestoreResp{Success: false, Message: &message}, nil
	}

	secret, err := assembleRestoreSecret(req)
	if err != nil {
		logger.Error("Assemble restore secret failed", zap.Error(err))
		message := fmt.Sprintf("assemble restore secret failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.CreateRestoreResp{Success: false, Message: &message}, nil
	}

	restore, err := assembleRestore(req)
	if err != nil {
		logger.Error("Assemble restore CR failed", zap.Error(err))
		message := fmt.Sprintf("assemble restore CR failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.CreateRestoreResp{Success: false, Message: &message}, nil
	}

	// ensure RBAC
	if err = ensureBackupRestoreRBAC(ctx, kubeCli, req.ClusterId); err != nil {
		logger.Error("Ensure restore RBAC failed", zap.Error(err))
		message := fmt.Sprintf("ensure restore RBAC failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.CreateRestoreResp{Success: false, Message: &message}, nil
	}

	// create secret
	_, err = kubeCli.CoreV1().Secrets(req.ClusterId).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		logger.Error("Create restore secret failed", zap.Error(err))
		message := fmt.Sprintf("create restore secret failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.CreateRestoreResp{Success: false, Message: &message}, nil
	}

	// create restore
	_, err = opCli.PingcapV1alpha1().Restores(req.ClusterId).Create(ctx, restore, metav1.CreateOptions{})
	if err != nil {
		logger.Error("Create restore failed", zap.Error(err))
		message := fmt.Sprintf("create restore failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.CreateRestoreResp{Success: false, Message: &message}, nil
	}

	return &api.CreateRestoreResp{Success: true}, nil
}

func assembleRestoreSecret(req *api.CreateRestoreReq) (*corev1.Secret, error) {
	if req.AccessKey == "" || req.SecretKey == "" {
		return nil, errors.New("access_key and secret_key must be specified")
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.ClusterId,
			Name:      req.RestoreId, // same as restore name
		},
		Data: map[string][]byte{
			"access_key": []byte(req.AccessKey),
			"secret_key": []byte(req.SecretKey),
		},
	}, nil
}

func assembleRestore(req *api.CreateRestoreReq) (*v1alpha1.Restore, error) {
	if req.Bucket == "" || req.Prefix == "" {
		return nil, errors.New("bucket and prefix must be specified")
	}

	var endpoint string
	if req.Endpoint != nil {
		endpoint = *req.Endpoint
	}

	return &v1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.ClusterId,
			Name:      req.RestoreId,
		},
		Spec: v1alpha1.RestoreSpec{
			BR: &v1alpha1.BRConfig{
				ClusterNamespace: req.ClusterId,
				Cluster:          tidbClusterName,
			},
			StorageProvider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					// NOTE: update these fields if necessary
					Provider:     v1alpha1.S3StorageProviderTypeAWS,
					StorageClass: s3StorageClass,
					SecretName:   req.RestoreId, // same as restore name
					Endpoint:     endpoint,
					Bucket:       req.Bucket,
					Prefix:       req.Prefix,
				},
			},
		},
	}, nil
}

func (s *ClusterServer) GetBackup(ctx context.Context, req *api.GetBackupReq) (*api.GetBackupResp, error) {
	k8sID := getKubernetesID(ctx)
	opCli := s.KubeClient.GetOperatorClient(k8sID)
	logger := log.L().With(zap.String("request", "GetBackup"), zap.String("k8sID", k8sID),
		zap.String("clusterID", req.ClusterId), zap.String("backupID", req.BackupId))
	if opCli == nil {
		logger.Error("K8s client not found")
		message := fmt.Sprintf("no %s is specified in the request header or the kubeconfig context not exists", HeaderKeyKubernetesID)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.GetBackupResp{Success: false, Message: &message}, nil
	}

	backup, err := opCli.PingcapV1alpha1().Backups(req.ClusterId).Get(ctx, req.BackupId, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Backup not found", zap.Error(err))
			message := fmt.Sprintf("Backup %s not found", req.BackupId)
			setResponseStatusCodes(ctx, http.StatusNotFound)
			return &api.GetBackupResp{Success: false, Message: &message}, nil
		}
		logger.Error("Get backup failed", zap.Error(err))
		message := fmt.Sprintf("get backup failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.GetBackupResp{Success: false, Message: &message}, nil
	}

	info := convertToBackupInfo(backup)
	// get job status
	kubeCli := s.KubeClient.GetKubeClient(k8sID)
	_, err = kubeCli.BatchV1().Jobs(backup.GetNamespace()).Get(ctx, backup.GetBackupJobName(), metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			info.Status = "Stopped"
			logger.Info("Backup Job not found, Backup Job is Stopped.", zap.Error(err))
		} else {
			logger.Error("Get backup job failed", zap.Error(err))
			message := fmt.Sprintf("Get backup job failed: %s", err.Error())
			setResponseStatusCodes(ctx, http.StatusInternalServerError)
			return &api.GetBackupResp{Success: false, Message: &message}, nil
		}
	}

	return &api.GetBackupResp{Success: true, Data: info}, nil
}

func convertToBackupInfo(backup *v1alpha1.Backup) *api.BackupInfo {
	info := &api.BackupInfo{
		ClusterId:  backup.Namespace,
		BackupId:   backup.Name,
		Status:     string(backup.Status.Phase),
		Size:       backup.Status.BackupSizeReadable,
		BackupPath: backup.Status.BackupPath,
		CommitTs:   backup.Status.CommitTs,
	}

	if !backup.Status.TimeStarted.IsZero() {
		info.StartedAt = backup.Status.TimeStarted.Format(time.RFC3339)
	}
	if !backup.Status.TimeCompleted.IsZero() {
		info.CompletedAt = backup.Status.TimeCompleted.Format(time.RFC3339)
	}

	return info
}

func (s *ClusterServer) GetRestore(ctx context.Context, req *api.GetRestoreReq) (*api.GetRestoreResp, error) {
	k8sID := getKubernetesID(ctx)
	opCli := s.KubeClient.GetOperatorClient(k8sID)
	logger := log.L().With(zap.String("request", "GetRestore"), zap.String("k8sID", k8sID),
		zap.String("clusterID", req.ClusterId), zap.String("restoreID", req.RestoreId))
	if opCli == nil {
		logger.Error("K8s client not found")
		message := fmt.Sprintf("no %s is specified in the request header or the kubeconfig context not exists", HeaderKeyKubernetesID)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.GetRestoreResp{Success: false, Message: &message}, nil
	}

	restore, err := opCli.PingcapV1alpha1().Restores(req.ClusterId).Get(ctx, req.RestoreId, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Restore not found", zap.Error(err))
			message := fmt.Sprintf("Restore %s not found", req.RestoreId)
			setResponseStatusCodes(ctx, http.StatusNotFound)
			return &api.GetRestoreResp{Success: false, Message: &message}, nil
		}
		logger.Error("Get restore failed", zap.Error(err))
		message := fmt.Sprintf("get restore failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.GetRestoreResp{Success: false, Message: &message}, nil
	}

	info := convertToRestoreInfo(restore)
	// get job status
	kubeCli := s.KubeClient.GetKubeClient(k8sID)
	_, err = kubeCli.BatchV1().Jobs(restore.GetNamespace()).Get(ctx, restore.GetRestoreJobName(), metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			info.Status = "Stopped"
			logger.Info("Restore Job not found, Restore Job is Stopped.", zap.Error(err))
		} else {
			logger.Error("Get Restore job failed", zap.Error(err))
			message := fmt.Sprintf("Get restore job failed: %s", err.Error())
			setResponseStatusCodes(ctx, http.StatusInternalServerError)
			return &api.GetRestoreResp{Success: false, Message: &message}, nil
		}
	}

	return &api.GetRestoreResp{Success: true, Data: info}, nil
}

func convertToRestoreInfo(restore *v1alpha1.Restore) *api.RestoreInfo {
	info := &api.RestoreInfo{
		ClusterId: restore.Namespace,
		RestoreId: restore.Name,
		Status:    string(restore.Status.Phase),
		CommitTs:  restore.Status.CommitTs,
	}

	if !restore.Status.TimeStarted.IsZero() {
		info.StartedAt = restore.Status.TimeStarted.Format(time.RFC3339)
	}
	if !restore.Status.TimeCompleted.IsZero() {
		info.CompletedAt = restore.Status.TimeCompleted.Format(time.RFC3339)
	}

	return info
}

func (s *ClusterServer) StopBackup(ctx context.Context, req *api.StopBackupReq) (*api.StopBackupResp, error) {
	k8sID := getKubernetesID(ctx)
	opCli := s.KubeClient.GetOperatorClient(k8sID)
	kubeCli := s.KubeClient.GetKubeClient(k8sID)
	logger := log.L().With(zap.String("request", "StopBackup"), zap.String("k8sID", k8sID),
		zap.String("clusterID", req.ClusterId), zap.String("backupID", req.BackupId))
	if opCli == nil || kubeCli == nil {
		logger.Error("K8s client not found")
		message := fmt.Sprintf("no %s is specified in the request header or the kubeconfig context not exists", HeaderKeyKubernetesID)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.StopBackupResp{Success: false, Message: &message}, nil
	}

	// check whether the backup exists
	backup, err := opCli.PingcapV1alpha1().Backups(req.ClusterId).Get(ctx, req.BackupId, metav1.GetOptions{})
	if err != nil {
		logger.Error("Backup not found", zap.Error(err))
		message := fmt.Sprintf("Backup %s not found", req.BackupId)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.StopBackupResp{Success: false, Message: &message}, nil
	}

	// stop backup
	if backup.Spec.Mode == v1alpha1.BackupModeLog {
		backup.Spec.LogStop = true
		_, err = opCli.PingcapV1alpha1().Backups(req.ClusterId).Update(ctx, backup, metav1.UpdateOptions{})
		if err != nil {
			logger.Error("Stop log backup failed", zap.Error(err))
			message := fmt.Sprintf("Stop log backup failed: %s", err.Error())
			setResponseStatusCodes(ctx, http.StatusInternalServerError)
			return &api.StopBackupResp{Success: false, Message: &message}, nil
		}
	} else {
		_, err := kubeCli.BatchV1().Jobs(backup.GetNamespace()).Get(ctx, backup.GetBackupJobName(), metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				logger.Warn("Backup is already Stopped", zap.Error(err))
				message := fmt.Sprintf("Backup %s is already Stopped", req.BackupId)
				setResponseStatusCodes(ctx, http.StatusNotFound)
				return &api.StopBackupResp{Success: false, Message: &message}, nil
			}
			logger.Error("Get backup job failed", zap.Error(err))
			message := fmt.Sprintf("Get backup job failed: %s", err.Error())
			setResponseStatusCodes(ctx, http.StatusInternalServerError)
			return &api.StopBackupResp{Success: false, Message: &message}, nil
		}

		err = kubeCli.BatchV1().Jobs(backup.GetNamespace()).Delete(ctx, backup.GetBackupJobName(), metav1.DeleteOptions{})
		if err != nil {
			logger.Error("Stop backup failed", zap.Error(err))
			message := fmt.Sprintf("Stop backup failed: %s", err.Error())
			setResponseStatusCodes(ctx, http.StatusInternalServerError)
			return &api.StopBackupResp{Success: false, Message: &message}, nil
		}
	}

	return &api.StopBackupResp{Success: true}, nil
}

func (s *ClusterServer) StopRestore(ctx context.Context, req *api.StopRestoreReq) (*api.StopRestoreResp, error) {
	k8sID := getKubernetesID(ctx)
	opCli := s.KubeClient.GetOperatorClient(k8sID)
	kubeCli := s.KubeClient.GetKubeClient(k8sID)
	logger := log.L().With(zap.String("request", "StopRestore"), zap.String("k8sID", k8sID),
		zap.String("clusterID", req.ClusterId), zap.String("storeID", req.RestoreId))
	if opCli == nil || kubeCli == nil {
		logger.Error("K8s client not found")
		message := fmt.Sprintf("no %s is specified in the request header or the kubeconfig context not exists", HeaderKeyKubernetesID)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.StopRestoreResp{Success: false, Message: &message}, nil
	}

	// check whether the restore exists
	restore, err := opCli.PingcapV1alpha1().Restores(req.ClusterId).Get(ctx, req.RestoreId, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Restore is already Stopped", zap.Error(err))
			message := fmt.Sprintf("Restore %s is already Stopped", req.RestoreId)
			setResponseStatusCodes(ctx, http.StatusNotFound)
			return &api.StopRestoreResp{Success: false, Message: &message}, nil
		}
		logger.Error("Restore not found", zap.Error(err))
		message := fmt.Sprintf("Restore %s not found", req.RestoreId)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.StopRestoreResp{Success: false, Message: &message}, nil
	}

	// stop restore
	_, err = kubeCli.BatchV1().Jobs(restore.GetNamespace()).Get(ctx, restore.GetRestoreJobName(), metav1.GetOptions{})
	if err != nil {
		logger.Error("Get restore job failed", zap.Error(err))
		message := fmt.Sprintf("Get restore job failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.StopRestoreResp{Success: false, Message: &message}, nil
	}

	err = kubeCli.BatchV1().Jobs(restore.GetNamespace()).Delete(ctx, restore.GetRestoreJobName(), metav1.DeleteOptions{})
	if err != nil {
		logger.Error("Stop restore failed", zap.Error(err))
		message := fmt.Sprintf("Stop restore failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.StopRestoreResp{Success: false, Message: &message}, nil
	}

	return &api.StopRestoreResp{Success: true}, nil
}

func (s *ClusterServer) DeleteBackup(ctx context.Context, req *api.DeleteBackupReq) (*api.DeleteBackupResp, error) {
	k8sID := getKubernetesID(ctx)
	opCli := s.KubeClient.GetOperatorClient(k8sID)
	logger := log.L().With(zap.String("request", "DeleteBackup"), zap.String("k8sID", k8sID),
		zap.String("clusterID", req.ClusterId), zap.String("backupID", req.BackupId))
	if opCli == nil {
		logger.Error("K8s client not found")
		message := fmt.Sprintf("no %s is specified in the request header or the kubeconfig context not exists", HeaderKeyKubernetesID)
		setResponseStatusCodes(ctx, http.StatusBadRequest)
		return &api.DeleteBackupResp{Success: false, Message: &message}, nil
	}

	// check whether the backup exists first
	_, err := opCli.PingcapV1alpha1().Backups(req.ClusterId).Get(ctx, req.BackupId, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Backup not found", zap.Error(err))
			message := fmt.Sprintf("Backup %s not found", req.BackupId)
			setResponseStatusCodes(ctx, http.StatusNotFound)
			return &api.DeleteBackupResp{Success: false, Message: &message}, nil
		}
		logger.Error("Get backup failed", zap.Error(err))
		message := fmt.Sprintf("get backup failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.DeleteBackupResp{Success: false, Message: &message}, nil
	}

	// delete backup
	err = opCli.PingcapV1alpha1().Backups(req.ClusterId).Delete(ctx, req.BackupId, metav1.DeleteOptions{})
	if err != nil {
		logger.Error("Delete backup failed", zap.Error(err))
		message := fmt.Sprintf("delete backup failed: %s", err.Error())
		setResponseStatusCodes(ctx, http.StatusInternalServerError)
		return &api.DeleteBackupResp{Success: false, Message: &message}, nil
	}

	return &api.DeleteBackupResp{Success: true}, nil
}
