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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"time"

	v1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	scheme "github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// BackupSchedulesGetter has a method to return a BackupScheduleInterface.
// A group's client should implement this interface.
type BackupSchedulesGetter interface {
	BackupSchedules(namespace string) BackupScheduleInterface
}

// BackupScheduleInterface has methods to work with BackupSchedule resources.
type BackupScheduleInterface interface {
	Create(*v1alpha1.BackupSchedule) (*v1alpha1.BackupSchedule, error)
	Update(*v1alpha1.BackupSchedule) (*v1alpha1.BackupSchedule, error)
	UpdateStatus(*v1alpha1.BackupSchedule) (*v1alpha1.BackupSchedule, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.BackupSchedule, error)
	List(opts v1.ListOptions) (*v1alpha1.BackupScheduleList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.BackupSchedule, err error)
	BackupScheduleExpansion
}

// backupSchedules implements BackupScheduleInterface
type backupSchedules struct {
	client rest.Interface
	ns     string
}

// newBackupSchedules returns a BackupSchedules
func newBackupSchedules(c *PingcapV1alpha1Client, namespace string) *backupSchedules {
	return &backupSchedules{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the backupSchedule, and returns the corresponding backupSchedule object, and an error if there is any.
func (c *backupSchedules) Get(name string, options v1.GetOptions) (result *v1alpha1.BackupSchedule, err error) {
	result = &v1alpha1.BackupSchedule{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("backupschedules").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of BackupSchedules that match those selectors.
func (c *backupSchedules) List(opts v1.ListOptions) (result *v1alpha1.BackupScheduleList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.BackupScheduleList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("backupschedules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested backupSchedules.
func (c *backupSchedules) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("backupschedules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a backupSchedule and creates it.  Returns the server's representation of the backupSchedule, and an error, if there is any.
func (c *backupSchedules) Create(backupSchedule *v1alpha1.BackupSchedule) (result *v1alpha1.BackupSchedule, err error) {
	result = &v1alpha1.BackupSchedule{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("backupschedules").
		Body(backupSchedule).
		Do().
		Into(result)
	return
}

// Update takes the representation of a backupSchedule and updates it. Returns the server's representation of the backupSchedule, and an error, if there is any.
func (c *backupSchedules) Update(backupSchedule *v1alpha1.BackupSchedule) (result *v1alpha1.BackupSchedule, err error) {
	result = &v1alpha1.BackupSchedule{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("backupschedules").
		Name(backupSchedule.Name).
		Body(backupSchedule).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *backupSchedules) UpdateStatus(backupSchedule *v1alpha1.BackupSchedule) (result *v1alpha1.BackupSchedule, err error) {
	result = &v1alpha1.BackupSchedule{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("backupschedules").
		Name(backupSchedule.Name).
		SubResource("status").
		Body(backupSchedule).
		Do().
		Into(result)
	return
}

// Delete takes name of the backupSchedule and deletes it. Returns an error if one occurs.
func (c *backupSchedules) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("backupschedules").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *backupSchedules) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("backupschedules").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched backupSchedule.
func (c *backupSchedules) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.BackupSchedule, err error) {
	result = &v1alpha1.BackupSchedule{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("backupschedules").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
