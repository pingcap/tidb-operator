// Copyright 2020 PingCAP, Inc.
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

package storage

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/minio/minio-go/v6"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
)

const (
	minioPodName = "minio"
)

type minioStorage struct {
	kubecli     clientset.Interface
	minioClient *minio.Client
	accessKey   string
	secretKey   string
	s3config    *v1alpha1.S3StorageProvider
}

func NewMinioStorage(fw portforward.PortForward, ns, accessKey, secretKey string, cli clientset.Interface, s3config *v1alpha1.S3StorageProvider) (*minioStorage, context.CancelFunc, error) {
	cmd := fmt.Sprintf(`kubectl apply -f /minio/minio.yaml -n %s`, ns)
	if data, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
		return nil, nil, fmt.Errorf("failed to install minio %s %v", string(data), err)
	}
	err := e2epod.WaitTimeoutForPodReadyInNamespace(cli, minioPodName, ns, 5*time.Minute)
	if err != nil {
		return nil, nil, err
	}
	localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, ns, "svc/minio-service", 9000)
	if err != nil {
		return nil, nil, err
	}
	ep := fmt.Sprintf("%s:%d", localHost, localPort)
	client, err := minio.New(ep, accessKey, secretKey, false)
	if err != nil {
		return nil, nil, err
	}
	m := &minioStorage{
		kubecli:     cli,
		minioClient: client,
		accessKey:   accessKey,
		secretKey:   secretKey,
		s3config:    s3config,
	}
	bucketName := m.s3config.Bucket
	err = m.minioClient.MakeBucket(bucketName, "")
	if err != nil {
		return nil, nil, err
	}
	return m, cancel, nil
}

func (m *minioStorage) ProvideCredential(ns string) *corev1.Secret {
	return fixture.GetS3Secret(ns, m.accessKey, m.secretKey)
}

func (m *minioStorage) ProvideBackup(tc *v1alpha1.TidbCluster, fromSecret *corev1.Secret, brType string) *v1alpha1.Backup {
	return fixture.GetBackupCRDWithS3(tc, fromSecret.Name, brType, m.s3config)
}

func (m *minioStorage) ProvideRestore(tc *v1alpha1.TidbCluster, toSecret *corev1.Secret, restoreType string) *v1alpha1.Restore {
	return fixture.GetRestoreCRDWithS3(tc, toSecret.Name, restoreType, m.s3config)
}

func (m *minioStorage) CheckDataCleaned() error {
	return wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
		doneCh := make(chan struct{})
		defer close(doneCh)
		objs := m.minioClient.ListObjects(m.s3config.Bucket, m.s3config.Prefix, true, doneCh)
		if len(objs) == 0 {
			return true, nil
		}
		return false, nil
	})
}
