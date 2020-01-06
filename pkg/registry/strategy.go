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

package registry

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// CreateUpdateStrategy is a sub set of the RESTCreateUpdateStrategy interface of kube-apiserver, which abstracts the
// defaulting and validation logic of each custom resources in the kube-apiserver way but allow using webhook as
// an alternative implementation.
// Note that PrepareForCreate/Update method is different with the defaultingFunc of kube-apiserver, the latter one
// is applied to versioned resource, and PrepareForCreate/Update is applied to resource after conversion.
// MutatingAdmissionWebhook is also applied to resource after conversion, which allows it to implement the correct
// semantic of PrepareForCreate/Update as the following figure shows:
//             +
//          Resource
//             |
//    +--------v---------+
//    |    Conversion    |
//    +--------+---------+
//             |     MutatingWebhook   +---------------------+
//             +---------------------->+   Prepare(Webhook)  |
//                                     +----------+----------+
//                                                |
//             +----------------------------------+
//             |
//    +--------v---------+
//    |  Prepare(Server) |
//    +--------+---------+
//             |
//    +--------v---------+
//    | Validate(Server) |
//    +--------+---------+
//             |    ValidatingWebhook  +---------------------+
//             +---------------------->+  Validate(Webhook)  |
//                                     +----------+----------+
//                                                |
//             +----------------------------------+
//             |
//    +--------v---------+
//    |       ETCD       |
//    +------------------+
//
// There is a special case for custom resource, if there is only one version specified, the conversion is actually no-op.
// And if there is multiple versions specified, a storage version could be specified in CustomResourceDefinition which acts
// the conversion target. So, the strategy is always applied to a certain version of CustomResource instead of an "internal"
// version.
//
// TODO(aylei): we may place this definition in a more reasonable package
type CreateUpdateStrategy interface {
	// NewObject create an empty object that the strategt applied to
	NewObject() runtime.Object
	// PrepareForCreate mutate a new resource before persistent it
	PrepareForCreate(ctx context.Context, obj runtime.Object)
	// PrepareForUpdate mutate a new resource before it replace the existing on in storage
	PrepareForUpdate(ctx context.Context, obj, old runtime.Object)
	// Validate validates a new resource
	Validate(ctx context.Context, obj runtime.Object) field.ErrorList
	// ValidateUpdate validates an update request for existing resource
	ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList
}
