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
	// This value can be between 5 and 1,000; if MaxResults is given a value larger than 1,000, only 1,000 results are returned.
	DescribeSnapMaxReturnResult = 1000

	// This value can be between 100 and 1,0000, and charge ~0.6$/1 million request
	ListSnapMaxReturnResult = 10000

	// This value can be between 100 and 1,0000, and charge ~0.6$/1 million request
	ListBlocksMaxReturnResult = 10000

	// This value can be between 1 and 50 due to aws service quota
	EbsApiConcurrency = 40
)

// CalcVolSnapBackupSize get snapshots from backup meta and then calc the backup size of snapshots.
func CalcVolSnapBackupSize(ctx context.Context, provider v1alpha1.StorageProvider) (int64, error) {
	start := time.Now()
	// retrieves all snapshots from backup meta file
	volSnapshots, err := getSnapshotsFromBackupmeta(ctx, provider)
	if err != nil {
		return 0, err
	}

	if err != nil {
		return 0, err
	}

	backupSize, err := calcBackupSize(ctx, volSnapshots)

	if err != nil {
		return 0, err
	}
	elapsed := time.Since(start)
	klog.Infof("calculate volume-snapshot backup size takes %v", elapsed)
	return int64(backupSize), nil
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

// getBackupVolSnapshots get a volume-snapshots map contains map[volumeId]{snapshot1, snapshot2, snapshot3}
func getBackupVolSnapshots(volumes map[string]string) (map[string][]*ec2.Snapshot, error) {
	volWithTheirSnapshots := make(map[string][]*ec2.Snapshot)

	// read all snapshots from aws
	ec2Session, err := util.NewEC2Session(util.CloudAPIConcurrency)
	if err != nil {
		klog.Errorf("new a ec2 session failure.")
		return nil, err
	}

	// init search filter.Values
	// init volWithTheirSnapshots
	volValues := make([]*string, 0)
	for volumeId := range volumes {
		volValues = append(volValues, aws.String(volumeId))
		if volWithTheirSnapshots[volumeId] == nil {
			volWithTheirSnapshots[volumeId] = make([]*ec2.Snapshot, 0)
		}
	}

	filters := []*ec2.Filter{{Name: aws.String("volume-id"), Values: volValues}}
	// describe snapshot is heavy operator, try to call only once
	// api has limit with max 1000 snapshots
	// search with filter volume id the backupmeta contains
	var nextToken *string
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

		for i, s := range resp.Snapshots {
			if *s.State == ec2.SnapshotStateCompleted {
				if volWithTheirSnapshots[*s.VolumeId] == nil {
					klog.Errorf("search with filter[volume-id] received unexpected result, volumeId:%s, snapshotId:%s", *s.VolumeId, *s.SnapshotId)
					break
				}
				klog.Infof("the snapshot#%d %s created for volume %s", i, *s.SnapshotId, *s.VolumeId)
				volWithTheirSnapshots[*s.VolumeId] = append(volWithTheirSnapshots[*s.VolumeId], s)
			} else { // skip ongoing snapshots
				klog.Infof("the snapshot#%d %s creating... skip it", i, *s.SnapshotId)
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

	return volWithTheirSnapshots, nil
}

// calcBackupSize get a volume-snapshots backup size
func calcBackupSize(ctx context.Context, volumes map[string]string) (uint64, error) {
	var backupSize uint64
	var apiReqCount uint64

	workerPool := util.NewWorkerPool(EbsApiConcurrency, "list snapshot size")
	eg, _ := errgroup.WithContext(ctx)

	for _, id := range volumes {
		snapshotId := id
		// sort snapshots by timestamp
		workerPool.ApplyOnErrorGroup(eg, func() error {
			snapSize, apiReq, err := initialSnapshotSize(snapshotId)
			if err != nil {
				return err
			}
			atomic.AddUint64(&backupSize, snapSize)
			atomic.AddUint64(&apiReqCount, apiReq)
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		klog.Errorf("failed to get snapshots size %d, number of api request %d", backupSize, apiReqCount)
		return 0, err
	}

	// currently, we do not count api request fees, since it is very few of cost, however, we print it in log in case "very few" is not correct
	klog.Infof("backup size %d bytes, number of api request %d", backupSize, apiReqCount)
	return backupSize, nil
}

// initialSnapshotSize calculate size of an initial snapshot in bytes by listing its blocks.
// initial snapshot always a ful backup of volume
func initialSnapshotSize(snapshotId string) (uint64, uint64, error) {
	var snapshotSize uint64
	var numApiReq uint64
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
	klog.Infof("full backup snapshot size %d bytes, num of ListSnapshotBlocks request %d", snapshotSize, numApiReq)
	return snapshotSize, numApiReq, nil
}

func getPrevSnapshotId(snapshotId string, volSnapshots []*ec2.Snapshot) (string, error) {
	var prevSnapshotId string

	sort.Slice(volSnapshots, func(i, j int) bool {
		return volSnapshots[i].StartTime.Before(*volSnapshots[j].StartTime)
	})

	for i, snapshot := range volSnapshots {
		if snapshotId == *snapshot.SnapshotId {
			// the first snapshot for the volume
			if i == 0 {
				return "", nil
			}
			prevSnapshotId = *volSnapshots[i-1].SnapshotId
			klog.Infof("the prevSnapshot index is %d, ID is %s", i, *snapshot.SnapshotId)
			break
		}
	}
	if len(prevSnapshotId) == 0 {
		return "", fmt.Errorf("Could not find the prevousely snapshot id, current snapshotId: %s.", snapshotId)
	}
	return prevSnapshotId, nil
}

// changedBlocksSize calculates changed blocks total size in bytes between two snapshots with common ancestry.
func changedBlocksSize(preSnapshotId string, snapshotId string) (uint64, uint64, error) {
	var numBlocks int
	var snapshotSize uint64
	var numApiReq uint64

	klog.Infof("the calc snapshot size for %s, base on prev snapshot %s", snapshotId, preSnapshotId)
	ebsSession, err := util.NewEBSSession(util.CloudAPIConcurrency)
	if err != nil {
		klog.Errorf("new a ebs session failure.")
		return 0, numApiReq, err
	}

	var nextToken *string

	for {
		resp, err := ebsSession.EBS.ListChangedBlocks(&ebs.ListChangedBlocksInput{
			FirstSnapshotId:  aws.String(preSnapshotId),
			MaxResults:       aws.Int64(ListBlocksMaxReturnResult),
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
	klog.Infof("the total size of snapshot %d, num of api ListChangedBlocks request %d, snapshot id %s", snapshotSize, numApiReq, snapshotId)
	return snapshotSize, numApiReq, nil
}
