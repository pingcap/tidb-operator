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
	"io"
	"path"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"gocloud.dev/blob"
	"gocloud.dev/blob/fileblob"
	"gocloud.dev/blob/gcsblob"
	"gocloud.dev/blob/s3blob"
	"gocloud.dev/gcp"
	"k8s.io/client-go/util/workqueue"

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
	path         string
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

// StorageBackend wrap blob.Bucket and provide more function
type StorageBackend struct {
	*blob.Bucket

	provider v1alpha1.StorageProvider
}

// NewStorageBackend creates new storage backend, now supports S3/GCS/Local
func NewStorageBackend(provider v1alpha1.StorageProvider) (*StorageBackend, error) {
	var bucket *blob.Bucket
	var err error

	st := util.GetStorageType(provider)
	switch st {
	case v1alpha1.BackupStorageTypeS3:
		conf := makeS3Config(provider.S3, true)
		bucket, err = newS3Storage(conf)
	case v1alpha1.BackupStorageTypeGcs:
		conf := makeGcsConfig(provider.Gcs, true)
		bucket, err = newGcsStorage(conf)
	case v1alpha1.BackupStorageTypeLocal:
		conf := makeLocalConfig(provider.Local)
		bucket, err = newLocalStorage(conf)
	default:
		err = fmt.Errorf("storage %s not supported yet", st)
	}

	if err != nil {
		return nil, err
	}

	b := &StorageBackend{
		Bucket:   bucket,
		provider: provider,
	}

	return b, nil
}

func (b *StorageBackend) StorageType() v1alpha1.BackupStorageType {
	return util.GetStorageType(b.provider)
}

func (b *StorageBackend) ListPage(opts *blob.ListOptions) *PageIterator {
	return &PageIterator{
		iter: b.Bucket.List(opts),
	}
}

func (b *StorageBackend) AsS3() (*s3.S3, bool) {
	var s3cli *s3.S3
	if ok := b.Bucket.As(&s3cli); !ok {
		return nil, false
	}

	return s3cli, true
}

func (b *StorageBackend) AsGCS() (*storage.Client, bool) {
	var gcsClient *storage.Client
	if ok := b.As(&gcsClient); !ok {
		return nil, false
	}

	return gcsClient, true
}

// GetBucket return bucket name
//
// If provider is S3/GCS, return bucket. Otherwise return empty string
func (b *StorageBackend) GetBucket() string {
	if b.provider.S3 != nil {
		return b.provider.S3.Bucket
	} else if b.provider.Gcs != nil {
		return b.provider.Gcs.Bucket
	}

	return ""
}

// GetPrefix return prefix
func (b *StorageBackend) GetPrefix() string {
	if b.provider.S3 != nil {
		return b.provider.S3.Prefix
	} else if b.provider.Gcs != nil {
		return b.provider.Gcs.Prefix
	} else if b.provider.Local != nil {
		return b.provider.Local.Prefix
	}

	return ""
}

type ObjectError struct {
	Key string
	Err error
}

type BatchDeleteObjectsOption struct {
	BatchConcurrency   int
	RoutineConcurrency int
}

type BatchDeleteObjectsResult struct {
	Deleted []string
	Errors  []ObjectError
}

// BatchDeleteObjects delete mutli objects
//
// Depending on storage type, it use function 'BatchDeleteObjectsOfS3' or 'BatchDeleteObjectsConcurrently'
func (b *StorageBackend) BatchDeleteObjects(ctx context.Context, objs []*blob.ListObject, opt *BatchDeleteObjectsOption) *BatchDeleteObjectsResult {
	var result *BatchDeleteObjectsResult

	s3cli, ok := b.AsS3()
	if ok {
		concurrency := 10
		if opt != nil && opt.BatchConcurrency != 0 {
			concurrency = opt.BatchConcurrency
		}
		result = BatchDeleteObjectsOfS3(ctx, s3cli, objs, b.GetBucket(), b.GetPrefix(), concurrency)
	} else {
		concurrency := 100
		if opt != nil && opt.RoutineConcurrency != 0 {
			concurrency = opt.RoutineConcurrency
		}
		result = BatchDeleteObjectsConcurrently(ctx, b.Bucket, objs, concurrency)
	}

	return result
}

