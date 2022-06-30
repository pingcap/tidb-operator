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

package snapshotter

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
)

// The snapshotter for creating snapshots from volumes (during a backup)
// and volumes from snapshots (during a restore) on AWS EBS.
type AWSSnapshotter struct {
	BaseSnapshotter
}

func (s *AWSSnapshotter) Init(deps *controller.Dependencies, conf map[string]string) error {
	s.BaseSnapshotter.Init(deps, conf)
	s.volRegexp = regexp.MustCompile("vol-.*")
	return nil
}

func (s *AWSSnapshotter) GetVolumeID(pv *corev1.PersistentVolume) (string, error) {
	if pv == nil {
		return "", nil
	}

	if pv.Spec.CSI != nil {
		driver := pv.Spec.CSI.Driver
		if driver == constants.EbsCSIDriver {
			return s.volRegexp.FindString(pv.Spec.CSI.VolumeHandle), nil
		}
		return "", fmt.Errorf("unable to handle CSI driver: %s", driver)
	}
	if pv.Spec.AWSElasticBlockStore != nil {
		if pv.Spec.AWSElasticBlockStore.VolumeID == "" {
			return "", fmt.Errorf("spec.awsElasticBlockStore.volumeID not found")
		}
		return s.volRegexp.FindString(pv.Spec.AWSElasticBlockStore.VolumeID), nil
	}

	return "", nil
}

func (s *AWSSnapshotter) PrepareBackupMetadata(b *v1alpha1.Backup, tc *v1alpha1.TidbCluster, ns string) (string, error) {
	return s.BaseSnapshotter.prepareBackupMetadata(b, tc, ns, s)
}

func (s *AWSSnapshotter) SetVolumeID(pv *corev1.PersistentVolume, volumeID string) (*corev1.PersistentVolume, error) {
	newPV := pv.DeepCopy()

	if pv.Spec.CSI != nil {
		// PV is provisioned by CSI driver
		driver := pv.Spec.CSI.Driver
		if driver == constants.EbsCSIDriver {
			newPV.Spec.CSI.VolumeHandle = volumeID
		} else {
			return nil, fmt.Errorf("unable to handle CSI driver: %s", driver)
		}
	} else if pv.Spec.AWSElasticBlockStore != nil {
		// PV is provisioned by in-tree driver
		pvFailureDomainZone := pv.Labels["failure-domain.beta.kubernetes.io/zone"]
		if len(pvFailureDomainZone) > 0 {
			newPV.Spec.AWSElasticBlockStore.VolumeID = fmt.Sprintf("aws://%s/%s", pvFailureDomainZone, volumeID)
		} else {
			newPV.Spec.AWSElasticBlockStore.VolumeID = volumeID
		}
	} else {
		return nil, errors.New("spec.csi and spec.awsElasticBlockStore not found")
	}

	return newPV, nil
}

func (s *AWSSnapshotter) PrepareRestoreMetadata(r *v1alpha1.Restore) (string, error) {
	return s.BaseSnapshotter.prepareRestoreMetadata(r, s)
}
