#!/bin/bash

# Copyright 2021 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# See the License for the specific language governing permissions and
# limitations under the License.

# This script patches the chart templates of tidb-operator to use
# in E2E tests.

set -e

echo "hack/e2e-patch-codecov.sh: PWD $PWD"

CONTROLLER_MANAGER_DEPLOYMENT=charts/tidb-operator/templates/controller-manager-deployment.yaml
DISCOVERY_DEPLOYMENT=charts/tidb-cluster/templates/discovery-deployment.yaml
ADMISSION_WEBHOOK_DEPLOYMENT=charts/tidb-operator/templates/admission/admission-webhook-deployment.yaml
TMP_ADMISSION_WEBHOOK_DEPLOYMENT=/tmp/admission-webhook-deployment.yaml

DISCOVERY_MANAGER=pkg/manager/member/tidb_discovery_manager.go
TMP_DISCOVERY_MANAGER=/tmp/tidb_discovery_manager.go
RESTORE_MANAGER=pkg/backup/restore/restore_manager.go
TMP_RESTORE_MANAGER=/tmp/restore_manager.go
BACKUP_MANAGER=pkg/backup/backup/backup_manager.go
TMP_BACKUP_MANAGER=/tmp/backup_manager.go
BACKUP_CLEANER=pkg/backup/backup/backup_cleaner.go
TMP_BACKUP_CLEANER=/tmp/backup_cleaner.go

echo "replace the entrypoint to generate and upload the coverage profile"
sed -i 's/\/usr\/local\/bin\/tidb-controller-manager/\/e2e-entrypoint.sh\n          - \/usr\/local\/bin\/tidb-controller-manager\n          - -test.coverprofile=\/coverage\/tidb-controller-manager.cov\n          - E2E/g' \
    $CONTROLLER_MANAGER_DEPLOYMENT
sed -i 's/\/usr\/local\/bin\/tidb-discovery/\/e2e-entrypoint.sh\n          - \/usr\/local\/bin\/tidb-discovery\n          - -test.coverprofile=\/coverage\/tidb-discovery.cov\n          - E2E/g' \
    $DISCOVERY_DEPLOYMENT
sed -i 's/\/usr\/local\/bin\/tidb-admission-webhook/\/e2e-entrypoint.sh\n            - \/usr\/local\/bin\/tidb-admission-webhook\n            - -test.coverprofile=\/coverage\/tidb-admission-webhook.cov\n            - E2E/g' \
    $ADMISSION_WEBHOOK_DEPLOYMENT

# -v is duplicated for operator and go test
sed -i '/\-v=/d' $CONTROLLER_MANAGER_DEPLOYMENT
sed -i '/\-v=/d' $ADMISSION_WEBHOOK_DEPLOYMENT

# populate needed environment variables and local-path volumes
echo "hack/e2e-patch-codecov.sh: setting environment variables and volumes in charts"
sed -i 's/^{{\- end }}$//g' $CONTROLLER_MANAGER_DEPLOYMENT
cat >> $CONTROLLER_MANAGER_DEPLOYMENT << EOF
          - name: COMPONENT
            value: "controller-manager"
        volumeMounts:
          - mountPath: /coverage
            name: coverage
      volumes:
        - name: coverage
          hostPath:
            path: /mnt/disks/coverage
            type: Directory
{{- end }}
EOF

line=$(grep -n 'volumeMounts:' $ADMISSION_WEBHOOK_DEPLOYMENT | cut -d ":" -f 1)
head -n $(($line-1)) $ADMISSION_WEBHOOK_DEPLOYMENT > $TMP_ADMISSION_WEBHOOK_DEPLOYMENT
cat >> $TMP_ADMISSION_WEBHOOK_DEPLOYMENT <<EOF
          - name: COMPONENT
            value: "admission-webhook"
          volumeMounts:
            - mountPath: /coverage
              name: coverage
