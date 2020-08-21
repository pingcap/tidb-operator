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

package executor

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	watchapi "k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/watch"
	cmdattach "k8s.io/kubectl/pkg/cmd/attach"
	cmdrun "k8s.io/kubectl/pkg/cmd/run"
	"k8s.io/kubernetes/pkg/util/interrupt"
)

const (
	defaultWaitTimeOutSeconds = 5 * 60
)

// PodExecutor run pod once, attach to it, and clean pod when session complete
type PodExecutor struct {
	Pod     *v1.Pod
	KubeCli *kubernetes.Clientset

	RestConfig *rest.Config

	genericclioptions.IOStreams
}

func NewPodExecutor(kubeCli *kubernetes.Clientset, pod *v1.Pod, restConfig *rest.Config, streams genericclioptions.IOStreams) *PodExecutor {
	return &PodExecutor{
		KubeCli: kubeCli,
		Pod:     pod,

		RestConfig: restConfig,

		IOStreams: streams,
	}
}

func (t *PodExecutor) Execute() error {
	pod, err := t.KubeCli.CoreV1().Pods(t.Pod.Namespace).Create(t.Pod)
	if err != nil {
		return err
	}

	defer t.removePod(pod)
	return t.attachPod(pod)
}

// podRunningAndReady returns true if the pod is running and ready, false if the pod has not
// yet reached those states, returns ErrPodCompleted if the pod has run to completion, or
// an error in any other case.
func podRunningAndReady(event watchapi.Event) (bool, error) {
	switch event.Type {
	case watchapi.Deleted:
		return false, errors.NewNotFound(schema.GroupResource{Resource: "pods"}, "")
	}
	switch t := event.Object.(type) {
	case *v1.Pod:
		switch t.Status.Phase {
		case v1.PodFailed, v1.PodSucceeded:
			return false, cmdrun.ErrPodCompleted
		case v1.PodRunning:
			conditions := t.Status.Conditions
			if conditions == nil {
				return false, nil
			}
			for i := range conditions {
				if conditions[i].Type == v1.PodReady &&
					conditions[i].Status == v1.ConditionTrue {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (t *PodExecutor) attachPod(pod *v1.Pod) error {

	// TODO: currently, if a pod stuck in ImagePullBackoff state, watch thinks it still has chance to start so will
	// keep waiting, but most likely the pod cannot really start. For better experience, We should periodically
	// print pod current state, so that user can interrupt waiting when seeing 'ImagePullBackoff' or other unexpected states.
	fmt.Fprintf(t.Out, "waiting for pod %s running...\n", pod.Name)
	watched, err := t.waitForPod(pod, defaultWaitTimeOutSeconds, podRunningAndReady)
	if err != nil {
		return err
	}
	if watched.Status.Phase == v1.PodSucceeded || watched.Status.Phase == v1.PodFailed {
		return fmt.Errorf("pod unexpcetedly exits, status: %s", watched.Status.Phase)
	}
	// reuse `kubectl attach` facility
	attachOpts := cmdattach.NewAttachOptions(t.IOStreams)
	attachOpts.TTY = true
	attachOpts.Stdin = true
	attachOpts.Pod = watched
	attachOpts.PodName = watched.Name
	attachOpts.Namespace = watched.Namespace
	attachOpts.Config = t.RestConfig
	setKubernetesDefaults(attachOpts.Config)

	return attachOpts.Run()
}

func (t *PodExecutor) removePod(pod *v1.Pod) error {
	return t.KubeCli.CoreV1().
		Pods(pod.Namespace).
		Delete(pod.Name, &metav1.DeleteOptions{})
}

func (t *PodExecutor) waitForPod(pod *v1.Pod, timeoutSeconds int64, exitCondition watch.ConditionFunc) (*v1.Pod, error) {
	w, err := t.KubeCli.CoreV1().Pods(pod.Namespace).Watch(metav1.SingleObject(metav1.ObjectMeta{Name: pod.Name}))
	if err != nil {
		return nil, err
	}
	ctx, cancel := watch.ContextWithOptionalTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	intr := interrupt.New(nil, cancel)
	var result *v1.Pod
	err = intr.Run(func() error {
		event, err := watch.UntilWithoutRetry(ctx, w, func(event watchapi.Event) (bool, error) {
			return exitCondition(event)
		})
		if event != nil {
			result = event.Object.(*v1.Pod)
		}
		return err
	})
	if err != nil && errors.IsNotFound(err) {
		err = errors.NewNotFound(v1.Resource("pods"), pod.Name)
	}
	return result, err
}

func setKubernetesDefaults(config *rest.Config) error {
	config.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}

	if config.APIPath == "" {
		config.APIPath = "/api"
	}
	if config.NegotiatedSerializer == nil {
		// This codec factory ensures the resources are not converted. Therefore, resources
		// will not be round-tripped through internal versions. Defaulting does not happen
		// on the client.
		config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	}
	return rest.SetKubernetesDefaults(config)
}
