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
	MatchingLabels         = client.MatchingLabels
	MatchingLabelsSelector = client.MatchingLabelsSelector
	MatchingFields         = client.MatchingFields
	InNamespace            = client.InNamespace

	// Options = client.Options

	// ListOptions from client
	ListOptions = client.ListOptions
	ListOption  = client.ListOption

	// DeleteOptions from client
	DeleteOptions      = client.DeleteOptions
	DeleteOption       = client.DeleteOption
	Preconditions      = client.Preconditions
	GracePeriodSeconds = client.GracePeriodSeconds
	PropagationPolicy  = client.PropagationPolicy

	// MergeFromOption from client
	MergeFromOption = client.MergeFromOption
)

var (
	ObjectKeyFromObject = client.ObjectKeyFromObject
	IgnoreNotFound      = client.IgnoreNotFound
	RawPatch            = client.RawPatch
	WithSubResourceBody = client.WithSubResourceBody
	MergeFrom           = client.MergeFrom
)
