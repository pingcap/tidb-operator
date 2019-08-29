// Copyright 2019 PingCAP, Inc.
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
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// CheckAllKeysExistInSecret check if all keys are included in the specific secret
func CheckAllKeysExistInSecret(secret *corev1.Secret, keys ...string) (string, bool) {
	var notExistKeys []string

	for _, key := range keys {
		if _, exist := secret.Data[key]; !exist {
			notExistKeys = append(notExistKeys, key)
		}
	}

	return strings.Join(notExistKeys, ","), len(notExistKeys) == 0
}

// generateCephCertEnvVar generate the env info for ceph
func generateCephCertEnvVar(secret *corev1.Secret, endpoint string) ([]corev1.EnvVar, error) {
	var envVars []corev1.EnvVar

	if !strings.Contains(endpoint, "://") {
		// convert xxx.xxx.xxx.xxx:port to http://xxx.xxx.xxx.xxx:port
		// the endpoint must start with http://
		endpoint = fmt.Sprintf("http://%s", endpoint)
	}

	if !strings.HasPrefix(endpoint, "http://") {
		return envVars, fmt.Errorf("cenph endpoint URI %s must start with http:// scheme", endpoint)
	}
	envVars = []corev1.EnvVar{
		{
			Name:  "S3_ENDPOINT",
			Value: endpoint,
		},
		{
			Name:  "AWS_ACCESS_KEY_ID",
			Value: string(secret.Data[constants.S3AccessKey]),
		},
		{
			Name:  "AWS_SECRET_ACCESS_KEY",
			Value: string(secret.Data[constants.S3SecretKey]),
		},
	}
	return envVars, nil
}

// GenerateStorageCertEnv generate the env info in order to access backend backup storage
func GenerateStorageCertEnv(backup *v1alpha1.Backup, secretLister corelisters.SecretLister) ([]corev1.EnvVar, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()

	var certEnv []corev1.EnvVar

	switch backup.Spec.StorageType {
	case v1alpha1.BackupStorageTypeCeph:
		cephSecretName := backup.Spec.Ceph.SecretName
		secret, err := secretLister.Secrets(ns).Get(cephSecretName)
		if err != nil {
			err := fmt.Errorf("backup %s/%s get ceph secret %s failed, err: %v", ns, name, cephSecretName, err)
			return certEnv, "GetCephSecretFailed", err
		}

		keyStr, exist := CheckAllKeysExistInSecret(secret, constants.S3AccessKey, constants.S3SecretKey)
		if !exist {
			err := fmt.Errorf("backup %s/%s, The secret %s missing some keys %s", ns, name, cephSecretName, keyStr)
			return certEnv, "KeyNotExist", err
		}

		certEnv, err = generateCephCertEnvVar(secret, backup.Spec.Ceph.Endpoint)
		if err != nil {
			return certEnv, "InvalidCephEndpoint", err
		}
	case v1alpha1.BackupStorageTypeLocal:
		// if the backup storage type is local, do nothing
		break
	default:
		err := fmt.Errorf("backup %s/%s don't support storage type %s", ns, name, backup.Spec.StorageType)
		return certEnv, "NotSupportStorageType", err
	}
	return certEnv, "", nil
}

// GetTidbUserAndPassword get the tidb user and password from specific secret
func GetTidbUserAndPassword(backup *v1alpha1.Backup, secretLister corelisters.SecretLister) (user, password, reason string, err error) {
	ns := backup.GetNamespace()
	name := backup.GetName()
	tidbSecretName := backup.Spec.TidbSecretName

	secret, err := secretLister.Secrets(ns).Get(tidbSecretName)
	if err != nil {
		err = fmt.Errorf("backup %s/%s get tidb secret %s failed, err: %v", ns, name, tidbSecretName, err)
		reason = "GetTidbSecretFailed"
		return
	}

	keyStr, exist := CheckAllKeysExistInSecret(secret, constants.TidbUserKey, constants.TidbPasswordKey)
	if !exist {
		err = fmt.Errorf("backup %s/%s, tidb secret %s missing some keys %s", ns, name, tidbSecretName, keyStr)
		reason = "KeyNotExist"
		return
	}

	user = string(secret.Data[constants.TidbUserKey])
	password = string(secret.Data[constants.TidbPasswordKey])
	return
}

func DeleteOccupyJob(job *batchv1.Job, jobControl controller.JobControlInterface) error {
	if job == nil {
		// PVC is not occupied by backup or restore jobs, do nothing
		return nil
	}

	ns := job.GetNamespace()
	jobName := job.GetName()

	controllerRef := metav1.GetControllerOf(job)
	if controllerRef == nil {
		return fmt.Errorf("can't find job %s/%s owner reference", ns, jobName)
	}

	pvcName := job.Annotations[label.AnnBackupPVC]
	if job.Status.Failed == 0 && job.Status.Succeeded == 0 {
		// job is still running, return a requeue error
		return controller.RequeueErrorf("the job %s/%s that occupies PVC %s is still running", ns, jobName, pvcName)
	}

	job.SetGroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind(controllerRef.Kind))
	return jobControl.DeleteJob(job, job)
}

// GetOccupyJobOfPVC return the job that used the specific PVC
func GetOccupyJobOfPVC(jobList batchlisters.JobLister, podLister corelisters.PodLister, ns, pvc string) (*batchv1.Job, string, error) {
	pods, err := podLister.Pods(ns).List(labels.Everything())
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, "", nil
		}
		return nil, "ListPodsFailed", err
	}

	for _, pod := range pods {
		if pod.Spec.NodeName == "" {
			// This pod is not scheduled. Therefore this pod does not block the PVC used by other pod.
			continue
		}
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim == nil {
				continue
			}
			if volume.PersistentVolumeClaim.ClaimName != pvc {
				continue
			}
			job := resolveJobFromPod(jobList, pod)
			if job == nil {
				err := fmt.Errorf("PVC %s/%s is occupied by an unexpected pod %s", ns, pvc, pod.GetName())
				return nil, "UnknownPVCOccupation", err
			}
			// A PVC can only be occupied by a pod, so once we find the occupied job, return quickly.
			return job, "", nil
		}

	}

	return nil, "", nil
}

// resolveJobFromPod returns the Job by a Pod,
// or nil if the Pod could not be resolved to a matching Job
// of the correct Kind.
func resolveJobFromPod(jobList batchlisters.JobLister, pod *corev1.Pod) *batchv1.Job {
	ns := pod.GetNamespace()

	controllerRef := metav1.GetControllerOf(pod)
	if controllerRef == nil {
		return nil
	}

	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != "Job" {
		return nil
	}

	jobName := controllerRef.Name
	if !strings.HasPrefix(jobName, "backup") || !strings.HasPrefix(jobName, "restore") {
		// This pod does not belong to backup or restore job
		return nil
	}

	job, err := jobList.Jobs(ns).Get(jobName)
	if err != nil {
		return nil
	}

	if job.UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return job
}

type ByCreateTime []*v1alpha1.Backup

func (b ByCreateTime) Len() int      { return len(b) }
func (b ByCreateTime) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByCreateTime) Less(i, j int) bool {
	return b[j].ObjectMeta.CreationTimestamp.Before(&b[i].ObjectMeta.CreationTimestamp)
}
