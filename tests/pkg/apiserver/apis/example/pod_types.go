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

package example

/*
  The case we are simulating for e2e test is upgrading
    pod.v1alpha1.example.pingcap.com
  to
    pod.v1beta1.example.pingcap.com

  In v1beta1:
    - field .spec.container is removed
    - fields .spec.containers & .spec.tolerations are added
    - the default imagePullPolicy is changed from 'IfNotPreset' to 'Always', but the created objects should not be
      affected
*/

// PodSpec is the internal version of pod spec
type PodSpec struct {
	// New field introduced in v1beta1 to store multiple containers
	Containers []ContainerSpec
	// In real case, we design the v1alpha1 API first, so we must keep this
	// field to be backward compatible with the objects written formerly
	Container    ContainerSpec
	NodeSelector map[string]string
	Tolerations  []string
	HostName     string

	// helper field to distinguish whether the .spec.tolerations is unset or empty
	HasTolerations bool
}

type ContainerSpec struct {
	Image           string
	ImagePullPolicy string
}
