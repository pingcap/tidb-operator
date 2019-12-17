// Copyright 2019 PingCAP, Inc.
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

package util

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/util"
)

const (
	accessKey          = "access_key"
	secretAccessKey    = "secret_access_key"
	regionKey          = "region"
	insecureKey        = "insecure"
	providerKey        = "provider"
	prefixKey          = "prefix"
	endpointKey        = "endpoint"
	awsKey             = "aws"
	aliKey             = "alibaba"
	accessKeyEnv       = "AWS_ACCESS_KEY_ID"
	secretAccessKeyEnv = "AWS_SECRET_ACCESS_KEY"

	maxRetries = 3 // number of retries to make of operations
)

type s3Query struct {
	region         string
	endpoint       string
	oriEndpoint    string
	bucket         string
	prefix         string
	provider       string
	sse            string
	acl            string
	storageClass   string
	forcePathStyle bool
}

// NewRemoteStorage creates new remote storage
func NewRemoteStorage(backup *v1alpha1.Backup) (*blob.Bucket, error) {
	st := util.GetStorageType(backup.Spec.StorageProvider)
	switch st {
	case v1alpha1.BackupStorageTypeS3:
		qs, err := checkS3Config(backup, true)
		if err != nil {
			return nil, err
		}
		bucket, err := newS3Storage(qs)
		if err != nil {
			return nil, err
		}
		return bucket, nil
	default:
		return nil, fmt.Errorf("storage %s not support yet", st)
	}
}

// getRemoteStorage returns the arg for --storage option and the remote path for br
func getRemoteStorage(backup *v1alpha1.Backup) ([]string, string, error) {
	st := util.GetStorageType(backup.Spec.StorageProvider)
	switch st {
	case "s3":
		qs, err := checkS3Config(backup, false)
		if err != nil {
			return nil, "", err
		}
		s, path := newS3StorageOption(qs)
		return s, path, nil
	default:
		return nil, "", fmt.Errorf("storage %s not support yet", st)
	}
}

// newS3StorageOption constructs the arg for --storage option and the remote path for br
func newS3StorageOption(qs *s3Query) ([]string, string) {
	var s3options []string
	var path string
	if qs.prefix == "/" {
		path = fmt.Sprintf("s3://%s%s", qs.bucket, qs.prefix)
	} else {
		path = fmt.Sprintf("s3://%s/%s", qs.bucket, qs.prefix)
	}
	s3options = append(s3options, fmt.Sprintf("--storage=%s", path))
	if qs.region != "" {
		s3options = append(s3options, fmt.Sprintf("--s3.region=%s", qs.region))
	}
	if qs.provider != "" {
		s3options = append(s3options, fmt.Sprintf("--s3.provider=%s", qs.provider))
	}
	if qs.endpoint != "" {
		s3options = append(s3options, fmt.Sprintf("--s3.endpoint=%s", qs.endpoint))
	}
	if qs.sse != "" {
		s3options = append(s3options, fmt.Sprintf("--s3.sse=%s", qs.sse))
	}
	if qs.acl != "" {
		s3options = append(s3options, fmt.Sprintf("--s3.acl=%s", qs.acl))
	}
	if qs.storageClass != "" {
		s3options = append(s3options, fmt.Sprintf("--s3.storage-class=%s", qs.storageClass))
	}
	return s3options, path
}

// newS3Storage initialize a new s3 storage
func newS3Storage(qs *s3Query) (*blob.Bucket, error) {
	awsConfig := aws.NewConfig().WithMaxRetries(maxRetries).WithS3ForcePathStyle(qs.forcePathStyle)
	if qs.region != "" {
		awsConfig.WithRegion(qs.region)
	}
	if qs.oriEndpoint != "" {
		awsConfig.WithEndpoint(qs.oriEndpoint)
	}
	// awsConfig.WithLogLevel(aws.LogDebugWithSigning)
	awsSessionOpts := session.Options{
		Config: *awsConfig,
	}
	ses, err := session.NewSessionWithOptions(awsSessionOpts)
	if err != nil {
		return nil, err
	}

	// Create a *blob.Bucket.
	bkt, err := s3blob.OpenBucket(context.Background(), ses, qs.bucket, nil)
	if err != nil {
		return nil, err
	}
	return blob.PrefixedBucket(bkt, qs.prefix), nil

}

// checkS3Config constructs s3Query parameters
func checkS3Config(backup *v1alpha1.Backup, fakeRegion bool) (*s3Query, error) {
	sqs := s3Query{}

	if backup.Spec.S3 == nil {
		return nil, fmt.Errorf("no s3 config in backup %s/%s", backup.Namespace, backup.Name)
	}
	sqs.bucket = backup.Spec.S3.Bucket
	sqs.region = backup.Spec.S3.Region
	sqs.provider = string(backup.Spec.S3.Provider)
	sqs.prefix = backup.Spec.S3.Prefix
	sqs.endpoint = backup.Spec.S3.Endpoint
	sqs.oriEndpoint = backup.Spec.S3.Endpoint
	sqs.sse = backup.Spec.S3.SSE
	sqs.acl = backup.Spec.S3.Acl
	sqs.storageClass = backup.Spec.S3.StorageClass
	sqs.forcePathStyle = true
	// In some cases, we need to set ForcePathStyle to false.
	// Refer to: https://rclone.org/s3/#s3-force-path-style
	if sqs.provider == "alibaba" || sqs.provider == "netease" {
		sqs.forcePathStyle = false
	}
	if fakeRegion && sqs.region == "" {
		sqs.region = "us-east-1"
	}
	sqs.prefix = strings.Trim(sqs.prefix, "/")
	sqs.prefix += "/"

	return &sqs, nil
}

// ConstructBRGlobalOptions constructs global options for BR and also return the remote path
func ConstructBRGlobalOptions(backup *v1alpha1.Backup) ([]string, string, error) {
	var args []string
	config := backup.Spec.BR
	if config == nil {
		return nil, "", fmt.Errorf("no config for br in backup %s/%s", backup.Namespace, backup.Name)
	}
	args = append(args, fmt.Sprintf("--pd=%s", config.PDAddress))
	if config.CA != "" {
		args = append(args, fmt.Sprintf("--ca=%s", config.CA))
	}
	if config.Cert != "" {
		args = append(args, fmt.Sprintf("--cert=%s", config.Cert))
	}
	if config.Key != "" {
		args = append(args, fmt.Sprintf("--key=%s", config.Key))
	}
	// Do not set log-file, backup-manager needs to get backup
	// position from the output of BR with info log-level
	// if config.LogFile != "" {
	// 	args = append(args, fmt.Sprintf("--log-file=%s", config.LogFile))
	// }
	args = append(args, "--log-level=info")
	if config.StatusAddr != "" {
		args = append(args, fmt.Sprintf("--status-addr=%s", config.StatusAddr))
	}
	s, path, err := getRemoteStorage(backup)
	if err != nil {
		return nil, "", err
	}
	args = append(args, s...)
	return args, path, nil
}
