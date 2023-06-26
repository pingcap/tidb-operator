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
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackup":             schema_apis_federation_pingcap_v1alpha1_VolumeBackup(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupList":         schema_apis_federation_pingcap_v1alpha1_VolumeBackupList(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupSchedule":     schema_apis_federation_pingcap_v1alpha1_VolumeBackupSchedule(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupScheduleList": schema_apis_federation_pingcap_v1alpha1_VolumeBackupScheduleList(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupScheduleSpec": schema_apis_federation_pingcap_v1alpha1_VolumeBackupScheduleSpec(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeBackupSpec":         schema_apis_federation_pingcap_v1alpha1_VolumeBackupSpec(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestore":            schema_apis_federation_pingcap_v1alpha1_VolumeRestore(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestoreList":        schema_apis_federation_pingcap_v1alpha1_VolumeRestoreList(ref),
		"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1.VolumeRestoreSpec":        schema_apis_federation_pingcap_v1alpha1_VolumeRestoreSpec(ref),
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
			},
		},
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

func schema_apis_federation_pingcap_v1alpha1_VolumeRestoreSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "VolumeRestoreSpec describes the attributes that a user creates on a volume restore.",
				Type:        []string{"object"},
			},
		},
	}
}
