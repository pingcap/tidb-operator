// Copyright 2021 PingCAP, Inc.
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
	"math"
	"strconv"
	"strings"
	"sync"
	"testing"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/onsi/gomega"
	"gocloud.dev/blob"
	"gocloud.dev/blob/driver"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

type mockS3Client struct {
	s3iface.S3API

	deleteObjects func(*s3.DeleteObjectsInput) (*s3.DeleteObjectsOutput, error)
}

func (c *mockS3Client) DeleteObjectsWithContext(_ aws.Context, input *s3.DeleteObjectsInput, _ ...request.Option) (*s3.DeleteObjectsOutput, error) {
	return c.deleteObjects(input)
}

func TestPageIterator(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type testcase struct {
		size     int
		rerr     error
		pageSize int
	}
	cases := []testcase{
		{
			size:     10000,
			pageSize: 1000,
		},
		{
			size:     12345,
			pageSize: 1000,
		},
		{
			size:     12345,
			pageSize: 987,
		},
		{
			size:     999,
			pageSize: 1000,
		},
		{
			size:     12345,
			pageSize: 10000,
		},
	}

	// basic function
	for _, tcase := range cases {
		drv := &MockDriver{}
		backend := &StorageBackend{}
		backend.Bucket = blob.NewBucket(drv)

		traveled := make([]bool, tcase.size)
		orginObjs := make([]*driver.ListObject, 0, tcase.size)
		for i := 0; i < tcase.size; i++ {
			orginObjs = append(orginObjs, &driver.ListObject{
				Key: strconv.Itoa(i),
			})
		}
		drv.SetListPaged(orginObjs, tcase.rerr)

		iter := backend.ListPage(nil)
		count := 0
		maxCount := int(math.Ceil(float64(len(orginObjs)) / float64(tcase.pageSize))) // when it should return io.EOF
		for {
			objs, err := iter.Next(context.Background(), tcase.pageSize)

			if count == maxCount {
				g.Expect(err).To(gomega.Equal(io.EOF))
			} else {
				g.Expect(err).To(gomega.Succeed())
				for index := range objs {
					g.Expect(objs[index].Key).To(gomega.Equal(orginObjs[count*tcase.pageSize+index].Key))
					traveled[count*tcase.pageSize+index] = true
				}
			}

			if err == io.EOF {
				break
			}
			count++
		}

		for _, ok := range traveled {
			g.Expect(ok).To(gomega.BeTrue())
		}
	}

	// err handle
	errcases := []testcase{
		{
			size:     10000,
			rerr:     io.EOF,
			pageSize: 1000,
		},
		{
			size:     12345,
			rerr:     fmt.Errorf("test case err"),
			pageSize: 1000,
		},
	}
	for _, tcase := range errcases {
		drv := &MockDriver{}
		backend := &StorageBackend{}
		backend.Bucket = blob.NewBucket(drv)

		orginObjs := make([]*driver.ListObject, 0, tcase.size)
		for i := 0; i < tcase.size; i++ {
			orginObjs = append(orginObjs, &driver.ListObject{
				Key: strconv.Itoa(i),
			})
		}
		drv.SetListPaged(orginObjs, tcase.rerr)

		iter := backend.ListPage(nil)
		objs, err := iter.Next(context.Background(), tcase.pageSize)

		g.Expect(err.Error()).To(gomega.ContainSubstring(tcase.rerr.Error())) // can't find any func to convert error to gcerr.Error
		g.Expect(objs).To(gomega.BeNil())
	}
}

