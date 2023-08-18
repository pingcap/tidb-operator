//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Copyright PingCAP, Inc.
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

// Code generated by openapi-gen. DO NOT EDIT.

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1alpha1

import (
	common "k8s.io/kube-openapi/pkg/common"
	spec "k8s.io/kube-openapi/pkg/validation/spec"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.BRConfig":                   schema_apis_federation_pingcap_v1alpha1_BRConfig(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackup":               schema_apis_federation_pingcap_v1alpha1_VolumeBackup(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupList":           schema_apis_federation_pingcap_v1alpha1_VolumeBackupList(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupMemberCluster":  schema_apis_federation_pingcap_v1alpha1_VolumeBackupMemberCluster(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupMemberSpec":     schema_apis_federation_pingcap_v1alpha1_VolumeBackupMemberSpec(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupSchedule":       schema_apis_federation_pingcap_v1alpha1_VolumeBackupSchedule(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupScheduleList":   schema_apis_federation_pingcap_v1alpha1_VolumeBackupScheduleList(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupScheduleSpec":   schema_apis_federation_pingcap_v1alpha1_VolumeBackupScheduleSpec(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupSpec":           schema_apis_federation_pingcap_v1alpha1_VolumeBackupSpec(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestore":              schema_apis_federation_pingcap_v1alpha1_VolumeRestore(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestoreList":          schema_apis_federation_pingcap_v1alpha1_VolumeRestoreList(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestoreMemberCluster": schema_apis_federation_pingcap_v1alpha1_VolumeRestoreMemberCluster(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestoreMemberSpec":    schema_apis_federation_pingcap_v1alpha1_VolumeRestoreMemberSpec(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestoreSpec":          schema_apis_federation_pingcap_v1alpha1_VolumeRestoreSpec(ref),
	}
}

func schema_apis_federation_pingcap_v1alpha1_BRConfig(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "BRConfig contains config for BR",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"concurrency": {
						SchemaProps: spec.SchemaProps{
							Description: "Concurrency is the size of thread pool on each node that execute the backup task",
							Type:        []string{"integer"},
							Format:      "int64",
						},
					},
					"checkRequirements": {
						SchemaProps: spec.SchemaProps{
							Description: "CheckRequirements specifies whether to check requirements",
							Type:        []string{"boolean"},
							Format:      "",
						},
					},
					"sendCredToTikv": {
						SchemaProps: spec.SchemaProps{
							Description: "SendCredToTikv specifies whether to send credentials to TiKV",
							Type:        []string{"boolean"},
							Format:      "",
						},
					},
					"options": {
						SchemaProps: spec.SchemaProps{
							Description: "Options means options for backup data to remote storage with BR. These options has highest priority.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Default: "",
										Type:    []string{"string"},
										Format:  "",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func schema_apis_federation_pingcap_v1alpha1_VolumeBackup(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "VolumeBackup is the control script's spec",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Default: map[string]interface{}{},
							Ref:     ref("github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupSpec"),
						},
					},
				},
				Required: []string{"spec"},
			},
		},
		Dependencies: []string{
			"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupSpec"},
	}
}

func schema_apis_federation_pingcap_v1alpha1_VolumeBackupList(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "VolumeBackupList is VolumeBackup list",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Default: map[string]interface{}{},
							Ref:     ref("k8s.io/apimachinery/pkg/apis/meta/v1.ListMeta"),
						},
					},
					"items": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Default: map[string]interface{}{},
										Ref:     ref("github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackup"),
									},
								},
							},
						},
					},
				},
				Required: []string{"metadata", "items"},
			},
		},
		Dependencies: []string{
			"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackup", "k8s.io/apimachinery/pkg/apis/meta/v1.ListMeta"},
	}
}

func schema_apis_federation_pingcap_v1alpha1_VolumeBackupMemberCluster(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "VolumeBackupMemberCluster contains the TiDB cluster which need to execute volume backup",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"k8sClusterName": {
						SchemaProps: spec.SchemaProps{
							Description: "K8sClusterName is the name of the k8s cluster where the tc locates",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"tcName": {
						SchemaProps: spec.SchemaProps{
							Description: "TCName is the name of the TiDBCluster CR which need to execute volume backup",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"tcNamespace": {
						SchemaProps: spec.SchemaProps{
							Description: "TCNamespace is the namespace of the TiDBCluster CR",
							Type:        []string{"string"},
							Format:      "",
						},
					},
				},
			},
		},
	}
}

