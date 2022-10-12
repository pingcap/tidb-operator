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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/pingcap/errors"
	"go.uber.org/atomic"
	"golang.org/x/sync/errgroup"
	"k8s.io/klog/v2"
)

// TODO: shall this structure be refactor or reserved for future use?
type EBSVolumeType string

const (
	GP3Volume EBSVolumeType = "gp3"
	IO1Volume EBSVolumeType = "io1"
	IO2Volume EBSVolumeType = "io2"
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

type Kubernetes struct {
	PVs     []interface{}          `json:"pvs" toml:"pvs"`
	PVCs    []interface{}          `json:"pvcs" toml:"pvcs"`
	CRD     interface{}            `json:"crd_tidb_cluster" toml:"crd_tidb_cluster"`
	Options map[string]interface{} `json:"options" toml:"options"`
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
	KubernetesMeta *Kubernetes            `json:"kubernetes" toml:"kubernetes"`
	Options        map[string]interface{} `json:"options" toml:"options"`
	Region         string                 `json:"region" toml:"region"`
}

type EC2Session struct {
	ec2 ec2iface.EC2API
	// aws operation concurrency
	concurrency uint
}

type VolumeAZs map[string]string

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
	ec2Session := ec2.New(sess)
	return &EC2Session{ec2: ec2Session, concurrency: concurrency}, nil
}

func (e *EC2Session) DeleteSnapshots(snapIDMap map[string]string) error {

	var deletedCnt atomic.Int32
	eg := new(errgroup.Group)
	for volID := range snapIDMap {
		snapID := snapIDMap[volID]
		eg.Go(func() error {
			_, err := e.ec2.DeleteSnapshot(&ec2.DeleteSnapshotInput{
				SnapshotId: &snapID,
			})
			if err != nil {
				klog.Errorf("failed to delete snapshot id=%s, error=%s", snapID, err)
				// todo: we can only retry for a few times, might fail still, need to handle error from outside.
				// we don't return error if it fails to make sure all snapshot got chance to delete.
			} else {
				deletedCnt.Add(1)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		klog.Errorf("failed to delete snapshots error=%s, already delete=%d", err, deletedCnt.Load())
		return err
	}
	return nil
}