func TestStorageBackendBasic(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// basic
	type testcase struct {
		name     string
		provider v1alpha1.StorageProvider

		expectStorageType v1alpha1.BackupStorageType
		expectedBucket    string
		expectedPrefix    string
	}
	cases := []testcase{
		{
			name: "basic s3 storage backend",
			provider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					Provider: v1alpha1.S3StorageProviderTypeAWS,
					Bucket:   "s3-bucket",
					Prefix:   "s3-prefix",
				},
			},
			expectStorageType: v1alpha1.BackupStorageTypeS3,
			expectedBucket:    "s3-bucket",
			expectedPrefix:    "s3-prefix",
		},
		{
			name: "basic gcs storage backend",
			provider: v1alpha1.StorageProvider{
				Gcs: &v1alpha1.GcsStorageProvider{
					Bucket: "gcs-bucket",
					Prefix: "gcs-prefix",
				},
			},
			expectStorageType: v1alpha1.BackupStorageTypeGcs,
			expectedBucket:    "gcs-bucket",
			expectedPrefix:    "gcs-prefix",
		},
		{
			name: "basic azblob storage backend",
			provider: v1alpha1.StorageProvider{
				Azblob: &v1alpha1.AzblobStorageProvider{
					Container: "azblob-bucket",
					Prefix:    "azblob-prefix",
				},
			},
			expectStorageType: v1alpha1.BackupStorageTypeAzblob,
			expectedBucket:    "azblob-bucket",
			expectedPrefix:    "azblob-prefix",
		},
		{
			name: "basic local storage backend",
			provider: v1alpha1.StorageProvider{
				Local: &v1alpha1.LocalStorageProvider{
					Prefix: "local-prefix",
				},
			},
			expectStorageType: v1alpha1.BackupStorageTypeLocal,
			expectedBucket:    "",
			expectedPrefix:    "local-prefix",
		},
		{
			name: "s3 storage backend and bucket contains prefix",
			provider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					Provider: v1alpha1.S3StorageProviderTypeAWS,
					Bucket:   "s3-bucket/a/b",
					Prefix:   "s3-prefix/c/d",
				},
			},
			expectStorageType: v1alpha1.BackupStorageTypeS3,
			expectedBucket:    "s3-bucket",
			expectedPrefix:    "a/b/s3-prefix/c/d",
		},
		{
			name: "gcs storage backend and bucket contains prefix",
			provider: v1alpha1.StorageProvider{
				Gcs: &v1alpha1.GcsStorageProvider{
					Bucket: "gcs-bucket/a/b",
					Prefix: "gcs-prefix/c/d",
				},
			},
			expectStorageType: v1alpha1.BackupStorageTypeGcs,
			expectedBucket:    "gcs-bucket",
			expectedPrefix:    "a/b/gcs-prefix/c/d",
		},
		{
			name: "azblob storage backend and bucket contains prefix",
			provider: v1alpha1.StorageProvider{
				Azblob: &v1alpha1.AzblobStorageProvider{
					Container: "azblob-bucket/a/b",
					Prefix:    "azblob-prefix/c/d",
				},
			},
			expectStorageType: v1alpha1.BackupStorageTypeAzblob,
			expectedBucket:    "azblob-bucket",
			expectedPrefix:    "a/b/azblob-prefix/c/d",
		},
	}
	for _, tcases := range cases {
		t.Log(tcases.name)
		provider := tcases.provider

		// mock function
		s3patches := gomonkey.ApplyFunc(newS3Storage, func(conf *s3Config, cred *StorageCredential) (*blob.Bucket, error) {
			return nil, nil
		})
		defer s3patches.Reset()
		gcsPatches := gomonkey.ApplyFunc(newGcsStorage, func(conf *gcsConfig) (*blob.Bucket, error) {
			return nil, nil
		})
		defer gcsPatches.Reset()
		azblobPatches := gomonkey.ApplyFunc(newAzblobStorage, func(conf *azblobConfig) (*blob.Bucket, error) {
			return nil, nil
		})
		defer azblobPatches.Reset()
		localPatches := gomonkey.ApplyFunc(newLocalStorage, func(conf *localConfig) (*blob.Bucket, error) {
			return nil, nil
		})
		defer localPatches.Reset()

		backend, err := NewStorageBackend(provider, &StorageCredential{})
		g.Expect(err).Should(gomega.Succeed())

		g.Expect(backend.StorageType()).Should(gomega.Equal(tcases.expectStorageType))

		g.Expect(backend.GetBucket()).Should(gomega.Equal(tcases.expectedBucket))

		g.Expect(backend.GetPrefix()).Should(gomega.Equal(tcases.expectedPrefix))
	}
}