func schema_apis_federation_pingcap_v1alpha1_VolumeBackupMemberSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "VolumeBackupMemberSpec contains the backup specification for one tidb cluster",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"resources": {
						SchemaProps: spec.SchemaProps{
							Default: map[string]interface{}{},
							Ref:     ref("k8s.io/api/core/v1.ResourceRequirements"),
						},
					},
					"env": {
						SchemaProps: spec.SchemaProps{
							Description: "List of environment variables to set in the container, like v1.Container.Env. Note that the following builtin env vars will be overwritten by values set here - S3_PROVIDER - S3_ENDPOINT - AWS_REGION - AWS_ACL - AWS_STORAGE_CLASS - AWS_DEFAULT_REGION - AWS_ACCESS_KEY_ID - AWS_SECRET_ACCESS_KEY - GCS_PROJECT_ID - GCS_OBJECT_ACL - GCS_BUCKET_ACL - GCS_LOCATION - GCS_STORAGE_CLASS - GCS_SERVICE_ACCOUNT_JSON_KEY - BR_LOG_TO_TERM",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Default: map[string]interface{}{},
										Ref:     ref("k8s.io/api/core/v1.EnvVar"),
									},
								},
							},
						},
					},
					"br": {
						SchemaProps: spec.SchemaProps{
							Description: "BRConfig is the configs for BR",
							Ref:         ref("github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.BRConfig"),
						},
					},
					"s3": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.S3StorageProvider"),
						},
					},
					"gcs": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.GcsStorageProvider"),
						},
					},
					"azblob": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.AzblobStorageProvider"),
						},
					},
					"local": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.LocalStorageProvider"),
						},
					},
					"tolerations": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Default: map[string]interface{}{},
										Ref:     ref("k8s.io/api/core/v1.Toleration"),
									},
								},
							},
						},
					},
					"toolImage": {
						SchemaProps: spec.SchemaProps{
							Description: "ToolImage specifies the tool image used in `Backup`, which supports BR. For examples `spec.toolImage: pingcap/br:v6.5.0` For BR image, if it does not contain tag, Pod will use image 'ToolImage:${TiKV_Version}'.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"imagePullSecrets": {
						SchemaProps: spec.SchemaProps{
							Description: "ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Default: map[string]interface{}{},
										Ref:     ref("k8s.io/api/core/v1.LocalObjectReference"),
									},
								},
							},
						},
					},
					"serviceAccount": {
						SchemaProps: spec.SchemaProps{
							Description: "Specify service account of backup",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"cleanPolicy": {
						SchemaProps: spec.SchemaProps{
							Description: "CleanPolicy denotes whether to clean backup data when the object is deleted from the cluster, if not set, the backup data will be retained",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"priorityClassName": {
						SchemaProps: spec.SchemaProps{
							Description: "PriorityClassName of Backup Job Pods",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"calcSizeLevel": {
						SchemaProps: spec.SchemaProps{
							Description: "CalcSizeLevel determines how to size calculation of snapshots for EBS volume snapshot backup",
							Type:        []string{"string"},
							Format:      "",
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.BRConfig", "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.AzblobStorageProvider", "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.GcsStorageProvider", "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.LocalStorageProvider", "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1.S3StorageProvider", "k8s.io/api/core/v1.EnvVar", "k8s.io/api/core/v1.LocalObjectReference", "k8s.io/api/core/v1.ResourceRequirements", "k8s.io/api/core/v1.Toleration"},
	}
}

func schema_apis_federation_pingcap_v1alpha1_VolumeBackupSchedule(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "VolumeBackupSchedule is the control script's spec",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Default: map[string]interface{}{},
							Ref:     ref("github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupScheduleSpec"),
						},
					},
				},
				Required: []string{"spec"},
			},
		},
		Dependencies: []string{
			"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupScheduleSpec"},
	}
}

