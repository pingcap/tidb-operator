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
	maxRetries = 3 // number of retries to make of operations
)

type s3Query struct {
	region         string
	endpoint       string
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
		qs := checkS3Config(backup.Spec.S3, true)
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
func getRemoteStorage(provider v1alpha1.StorageProvider) ([]string, string, error) {
	st := util.GetStorageType(provider)
	switch st {
	case v1alpha1.BackupStorageTypeS3:
		qs := checkS3Config(provider.S3, false)
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
	awsConfig := aws.NewConfig().WithMaxRetries(maxRetries).
		WithS3ForcePathStyle(qs.forcePathStyle)
	if qs.region != "" {
		awsConfig.WithRegion(qs.region)
	}
	if qs.endpoint != "" {
		awsConfig.WithEndpoint(qs.endpoint)
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
func checkS3Config(s3 *v1alpha1.S3StorageProvider, fakeRegion bool) *s3Query {
	sqs := s3Query{}

	sqs.bucket = s3.Bucket
	sqs.region = s3.Region
	sqs.provider = string(s3.Provider)
	sqs.prefix = s3.Prefix
	sqs.endpoint = s3.Endpoint
	sqs.sse = s3.SSE
	sqs.acl = s3.Acl
	sqs.storageClass = s3.StorageClass
	sqs.forcePathStyle = true
	// In some cases, we need to set ForcePathStyle to false.
	// Refer to: https://rclone.org/s3/#s3-force-path-style
	// if UseAccelerateEndpoint is supported for AWS s3 in future,
	// need to set forcePathStyle = false too.
	if sqs.provider == "alibaba" || sqs.provider == "netease" {
		sqs.forcePathStyle = false
	}
	if fakeRegion && sqs.region == "" {
		sqs.region = "us-east-1"
	}
	sqs.prefix = strings.Trim(sqs.prefix, "/")
	sqs.prefix += "/"

	return &sqs
}
