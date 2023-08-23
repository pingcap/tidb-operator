// Copyright 2023 PingCAP, Inc.
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
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ebs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/dustin/go-humanize"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/util"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

// TODO#1: there shall be a abstract interface for diff cloud platform. e.g. GetBackupSize
// it may better to do this work after gcp support volume-snapshot

// for EBS Direct API permissions, refer to https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebsapi-permissions.html
// for EBS Direct API, refer to https://docs.aws.amazon.com/sdk-for-go/api/service/ebs/

// interface CalcVolSnapBackupSize called by backup and backup clean.

const (
	// DescribeSnapMaxReturnResult can be between 5 and 1,000; if MaxResults is given a value larger than 1,000, only 1,000 results are returned.
	DescribeSnapMaxReturnResult = 1000
	// ListSnapMaxReturnResult can be between 100 and 10,000, and charge ~0.6$/1 million request
	ListSnapMaxReturnResult = 10000
	// EbsApiConcurrency can be between 1 and 50 due to aws service quota
	EbsApiConcurrency = 40

	CalculateFullSize    = "full"
	CalculateIncremental = "incremental"
	CalculateAll         = "all"
)

// CalcVolSnapBackupSize get snapshots from backup meta and then calc the backup size of snapshots.
func CalcVolSnapBackupSize(ctx context.Context, provider v1alpha1.StorageProvider, level string) (fullBackupSize int64, incrementalBackupSize int64, err error) {
	start := time.Now()
	// retrieves all snapshots from backup meta file
	volSnapshots, err := getSnapshotsFromBackupmeta(ctx, provider)
	if err != nil {
		return 0, 0, err
	}

	if err != nil {
		return 0, 0, err
	}

	fullBackupSize, incrementalBackupSize, err = calcBackupSize(ctx, volSnapshots, level)

	if err != nil {
		return 0, 0, err
	}
	elapsed := time.Since(start)
	klog.Infof("calculate volume-snapshot backup size takes %v", elapsed)
	return
}

// getSnapshotsFromBackupmeta read all snapshots from backupmeta
// return volume - snapshot map
func getSnapshotsFromBackupmeta(ctx context.Context, provider v1alpha1.StorageProvider) (map[string]string, error) {
	volumeIDMap := make(map[string]string)

	// read backup meta
	s, err := util.NewStorageBackend(provider, &util.StorageCredential{})
	if err != nil {
		return volumeIDMap, err
	}
	defer s.Close()

	var contents []byte
	// use exponential backoff, every retry duration is duration * factor ^ (used_step - 1)
	backoff := wait.Backoff{
		Duration: time.Second,
		Steps:    6,
		Factor:   2.0,
		Cap:      time.Minute,
	}
	readBackupMeta := func() error {
		exist, err := s.Exists(ctx, constants.MetaFile)
		if err != nil {
			return err
		}
		if !exist {
			return fmt.Errorf("%s not exist", constants.MetaFile)
		}
		contents, err = s.ReadAll(ctx, constants.MetaFile)
		if err != nil {
			return err
		}
		return nil
	}
	isRetry := func(err error) bool {
		return !strings.Contains(err.Error(), "not exist")
	}
	err = retry.OnError(backoff, isRetry, readBackupMeta)
	if err != nil {
		return nil, errors.Annotatef(err, "read backup meta from bucket %s and prefix %s", s.GetBucket(), s.GetPrefix())
	}

	metaInfo := &util.EBSBasedBRMeta{}
	if err = json.Unmarshal(contents, metaInfo); err != nil {
		return volumeIDMap, errors.Annotatef(err, "read backup meta from bucket %s and prefix %s", s.GetBucket(), s.GetPrefix())
	}

	// get volume-snapshot map
	for i := range metaInfo.TiKVComponent.Stores {
		store := metaInfo.TiKVComponent.Stores[i]
		for j := range store.Volumes {
			vol := store.Volumes[j]
			volumeIDMap[vol.ID] = vol.SnapshotID
		}
	}

	return volumeIDMap, nil
}