func schema_apis_federation_pingcap_v1alpha1_VolumeBackupScheduleList(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "VolumeBackupScheduleList is VolumeBackupSchedule list",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Default: map[string]interface{}{},
							Ref:     ref("k8s.io/apimachinery/pkg/apis/meta/v1.ListMeta"),
						},
					},
					"items": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Default: map[string]interface{}{},
										Ref:     ref("github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupSchedule"),
									},
								},
							},
						},
					},
				},
				Required: []string{"metadata", "items"},
			},
		},
		Dependencies: []string{
			"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupSchedule", "k8s.io/apimachinery/pkg/apis/meta/v1.ListMeta"},
	}
}

func schema_apis_federation_pingcap_v1alpha1_VolumeBackupScheduleSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "VolumeBackupScheduleSpec describes the attributes that a user creates on a volume backup schedule.",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"schedule": {
						SchemaProps: spec.SchemaProps{
							Description: "Schedule specifies the cron string used for backup scheduling.",
							Default:     "",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"pause": {
						SchemaProps: spec.SchemaProps{
							Description: "Pause means paused backupSchedule",
							Type:        []string{"boolean"},
							Format:      "",
						},
					},
					"maxBackups": {
						SchemaProps: spec.SchemaProps{
							Description: "MaxBackups is to specify how many backups we want to keep 0 is magic number to indicate un-limited backups. if MaxBackups and MaxReservedTime are set at the same time, MaxReservedTime is preferred and MaxBackups is ignored.",
							Type:        []string{"integer"},
							Format:      "int32",
						},
					},
					"maxReservedTime": {
						SchemaProps: spec.SchemaProps{
							Description: "MaxReservedTime is to specify how long backups we want to keep.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"backupTemplate": {
						SchemaProps: spec.SchemaProps{
							Description: "BackupTemplate is the specification of the volume backup structure to get scheduled.",
							Default:     map[string]interface{}{},
							Ref:         ref("github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupSpec"),
						},
					},
				},
				Required: []string{"schedule", "backupTemplate"},
			},
		},
		Dependencies: []string{
			"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupSpec"},
	}
}

func schema_apis_federation_pingcap_v1alpha1_VolumeBackupSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "VolumeBackupSpec describes the attributes that a user creates on a volume backup.",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"clusters": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Default: map[string]interface{}{},
										Ref:     ref("github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupMemberCluster"),
									},
								},
							},
						},
					},
					"template": {
						SchemaProps: spec.SchemaProps{
							Default: map[string]interface{}{},
							Ref:     ref("github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupMemberSpec"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupMemberCluster", "github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupMemberSpec"},
	}
}

func schema_apis_federation_pingcap_v1alpha1_VolumeRestore(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "VolumeRestore is the control script's spec",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Default: map[string]interface{}{},
							Ref:     ref("github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestoreSpec"),
						},
					},
				},
				Required: []string{"spec"},
			},
		},
		Dependencies: []string{
			"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestoreSpec"},
	}
}

func schema_apis_federation_pingcap_v1alpha1_VolumeRestoreList(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "VolumeRestoreList is VolumeRestore list",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Default: map[string]interface{}{},
							Ref:     ref("k8s.io/apimachinery/pkg/apis/meta/v1.ListMeta"),
						},
					},
					"items": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Default: map[string]interface{}{},
										Ref:     ref("github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestore"),
									},
								},
							},
						},
					},
				},
				Required: []string{"metadata", "items"},
			},
		},
		Dependencies: []string{
			"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestore", "k8s.io/apimachinery/pkg/apis/meta/v1.ListMeta"},
	}
}

