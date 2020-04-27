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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/scheme"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// GenericControlInterface is a wrapper to manage typed object that managed by an arbitrary controller
type TypedControlInterface interface {
	// CreateOrUpdateSecret create the desired secret or update the current one to desired state if already existed
	CreateOrUpdateSecret(controller runtime.Object, secret *corev1.Secret) (*corev1.Secret, error)
	// CreateOrUpdateConfigMap create the desired configmap or update the current one to desired state if already existed
	CreateOrUpdateConfigMap(controller runtime.Object, cm *corev1.ConfigMap) (*corev1.ConfigMap, error)
	// CreateOrUpdateClusterRole the desired clusterRole or update the current one to desired state if already existed
	CreateOrUpdateClusterRole(controller runtime.Object, clusterRole *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error)
	// CreateOrUpdateClusterRoleBinding create the desired clusterRoleBinding or update the current one to desired state if already existed
	CreateOrUpdateClusterRoleBinding(controller runtime.Object, crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error)
	// CreateOrUpdateRole create the desired role or update the current one to desired state if already existed
	CreateOrUpdateRole(controller runtime.Object, role *rbacv1.Role) (*rbacv1.Role, error)
	// CreateOrUpdateRoleBinding create the desired rolebinding or update the current one to desired state if already existed
	CreateOrUpdateRoleBinding(controller runtime.Object, cr *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error)
	// CreateOrUpdateServiceAccount create the desired serviceaccount or update the current one to desired state if already existed
	CreateOrUpdateServiceAccount(controller runtime.Object, sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error)
	// CreateOrUpdateService create the desired service or update the current one to desired state if already existed
	CreateOrUpdateService(controller runtime.Object, svc *corev1.Service) (*corev1.Service, error)
	// CreateOrUpdateDeployment create the desired deployment or update the current one to desired state if already existed
	CreateOrUpdateDeployment(controller runtime.Object, deploy *appsv1.Deployment) (*appsv1.Deployment, error)
	// CreateOrUpdatePVC create the desired pvc or update the current one to desired state if already existed
	CreateOrUpdatePVC(controller runtime.Object, pvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, error)
	// CreateOrUpdateIngress create the desired ingress or update the current one to desired state if already existed
	CreateOrUpdateIngress(controller runtime.Object, ingress *extensionsv1beta1.Ingress) (*extensionsv1beta1.Ingress, error)
	// UpdateStatus update the /status subresource of the object
	UpdateStatus(newStatus runtime.Object) error
	// Delete delete the given object from the cluster
	Delete(controller, obj runtime.Object) error
	// Create create the given object for the controller
	Create(controller, obj runtime.Object) error
	// Exist check whether object exists
	Exist(key client.ObjectKey, obj runtime.Object) (bool, error)
}
type typedWrapper struct {
	GenericControlInterface
}

// NewTypedControl wraps a GenericControlInterface to a TypedControlInterface
func NewTypedControl(control GenericControlInterface) TypedControlInterface {
	return &typedWrapper{control}
}

