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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type s3Storage struct {
	cred      *credentials.Credentials
	s3config  *v1alpha1.S3StorageProvider
	accessKey string
	secretKey string
}

func NewS3Storage(cred *credentials.Credentials, s3config *v1alpha1.S3StorageProvider) (*s3Storage, error) {
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

func (s *s3Storage) ProvideCredential(ns string) *corev1.Secret {
	return fixture.GetS3Secret(ns, s.accessKey, s.secretKey)
}

func (s *s3Storage) ProvideBackup(tc *v1alpha1.TidbCluster, fromSecret *corev1.Secret, t string) *v1alpha1.Backup {
	return fixture.GetBackupCRDWithS3(tc, fromSecret.Name, t, s.s3config)
}

func (s *s3Storage) ProvideRestore(tc *v1alpha1.TidbCluster, toSecret *corev1.Secret, t string) *v1alpha1.Restore {
	return fixture.GetRestoreCRDWithS3(tc, toSecret.Name, t, s.s3config)
}

func (s *s3Storage) CheckDataCleaned() error {
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