// BatchDeleteObjectsOfS3 delete objects by batch delete api
func BatchDeleteObjectsOfS3(ctx context.Context, s3cli s3iface.S3API, objs []*blob.ListObject, bucket string, prefix string, concurrency int) *BatchDeleteObjectsResult {
	mu := &sync.Mutex{}
	result := &BatchDeleteObjectsResult{}
	batchSize := 1000
	if prefix != "" {
		prefix = strings.Trim(prefix, "/") + "/"
	}

	workqueue.ParallelizeUntil(ctx, concurrency, len(objs)/batchSize+1, func(piece int) {
		start := piece * batchSize
		end := (piece + 1) * batchSize
		if end > len(objs) {
			end = len(objs)
		}

		if len(objs[start:end]) == 0 {
			return
		}

		delete := &s3.Delete{}
		for _, obj := range objs[start:end] {
			delete.Objects = append(delete.Objects, &s3.ObjectIdentifier{
				Key: aws.String(prefix + obj.Key),
			})
		}
		input := &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: delete,
		}
		resp, err := s3cli.DeleteObjectsWithContext(ctx, input)

		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			// send request failed, consider that all deletion is failed
			for _, obj := range objs[start:end] {
				result.Errors = append(result.Errors, ObjectError{
					Key: obj.Key,
					Err: err,
				})
			}
		} else {
			for _, deleted := range resp.Deleted {
				result.Deleted = append(result.Deleted, *deleted.Key)
			}
			for _, rerr := range resp.Errors {
				result.Errors = append(result.Errors, ObjectError{
					Key: *rerr.Key,
					Err: fmt.Errorf("code:%s msg:%s", *rerr.Code, *rerr.Message),
				})
			}
		}
	})

	return result
}

// BatchDeleteObjectsConcurrently delete objects by multiple goroutine concurrently
func BatchDeleteObjectsConcurrently(ctx context.Context, bucket *blob.Bucket, objs []*blob.ListObject, concurrency int) *BatchDeleteObjectsResult {
	mu := &sync.Mutex{}
	result := &BatchDeleteObjectsResult{}

	workqueue.ParallelizeUntil(ctx, concurrency, len(objs), func(piece int) {
		key := objs[piece].Key
		err := bucket.Delete(ctx, key)

		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			result.Errors = append(result.Errors, ObjectError{
				Key: key,
				Err: err,
			})
		} else {
			result.Deleted = append(result.Deleted, key)
		}
	})

	return result
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

	conf.bucket = s3.Bucket
	conf.region = s3.Region
	conf.provider = string(s3.Provider)
	conf.prefix = s3.Prefix
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

	conf.bucket = gcs.Bucket
	conf.location = gcs.Location
	conf.path = gcs.Path
	conf.projectId = gcs.ProjectId
	conf.storageClass = gcs.StorageClass
	conf.objectAcl = gcs.ObjectAcl
	conf.bucketAcl = gcs.BucketAcl
	conf.secretName = gcs.SecretName
	conf.prefix = gcs.Prefix

	return &conf
}

func makeLocalConfig(local *v1alpha1.LocalStorageProvider) localConfig {
	return localConfig{
		mountPath: local.VolumeMount.MountPath,
		prefix:    local.Prefix,
	}
}

// PageIterator iterates a page of objects via 'ListIterator'
type PageIterator struct {
	iter *blob.ListIterator
}

// Next list a page of objects
//
// If err == io.EOF, all objects of bucket have been read.
// If err == nil, a page of objects have been read.
// Otherwise, err occurs and return objects have been read.
func (i *PageIterator) Next(ctx context.Context, pageSize int) ([]*blob.ListObject, error) {
	objs := make([]*blob.ListObject, 0, pageSize)

	for len(objs) < pageSize {
		obj, err := i.iter.Next(ctx)
		if err == io.EOF && len(objs) != 0 {
			// if err is EOF, but objects has been read, return nil.
			// and the next call will return io.EOF.
			break
		}

		if err != nil {
			return nil, err
		}

		objs = append(objs, obj)
	}

	return objs, nil
}
