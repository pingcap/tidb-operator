// Copyright 2018 PingCAP, Inc.
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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// SchemeBuilder and AddToScheme will stay in k8s.io/kubernetes.
	SchemeBuilder      runtime.SchemeBuilder
	localSchemeBuilder = &SchemeBuilder
	// AddToScheme applies all the stored functions to the scheme.
	AddToScheme = localSchemeBuilder.AddToScheme
	// Scheme is the scheme instance of operator
	Scheme *runtime.Scheme

	groupName = "pingcap.com"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: groupName, Version: "v1alpha1"}

func init() {
	// We only register manually written functions here. The registration of the
	// generated functions takes place in the generated files. The separation
	// makes the code compile even when the generated files are missing.
	localSchemeBuilder.Register(addKnownTypes)
}

// Resource takes an unqualified resource and returns back a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	Scheme = scheme
	scheme.AddKnownTypes(SchemeGroupVersion,
		&TidbCluster{},
		&TidbClusterList{},
		&Backup{},
		&BackupList{},
		&BackupSchedule{},
		&BackupScheduleList{},
		&Restore{},
		&RestoreList{},
		&DataResource{},
		&DataResourceList{},
		&TidbInitializer{},
		&TidbInitializerList{},
		&TidbMonitor{},
		&TidbMonitorList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
