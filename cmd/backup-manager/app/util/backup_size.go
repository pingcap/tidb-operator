package util

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ebs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/util"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

// TODO: there shall be a abstract interface for diff cloud platform. e.g. GetBackupSize
// it may better to do this work after gcp support volume-snapshot

// CalcBackupSizeFromBackupmeta get backup size from backup meta
func CalcBackupSizeFromBackupmeta(ctx context.Context, provider v1alpha1.StorageProvider) (int64, error) {
	// read all snapshots from backup meta file
	volSnapshots, err := getSnapshotsFromBackupmeta(ctx, provider)
	if err != nil {
		return 0, err
	}
	// get all snapshots per backup volume from aws
	snapshots, err := getBackupVolSnapshots(volSnapshots)

	if err != nil {
		return 0, err
	}

	backupSize, err := calcBackupVolSnapshotSize(volSnapshots, snapshots)

	if err != nil {
		return 0, err
	}

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

// getBackupVolSnapshots get a volue-snapshots map contains map[volumeId]{snapshot1, snapshot2, snapshot3}
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
	resp, err := ec2Session.EC2.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
		OwnerIds:   aws.StringSlice([]string{"self"}),
		MaxResults: aws.Int64(1000),
		Filters:    filters,
	})

	if err != nil {
		return nil, err
	}

	for {
		for _, s := range resp.Snapshots {
			if *s.State == ec2.SnapshotStateCompleted {
				if volWithTheirSnapshots[*s.VolumeId] == nil {
					klog.Errorf("search with filter[volume-id] received unexpected result, volumeId:%s, snapshotId:%s", *s.VolumeId, *s.SnapshotId)
					break
				}
				klog.Infof("the snapshot %s created for volume %s", *s.SnapshotId, *s.VolumeId)
				volWithTheirSnapshots[*s.VolumeId] = append(volWithTheirSnapshots[*s.VolumeId], s)
			} else { // skip ongoing snapshots
				klog.Infof("the snapshot %s creating... skip it", *s.SnapshotId)
				continue
			}
		}

		// check if there's more to retrieve
		if resp.NextToken == nil {
			break
		}

		resp, err = ec2Session.EC2.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
			OwnerIds:   aws.StringSlice([]string{"self"}),
			MaxResults: aws.Int64(1000),
			Filters:    filters,
			NextToken:  resp.NextToken,
		})

		if err != nil {
			return nil, err
		}
	}

	return volWithTheirSnapshots, nil
}

// CalcBackupVolSnapshotSize get a volue-snapshots map contains map[volumeId]{snapshot1, snapshot2, snapshot3}
func calcBackupVolSnapshotSize(volumes map[string]string, snapshots map[string][]*ec2.Snapshot) (uint64, error) {
	var backupSize uint64

	for volumeId, snapshotId := range volumes {
		volSnapshots := snapshots[volumeId]
		// full snapshot backup
		if len(volSnapshots) == 1 {
			snapSize, err := initialSnapshotSize(snapshotId)
			if err != nil {
				return 0, err
			}

			backupSize += snapSize
			continue
		}

		// incremental snapshot backup
		prevSnapshot, err := getPrevSnapshotId(snapshotId, volSnapshots)
		if err != nil {
			return 0, err
		}

		// snapshot is full snapshot / first snapshot
		if prevSnapshot == "" {
			snapSize, err := initialSnapshotSize(snapshotId)
			if err != nil {
				return 0, err
			}

			backupSize += snapSize
			continue
		}
		snapSize, err := changedBlocksSize(prevSnapshot, snapshotId)
		if err != nil {
			return 0, err
		}

		backupSize += snapSize
	}

	klog.Infof("backup size %d bytes", backupSize)
	return backupSize, nil
}

// initialSnapshotSize calculate size of an initial snapshot in bytes by listing its blocks.
func initialSnapshotSize(snapshotId string) (uint64, error) {
	var numBlocks uint64
	ebsSession, err := util.NewEBSSession(util.CloudAPIConcurrency)
	if err != nil {
		klog.Errorf("new a ebs session failure.")
		return 0, err
	}
	resp, err := ebsSession.EBS.ListSnapshotBlocks(&ebs.ListSnapshotBlocksInput{
		SnapshotId: aws.String(snapshotId),
		MaxResults: aws.Int64(10000),
	})

	if err != nil {
		return 0, err
	}

	blockSize := uint64(*resp.BlockSize)
	for {
		numBlocks += uint64(len(resp.Blocks))
		// check if there is more to retrieve
		if resp.NextToken == nil {
			break
		}
		resp, err = ebsSession.EBS.ListSnapshotBlocks(&ebs.ListSnapshotBlocksInput{
			SnapshotId: aws.String(snapshotId),
			MaxResults: aws.Int64(10000),
			NextToken:  resp.NextToken,
		})

		if err != nil {
			return 0, err
		}
	}
	klog.Infof("full backup snapshot num block %d, block size %d", numBlocks, blockSize)
	return numBlocks * blockSize, nil
}

func getPrevSnapshotId(snapshotId string, volSnapshots []*ec2.Snapshot) (string, error) {
	// sort snapshots by timestamp
	sort.Slice(volSnapshots, func(i, j int) bool {
		return volSnapshots[i].StartTime.Before(*volSnapshots[j].StartTime)
	})
	var prevSnapshotId string
	for i, snapshot := range volSnapshots {
		klog.Infof("the snapshot %s", *snapshot.SnapshotId)
		if snapshotId == *snapshot.SnapshotId {
			// first snapshot
			if i == 0 {
				return "", nil
			}
			prevSnapshotId = *volSnapshots[i-1].SnapshotId
			break
		}
	}
	if len(prevSnapshotId) == 0 {
		return "", fmt.Errorf("Could not find the prevousely snapshot id, current snapshotId: %s.", snapshotId)
	}
	return prevSnapshotId, nil
}

// changedBlocksSize calculates changed blocks total size in bytes between two snapshots with common ancestry.
func changedBlocksSize(preSnapshotId string, snapshotId string) (uint64, error) {
	var numBlocks uint64
	ebsSession, err := util.NewEBSSession(util.CloudAPIConcurrency)
	if err != nil {
		klog.Errorf("new a ebs session failure.")
		return 0, err
	}
	resp, err := ebsSession.EBS.ListChangedBlocks(&ebs.ListChangedBlocksInput{
		FirstSnapshotId:  aws.String(preSnapshotId),
		MaxResults:       aws.Int64(10000),
		SecondSnapshotId: aws.String(snapshotId),
	})

	if err != nil {
		return 0, err
	}

	blockSize := uint64(*resp.BlockSize)
	klog.Infof("the preSnapshotId %s, the current snapshotId %s", preSnapshotId, snapshotId)
	for {
		// retrieve only changed block and blocks only existed in current snapshot (new add blocks)
		for _, block := range resp.ChangedBlocks {
			if block.SecondBlockToken != nil && len(aws.StringValue(block.SecondBlockToken)) != 0 {
				numBlocks += 1
			}
		}

		klog.Infof("the current num blocks %d", numBlocks)
		// check if there is more to retrieve
		if resp.NextToken == nil {
			break
		}

		resp, err = ebsSession.EBS.ListChangedBlocks(&ebs.ListChangedBlocksInput{
			FirstSnapshotId:  aws.String(preSnapshotId),
			MaxResults:       aws.Int64(10000),
			SecondSnapshotId: aws.String(snapshotId),
			NextToken:        resp.NextToken,
		})

		if err != nil {
			return 0, err
		}
	}
	klog.Infof("the total num of blocks %d", numBlocks)
	return numBlocks * blockSize, nil
}
