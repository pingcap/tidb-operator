// Copyright 2022 PingCAP, Inc.
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
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ebs"
	"github.com/aws/aws-sdk-go/service/ebs/ebsiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

// TODO: shall this structure be refactor or reserved for future use?
type EBSVolumeType string

const (
	GP3Volume                           EBSVolumeType = "gp3"
	IO1Volume                           EBSVolumeType = "io1"
	IO2Volume                           EBSVolumeType = "io2"
	CloudAPIConcurrency                               = 3
	SnapshotDeletionFlowControlInterval               = 10
)

func (t EBSVolumeType) Valid() bool {
	return t == GP3Volume || t == IO1Volume || t == IO2Volume
}

// EBSVolume is passed by TiDB deployment tools: TiDB Operator and TiUP(in future)
// we should do snapshot inside BR, because we need some logic to determine the order of snapshot starts.
// TODO finish the info with TiDB Operator developer.
type EBSVolume struct {
	ID              string `json:"volume_id" toml:"volume_id"`
	Type            string `json:"type" toml:"type"`
	SnapshotID      string `json:"snapshot_id" toml:"snapshot_id"`
	RestoreVolumeId string `json:"restore_volume_id" toml:"restore_volume_id"`
	VolumeAZ        string `json:"volume_az" toml:"volume_az"`
	Status          string `json:"status" toml:"status"`
}

type EBSStore struct {
	StoreID uint64       `json:"store_id" toml:"store_id"`
	Volumes []*EBSVolume `json:"volumes" toml:"volumes"`
}

// ClusterInfo represents the tidb cluster level meta infos. such as
// pd cluster id/alloc id, cluster resolved ts and tikv configuration.
type ClusterInfo struct {
	Version        string            `json:"cluster_version" toml:"cluster_version"`
	FullBackupType string            `json:"full_backup_type" toml:"full_backup_type"`
	ResolvedTS     uint64            `json:"resolved_ts" toml:"resolved_ts"`
	Replicas       map[string]uint64 `json:"replicas" toml:"replicas"`
}

type KubernetesBackup struct {
	PVCs         []*corev1.PersistentVolumeClaim `json:"pvcs"`
	PVs          []*corev1.PersistentVolume      `json:"pvs"`
	TiDBCluster  *v1alpha1.TidbCluster           `json:"crd_tidb_cluster"`
	Unstructured *unstructured.Unstructured      `json:"options"`
}

type TiKVComponent struct {
	Replicas int         `json:"replicas"`
	Stores   []*EBSStore `json:"stores"`
}

type PDComponent struct {
	Replicas int `json:"replicas"`
}

type TiDBComponent struct {
	Replicas int `json:"replicas"`
}

type EBSBasedBRMeta struct {
	ClusterInfo    *ClusterInfo           `json:"cluster_info" toml:"cluster_info"`
	TiKVComponent  *TiKVComponent         `json:"tikv" toml:"tikv"`
	TiDBComponent  *TiDBComponent         `json:"tidb" toml:"tidb"`
	PDComponent    *PDComponent           `json:"pd" toml:"pd"`
	KubernetesMeta *KubernetesBackup      `json:"kubernetes" toml:"kubernetes"`
	Options        map[string]interface{} `json:"options" toml:"options"`
	Region         string                 `json:"region" toml:"region"`
}

type EC2Session struct {
	EC2 ec2iface.EC2API
	// aws operation concurrency
	concurrency uint
}

type TagMap map[string]string

func NewEC2Session(concurrency uint) (*EC2Session, error) {
	// aws-sdk has builtin exponential backoff retry mechanism, see:
	// https://github.com/aws/aws-sdk-go/blob/db4388e8b9b19d34dcde76c492b17607cd5651e2/aws/client/default_retryer.go#L12-L16
	// with default retryer & max-retry=9, we will wait for at least 30s in total
	awsConfig := aws.NewConfig().WithMaxRetries(9)
	// TiDB Operator need make sure we have the correct permission to call aws api(through aws env variables)
	// we may change this behaviour in the future.
	sessionOptions := session.Options{Config: *awsConfig}
	sess, err := session.NewSessionWithOptions(sessionOptions)
	if err != nil {
		return nil, errors.Trace(err)
	}

	region := os.Getenv(constants.AWSRegionEnv)
	if region == "" {
		ec2Metadata := ec2metadata.New(sess)
		region, err = ec2Metadata.Region()
		if err != nil {
			return nil, errors.Annotate(err, "get ec2 region")
		}
	}

	ec2Session := ec2.New(sess, aws.NewConfig().WithRegion(region))
	return &EC2Session{EC2: ec2Session, concurrency: concurrency}, nil
}