func schema_apis_federation_pingcap_v1alpha1_VolumeRestoreMemberCluster(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "VolumeRestoreMemberCluster contains the TiDB cluster which need to execute volume restore",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"k8sClusterName": {
						SchemaProps: spec.SchemaProps{
							Description: "K8sClusterName is the name of the k8s cluster where the tc locates",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"tcName": {
						SchemaProps: spec.SchemaProps{
							Description: "TCName is the name of the TiDBCluster CR which need to execute volume backup",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"tcNamespace": {
						SchemaProps: spec.SchemaProps{
							Description: "TCNamespace is the namespace of the TiDBCluster CR",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"azName": {
						SchemaProps: spec.SchemaProps{
							Description: "AZName is the available zone which the volume snapshots restore to",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"backup": {
						SchemaProps: spec.SchemaProps{
							Description: "Backup is the volume backup information",
							Default:     map[string]interface{}{},
							Ref:         ref("github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestoreMemberBackupInfo"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestoreMemberBackupInfo"},
	}
}

func schema_apis_federation_pingcap_v1alpha1_VolumeRestoreMemberSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "VolumeRestoreMemberSpec contains the restore specification for one tidb cluster",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"resources": {
						SchemaProps: spec.SchemaProps{
							Default: map[string]interface{}{},
							Ref:     ref("k8s.io/api/core/v1.ResourceRequirements"),
						},
					},
					"env": {
						SchemaProps: spec.SchemaProps{
							Description: "List of environment variables to set in the container, like v1.Container.Env. Note that the following builtin env vars will be overwritten by values set here - S3_PROVIDER - S3_ENDPOINT - AWS_REGION - AWS_ACL - AWS_STORAGE_CLASS - AWS_DEFAULT_REGION - AWS_ACCESS_KEY_ID - AWS_SECRET_ACCESS_KEY - GCS_PROJECT_ID - GCS_OBJECT_ACL - GCS_BUCKET_ACL - GCS_LOCATION - GCS_STORAGE_CLASS - GCS_SERVICE_ACCOUNT_JSON_KEY - BR_LOG_TO_TERM",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Default: map[string]interface{}{},
										Ref:     ref("k8s.io/api/core/v1.EnvVar"),
									},
								},
							},
						},
					},
					"br": {
						SchemaProps: spec.SchemaProps{
							Description: "BRConfig is the configs for BR",
							Ref:         ref("github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.BRConfig"),
						},
					},
					"tolerations": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Default: map[string]interface{}{},
										Ref:     ref("k8s.io/api/core/v1.Toleration"),
									},
								},
							},
						},
					},
					"toolImage": {
						SchemaProps: spec.SchemaProps{
							Description: "ToolImage specifies the tool image used in `Restore`, which supports BR image. For examples `spec.toolImage: pingcap/br:v6.5.0` For BR image, if it does not contain tag, Pod will use image 'ToolImage:${TiKV_Version}'.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"imagePullSecrets": {
						SchemaProps: spec.SchemaProps{
							Description: "ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Default: map[string]interface{}{},
										Ref:     ref("k8s.io/api/core/v1.LocalObjectReference"),
									},
								},
							},
						},
					},
					"serviceAccount": {
						SchemaProps: spec.SchemaProps{
							Description: "Specify service account of restore",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"priorityClassName": {
						SchemaProps: spec.SchemaProps{
							Description: "PriorityClassName of Restore Job Pods",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"warmup": {
						SchemaProps: spec.SchemaProps{
							Description: "Warmup represents whether to initialize TiKV volumes after volume snapshot restore",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"warmupImage": {
						SchemaProps: spec.SchemaProps{
							Description: "WarmupImage represents using what image to initialize TiKV volumes",
							Type:        []string{"string"},
							Format:      "",
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.BRConfig", "k8s.io/api/core/v1.EnvVar", "k8s.io/api/core/v1.LocalObjectReference", "k8s.io/api/core/v1.ResourceRequirements", "k8s.io/api/core/v1.Toleration"},
	}
}

func schema_apis_federation_pingcap_v1alpha1_VolumeRestoreSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "VolumeRestoreSpec describes the attributes that a user creates on a volume restore.",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"clusters": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Default: map[string]interface{}{},
										Ref:     ref("github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestoreMemberCluster"),
									},
								},
							},
						},
					},
					"template": {
						SchemaProps: spec.SchemaProps{
							Default: map[string]interface{}{},
							Ref:     ref("github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestoreMemberSpec"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestoreMemberCluster", "github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestoreMemberSpec"},
	}
}