func TestStorageBackendBatchDeleteObjects(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type testcase struct {
		name    string
		backend *StorageBackend
		opt     v1alpha1.BatchDeleteOption

		expectedConcurrency       int
		useBatchDeleteObjectsOfS3 bool
	}

	s3drv := &MockDriver{
		Type: v1alpha1.BackupStorageTypeS3,
	}
	gcsdrv := &MockDriver{
		Type: v1alpha1.BackupStorageTypeGcs,
	}
	azblobdrv := &MockDriver{
		Type: v1alpha1.BackupStorageTypeAzblob,
	}
	localdrv := &MockDriver{
		Type: v1alpha1.BackupStorageTypeLocal,
	}

	cases := []testcase{
		{
			name: "s3 with default option",
			backend: &StorageBackend{
				Bucket: blob.NewBucket(s3drv),
			},
			opt:                       v1alpha1.BatchDeleteOption{},
			expectedConcurrency:       int(v1alpha1.DefaultBatchDeleteOption.BatchConcurrency),
			useBatchDeleteObjectsOfS3: true,
		},
		{
			name: "gcs with default option",
			backend: &StorageBackend{
				Bucket: blob.NewBucket(gcsdrv),
			},
			opt:                       v1alpha1.BatchDeleteOption{},
			expectedConcurrency:       int(v1alpha1.DefaultBatchDeleteOption.RoutineConcurrency),
			useBatchDeleteObjectsOfS3: false,
		},
		{
			name: "azblob with default option",
			backend: &StorageBackend{
				Bucket: blob.NewBucket(azblobdrv),
			},
			opt:                       v1alpha1.BatchDeleteOption{},
			expectedConcurrency:       int(v1alpha1.DefaultBatchDeleteOption.RoutineConcurrency),
			useBatchDeleteObjectsOfS3: false,
		},
		{
			name: "local with default option",
			backend: &StorageBackend{
				Bucket: blob.NewBucket(localdrv),
			},
			opt:                       v1alpha1.BatchDeleteOption{},
			expectedConcurrency:       int(v1alpha1.DefaultBatchDeleteOption.RoutineConcurrency),
			useBatchDeleteObjectsOfS3: false,
		},
		{
			name: "s3 with option",
			backend: &StorageBackend{
				Bucket: blob.NewBucket(s3drv),
			},
			opt: v1alpha1.BatchDeleteOption{
				BatchConcurrency: 10000,
			},
			expectedConcurrency:       10000,
			useBatchDeleteObjectsOfS3: true,
		},
		{
			name: "gcs with option",
			backend: &StorageBackend{
				Bucket: blob.NewBucket(gcsdrv),
			},
			opt: v1alpha1.BatchDeleteOption{
				RoutineConcurrency: 10000,
			},
			expectedConcurrency:       10000,
			useBatchDeleteObjectsOfS3: false,
		},
		{
			name: "azblob with option",
			backend: &StorageBackend{
				Bucket: blob.NewBucket(azblobdrv),
			},
			opt: v1alpha1.BatchDeleteOption{
				RoutineConcurrency: 10000,
			},
			expectedConcurrency:       10000,
			useBatchDeleteObjectsOfS3: false,
		},
		{
			name: "local with default option",
			backend: &StorageBackend{
				Bucket: blob.NewBucket(localdrv),
			},
			opt: v1alpha1.BatchDeleteOption{
				RoutineConcurrency: 10000,
			},
			expectedConcurrency:       10000,
			useBatchDeleteObjectsOfS3: false,
		},
		{
			name: "s3 with disable option",
			backend: &StorageBackend{
				Bucket: blob.NewBucket(s3drv),
			},
			opt: v1alpha1.BatchDeleteOption{
				DisableBatchConcurrency: true,
			},
			expectedConcurrency:       int(v1alpha1.DefaultBatchDeleteOption.RoutineConcurrency),
			useBatchDeleteObjectsOfS3: false,
		},
	}

	for _, tcase := range cases {
		t.Log(tcase.name)

		expectedResult := &BatchDeleteObjectsResult{}

		// mock function
		s3patch := gomonkey.ApplyFunc(BatchDeleteObjectsOfS3, func(ctx context.Context, s3cli s3iface.S3API, objs []*blob.ListObject, bucket string, prefix string, concurrency int) *BatchDeleteObjectsResult {
			if !tcase.useBatchDeleteObjectsOfS3 {
				t.Fatal("should not use 'BatchDeleteObjectsOfS3'")
			}
			g.Expect(concurrency).Should(gomega.Equal(tcase.expectedConcurrency))
			return expectedResult
		})
		defer s3patch.Reset()
		patch := gomonkey.ApplyFunc(BatchDeleteObjectsConcurrently, func(ctx context.Context, bucket *blob.Bucket, objs []*blob.ListObject, concurrency int) *BatchDeleteObjectsResult {
			if tcase.useBatchDeleteObjectsOfS3 {
				t.Fatal("should not use 'BatchDeleteObjectsConcurrently'")
			}
			g.Expect(concurrency).Should(gomega.Equal(tcase.expectedConcurrency))
			return expectedResult
		})
		defer patch.Reset()

		result := tcase.backend.BatchDeleteObjects(context.TODO(), nil, tcase.opt)
		g.Expect(result == expectedResult).Should(gomega.BeTrue())

	}
}