func (e *EC2Session) DeleteSnapshots(snapIDMap map[string]string) error {
	var deletedCnt int32
	lastFlowCheck := time.Now()
	klog.Infof("Start deleting snapshots, total is %d", len(snapIDMap))
	for volID := range snapIDMap {
		snapID := snapIDMap[volID]
		klog.Infof("deleting snapshot %s ", snapID)
		// use exponential backoff, every retry duration is duration * factor ^ (used_step - 1)
		backoff := wait.Backoff{
			Duration: time.Second,
			Steps:    8,
			Factor:   2.0,
			Cap:      time.Minute,
		}
		delSnapshots := func() error {
			_, err := e.EC2.DeleteSnapshot(&ec2.DeleteSnapshotInput{
				SnapshotId: &snapID,
			})
			if err != nil {
				if aErr, ok := err.(awserr.Error); ok {
					if aErr.Code() == "InvalidSnapshot.NotFound" {
						klog.Warningf("snapshot %s not found", snapID, err.Error())
						return nil
					}
				}
				klog.Warningf("delete snapshot %s failed, err: %s", snapID, err.Error())
				return err
			} else {
				klog.Infof("snapshot %s is deleted", snapID)
				deletedCnt++
				// Check flow every 10 deletions, we try to make no more than 1 deletion/second.
				if deletedCnt%SnapshotDeletionFlowControlInterval == 0 {
					lastRoundDuration := time.Since(lastFlowCheck)
					klog.Infof("deletion count is %d, last round costs %s", deletedCnt, lastRoundDuration)
					if lastRoundDuration < SnapshotDeletionFlowControlInterval*time.Second {
						suspension := SnapshotDeletionFlowControlInterval*time.Second - lastRoundDuration
						klog.Infof("Snapshot deletion flow control for %s", suspension)
						time.Sleep(suspension)
					}
					lastFlowCheck = time.Now()
				}
				return nil
			}
		}

		isRetry := func(err error) bool {
			return request.IsErrorThrottle(err)
		}

		err := retry.OnError(backoff, isRetry, delSnapshots)
		if err != nil {
			klog.Errorf("failed to delete snapshot id=%s, error=%s", snapID, err.Error())
			return err
		}
	}

	return nil
}

func (e *EC2Session) AddTags(resourcesTags map[string]TagMap) error {

	eg, _ := errgroup.WithContext(context.Background())
	workerPool := NewWorkerPool(e.concurrency, "add tags")
	for resourceID := range resourcesTags {
		id := resourceID
		tagMap := resourcesTags[resourceID]
		var tags []*ec2.Tag
		for tag := range tagMap {
			tagKey := tag
			value := tagMap[tag]
			tags = append(tags, &ec2.Tag{Key: &tagKey, Value: &value})
		}

		// Create the input for adding the tag
		input := &ec2.CreateTagsInput{
			Resources: []*string{aws.String(id)},
			Tags:      tags,
		}

		workerPool.ApplyOnErrorGroup(eg, func() error {
			_, err := e.EC2.CreateTags(input)
			if err != nil {
				klog.Errorf("failed to create tags for resource id=%s, %v", id, err)
				return err
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		klog.Errorf("failed to create tags for all resources")
		return err
	}
	return nil
}

type EBSSession struct {
	EBS ebsiface.EBSAPI
	// aws operation concurrency
	concurrency uint
}

func NewEBSSession(concurrency uint) (*EBSSession, error) {
	awsConfig := aws.NewConfig().WithMaxRetries(9)
	// TiDB Operator need make sure we have the correct permission to call aws api(through aws env variables)
	sessionOptions := session.Options{Config: *awsConfig}
	sess, err := session.NewSessionWithOptions(sessionOptions)
	if err != nil {
		return nil, errors.Trace(err)
	}
	region := os.Getenv(constants.AWSRegionEnv)
	if region == "" {
		ec2Metadata := ec2metadata.New(sess)
		region, err = ec2Metadata.Region()
		if err != nil {
			return nil, errors.Annotate(err, "get ec2 region")
		}
	}

	ebsSession := ebs.New(sess, aws.NewConfig().WithRegion(region))
	return &EBSSession{EBS: ebsSession, concurrency: concurrency}, nil
}