// calcBackupSize get a volume-snapshots backup size
func calcBackupSize(ctx context.Context, volumes map[string]string, level string) (fullBackupSize int64, incrementalBackupSize int64, err error) {
	var apiReqCount, incrementalApiReqCount uint64

	workerPool := util.NewWorkerPool(EbsApiConcurrency, "list snapshot size")
	eg, _ := errgroup.WithContext(ctx)

	for vid, sid := range volumes {
		snapshotId := sid
		volumeId := vid
		// sort snapshots by timestamp
		workerPool.ApplyOnErrorGroup(eg, func() error {
			var snapSize, apiReq uint64
			if level == CalculateAll || level == CalculateFullSize {
				snapSize, apiReq, err = calculateSnapshotSize(volumeId, snapshotId)
				if err != nil {
					return err
				}
				atomic.AddInt64(&fullBackupSize, int64(snapSize))
				atomic.AddUint64(&apiReqCount, apiReq)
			}

			if level == CalculateAll || level == CalculateIncremental {
				volSnapshots, err := getVolSnapshots(volumeId)
				if err != nil {
					return err
				}
				prevSnapshotId, existed := getPrevSnapshotId(snapshotId, volSnapshots)
				if !existed {
					// if there is no previous snapshot, means it's the first snapshot, uses its full size as incremental size
					atomic.AddInt64(&incrementalBackupSize, int64(snapSize))
					return nil
				}
				klog.Infof("get previous snapshot %s of snapshot %s, volume %s", prevSnapshotId, snapshotId, volumeId)
				incrementalSnapSize, incrementalApiReq, err := calculateChangedBlocksSize(volumeId, prevSnapshotId, snapshotId)
				if err != nil {
					return err
				}
				atomic.AddInt64(&incrementalBackupSize, int64(incrementalSnapSize))
				atomic.AddUint64(&incrementalApiReqCount, incrementalApiReq)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		klog.Errorf("failed to get snapshots size %d, number of api request %d", fullBackupSize, apiReqCount)
		return 0, 0, err
	}

	// currently, we do not count api request fees, since it is very few of cost, however, we print it in log in case "very few" is not correct
	klog.Infof("backup size %d bytes, number of api request %d, incremental backup size %d bytes, numbers of incremental size's api request %d",
		fullBackupSize, apiReqCount, incrementalBackupSize, incrementalApiReqCount)
	return
}

// calculateSnapshotSize calculate size of an snapshot in bytes by listing its blocks.
func calculateSnapshotSize(volumeId, snapshotId string) (uint64, uint64, error) {
	var snapshotSize uint64
	var numApiReq uint64

	start := time.Now()

	klog.Infof("start to calculate snapshot size for %s, base on snapshot %s, volume id %s",
		snapshotId, volumeId)

	ebsSession, err := util.NewEBSSession(util.CloudAPIConcurrency)
	if err != nil {
		klog.Errorf("new a ebs session failure.")
		return 0, numApiReq, err
	}

	var nextToken *string
	for {
		resp, err := ebsSession.EBS.ListSnapshotBlocks(&ebs.ListSnapshotBlocksInput{
			SnapshotId: aws.String(snapshotId),
			MaxResults: aws.Int64(ListSnapMaxReturnResult),
			NextToken:  nextToken,
		})
		numApiReq += 1
		if err != nil {
			return 0, numApiReq, err
		}
		if resp.BlockSize != nil {
			snapshotSize += uint64(len(resp.Blocks)) * uint64(*resp.BlockSize)
		}

		// check if there is more to retrieve
		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}

	elapsed := time.Since(start)

	klog.Infof("full snapshot size %s, num of ListSnapshotBlocks request %d, snapshot id %s, volume id %s, takes %v",
		humanize.Bytes(snapshotSize), numApiReq, snapshotId, volumeId, elapsed)

	return snapshotSize, numApiReq, nil
}

// calculateChangedBlocksSize calculates changed blocks total size in bytes between two snapshots with common ancestry.
func calculateChangedBlocksSize(volumeId, preSnapshotId, snapshotId string) (uint64, uint64, error) {
	var numBlocks int
	var snapshotSize uint64
	var numApiReq uint64

	start := time.Now()

	klog.Infof("start to calculate incremental snapshot size for %s, base on prev snapshot %s, volume id %s",
		snapshotId, preSnapshotId, volumeId)

	ebsSession, err := util.NewEBSSession(util.CloudAPIConcurrency)
	if err != nil {
		klog.Errorf("new a ebs session failure.")
		return 0, numApiReq, err
	}

	var nextToken *string

	for {
		resp, err := ebsSession.EBS.ListChangedBlocks(&ebs.ListChangedBlocksInput{
			FirstSnapshotId:  aws.String(preSnapshotId),
			MaxResults:       aws.Int64(ListSnapMaxReturnResult),
			SecondSnapshotId: aws.String(snapshotId),
			NextToken:        nextToken,
		})
		numApiReq += 1
		if err != nil {
			return 0, numApiReq, err
		}
		numBlocks += len(resp.ChangedBlocks)

		// retrieve only changed block and blocks only existed in current snapshot (new add blocks)
		for _, block := range resp.ChangedBlocks {
			if block.SecondBlockToken != nil && resp.BlockSize != nil {
				snapshotSize += uint64(*resp.BlockSize)
			}
		}

		// check if there is more to retrieve
		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}

	elapsed := time.Since(start)

	klog.Infof("incremental snapshot size %s, num of api ListChangedBlocks request %d, snapshot id %s, volume id %s, takes %v",
		humanize.Bytes(snapshotSize), numApiReq, snapshotId, volumeId, elapsed)

	return snapshotSize, numApiReq, nil
}

// getBackupVolSnapshots get a volume-snapshots map contains map[volumeId]{snapshot1, snapshot2, snapshot3}
func getVolSnapshots(volumeId string) ([]*ec2.Snapshot, error) {
	// read all snapshots from aws
	ec2Session, err := util.NewEC2Session(util.CloudAPIConcurrency)
	if err != nil {
		klog.Errorf("new a ec2 session failure.")
		return nil, err
	}

	filters := []*ec2.Filter{{Name: aws.String("volume-id"), Values: []*string{aws.String(volumeId)}}}
	// describe snapshot is heavy operator, try to call only once
	// api has limit with max 1000 snapshots
	// search with filter volume id the backupmeta contains
	var nextToken *string
	var snapshots []*ec2.Snapshot
	for {
		resp, err := ec2Session.EC2.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
			OwnerIds:   aws.StringSlice([]string{"self"}),
			MaxResults: aws.Int64(DescribeSnapMaxReturnResult),
			Filters:    filters,
			NextToken:  nextToken,
		})

		if err != nil {
			return nil, err
		}

		for _, s := range resp.Snapshots {
			if *s.State == ec2.SnapshotStateCompleted {
				klog.Infof("get the snapshot %s created for volume %s", *s.SnapshotId, *s.VolumeId)
				snapshots = append(snapshots, s)
			} else { // skip ongoing snapshots
				klog.Infof("the snapshot %s is creating... skip it, volume %s", *s.SnapshotId, *s.VolumeId)
				continue
			}
		}

		// check if there's more to retrieve
		if resp.NextToken == nil {
			break
		}
		klog.Infof("the total number of snapshot is %d", len(resp.Snapshots))
		nextToken = resp.NextToken
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].StartTime.Before(*snapshots[j].StartTime)
	})
	return snapshots, nil
}

func getPrevSnapshotId(snapshotId string, sortedVolSnapshots []*ec2.Snapshot) (string, bool) {
	for i, snapshot := range sortedVolSnapshots {
		if snapshotId == *snapshot.SnapshotId {
			if i == 0 {
				return "", false
			} else {
				return *sortedVolSnapshots[i-1].SnapshotId, true
			}
		}
	}
	return "", false
}