func (w *typedWrapper) CreateOrUpdatePVC(controller runtime.Object, pvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, error) {
	result, err := w.GenericControlInterface.CreateOrUpdate(controller, pvc, func(existing, desired runtime.Object) error {
		existingPVC := existing.(*corev1.PersistentVolumeClaim)
		desiredPVC := desired.(*corev1.PersistentVolumeClaim)

		existingPVC.Spec.Resources.Requests = desiredPVC.Spec.Resources.Requests
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*corev1.PersistentVolumeClaim), err
}

func (w *typedWrapper) CreateOrUpdateClusterRoleBinding(controller runtime.Object, crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	result, err := w.GenericControlInterface.CreateOrUpdate(controller, crb, func(existing, desired runtime.Object) error {
		existingCRB := existing.(*rbacv1.ClusterRoleBinding)
		desiredCRB := desired.(*rbacv1.ClusterRoleBinding)

		existingCRB.Labels = desiredCRB.Labels
		existingCRB.RoleRef = desiredCRB.RoleRef
		existingCRB.Subjects = desiredCRB.Subjects
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*rbacv1.ClusterRoleBinding), err
}

func (w *typedWrapper) CreateOrUpdateClusterRole(controller runtime.Object, clusterRole *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	result, err := w.GenericControlInterface.CreateOrUpdate(controller, clusterRole, func(existing, desired runtime.Object) error {
		existingCRole := existing.(*rbacv1.ClusterRole)
		desiredCRole := desired.(*rbacv1.ClusterRole)

		existingCRole.Labels = desiredCRole.Labels
		existingCRole.Rules = desiredCRole.Rules
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*rbacv1.ClusterRole), err
}

func (w *typedWrapper) CreateOrUpdateSecret(controller runtime.Object, secret *corev1.Secret) (*corev1.Secret, error) {
	result, err := w.GenericControlInterface.CreateOrUpdate(controller, secret, func(existing, desired runtime.Object) error {
		existingSecret := existing.(*corev1.Secret)
		desiredSecret := desired.(*corev1.Secret)

		existingSecret.Data = desiredSecret.Data
		existingSecret.Labels = desiredSecret.Labels
		for k, v := range desiredSecret.Annotations {
			existingSecret.Annotations[k] = v
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*corev1.Secret), nil
}

func (w *typedWrapper) Delete(controller, obj runtime.Object) error {
	return w.GenericControlInterface.Delete(controller, obj)
}

func (w *typedWrapper) CreateOrUpdateDeployment(controller runtime.Object, deploy *appsv1.Deployment) (*appsv1.Deployment, error) {
	result, err := w.GenericControlInterface.CreateOrUpdate(controller, deploy, func(existing, desired runtime.Object) error {
		existingDep := existing.(*appsv1.Deployment)
		desiredDep := desired.(*appsv1.Deployment)

		existingDep.Spec.Replicas = desiredDep.Spec.Replicas
		existingDep.Labels = desiredDep.Labels

		if existingDep.Annotations == nil {
			existingDep.Annotations = map[string]string{}
		}
		for k, v := range desiredDep.Annotations {
			existingDep.Annotations[k] = v
		}
		// only override the default strategy if it is explicitly set in the desiredDep
		if string(desiredDep.Spec.Strategy.Type) != "" {
			existingDep.Spec.Strategy.Type = desiredDep.Spec.Strategy.Type
			if existingDep.Spec.Strategy.RollingUpdate != nil {
				existingDep.Spec.Strategy.RollingUpdate = desiredDep.Spec.Strategy.RollingUpdate
			}
		}
		// pod selector of deployment is immutable, so we don't mutate the labels of pod
		for k, v := range desiredDep.Spec.Template.Annotations {
			existingDep.Spec.Template.Annotations[k] = v
		}
		// podSpec of deployment is hard to merge, use an annotation to assist
		if DeploymentPodSpecChanged(desiredDep, existingDep) {
			// Record last applied spec in favor of future equality check
			b, err := json.Marshal(desiredDep.Spec.Template.Spec)
			if err != nil {
				return err
			}
			existingDep.Annotations[LastAppliedConfigAnnotation] = string(b)
			existingDep.Spec.Template.Spec = desiredDep.Spec.Template.Spec
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*appsv1.Deployment), err
}

func (w *typedWrapper) CreateOrUpdateRole(controller runtime.Object, role *rbacv1.Role) (*rbacv1.Role, error) {
	result, err := w.GenericControlInterface.CreateOrUpdate(controller, role, func(existing, desired runtime.Object) error {
		existingRole := existing.(*rbacv1.Role)
		desiredCRole := desired.(*rbacv1.Role)

		existingRole.Labels = desiredCRole.Labels
		existingRole.Rules = desiredCRole.Rules
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*rbacv1.Role), err
}

func (w *typedWrapper) CreateOrUpdateRoleBinding(controller runtime.Object, rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	result, err := w.GenericControlInterface.CreateOrUpdate(controller, rb, func(existing, desired runtime.Object) error {
		existingRB := existing.(*rbacv1.RoleBinding)
		desiredRB := desired.(*rbacv1.RoleBinding)

		existingRB.Labels = desiredRB.Labels
		existingRB.RoleRef = desiredRB.RoleRef
		existingRB.Subjects = desiredRB.Subjects
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*rbacv1.RoleBinding), err
}

func (w *typedWrapper) CreateOrUpdateServiceAccount(controller runtime.Object, sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
	result, err := w.GenericControlInterface.CreateOrUpdate(controller, sa, func(existing, desired runtime.Object) error {
		existingSA := existing.(*corev1.ServiceAccount)
		desiredSA := desired.(*corev1.ServiceAccount)

		existingSA.Labels = desiredSA.Labels
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*corev1.ServiceAccount), err
}

func (w *typedWrapper) CreateOrUpdateConfigMap(controller runtime.Object, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	result, err := w.GenericControlInterface.CreateOrUpdate(controller, cm, func(existing, desired runtime.Object) error {
		existingCm := existing.(*corev1.ConfigMap)
		desiredCm := desired.(*corev1.ConfigMap)

		existingCm.Data = desiredCm.Data
		existingCm.Labels = desiredCm.Labels
		for k, v := range desiredCm.Annotations {
			existingCm.Annotations[k] = v
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*corev1.ConfigMap), nil
}

func (w *typedWrapper) CreateOrUpdateService(controller runtime.Object, svc *corev1.Service) (*corev1.Service, error) {
	result, err := w.GenericControlInterface.CreateOrUpdate(controller, svc, func(existing, desired runtime.Object) error {
		existingSvc := existing.(*corev1.Service)
		desiredSvc := desired.(*corev1.Service)

		if existingSvc.Annotations == nil {
			existingSvc.Annotations = map[string]string{}
		}
		for k, v := range desiredSvc.Annotations {
			existingSvc.Annotations[k] = v
		}
		existingSvc.Labels = desiredSvc.Labels
		equal, err := ServiceEqual(desiredSvc, existingSvc)
		if err != nil {
			return err
		}
		if !equal {
			// record desiredSvc Spec in annotations in favor of future equality checks
			b, err := json.Marshal(desiredSvc.Spec)
			if err != nil {
				return err
			}
			existingSvc.Annotations[LastAppliedConfigAnnotation] = string(b)
			clusterIp := existingSvc.Spec.ClusterIP
			ports := existingSvc.Spec.Ports
			serviceType := existingSvc.Spec.Type

			existingSvc.Spec = desiredSvc.Spec
			existingSvc.Spec.ClusterIP = clusterIp

			// If the existed service and the desired service is NodePort or LoadBalancerType, we should keep the nodePort unchanged.
			if (serviceType == corev1.ServiceTypeNodePort || serviceType == corev1.ServiceTypeLoadBalancer) &&
				(desiredSvc.Spec.Type == corev1.ServiceTypeNodePort || desiredSvc.Spec.Type == corev1.ServiceTypeLoadBalancer) {
				for i, dport := range existingSvc.Spec.Ports {
					for _, eport := range ports {
						// Because the portName could be edited,
						// we use Port number to link the desired Service Port and the existed Service Port in the nested loop
						if dport.Port == eport.Port && dport.Protocol == eport.Protocol {
							dport.NodePort = eport.NodePort
							existingSvc.Spec.Ports[i] = dport
							break
						}
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*corev1.Service), nil
}

func (w *typedWrapper) CreateOrUpdateIngress(controller runtime.Object, ingress *extensionsv1beta1.Ingress) (*extensionsv1beta1.Ingress, error) {
	result, err := w.GenericControlInterface.CreateOrUpdate(controller, ingress, func(existing, desired runtime.Object) error {
		existingIngress := existing.(*extensionsv1beta1.Ingress)
		desiredIngress := desired.(*extensionsv1beta1.Ingress)

		if existingIngress.Annotations == nil {
			existingIngress.Annotations = map[string]string{}
		}
		for k, v := range desiredIngress.Annotations {
			desiredIngress.Annotations[k] = v
		}
		existingIngress.Labels = desiredIngress.Labels
		equal, err := IngressEqual(desiredIngress, existingIngress)
		if err != nil {
			return err
		}
		if !equal {
			// record desiredSvc Spec in annotations in favor of future equality checks
			b, err := json.Marshal(desiredIngress.Spec)
			if err != nil {
				return err
			}
			existingIngress.Annotations[LastAppliedConfigAnnotation] = string(b)
			existingIngress.Spec = desiredIngress.Spec
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*extensionsv1beta1.Ingress), nil
}

func (w *typedWrapper) Create(controller, obj runtime.Object) error {
	return w.GenericControlInterface.Create(controller, obj)
}
func (w *typedWrapper) Exist(key client.ObjectKey, obj runtime.Object) (bool, error) {
	return w.GenericControlInterface.Exist(key, obj)
}

// GenericControlInterface manages generic object that managed by an arbitrary controller
type GenericControlInterface interface {
	CreateOrUpdate(controller, obj runtime.Object, mergeFn MergeFn) (runtime.Object, error)
	Create(controller, obj runtime.Object) error
	UpdateStatus(obj runtime.Object) error
	Exist(key client.ObjectKey, obj runtime.Object) (bool, error)
	Delete(controller, obj runtime.Object) error
}

// MergeFn knows how to merge a desired object into the current object.
// Typically, merge should only set the specific fields the caller wants to control to the existing object
// instead of override a whole struct. e.g.
//
// Prefer:
//     existing.spec.type = desired.spec.type
//     existing.spec.externalTrafficPolicy = desired.spec.externalTrafficPolicy
// Instead of:
//     existing.spec = desired.spec
//
// However, this could be tedious for large object if the caller want to control lots of the fields,
// if there is no one else will mutate this object or cooperation is not needed, it is okay to do aggressive
// override. Note that aggressive override usually causes unnecessary updates because the object will be mutated
// after POST/PUT to api-server (e.g. Defaulting), an annotation based technique could be used to avoid such
// updating: set a last-applied-config annotation and diff the annotation instead of the real spec.
type MergeFn func(existing, desired runtime.Object) error

type realGenericControlInterface struct {
	client   client.Client
	recorder record.EventRecorder
}

func NewRealGenericControl(client client.Client, recorder record.EventRecorder) GenericControlInterface {
	return &realGenericControlInterface{client, recorder}
}

// UpdateStatus update the /status subresource of object
func (c *realGenericControlInterface) UpdateStatus(obj runtime.Object) error {
	return c.client.Status().Update(context.TODO(), obj)
}

// Exist checks whether object exists
func (c *realGenericControlInterface) Exist(key client.ObjectKey, obj runtime.Object) (bool, error) {
	err := c.client.Get(context.TODO(), key, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return true, err
	}
	return true, nil
}

// CreateOrUpdate create an object to the Kubernetes cluster for controller, if the object to create is existed,
// call mergeFn to merge the change in new object to the existing object, then update the existing object.
// The object will also be adopted by the given controller.
func (c *realGenericControlInterface) CreateOrUpdate(controller, obj runtime.Object, mergeFn MergeFn) (runtime.Object, error) {

	// controller-runtime/client will mutate the object pointer in-place,
	// to be consistent with other methods in our controller, we copy the object
	// to avoid the in-place mutation here and hereafter.
	desired := obj.DeepCopyObject()
	if err := setControllerReference(controller, desired); err != nil {
		return desired, err
	}

	// 1. try to create and see if there is any conflicts
	err := c.client.Create(context.TODO(), desired)
	if errors.IsAlreadyExists(err) {

		// 2. object has already existed, merge our desired changes to it
		existing, err := EmptyClone(obj)
		if err != nil {
			return nil, err
		}
		key, err := client.ObjectKeyFromObject(existing)
		if err != nil {
			return nil, err
		}
		err = c.client.Get(context.TODO(), key, existing)
		if err != nil {
			return nil, err
		}

		// 3. try to adopt the existing object
		if err := setControllerReference(controller, existing); err != nil {
			return nil, err
		}

		mutated := existing.DeepCopyObject()
		// 4. invoke mergeFn to mutate a copy of the existing object
		if err := mergeFn(mutated, desired); err != nil {
			return nil, err
		}

		// 5. check if the copy is actually mutated
		if !apiequality.Semantic.DeepEqual(existing, mutated) {
			err := c.client.Update(context.TODO(), mutated)
			return mutated, err
		}

		return mutated, nil
	}

	// object do not exist, return the creation result
	if err == nil {
		c.RecordControllerEvent("create", controller, desired, err)
	}
	return desired, err
}

// Create create an object to the Kubernetes cluster for controller
func (c *realGenericControlInterface) Create(controller, obj runtime.Object) error {
	// controller-runtime/client will mutate the object pointer in-place,
	// to be consistent with other methods in our controller, we copy the object
	// to avoid the in-place mutation here and hereafter.
	desired := obj.DeepCopyObject()
	if err := setControllerReference(controller, desired); err != nil {
		return err
	}

	err := c.client.Create(context.TODO(), desired)
	c.RecordControllerEvent("create", controller, desired, err)
	return err
}

func (c *realGenericControlInterface) Delete(controller, obj runtime.Object) error {
	err := c.client.Delete(context.TODO(), obj)
	c.RecordControllerEvent("delete", controller, obj, err)
	return err
}

// RecordControllerEvent is a generic method to record event for controller
func (c *realGenericControlInterface) RecordControllerEvent(verb string, controller runtime.Object, obj runtime.Object, err error) {
	var controllerName string
	controllerGVK, err := InferObjectKind(controller)
	if err != nil {
		klog.Warningf("Cannot get GVK for controller %v: %v", controller, err)
	}
	if accessor, ok := controller.(metav1.ObjectMetaAccessor); ok {
		controllerName = accessor.GetObjectMeta().GetName()
	}
	objGVK, err := InferObjectKind(obj)
	if err != nil {
		klog.Warningf("Cannot get GVK for obj %v: %v", obj, err)
	}
	var name string
	if accessor, ok := obj.(metav1.ObjectMetaAccessor); ok {
		name = accessor.GetObjectMeta().GetName()
	}
	if err == nil {
		reason := fmt.Sprintf("Successfully %s", strings.Title(verb))
		msg := fmt.Sprintf("%s %s/%s for controller %s/%s successfully",
			strings.ToLower(verb),
			objGVK.Kind, name,
			controllerGVK.Kind, controllerName)
		c.recorder.Event(controller, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed to %s", strings.Title(verb))
		msg := fmt.Sprintf("%s %s/%s for controller %s/%s failed, error: %s",
			strings.ToLower(verb),
			objGVK.Kind, name,
			controllerGVK.Kind, controllerName,
			err)
		c.recorder.Event(controller, corev1.EventTypeWarning, reason, msg)
	}
}

func setControllerReference(controller, obj runtime.Object) error {
	controllerMo, ok := controller.(metav1.Object)
	if !ok {
		return fmt.Errorf("%T is not a metav1.Object, cannot call setControllerReference", controller)
	}
	objMo, ok := obj.(metav1.Object)
	if !ok {
		return fmt.Errorf("%T is not a metav1.Object, cannot call setControllerReference", obj)
	}
	return controllerutil.SetControllerReference(controllerMo, objMo, scheme.Scheme)
}

// FakeGenericControl is a fake GenericControlInterface
type FakeGenericControl struct {
	FakeCli               client.Client
	control               GenericControlInterface
	createOrUpdateTracker RequestTracker
	deleteTracker         RequestTracker
	updateStatusTracker   RequestTracker
	createTracker         RequestTracker
	existTracker          RequestTracker
}

// NewFakeGenericControl returns a FakeGenericControl
func NewFakeGenericControl(initObjects ...runtime.Object) *FakeGenericControl {
	fakeCli := fake.NewFakeClientWithScheme(scheme.Scheme, initObjects...)
	control := NewRealGenericControl(fakeCli, record.NewFakeRecorder(10))
	return &FakeGenericControl{
		fakeCli,
		control,
		RequestTracker{},
		RequestTracker{},
		RequestTracker{},
		RequestTracker{},
		RequestTracker{},
	}
}
func (gc *FakeGenericControl) Create(controller, obj runtime.Object) error {
	defer gc.createTracker.Inc()
	if gc.createTracker.ErrorReady() {
		defer gc.createTracker.Reset()
		return gc.createTracker.GetError()
	}

	return gc.control.Create(controller, obj)
}

func (gc *FakeGenericControl) Exist(key client.ObjectKey, obj runtime.Object) (bool, error) {
	defer gc.existTracker.Inc()
	if gc.existTracker.ErrorReady() {
		defer gc.existTracker.Reset()
		return true, gc.existTracker.GetError()
	}

	return gc.control.Exist(key, obj)
}

func (gc *FakeGenericControl) SetCreateError(err error, after int) {
	gc.createTracker.SetError(err).SetAfter(after)
}
func (gc *FakeGenericControl) SetExistError(err error, after int) {
	gc.existTracker.SetError(err).SetAfter(after)
}
func (gc *FakeGenericControl) SetUpdateStatusError(err error, after int) {
	gc.updateStatusTracker.SetError(err).SetAfter(after)
}

func (gc *FakeGenericControl) SetCreateOrUpdateError(err error, after int) {
	gc.createOrUpdateTracker.SetError(err).SetAfter(after)
}

func (gc *FakeGenericControl) SetDeleteError(err error, after int) {
	gc.deleteTracker.SetError(err).SetAfter(after)
}

// AddObject is used to prepare the indexer for fakeGenericControl
func (gc *FakeGenericControl) AddObject(object runtime.Object) error {
	return gc.FakeCli.Create(context.TODO(), object.DeepCopyObject())
}

// UpdateStatus update the /status subresource of object
func (gc *FakeGenericControl) UpdateStatus(obj runtime.Object) error {
	defer gc.updateStatusTracker.Inc()
	if gc.updateStatusTracker.ErrorReady() {
		defer gc.updateStatusTracker.Reset()
		return gc.updateStatusTracker.GetError()
	}

	return gc.FakeCli.Status().Update(context.TODO(), obj)
}

func (gc *FakeGenericControl) CreateOrUpdate(controller, obj runtime.Object, fn MergeFn) (runtime.Object, error) {
	defer gc.createOrUpdateTracker.Inc()
	if gc.createOrUpdateTracker.ErrorReady() {
		defer gc.createOrUpdateTracker.Reset()
		return nil, gc.createOrUpdateTracker.GetError()
	}

	return gc.control.CreateOrUpdate(controller, obj, fn)
}

func (gc *FakeGenericControl) Delete(controller, obj runtime.Object) error {
	defer gc.deleteTracker.Inc()
	if gc.deleteTracker.ErrorReady() {
		defer gc.deleteTracker.Reset()
		return gc.deleteTracker.GetError()
	}

	return gc.control.Delete(controller, obj)
}

var _ GenericControlInterface = &FakeGenericControl{}
