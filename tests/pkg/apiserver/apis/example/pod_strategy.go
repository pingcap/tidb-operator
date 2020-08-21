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

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (p PodStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	// call default
	p.DefaultStorageStrategy.PrepareForUpdate(ctx, obj, old)
	n := obj.(*Pod)
	o := old.(*Pod)
	// v1alpha1 do not set the tolerations field, retain the old one
	if !n.Spec.HasTolerations {
		n.Spec.Tolerations = o.Spec.Tolerations
	}
}

// Validate checks that an instance of Pod is well formed
func (PodStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*Pod)
	errors := field.ErrorList{}
	specPath := field.NewPath("spec")
	containerPath := specPath.Child("containers")
	if len(o.Spec.Containers) < 1 {
		errors = append(errors, field.Invalid(containerPath, o.Spec.Containers, "At least one container should be specified"))
	}
	for i := range o.Spec.Containers {
		if o.Spec.Containers[i].Image == "" {
			errors = append(errors, field.Invalid(containerPath.Index(i).Child("image"), o.Spec.Containers[i], "Image must not be empty for container"))
		}
	}
	return errors
}

// ValidateUpdate checks if the update is valid
func (p PodStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	errors := p.Validate(ctx, obj)
	n := obj.(*Pod)
	o := old.(*Pod)
	if o.Spec.HostName != "" && o.Spec.HostName != n.Spec.HostName {
		errors = append(errors, field.Invalid(field.NewPath("spec").Child("hostname"), o.Spec.HostName, "hostname is immutable after set"))
	}
	return errors
}
