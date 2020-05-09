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

package tidbcluster

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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

type storage interface {
	provideCredential(ns string) *corev1.Secret
	provideBackup(tc *v1alpha1.TidbCluster, fromSecret *corev1.Secret) *v1alpha1.Backup
	provideRestore(tc *v1alpha1.TidbCluster, toSecret *corev1.Secret) *v1alpha1.Restore
	checkDataCleaned() error
}

type minioStorage struct {
	kubecli     clientset.Interface
	minioClient *minio.Client
	accessKey   string
	secretKey   string
	s3config    *v1alpha1.S3StorageProvider
}

func newMinioStorage(fw portforward.PortForward, ns, accessKey, secretKey string, cli clientset.Interface, s3config *v1alpha1.S3StorageProvider) (*minioStorage, context.CancelFunc, error) {
	cmd := fmt.Sprintf(`kubectl apply -f /minio/minio.yaml -n %s`, ns)
	if data, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
		return nil, nil, fmt.Errorf("failed to install minio %s %v", string(data), err)
	}
	err := e2epod.WaitTimeoutForPodReadyInNamespace(cli, minioPodName, ns, 5*time.Minute)
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

func (m *minioStorage) provideCredential(ns string) *corev1.Secret {
	return fixture.GetS3Secret(ns, m.accessKey, m.secretKey)
}

func (m *minioStorage) provideBackup(tc *v1alpha1.TidbCluster, fromSecret *corev1.Secret) *v1alpha1.Backup {
	return fixture.GetBackupCRDForBRWithS3(tc, fromSecret.Name, m.s3config)
}

func (m *minioStorage) provideRestore(tc *v1alpha1.TidbCluster, toSecret *corev1.Secret) *v1alpha1.Restore {
	return fixture.GetRestoreCRDForBRWithS3(tc, toSecret.Name, m.s3config)
}

func (m *minioStorage) checkDataCleaned() error {
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

type s3Storage struct {
	cred      *credentials.Credentials
	s3config  *v1alpha1.S3StorageProvider
	accessKey string
	secretKey string
}

func newS3Storage(cred *credentials.Credentials, s3config *v1alpha1.S3StorageProvider) (*s3Storage, error) {
	val, err := cred.Get()
	if err != nil {
		return nil, err
	}
	return &s3Storage{
		cred:      cred,
		s3config:  s3config,
		accessKey: val.AccessKeyID,
		secretKey: val.SecretAccessKey,
	}, nil
}

func (s *s3Storage) provideCredential(ns string) *corev1.Secret {
	return fixture.GetS3Secret(ns, s.accessKey, s.secretKey)
}

func (s *s3Storage) provideBackup(tc *v1alpha1.TidbCluster, fromSecret *corev1.Secret) *v1alpha1.Backup {
	return fixture.GetBackupCRDForBRWithS3(tc, fromSecret.Name, s.s3config)
}

func (s *s3Storage) provideRestore(tc *v1alpha1.TidbCluster, toSecret *corev1.Secret) *v1alpha1.Restore {
	return fixture.GetRestoreCRDForBRWithS3(tc, toSecret.Name, s.s3config)
}

func (s *s3Storage) checkDataCleaned() error {
	return wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
		awsConfig := aws.NewConfig().
			WithRegion(s.s3config.Region).
			WithCredentials(s.cred)
		svc := s3.New(session.Must(session.NewSession(awsConfig)))
		input := &s3.ListObjectsV2Input{
			Bucket: aws.String(s.s3config.Bucket),
			Prefix: aws.String(s.s3config.Prefix),
		}
		result, err := svc.ListObjectsV2(input)
		if err != nil {
			return false, err
		}
		if *result.KeyCount != 0 {
			return false, nil
		}
		return true, nil
	})
}

func installAndWaitMinio(ns string, cli clientset.Interface) error {
	cmd := fmt.Sprintf(`kubectl apply -f /minio/minio.yaml -n %s`, ns)
	if data, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install minio %s %v", string(data), err)
	}
	return e2epod.WaitTimeoutForPodReadyInNamespace(cli, minioPodName, ns, 5*time.Minute)
}

func createMinioClient(fw portforward.PortForward, ns, accessKey, secretKey string, ssl bool) (*minio.Client, context.CancelFunc, error) {
	localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, ns, "svc/minio-service", 9000)
	if err != nil {
		return nil, nil, err
	}
	ep := fmt.Sprintf("%s:%d", localHost, localPort)
	client, err := minio.New(ep, accessKey, secretKey, ssl)
	if err != nil {
		return nil, nil, err
	}
	return client, cancel, nil
}
