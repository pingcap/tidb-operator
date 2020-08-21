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

package debug

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/tkctl/config"
	"github.com/pingcap/tidb-operator/pkg/tkctl/executor"
	"github.com/pingcap/tidb-operator/pkg/tkctl/util"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

const (
	debugExample = `
	# debug a container in the running pod, the first container will be picked by default
	tkctl debug POD_NAME

	# specify namespace or container
	tkctl debug --namespace foo POD_NAME -c CONTAINER_NAME

	# override the default troubleshooting image
	tkctl debug POD_NAME --image aylei/debug-jvm

	# override entrypoint of debug container
	tkctl debug POD_NAME --image aylei/debug-jvm /bin/bash

`
	debugLongDesc = `
Run a container in a running pod, this container will join the namespaces of an existing container of the pod.

Under the hood, 'debug' create a agent in target node, the agent is responsible for launching the actual debug
container which will join linux namespaces of the target container and proxying I/O connection. When debug session
ends, the agent destroy and cleanup the debug container and exited with code 0, 'tkc' will clean the finished agent 
in the defer manner if it is not interrupted by user.
`
	debugUsage    = "expected 'debug POD_NAME' for the debug command"
	defaultImage  = "pingcap/tidb-debug:latest"
	launcherImage = "pingcap/debug-launcher:latest"
	launcherName  = "debug-launcher"
)

var (
	defaultCommand = []string{"bash", "-l"}
)

// DebugOptions specify how to run debug container in a running pod
type DebugOptions struct {

	// Pod select options
	Namespace string
	PodName   string

	// Debug options
	Image            string
	ContainerName    string
	Command          []string
	HostDockerSocket string
	LauncherImage    string
	Privileged       bool

	KubeCli *kubernetes.Clientset

	RestConfig *rest.Config

	genericclioptions.IOStreams
}

func NewDebugOptions(iostreams genericclioptions.IOStreams) *DebugOptions {
	return &DebugOptions{
		Image:            defaultImage,
		HostDockerSocket: util.DockerSocket,
		LauncherImage:    launcherImage,

		IOStreams: iostreams,
	}
}

// NewCmdDebug creates the debug subcommand which helps container debugging
func NewCmdDebug(tkcContext *config.TkcContext, streams genericclioptions.IOStreams) *cobra.Command {
	options := NewDebugOptions(streams)

	cmd := &cobra.Command{
		Use:     "debug",
		Short:   "Run a debug container",
		Long:    debugLongDesc,
		Example: debugExample,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(options.Complete(tkcContext, cmd, args))
			cmdutil.CheckErr(options.Run())
		},
	}
	cmd.Flags().StringVar(&options.Image, "image", options.Image,
		"Container Image to run the debug container")
	cmd.Flags().StringVarP(&options.ContainerName, "container", "c", "",
		"Target container to debug, default to the first container in pod")
	cmd.Flags().StringVar(&options.HostDockerSocket, "docker-socketl", options.HostDockerSocket,
		"docker socket path of kubernetes node")
	cmd.Flags().StringVar(&options.LauncherImage, "launcher-image", options.LauncherImage,
		"image for launcher pod which is responsible to launch the debug container")
	cmd.Flags().BoolVar(&options.Privileged, "privileged", options.Privileged,
		"whether launch container in privileged mode (full container capabilities)")
	return cmd
}

// Complete populate default values for DebugOptions
func (o *DebugOptions) Complete(tkcContext *config.TkcContext, cmd *cobra.Command, argsIn []string) error {
	// select one pod to debug (required)
	if len(argsIn) == 0 {
		return cmdutil.UsageErrorf(cmd, debugUsage)
	}
	o.PodName = argsIn[0]

	o.Command = argsIn[1:]
	if len(o.Command) < 1 {
		o.Command = defaultCommand
	}
	clientConfig, err := tkcContext.ToTkcClientConfig()
	if err != nil {
		return err
	}
	ns, _, err := clientConfig.Namespace()
	if err != nil {
		return err
	}
	o.Namespace = ns
	restConfig, err := clientConfig.RestConfig()
	if err != nil {
		return err
	}
	kubeCli, err := kubernetes.NewForConfig(restConfig)
	o.RestConfig = restConfig
	if err != nil {
		return err
	}
	o.KubeCli = kubeCli
	return nil
}

func (o *DebugOptions) Run() error {

	// 0.Prepare debug context: get Pod and verify state
	pod, err := o.KubeCli.CoreV1().Pods(o.Namespace).Get(o.PodName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
		return fmt.Errorf("cannot debug in a completed pod; current phase is %s", pod.Status.Phase)
	}
	containerName := o.ContainerName
	if len(containerName) == 0 {
		if len(pod.Spec.Containers) > 1 {
			usageString := fmt.Sprintf("Defaulting container name to %s.", pod.Spec.Containers[0].Name)
			fmt.Fprintf(o.ErrOut, "%s\n\r", usageString)
		}
		containerName = pod.Spec.Containers[0].Name
	}

	nodeName := pod.Spec.NodeName
	targetContainerID, err := o.getContainerIDByName(pod, containerName)
	if err != nil {
		return err
	}

	launcher := o.makeLauncherPod(nodeName, targetContainerID, o.Command)
	podExecutor := executor.NewPodExecutor(o.KubeCli, launcher, o.RestConfig, o.IOStreams)
	return podExecutor.Execute()
}

func (o *DebugOptions) makeLauncherPod(nodeName, containerID string, command []string) *v1.Pod {

	volume, mount := util.MakeDockerSocketMount(o.HostDockerSocket, false)
	// we always mount docker socket to default path despite the host docker socket path
	launchArgs := []string{
		"--target-container",
		containerID,
		"--image",
		o.Image,
		"--docker-socket",
		fmt.Sprintf("unix://%s", util.DockerSocket),
	}
	if o.Privileged {
		launchArgs = append(launchArgs, "--privileged")
	}
	launchArgs = append(launchArgs, "--")
	launchArgs = append(launchArgs, command...)
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", launcherName, string(uuid.NewUUID())),
			Namespace: o.Namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            launcherName,
					Image:           o.LauncherImage,
					Args:            launchArgs,
					Stdin:           true,
					TTY:             true,
					VolumeMounts:    []v1.VolumeMount{mount},
					ImagePullPolicy: v1.PullAlways,
				},
			},
			Volumes:       []v1.Volume{volume},
			NodeName:      nodeName,
			RestartPolicy: v1.RestartPolicyNever,
		},
	}
}

func (o *DebugOptions) getContainerIDByName(pod *v1.Pod, containerName string) (string, error) {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name != containerName {
			continue
		}
		if !containerStatus.Ready {
			return "", fmt.Errorf("container [%s] not ready", containerName)
		}
		return containerStatus.ContainerID, nil
	}

	// also search init containers
	for _, initContainerStatus := range pod.Status.InitContainerStatuses {
		if initContainerStatus.Name != containerName {
			continue
		}
		if initContainerStatus.State.Running == nil {
			return "", fmt.Errorf("init container [%s] is not running", containerName)
		}
		return initContainerStatus.ContainerID, nil
	}

	return "", fmt.Errorf("cannot find specified container %s", containerName)
}
