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

package clean

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	. "github.com/onsi/gomega"
	"gocloud.dev/blob"
	"gocloud.dev/blob/driver"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/util"
)

func TestCleanBRRemoteBackupDataOnce(t *testing.T) {
	g := NewGomegaWithT(t)

	genTestObjs := func(size int) []*driver.ListObject {
		orginObjs := make([]*driver.ListObject, 0, size)
		for i := 0; i < size; i++ {
			orginObjs = append(orginObjs, &driver.ListObject{
				Key: strconv.Itoa(i),
			})
		}
		return orginObjs
	}
	pageSize := 10000

	type batchDel func(objs []*blob.ListObject) *util.BatchDeleteObjectsResult

	cases := map[string]struct {
		setup  func(drv *util.MockDriver, opt *v1alpha1.CleanOption) batchDel
		expect func(err error, backoff bool)
	}{
		"all objects deleted": {
			setup: func(drv *util.MockDriver, opt *v1alpha1.CleanOption) batchDel {
				size := 123456
				objs := genTestObjs(size)
				drv.SetListPaged(objs, nil)

				// mock delete function
				callCount := 0
				expectCallCount := size/pageSize + 1
				return func(objs []*blob.ListObject) *util.BatchDeleteObjectsResult {
					result := &util.BatchDeleteObjectsResult{}
					for _, obj := range objs {
						result.Deleted = append(result.Deleted, obj.Key) // all objects are deleted
					}

					callCount++
					g.Expect(callCount).Should(BeNumerically("<=", expectCallCount)) // check number of call BatchDeleteObjects
					if callCount == expectCallCount {
						drv.SetListPaged([]*driver.ListObject{}, nil) // mock delete all objects successfully
					}
					return result
				}
			},
			expect: func(err error, backoff bool) {
				g.Expect(err).Should(Succeed())
				g.Expect(backoff).Should(BeFalse())
			},
		},
		"some objects failed to be deleted": {
			setup: func(drv *util.MockDriver, opt *v1alpha1.CleanOption) batchDel {
				size := 123456
				objs := genTestObjs(size)
				drv.SetListPaged(objs, nil)

				// mock delete function
				callCount := 0
				expectCallCount := size/pageSize + 1
				return func(objs []*blob.ListObject) *util.BatchDeleteObjectsResult {
					result := &util.BatchDeleteObjectsResult{}
					if callCount%2 == 0 {
						for _, obj := range objs {
							result.Deleted = append(result.Deleted, obj.Key)
						}
					} else {
						for _, obj := range objs {
							result.Errors = append(result.Errors, util.ObjectError{Key: obj.Key, Err: fmt.Errorf("delete failed")})
						}
					}

					callCount++
					g.Expect(callCount).Should(BeNumerically("<=", expectCallCount)) // check number of call BatchDeleteObjects

					return result
				}
			},
			expect: func(err error, backoff bool) {
				g.Expect(err).Should(MatchError("some objects failed to be deleted"))
				g.Expect(backoff).Should(BeFalse())
			},
		},
		"some objects failed silently to be deleted": {
			setup: func(drv *util.MockDriver, opt *v1alpha1.CleanOption) batchDel {
				size := 123456
				objs := genTestObjs(size)
				drv.SetListPaged(objs, nil)

				// mock delete function
				callCount := 0
				expectCallCount := size/pageSize + 1
				return func(objs []*blob.ListObject) *util.BatchDeleteObjectsResult {
					result := &util.BatchDeleteObjectsResult{}
					if callCount%2 == 0 {
						for _, obj := range objs {
							result.Deleted = append(result.Deleted, obj.Key)
						}
					}
					// failed to delete but not report

					callCount++
					g.Expect(callCount).Should(BeNumerically("<=", expectCallCount)) // check number of call BatchDeleteObjects
					if callCount == expectCallCount {
						drv.SetListPaged(genTestObjs(123), nil) // mock some objects are missing
					}

					return result
				}
			},
			expect: func(err error, backoff bool) {
				g.Expect(err).Should(MatchError("some objects failed to be deleted"))
				g.Expect(backoff).Should(BeFalse())
			},
		},
		"some objects are missing": {
			setup: func(drv *util.MockDriver, opt *v1alpha1.CleanOption) batchDel {
				size := 123456
				objs := genTestObjs(size)
				drv.SetListPaged(objs, nil)

				// mock delete function
				callCount := 0
				expectCallCount := size/pageSize + 1
				return func(objs []*blob.ListObject) *util.BatchDeleteObjectsResult {
					result := &util.BatchDeleteObjectsResult{}
					for _, obj := range objs {
						result.Deleted = append(result.Deleted, obj.Key) // all objects are deleted
					}

					callCount++
					g.Expect(callCount).Should(BeNumerically("<=", expectCallCount)) // check number of call BatchDeleteObjects

					return result
				}
			},
			expect: func(err error, backoff bool) {
				g.Expect(err).Should(MatchError("some objects are missing to be deleted"))
				g.Expect(backoff).Should(BeFalse())
			},
		},
		"backoff not triggered when all objects deleted": {
			setup: func(drv *util.MockDriver, opt *v1alpha1.CleanOption) batchDel {
				size := 123456
				objs := genTestObjs(size)
				drv.SetListPaged(objs, nil)
				opt.BackoffEnabled = true

				// mock delete function
				callCount := 0
				expectCallCount := size/pageSize + 1
				return func(objs []*blob.ListObject) *util.BatchDeleteObjectsResult {
					result := &util.BatchDeleteObjectsResult{}
					for _, obj := range objs {
						result.Deleted = append(result.Deleted, obj.Key) // all objects are deleted
					}

					callCount++
					g.Expect(callCount).Should(BeNumerically("<=", expectCallCount)) // check number of call BatchDeleteObjects
					if callCount == expectCallCount {
						drv.SetListPaged([]*driver.ListObject{}, nil) // mock delete all objects successfully
					}
					return result
				}
			},
			expect: func(err error, backoff bool) {
				g.Expect(err).Should(Succeed())
				g.Expect(backoff).Should(BeFalse())
			},
		},
		"backoff triggered when some objects failed to be deleted": {
			setup: func(drv *util.MockDriver, opt *v1alpha1.CleanOption) batchDel {
				size := 123456
				objs := genTestObjs(size)
				drv.SetListPaged(objs, nil)
				opt.BackoffEnabled = true

				// mock delete function
				callCount := 0
				expectCallCount := size/pageSize + 1
				return func(objs []*blob.ListObject) *util.BatchDeleteObjectsResult {
					result := &util.BatchDeleteObjectsResult{}
					if callCount%2 == 0 {
						for _, obj := range objs {
							result.Deleted = append(result.Deleted, obj.Key)
						}
					} else {
						for _, obj := range objs {
							result.Errors = append(result.Errors, util.ObjectError{Key: obj.Key, Err: fmt.Errorf("delete failed")})
						}
					}

					callCount++
					g.Expect(callCount).Should(BeNumerically("<=", expectCallCount)) // check number of call BatchDeleteObjects

					return result
				}
			},
			expect: func(err error, backoff bool) {
				g.Expect(err).Should(MatchError("some objects failed to be deleted"))
				g.Expect(backoff).Should(BeTrue())
			},
		},
		"backoff triggered when some objects failed silently to be deleted": {
			setup: func(drv *util.MockDriver, opt *v1alpha1.CleanOption) batchDel {
				size := 123456
				objs := genTestObjs(size)
				drv.SetListPaged(objs, nil)
				opt.BackoffEnabled = true

				// mock delete function
				callCount := 0
				expectCallCount := size/pageSize + 1
				return func(objs []*blob.ListObject) *util.BatchDeleteObjectsResult {
					result := &util.BatchDeleteObjectsResult{}
					if callCount%2 == 0 {
						for _, obj := range objs {
							result.Deleted = append(result.Deleted, obj.Key)
						}
					}
					// failed to delete but not report

					callCount++
					g.Expect(callCount).Should(BeNumerically("<=", expectCallCount)) // check number of call BatchDeleteObjects
					if callCount == expectCallCount {
						drv.SetListPaged(genTestObjs(123), nil) // mock some objects are missing
					}

					return result
				}
			},
			expect: func(err error, backoff bool) {
				g.Expect(err).Should(MatchError("some objects failed to be deleted"))
				g.Expect(backoff).Should(BeTrue())
			},
		},
	}

	for name, tt := range cases {
		t.Logf("testcase: %s\n", name)

		drv := &util.MockDriver{
			Type: v1alpha1.BackupStorageTypeS3,
		}
		backend := &util.StorageBackend{}
		backend.Bucket = blob.NewBucket(drv)
		bo := &Options{
			Namespace:  "default",
			BackupName: "test",
		}
		opt := &v1alpha1.CleanOption{
			PageSize:       uint64(pageSize),
			BackoffEnabled: false,
		}

		batchDel := tt.setup(drv, opt)
		s3patch := gomonkey.ApplyFunc(util.BatchDeleteObjectsOfS3, func(ctx context.Context, s3cli s3iface.S3API, objs []*blob.ListObject, bucket string, prefix string, concurrency int) *util.BatchDeleteObjectsResult {
			return batchDel(objs)
		})
		defer s3patch.Reset()
		backoff := false
		timepatch := gomonkey.ApplyFunc(time.Sleep, func(d time.Duration) {
			backoff = true
		})
		defer timepatch.Reset()

		err := bo.cleanBRRemoteBackupDataOnce(context.TODO(), backend, *opt, 1)
		tt.expect(err, backoff)
	}
}
