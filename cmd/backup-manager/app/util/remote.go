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
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"gocloud.dev/blob"
	"gocloud.dev/blob/fileblob"
	"gocloud.dev/blob/gcsblob"
	"gocloud.dev/blob/s3blob"
	"gocloud.dev/gcp"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/util"
)

const (
	maxRetries = 3 // number of retries to make of operations
)

type s3Config struct {
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

type gcsConfig struct {
	projectId    string
	location     string
	bucket       string
	storageClass string
	objectAcl    string
	bucketAcl    string
	secretName   string
	prefix       string
}

type localConfig struct {
	mountPath string
	prefix    string
}

// NewStorageBackend creates new storage backend, now supports S3/GCS/Local
func NewStorageBackend(provider v1alpha1.StorageProvider) (*blob.Bucket, error) {
	st := util.GetStorageType(provider)
	switch st {
	case v1alpha1.BackupStorageTypeS3:
		conf := makeS3Config(provider.S3, true)
		bucket, err := newS3Storage(conf)
		if err != nil {
			return nil, err
		}
		return bucket, nil
	case v1alpha1.BackupStorageTypeGcs:
		conf := makeGcsConfig(provider.Gcs, true)
		bucket, err := newGcsStorage(conf)
		if err != nil {
			return nil, err
		}
		return bucket, nil
	case v1alpha1.BackupStorageTypeLocal:
		conf := makeLocalConfig(provider.Local)
		bucket, err := newLocalStorage(conf)
		if err != nil {
			return nil, err
		}
		return bucket, nil
	default:
		return nil, fmt.Errorf("storage %s not supported yet", st)
	}
}

// genStorageArgs returns the arg for --storage option and the remote/local path for br
// TODO: add unit test
func genStorageArgs(provider v1alpha1.StorageProvider) ([]string, error) {
	st := util.GetStorageType(provider)
	switch st {
	case v1alpha1.BackupStorageTypeS3:
		qs := makeS3Config(provider.S3, false)
		s := newS3StorageOption(qs)
		return s, nil
	case v1alpha1.BackupStorageTypeGcs:
		qs := makeGcsConfig(provider.Gcs, false)
		s := newGcsStorageOption(qs)
		return s, nil
	case v1alpha1.BackupStorageTypeLocal:
		localConfig := makeLocalConfig(provider.Local)
		cmdOpts, err := newLocalStorageOption(localConfig)
		if err != nil {
			return nil, err
		}
		return cmdOpts, nil
	default:
		return nil, fmt.Errorf("storage %s not supported yet", st)
	}
}

// newLocalStorageOption constructs `--storage local://$PATH` arg for br
func newLocalStorageOption(conf localConfig) ([]string, error) {
	return []string{fmt.Sprintf("--storage=local://%s", path.Join(conf.mountPath, conf.prefix))}, nil
}

// newS3StorageOption constructs the arg for --storage option and the remote path for br
func newS3StorageOption(conf *s3Config) []string {
	var s3options []string
	path := fmt.Sprintf("s3://%s", path.Join(conf.bucket, conf.prefix))
	s3options = append(s3options, fmt.Sprintf("--storage=%s", path))
	if conf.region != "" {
		s3options = append(s3options, fmt.Sprintf("--s3.region=%s", conf.region))
	}
	if conf.provider != "" {
		s3options = append(s3options, fmt.Sprintf("--s3.provider=%s", conf.provider))
	}
	if conf.endpoint != "" {
		s3options = append(s3options, fmt.Sprintf("--s3.endpoint=%s", conf.endpoint))
	}
	if conf.sse != "" {
		s3options = append(s3options, fmt.Sprintf("--s3.sse=%s", conf.sse))
	}
	if conf.acl != "" {
		s3options = append(s3options, fmt.Sprintf("--s3.acl=%s", conf.acl))
	}
	if conf.storageClass != "" {
		s3options = append(s3options, fmt.Sprintf("--s3.storage-class=%s", conf.storageClass))
	}
	return s3options
}

func newLocalStorage(conf localConfig) (*blob.Bucket, error) {
	dir := path.Join(conf.mountPath, conf.prefix)
	bucket, err := fileblob.OpenBucket(dir, nil)
	return bucket, err
}

// newS3Storage initialize a new s3 storage
func newS3Storage(conf *s3Config) (*blob.Bucket, error) {
	awsConfig := aws.NewConfig().WithMaxRetries(maxRetries).
		WithS3ForcePathStyle(conf.forcePathStyle)
	if conf.region != "" {
		awsConfig.WithRegion(conf.region)
	}
	if conf.endpoint != "" {
		awsConfig.WithEndpoint(conf.endpoint)
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
	bkt, err := s3blob.OpenBucket(context.Background(), ses, conf.bucket, nil)
	if err != nil {
		return nil, err
	}
	return blob.PrefixedBucket(bkt, strings.Trim(conf.prefix, "/")+"/"), nil

}

// newGcsStorage initialize a new gcs storage
func newGcsStorage(conf *gcsConfig) (*blob.Bucket, error) {
	ctx := context.Background()

	// Your GCP credentials.
	creds, err := gcp.DefaultCredentials(ctx)
	if err != nil {
		return nil, err
	}

	// Create an HTTP client.
	client, err := gcp.NewHTTPClient(
		gcp.DefaultTransport(),
		gcp.CredentialsTokenSource(creds))
	if err != nil {
		return nil, err
	}

	// Create a *blob.Bucket.
	bucket, err := gcsblob.OpenBucket(ctx, client, conf.bucket, nil)
	if err != nil {
		return nil, err
	}
	return blob.PrefixedBucket(bucket, strings.Trim(conf.prefix, "/")+"/"), nil
}

// newGcsStorageOption constructs the arg for --storage option and the remote path for br
func newGcsStorageOption(conf *gcsConfig) []string {
	var gcsoptions []string
	path := fmt.Sprintf("gcs://%s/", path.Join(conf.bucket, conf.prefix))
	gcsoptions = append(gcsoptions, fmt.Sprintf("--storage=%s", path))
	if conf.storageClass != "" {
		gcsoptions = append(gcsoptions, fmt.Sprintf("--gcs.storage-class=%s", conf.storageClass))
	}
	if conf.objectAcl != "" {
		gcsoptions = append(gcsoptions, fmt.Sprintf("--gcs.predefined-acl=%s", conf.objectAcl))
	}
	return gcsoptions
}

// makeS3Config constructs s3Config parameters
func makeS3Config(s3 *v1alpha1.S3StorageProvider, fakeRegion bool) *s3Config {
	conf := s3Config{}

	path := strings.Trim(s3.Bucket, "/") + "/" + strings.Trim(s3.Prefix, "/")
	fields := strings.SplitN(path, "/", 2)

	conf.bucket = fields[0]
	conf.region = s3.Region
	conf.provider = string(s3.Provider)
	conf.prefix = fields[1]
	conf.endpoint = s3.Endpoint
	conf.sse = s3.SSE
	conf.acl = s3.Acl
	conf.storageClass = s3.StorageClass
	conf.forcePathStyle = true
	// In some cases, we need to set ForcePathStyle to false.
	// Refer to: https://rclone.org/s3/#s3-force-path-style
	// if UseAccelerateEndpoint is supported for AWS s3 in future,
	// need to set forcePathStyle = false too.
	if conf.provider == "alibaba" || conf.provider == "netease" {
		conf.forcePathStyle = false
	}
	if fakeRegion && conf.region == "" {
		conf.region = "us-east-1"
	}

	return &conf
}

// makeGcsConfig constructs gcsConfig parameters
func makeGcsConfig(gcs *v1alpha1.GcsStorageProvider, fakeRegion bool) *gcsConfig {
	conf := gcsConfig{}

	path := strings.Trim(gcs.Bucket, "/") + "/" + strings.Trim(gcs.Prefix, "/")
	fields := strings.SplitN(path, "/", 2)

	conf.bucket = fields[0]
	conf.location = gcs.Location
	conf.projectId = gcs.ProjectId
	conf.storageClass = gcs.StorageClass
	conf.objectAcl = gcs.ObjectAcl
	conf.bucketAcl = gcs.BucketAcl
	conf.secretName = gcs.SecretName
	conf.prefix = fields[1]

	return &conf
}

func makeLocalConfig(local *v1alpha1.LocalStorageProvider) localConfig {
	return localConfig{
		mountPath: local.VolumeMount.MountPath,
		prefix:    local.Prefix,
	}
}