EOF
tail -n +$(($line+1)) $ADMISSION_WEBHOOK_DEPLOYMENT >> $TMP_ADMISSION_WEBHOOK_DEPLOYMENT
sed -i 's/^{{\- end }}$//g' $TMP_ADMISSION_WEBHOOK_DEPLOYMENT
cat >> $TMP_ADMISSION_WEBHOOK_DEPLOYMENT << EOF
        - name: coverage
          hostPath:
            path: /mnt/disks/coverage
            type: Directory
{{- end }}
EOF
mv -f $TMP_ADMISSION_WEBHOOK_DEPLOYMENT $ADMISSION_WEBHOOK_DEPLOYMENT

echo "hack/e2e-patch-codecov.sh: setting command, environment variables and volumes for golang code"

line=$(grep -n 'm.getTidbDiscoveryDeployment(metaObj)' $DISCOVERY_MANAGER | cut -d ":" -f 1)
head -n $(($line+3)) $DISCOVERY_MANAGER > $TMP_DISCOVERY_MANAGER
cat >> $TMP_DISCOVERY_MANAGER <<EOF
	d.Spec.Template.Spec.Containers[0].Command = []string{
		"/e2e-entrypoint.sh",
		"/usr/local/bin/tidb-discovery",
		"-test.coverprofile=/coverage/tidb-discovery.cov",
		"E2E",
	}
	d.Spec.Template.Spec.Containers[0].Env = append(d.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "COMPONENT",
		Value: "discovery",
	})
	volType := corev1.HostPathDirectory
	d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: "coverage",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/mnt/disks/coverage",
				Type: &volType,
			},
		},
	})
	d.Spec.Template.Spec.Containers[0].VolumeMounts = append(d.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
		Name:      "coverage",
		MountPath: "/coverage",
	})
EOF
tail -n +$(($line+4)) $DISCOVERY_MANAGER >> $TMP_DISCOVERY_MANAGER
mv -f $TMP_DISCOVERY_MANAGER $DISCOVERY_MANAGER

IFS= read -r -d '' PATCH_BR_JOB << EOF || true
	job.Spec.Template.Spec.Containers[0].Env = append(job.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "COMPONENT",
		Value: "backup-manager",
	})
	volType := corev1.HostPathDirectory
	job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: "coverage",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/mnt/disks/coverage",
				Type: &volType,
			},
		},
	})
	job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
		Name:      "coverage",
		MountPath: "/coverage",
	})
EOF

line=$(grep -n 'rm.deps.JobControl.CreateJob(restore, job)' $RESTORE_MANAGER | cut -d ":" -f 1)
head -n $(($line-1)) $RESTORE_MANAGER > $TMP_RESTORE_MANAGER
echo "$PATCH_BR_JOB" >> $TMP_RESTORE_MANAGER
tail -n +$line $RESTORE_MANAGER >> $TMP_RESTORE_MANAGER
mv -f $TMP_RESTORE_MANAGER $RESTORE_MANAGER

line=$(grep -n 'bm.deps.JobControl.CreateJob(backup, job)' $BACKUP_MANAGER | cut -d ":" -f 1)
head -n $(($line-1)) $BACKUP_MANAGER > $TMP_BACKUP_MANAGER
echo "$PATCH_BR_JOB"`` >> $TMP_BACKUP_MANAGER
tail -n +$line $BACKUP_MANAGER >> $TMP_BACKUP_MANAGER
mv -f $TMP_BACKUP_MANAGER $BACKUP_MANAGER

line=$(grep -n 'bc.deps.JobControl.CreateJob(backup, job)' $BACKUP_CLEANER | cut -d ":" -f 1)
head -n $(($line-1)) $BACKUP_CLEANER > $TMP_BACKUP_CLEANER
echo "$PATCH_BR_JOB"`` >> $TMP_BACKUP_CLEANER
tail -n +$line $BACKUP_CLEANER >> $TMP_BACKUP_CLEANER
mv -f $TMP_BACKUP_CLEANER $BACKUP_CLEANER