func TestBatchDeleteObjectsOfS3(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type testcase struct {
		size        int
		concurrency int
		prefix      string
		derr        error
	}
	cases := []testcase{
		{
			size:        10000,
			concurrency: 4,
			prefix:      "/a//",
		},
		{
			size:        12345,
			concurrency: 4,
		},
		{
			size:        1000,
			concurrency: 9,
			prefix:      "/",
		},
		{
			size:        10000,
			concurrency: 4,
			derr:        fmt.Errorf("test case err"),
		},
	}

	for _, tcase := range cases {
		cli := &mockS3Client{}

		bucket := "test"
		objs := objects(tcase.size)

		mu := &sync.Mutex{}
		deletedMap := map[string]bool{} // contain objects that are expected to be deleted
		errMap := map[string]bool{}     // contain objects that are expected to be failed
		setedErr := fmt.Errorf("error object")
		cli.deleteObjects = func(input *s3.DeleteObjectsInput) (*s3.DeleteObjectsOutput, error) {

			if *input.Bucket != bucket {
				panic("bucket isn't same")
			}

			// error case
			if tcase.derr != nil {
				for _, delObj := range input.Delete.Objects {
					key := *delObj.Key
					mu.Lock()
					errMap[key] = true
					mu.Unlock()
				}
				return nil, tcase.derr
			}

			// basic case
			output := &s3.DeleteObjectsOutput{}
			for i, delObj := range input.Delete.Objects {
				key := *delObj.Key

				mu.Lock()

				if i%2 == 0 {
					errMap[key] = true
					output.Errors = append(output.Errors, &s3.Error{
						Key:     &key,
						Code:    aws.String("1"),
						Message: aws.String(setedErr.Error()),
					})
				} else {
					deletedMap[key] = true
					output.Deleted = append(output.Deleted, &s3.DeletedObject{
						Key: &key,
					})
				}
				mu.Unlock()
			}

			return output, nil
		}

		result := BatchDeleteObjectsOfS3(context.Background(), cli, objs, bucket, tcase.prefix, tcase.concurrency)
		g.Expect(result.Errors).To(gomega.HaveLen(len(errMap)))
		g.Expect(result.Deleted).To(gomega.HaveLen(len(deletedMap)))
		for key := range errMap {
			var rerr *ObjectError
			for i := range result.Errors {
				err := result.Errors[i]
				if err.Key == key {
					rerr = &err
					break
				}
			}
			g.Expect(rerr).NotTo(gomega.BeNil())
			if tcase.derr != nil {
				g.Expect(rerr.Err).To(gomega.Equal(tcase.derr)) // check 'Error' of result
			} else {
				g.Expect(rerr.Err.Error()).To(gomega.ContainSubstring(setedErr.Error()))
			}
		}
		for key := range deletedMap {
			g.Expect(result.Deleted).To(gomega.ContainElement(key)) // check 'Deleted' of result
		}
		for _, obj := range objs {
			key := obj.Key
			if tcase.prefix != "" {
				key = strings.Trim(tcase.prefix, "/") + "/" + key
			}
			_, exist1 := deletedMap[key]
			_, exist2 := errMap[key]
			g.Expect(exist1 || exist2).To(gomega.BeTrue(), fmt.Sprintf("obj:%s", obj.Key)) // check if all key is deleted
		}
	}
}

func TestBatchDeleteObjectsConcurrently(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type testcase struct {
		size        int
		concurrency int
	}
	cases := []testcase{
		{
			size:        10000,
			concurrency: 100,
		},
		{
			size:        12345,
			concurrency: 100,
		},
		{
			size:        99,
			concurrency: 100,
		},
	}

	for _, tcase := range cases {
		drv := &MockDriver{}
		bucket := blob.NewBucket(drv)

		orginObjs := objects(tcase.size)

		mu := &sync.Mutex{}
		deletedMap := map[string]bool{} // contain objects that are expected to be deleted
		errMap := map[string]bool{}     // contain objects that are expected to be failed
		setedErr := fmt.Errorf("test case err")

		drv.SetDelete(func(key string) error {
			mu.Lock()
			defer mu.Unlock()
			if len(key)%2 == 0 {
				errMap[key] = true
				return setedErr
			}
			deletedMap[key] = true
			return nil
		})

		result := BatchDeleteObjectsConcurrently(context.Background(), bucket, orginObjs, tcase.concurrency)
		g.Expect(result.Errors).To(gomega.HaveLen(len(errMap)))
		g.Expect(result.Deleted).To(gomega.HaveLen(len(deletedMap)))
		for key := range errMap {
			var rerr *ObjectError
			for i := range result.Errors {
				err := result.Errors[i]
				if err.Key == key {
					rerr = &err
					break
				}
			}
			g.Expect(rerr).NotTo(gomega.BeNil())
			g.Expect(rerr.Err.Error()).To(gomega.ContainSubstring(setedErr.Error()))
		}
		for key := range deletedMap {
			g.Expect(result.Deleted).To(gomega.ContainElement(key)) // check if result is
		}
		for _, obj := range orginObjs {
			_, exist1 := deletedMap[obj.Key]
			_, exist2 := errMap[obj.Key]
			g.Expect(exist1 || exist2).To(gomega.BeTrue()) // check if all key is deleted
		}
	}
}

func objects(size int) []*blob.ListObject {
	objs := make([]*blob.ListObject, 0, size)
	for i := 0; i < size; i++ {
		objs = append(objs, &blob.ListObject{
			Key: strconv.Itoa(i),
		})
	}
	return objs
}
