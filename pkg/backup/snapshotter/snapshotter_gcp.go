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
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
)

// The GCPSnapshotter for creating snapshots from volumes (during a backup)
// and volumes from snapshots (during a restore) on Google Compute Engine Disks.
type GCPSnapshotter struct {
	BaseSnapshotter
}

func (s *GCPSnapshotter) Init(deps *controller.Dependencies, conf map[string]string) error {
	s.BaseSnapshotter.Init(deps, conf)
	s.volRegexp = regexp.MustCompile(`^projects\/[^\/]+\/(zones|regions)\/[^\/]+\/disks\/[^\/]+$`)
	return nil
}

func (s *GCPSnapshotter) GetVolumeID(pv *corev1.PersistentVolume) (string, error) {
	if pv == nil {
		return "", nil
	}

	if pv.Spec.CSI != nil {
		driver := pv.Spec.CSI.Driver
		if driver == constants.PdCSIDriver {
			handle := pv.Spec.CSI.VolumeHandle
			if !s.volRegexp.MatchString(handle) {
				return "", fmt.Errorf("invalid volumeHandle for CSI driver:%s, expected projects/{project}/zones/{zone}/disks/{name}, got %s",
					constants.PdCSIDriver, handle)
			}
			l := strings.Split(handle, "/")
			return l[len(l)-1], nil
		}
		return "", fmt.Errorf("unable to handle CSI driver: %s", driver)
	}

	if pv.Spec.GCEPersistentDisk != nil {
		if pv.Spec.GCEPersistentDisk.PDName == "" {
			return "", fmt.Errorf("spec.gcePersistentDisk.pdName not found")
		}
		return pv.Spec.GCEPersistentDisk.PDName, nil
	}

	return "", nil
}

func (s *GCPSnapshotter) GenerateBackupMetadata(b *v1alpha1.Backup, tc *v1alpha1.TidbCluster) (*CloudSnapBackup, string, error) {
	return s.BaseSnapshotter.generateBackupMetadata(b, tc, s)
}

func (s *GCPSnapshotter) SetVolumeID(pv *corev1.PersistentVolume, volumeID string) error {
	if pv.Spec.CSI != nil {
		// PV is provisioned by CSI driver
		driver := pv.Spec.CSI.Driver
		if driver == constants.PdCSIDriver {
			handle := pv.Spec.CSI.VolumeHandle
			// To restore in the same AZ, here we only replace the 'disk' chunk.
			if !s.volRegexp.MatchString(handle) {
				return fmt.Errorf("invalid volumeHandle for restore with CSI driver:%s, expected projects/{project}/zones/{zone}/disks/{name}, got %s",
					constants.PdCSIDriver, handle)
			}
			pv.Spec.CSI.VolumeHandle = handle[:strings.LastIndex(handle, "/")+1] + volumeID
		} else {
			return fmt.Errorf("unable to handle CSI driver: %s", driver)
		}
	} else if pv.Spec.GCEPersistentDisk != nil {
		// PV is provisioned by in-tree driver
		pv.Spec.GCEPersistentDisk.PDName = volumeID
	} else {
		return errors.New("spec.csi and spec.gcePersistentDisk not found")
	}

	return nil
}

func (s *GCPSnapshotter) PrepareRestoreMetadata(r *v1alpha1.Restore, csb *CloudSnapBackup) (string, error) {
	return s.BaseSnapshotter.prepareRestoreMetadata(r, csb, s)
}
