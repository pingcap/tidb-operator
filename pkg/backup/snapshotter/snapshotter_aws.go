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
	"github.com/pingcap/tidb-operator/pkg/backup/util"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
)

const (
	CloudAPIConcurrency = 8
	PVCTagKey           = "CSIVolumeName"
	PvNameTagKey        = "kubernetes.io/created-for/pv/name"
	PvcNameTagKey       = "kubernetes.io/created-for/pvc/name"
	PvcNSTagKey         = "kubernetes.io/created-for/pvc/namespace"
)

// AWSSnapshotter is the snapshotter for creating snapshots from volumes (during a backup)
// and volumes from snapshots (during a restore) on AWS EBS.
type AWSSnapshotter struct {
	BaseSnapshotter
}

func (s *AWSSnapshotter) Init(deps *controller.Dependencies, conf map[string]string) error {
	err := s.BaseSnapshotter.Init(deps, conf)
	s.volRegexp = regexp.MustCompile("vol-.*")
	return err
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

func (s *AWSSnapshotter) GenerateBackupMetadata(b *v1alpha1.Backup, tc *v1alpha1.TidbCluster) (*CloudSnapBackup, string, error) {
	return s.BaseSnapshotter.generateBackupMetadata(b, tc, s)
}

func (s *AWSSnapshotter) SetVolumeID(pv *corev1.PersistentVolume, volumeID string) error {
	if pv.Spec.CSI != nil {
		// PV is provisioned by CSI driver
		driver := pv.Spec.CSI.Driver
		if driver == constants.EbsCSIDriver {
			pv.Spec.CSI.VolumeHandle = volumeID
		} else {
			return fmt.Errorf("unable to handle CSI driver: %s", driver)
		}
	} else if pv.Spec.AWSElasticBlockStore != nil {
		// PV is provisioned by in-tree driver
		pvFailureDomainZone := pv.Labels["failure-domain.beta.kubernetes.io/zone"]
		if len(pvFailureDomainZone) > 0 {
			pv.Spec.AWSElasticBlockStore.VolumeID = fmt.Sprintf("aws://%s/%s", pvFailureDomainZone, volumeID)
		} else {
			pv.Spec.AWSElasticBlockStore.VolumeID = volumeID
		}
	} else {
		return errors.New("spec.csi and spec.awsElasticBlockStore not found")
	}

	return nil
}

func (s *AWSSnapshotter) PrepareRestoreMetadata(r *v1alpha1.Restore, csb *CloudSnapBackup) (string, error) {
	return s.BaseSnapshotter.prepareRestoreMetadata(r, csb, s)
}

func (s *AWSSnapshotter) AddVolumeTags(pvs []*corev1.PersistentVolume) error {
	resourcesTags := make(map[string]util.TagMap)

	for _, pv := range pvs {
		tags := make(map[string]string)
		tags[PVCTagKey] = pv.GetName()
		tags[PvNameTagKey] = pv.GetName()
		if pv.Spec.ClaimRef != nil {
			tags[PvcNameTagKey] = pv.Spec.ClaimRef.Name
			tags[PvcNSTagKey] = pv.Spec.ClaimRef.Namespace
		}

		resourcesTags[pv.GetAnnotations()[constants.AnnRestoredVolumeID]] = tags
	}
	ec2Session, err := util.NewEC2Session(CloudAPIConcurrency)
	if err != nil {
		return err
	}
	if err = ec2Session.AddTags(resourcesTags); err != nil {
		return err
	}

	return nil

}

func (s *AWSSnapshotter) ResetPvAvailableZone(r *v1alpha1.Restore, pv *corev1.PersistentVolume) {
	if r.Spec.VolumeAZ == "" {
		return
	}

	restoreAZ := r.Spec.VolumeAZ
	if pv.Spec.NodeAffinity == nil {
		return
	}
	if pv.Spec.NodeAffinity.Required == nil {
		return
	}
	for i, nodeSelector := range pv.Spec.NodeAffinity.Required.NodeSelectorTerms {
		for j, field := range nodeSelector.MatchFields {
			if field.Key == constants.NodeAffinityCsiEbsAzKey {
				pv.Spec.NodeAffinity.Required.NodeSelectorTerms[i].MatchFields[j].Values = []string{restoreAZ}
			}
		}
		for j, expr := range nodeSelector.MatchExpressions {
			if expr.Key == constants.NodeAffinityCsiEbsAzKey && expr.Operator == corev1.NodeSelectorOpIn {
				pv.Spec.NodeAffinity.Required.NodeSelectorTerms[i].MatchExpressions[j].Values = []string{restoreAZ}
			}
		}
	}
}

func (s *AWSSnapshotter) CleanVolumes(r *v1alpha1.Restore, csb *CloudSnapBackup) error {
	if !v1alpha1.IsRestoreVolumeFailed(r) {
		return errors.New("can't clean volumes if not restore volume failed")
	}

	volumeIDs := s.getRestoreVolumeIDs(csb)
	ec2Session, err := util.NewEC2Session(util.CloudAPIConcurrency)
	if err != nil {
		return fmt.Errorf("new ec2 session error: %w", err)
	}
	if err := ec2Session.DeleteVolumes(volumeIDs); err != nil {
		return fmt.Errorf("delete volumes error: %w", err)
	}
	return nil
}
