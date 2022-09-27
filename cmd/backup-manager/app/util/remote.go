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
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"gocloud.dev/blob"
	"gocloud.dev/blob/azureblob"
	"gocloud.dev/blob/driver"
	"gocloud.dev/blob/fileblob"
	"gocloud.dev/blob/gcsblob"
	"gocloud.dev/blob/s3blob"
	"gocloud.dev/gcerrors"
	"gocloud.dev/gcp"
	"k8s.io/client-go/util/workqueue"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/util"
)

const (
	maxRetries         = 3 // number of retries to make of operations
	defaultStorageFlag = "storage"
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

type azblobConfig struct {
	container  string
	accessTier string
	secretName string
	prefix     string
}

type localConfig struct {
	mountPath string
	prefix    string
}

// StorageBackend provide a generic storage backend
type StorageBackend struct {
	*blob.Bucket

	s3     *s3Config
	gcs    *gcsConfig
	azblob *azblobConfig
	local  *localConfig
}

// NewStorageBackend creates new storage backend, now supports S3/GCS/Azblob/Local
func NewStorageBackend(provider v1alpha1.StorageProvider) (*StorageBackend, error) {
	var bucket *blob.Bucket
	var err error

	b := &StorageBackend{}

	st := util.GetStorageType(provider)
	switch st {
	case v1alpha1.BackupStorageTypeS3:
		b.s3 = makeS3Config(provider.S3, true)
		bucket, err = newS3Storage(b.s3)
	case v1alpha1.BackupStorageTypeGcs:
		b.gcs = makeGcsConfig(provider.Gcs, true)
		bucket, err = newGcsStorage(b.gcs)
	case v1alpha1.BackupStorageTypeAzblob:
		b.azblob = makeAzblobConfig(provider.Azblob)
		bucket, err = newAzblobStorage(b.azblob)
	case v1alpha1.BackupStorageTypeLocal:
		b.local = makeLocalConfig(provider.Local)
		bucket, err = newLocalStorage(b.local)
	default:
		err = fmt.Errorf("storage %s not supported yet", st)
	}
	b.Bucket = bucket

	if err != nil {
		return nil, err
	}

	return b, nil
}

func (b *StorageBackend) StorageType() v1alpha1.BackupStorageType {
	if b.s3 != nil {
		return v1alpha1.BackupStorageTypeS3
	} else if b.gcs != nil {
		return v1alpha1.BackupStorageTypeGcs
	} else if b.azblob != nil {
		return v1alpha1.BackupStorageTypeAzblob
	} else if b.local != nil {
		return v1alpha1.BackupStorageTypeLocal
	}
	return v1alpha1.BackupStorageTypeUnknown
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
// If provider is S3/GCS/Azblob, return bucket. Otherwise return empty string
func (b *StorageBackend) GetBucket() string {
	if b.s3 != nil {
		return b.s3.bucket
	} else if b.gcs != nil {
		return b.gcs.bucket
	} else if b.azblob != nil {
		return b.azblob.container
	}

	return ""
}

// GetPrefix return prefix
func (b *StorageBackend) GetPrefix() string {
	if b.s3 != nil {
		return b.s3.prefix
	} else if b.gcs != nil {
		return b.gcs.prefix
	} else if b.azblob != nil {
		return b.azblob.prefix
	} else if b.local != nil {
		return b.local.prefix
	}

	return ""
}

type ObjectError struct {
	Key string
	Err error
}

type BatchDeleteObjectsResult struct {
	Deleted []string
	Errors  []ObjectError
}

// BatchDeleteObjects delete mutli objects
//
// Depending on storage type, it use function 'BatchDeleteObjectsOfS3' or 'BatchDeleteObjectsConcurrently'
func (b *StorageBackend) BatchDeleteObjects(ctx context.Context, objs []*blob.ListObject, opt v1alpha1.BatchDeleteOption) *BatchDeleteObjectsResult {
	var result *BatchDeleteObjectsResult

	s3cli, ok := b.AsS3()
	if !opt.DisableBatchConcurrency && ok {
		concurrency := v1alpha1.DefaultBatchDeleteOption.BatchConcurrency
		if opt.BatchConcurrency != 0 {
			concurrency = opt.BatchConcurrency
		}
		result = BatchDeleteObjectsOfS3(ctx, s3cli, objs, b.GetBucket(), b.GetPrefix(), int(concurrency))
	} else {
		concurrency := v1alpha1.DefaultBatchDeleteOption.RoutineConcurrency
		if opt.RoutineConcurrency != 0 {
			concurrency = opt.RoutineConcurrency
		}
		result = BatchDeleteObjectsConcurrently(ctx, b.Bucket, objs, int(concurrency))
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

		// delete objects
		delete := &s3.Delete{}
		for _, obj := range objs[start:end] {
			delete.Objects = append(delete.Objects, &s3.ObjectIdentifier{
				Key: aws.String(prefix + obj.Key), // key must be absolute path
			})
		}
		input := &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: delete,
		}
		resp, err := s3cli.DeleteObjectsWithContext(ctx, input)

		// record result
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

		// delete an object
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

// genStorageArgs returns the arg for --flag option and the remote/local path for br, default flag is storage.
// TODO: add unit test
func GenStorageArgsForFlag(provider v1alpha1.StorageProvider, flag string) ([]string, error) {
	st := util.GetStorageType(provider)
	switch st {
	case v1alpha1.BackupStorageTypeS3:
		qs := makeS3Config(provider.S3, false)
		s := newS3StorageOptionForFlag(qs, flag)
		return s, nil
	case v1alpha1.BackupStorageTypeGcs:
		qs := makeGcsConfig(provider.Gcs, false)
		s := newGcsStorageOptionForFlag(qs, flag)
		return s, nil
	case v1alpha1.BackupStorageTypeAzblob:
		conf := makeAzblobConfig(provider.Azblob)
		strs := newAzblobStorageOptionForFlag(conf, flag)
		return strs, nil
	case v1alpha1.BackupStorageTypeLocal:
		localConfig := makeLocalConfig(provider.Local)
		cmdOpts, err := newLocalStorageOptionForFlag(localConfig, flag)
		if err != nil {
			return nil, err
		}
		return cmdOpts, nil
	default:
		return nil, fmt.Errorf("storage %s not supported yet", st)
	}
}

// newLocalStorageOption constructs `--flag local://$PATH` arg for br
func newLocalStorageOptionForFlag(conf *localConfig, flag string) ([]string, error) {
	if flag != "" && flag != defaultStorageFlag {
		// now just set path to special flag
		return []string{fmt.Sprintf("--%s=local://%s", flag, path.Join(conf.mountPath, conf.prefix))}, nil
	}
	return []string{fmt.Sprintf("--storage=local://%s", path.Join(conf.mountPath, conf.prefix))}, nil
}

// newS3StorageOption constructs the arg for --flag option and the remote path for br
func newS3StorageOptionForFlag(conf *s3Config, flag string) []string {
	var s3options []string
	path := fmt.Sprintf("s3://%s", path.Join(conf.bucket, conf.prefix))
	if flag != "" && flag != defaultStorageFlag {
		// now just set path to special flag
		s3options = append(s3options, fmt.Sprintf("--%s=%s", flag, path))
		return s3options
	}
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

func newLocalStorage(conf *localConfig) (*blob.Bucket, error) {
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

// Azure Blob Storage using AAD credentials
type azblobAADCred struct {
	account      string
	clientID     string
	clientSecret string
	tenantID     string
}

// Azure Blob Storage using shared key credentials
type azblobSharedKeyCred struct {
	account   string
	sharedKey string
}

// newAzblobStorage initialize a new azblob storage
func newAzblobStorage(conf *azblobConfig) (*blob.Bucket, error) {
	account := os.Getenv("AZURE_STORAGE_ACCOUNT")
	if len(account) == 0 {
		return nil, errors.New("No AZURE_STORAGE_ACCOUNT")
	}

	// Azure AAD Service Principal with access to the storage account.
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	tenantID := os.Getenv("AZURE_TENANT_ID")

	// Azure shared key with access to the storage account
	accountKey := os.Getenv("AZURE_STORAGE_KEY")

	// check condition for using AAD credentials first
	var usingAAD bool
	if len(clientID) != 0 && len(clientSecret) != 0 && len(tenantID) != 0 {
		usingAAD = true
	} else if len(accountKey) != 0 {
		usingAAD = false
	} else {
		return nil, errors.New("Missing necessary key(s) for credentials")
	}

	// initialize a new azblob storage using AAD or shared key credentials
	var bucket *blob.Bucket
	var err error
	if usingAAD {
		bucket, err = newAzblobStorageUsingAAD(conf, &azblobAADCred{
			account:      account,
			clientID:     clientID,
			clientSecret: clientSecret,
			tenantID:     tenantID,
		})
	} else {
		bucket, err = newAzblobStorageUsingSharedKey(conf, &azblobSharedKeyCred{
			account:   account,
			sharedKey: accountKey,
		})
	}
	if err != nil {
		return nil, err
	}
	return blob.PrefixedBucket(bucket, strings.Trim(conf.prefix, "/")+"/"), nil
}

// newAzblobStorageUsingAAD initialize a new azblob storage using AAD credentials
func newAzblobStorageUsingAAD(conf *azblobConfig, cred *azblobAADCred) (*blob.Bucket, error) {
	// Azure Storage Account.
	accountName := azureblob.AccountName(cred.account)

	// Get an Oauth2 token for the account for use with Azure Storage.
	ccc := auth.NewClientCredentialsConfig(cred.clientID, cred.clientSecret, cred.tenantID)

	// Set the target resource to the Azure storage.
	ccc.Resource = "https://storage.azure.com/"
	token, err := ccc.ServicePrincipalToken()
	if err != nil {
		return nil, err
	}

	// Refresh OAuth2 token.
	if err := token.RefreshWithContext(context.Background()); err != nil {
		return nil, err
	}

	// Create the credential using the OAuth2 token.
	credential := azblob.NewTokenCredential(token.OAuthToken(), nil)

	// Create a Pipeline, using whatever PipelineOptions you need.
	pipeline := azureblob.NewPipeline(credential, azblob.PipelineOptions{})

	// Create a *blob.Bucket.
	ctx := context.Background()
	return azureblob.OpenBucket(ctx, pipeline, accountName, conf.container, new(azureblob.Options))
}

// newAzblobStorageUsingSharedKey initialize a new azblob storage using shared key credentials
func newAzblobStorageUsingSharedKey(conf *azblobConfig, cred *azblobSharedKeyCred) (*blob.Bucket, error) {
	ctx := context.Background()

	// Azure Storage Account and Access Key.
	accountName := azureblob.AccountName(cred.account)
	accountKey := azureblob.AccountKey(cred.sharedKey)

	// Create a credentials object.
	credential, err := azureblob.NewCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	// Create a Pipeline, using whatever PipelineOptions you need.
	pipeline := azureblob.NewPipeline(credential, azblob.PipelineOptions{})

	// Create a *blob.Bucket.
	// The credential Option is required if you're going to use blob.SignedURL.
	return azureblob.OpenBucket(ctx, pipeline, accountName, conf.container, &azureblob.Options{Credential: credential})
}

// newGcsStorageOption constructs the arg for --flag option and the remote path for br
func newGcsStorageOptionForFlag(conf *gcsConfig, flag string) []string {
	var gcsoptions []string
	path := fmt.Sprintf("gcs://%s/", path.Join(conf.bucket, conf.prefix))
	if flag != "" && flag != defaultStorageFlag {
		// now just set path to special flag
		gcsoptions = append(gcsoptions, fmt.Sprintf("--%s=%s", flag, path))
		return gcsoptions
	}
	gcsoptions = append(gcsoptions, fmt.Sprintf("--storage=%s", path))

	if conf.storageClass != "" {
		gcsoptions = append(gcsoptions, fmt.Sprintf("--gcs.storage-class=%s", conf.storageClass))
	}
	if conf.objectAcl != "" {
		gcsoptions = append(gcsoptions, fmt.Sprintf("--gcs.predefined-acl=%s", conf.objectAcl))
	}
	return gcsoptions
}

// newAzblobStorageOption constructs the arg for --flag option and the remote path for br
func newAzblobStorageOptionForFlag(conf *azblobConfig, flag string) []string {
	var azblobOptions []string
	path := fmt.Sprintf("azure://%s/", path.Join(conf.container, conf.prefix))
	if flag != "" && flag != defaultStorageFlag {
		// now just set path to special flag
		azblobOptions = append(azblobOptions, fmt.Sprintf("--%s=%s", flag, path))
		return azblobOptions
	}
	azblobOptions = append(azblobOptions, fmt.Sprintf("--storage=%s", path))

	if conf.accessTier != "" {
		azblobOptions = append(azblobOptions, fmt.Sprintf("--azblob.access-tier=%s", conf.accessTier))
	}
	return azblobOptions
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

// makeAzblobConfig constructs azblobConfig parameters
func makeAzblobConfig(azblob *v1alpha1.AzblobStorageProvider) *azblobConfig {
	conf := azblobConfig{}

	path := strings.Trim(azblob.Container, "/") + "/" + strings.Trim(azblob.Prefix, "/")
	fields := strings.SplitN(path, "/", 2)

	conf.container = fields[0]
	conf.accessTier = azblob.AccessTier
	conf.secretName = azblob.SecretName
	conf.prefix = fields[1]

	return &conf
}

func makeLocalConfig(local *v1alpha1.LocalStorageProvider) *localConfig {
	return &localConfig{
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
//
// TODO: use blob.ListPage after upgrade gocloud.dev denpendency to 0.21
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

// MockDriver implement driver.Bucket
type MockDriver struct {
	driver.Bucket

	errCode gcerrors.ErrorCode

	delete    func(key string) error
	listPaged func(opts *driver.ListOptions) (*driver.ListPage, error)

	Type v1alpha1.BackupStorageType
}

func (d *MockDriver) As(i interface{}) bool {
	switch d.Type {
	case v1alpha1.BackupStorageTypeS3:
		_, ok := i.(**s3.S3)
		if !ok {
			return false
		}
		return true
	case v1alpha1.BackupStorageTypeGcs:
		_, ok := i.(**storage.Client)
		if !ok {
			return false
		}
		return true
	}
	return false
}

func (d *MockDriver) ErrorCode(err error) gcerrors.ErrorCode {
	return d.errCode
}

func (d *MockDriver) Delete(_ context.Context, key string) error {
	return d.delete(key)
}

func (d *MockDriver) SetDelete(fn func(key string) error) {
	d.delete = fn
}

func (d *MockDriver) SetListPaged(objs []*driver.ListObject, rerr error) {
	d.listPaged = func(opts *driver.ListOptions) (*driver.ListPage, error) {
		if rerr != nil {
			return nil, rerr
		}

		var start int
		var err error
		if len(opts.PageToken) != 0 {
			start, err = strconv.Atoi(string(opts.PageToken))
			if err != nil {
				panic(err)
			}
		}

		pageSize := 1000
		if opts.PageSize != 0 {
			pageSize = opts.PageSize
		}
		end := start + pageSize
		if end > len(objs) {
			end = len(objs)
		}

		p := &driver.ListPage{
			Objects:       objs[start:end],
			NextPageToken: []byte(strconv.Itoa(end)),
		}
		if len(p.Objects) == 0 {
			p.NextPageToken = nil // it will return io.EOF
		}

		return p, nil
	}
}

func (d *MockDriver) ListPaged(ctx context.Context, opts *driver.ListOptions) (*driver.ListPage, error) {
	return d.listPaged(opts)
}
