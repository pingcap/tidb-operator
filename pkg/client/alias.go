// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import "sigs.k8s.io/controller-runtime/pkg/client"

// Add alias of client.XXX to avoid import
// two client pkgs
type (
	Object     = client.Object
	ObjectList = client.ObjectList
	ObjectKey  = client.ObjectKey
)

type (
	Options      = client.Options
	DeleteOption = client.DeleteOption
	ListOption   = client.ListOption
)

type (
	MatchingLabels = client.MatchingLabels
	MatchingFields = client.MatchingFields
	InNamespace    = client.InNamespace
	ListOptions    = client.ListOptions
	DeleteOptions  = client.DeleteOptions
)

type PropagationPolicy = client.PropagationPolicy

var ObjectKeyFromObject = client.ObjectKeyFromObject

var IgnoreNotFound = client.IgnoreNotFound

type GracePeriodSeconds = client.GracePeriodSeconds

type MergeFromOption = client.MergeFromOption

var RawPatch = client.RawPatch
