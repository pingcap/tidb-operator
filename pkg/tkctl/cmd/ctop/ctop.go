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

package ctop

import (
	"fmt"
	"strings"

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
	ctopExample = `
	# ctop the specified pod
	tkctl ctop POD_NAME

	# ctop the specified node
	tkctl ctop node/NODE_NAME
`
	ctopUsage    = "expected 'ctop POD_NAME' or 'ctop node/NODE_NAME' for the ctop command"
	defaultImage = "quay.io/vektorlab/ctop:0.7.2"
)

type CtopKind string

const (
	CtopPod  CtopKind = "pod"
	CtopNode CtopKind = "node"
)

// CtopOptions specify the target resource stats to show
type CtopOptions struct {
	Target           string
	Kind             CtopKind
	Namespace        string
	Image            string
	HostDockerSocket string

	KubeCli *kubernetes.Clientset

	RestConfig *rest.Config

	genericclioptions.IOStreams
}

// NewCtopOptions create a ctop options
func NewCtopOptions(iostreams genericclioptions.IOStreams) *CtopOptions {
	return &CtopOptions{
		Kind:             CtopPod,
		Image:            defaultImage,
		HostDockerSocket: util.DockerSocket,

		IOStreams: iostreams,
	}
}

// NewCmdTop creates the ctop subcommand
func NewCmdCtop(tkcContext *config.TkcContext, streams genericclioptions.IOStreams) *cobra.Command {
	options := NewCtopOptions(streams)

	cmd := &cobra.Command{
		Use:     "ctop",
		Example: ctopExample,
		Short:   "ctop shows top-like container stats for given pod",
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(options.Complete(cmd, tkcContext, args))
			cmdutil.CheckErr(options.Run())
		},
	}
	cmd.Flags().StringVar(&options.HostDockerSocket, "docker-socketl", options.HostDockerSocket,
		"docker socket path of kubernetes node")
	cmd.Flags().StringVar(&options.Image, "image", options.Image,
		"Container Image to run the debug container")
	return cmd
}

func (o *CtopOptions) Complete(cmd *cobra.Command, tkcContext *config.TkcContext, args []string) error {
	if len(args) < 1 {
		return cmdutil.UsageErrorf(cmd, ctopUsage)
	}
	o.Kind, o.Target = parseTarget(args[0])

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

func (o *CtopOptions) Run() error {

	var nodeName string
	var filter string
	switch o.Kind {
	case CtopPod:
		pod, err := o.KubeCli.CoreV1().Pods(o.Namespace).Get(o.Target, metav1.GetOptions{})
		if err != nil {
			return err
		}
		nodeName = pod.Spec.NodeName
		filter = pod.Name
	case CtopNode:
		nodeName = o.Target
	default:
		return fmt.Errorf("unknown resource type %s", string(o.Kind))
	}
	pod := o.makeCtopPod(nodeName, filter)
	podExecutor := executor.NewPodExecutor(o.KubeCli, pod, o.RestConfig, o.IOStreams)
	return podExecutor.Execute()
}

func (o *CtopOptions) makeCtopPod(nodeName, filter string) *v1.Pod {
	args := []string{"-a"}
	if len(filter) > 0 {
		args = []string{
			"-f",
			filter,
		}
	}
	volume, mount := util.MakeDockerSocketMount(o.HostDockerSocket, true)
	ctopPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("ctop-%s", string(uuid.NewUUID())),
			Namespace: o.Namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "ctop",
					Image:           o.Image,
					Args:            args,
					Stdin:           true,
					StdinOnce:       true,
					TTY:             true,
					VolumeMounts:    []v1.VolumeMount{mount},
					ImagePullPolicy: v1.PullIfNotPresent,
				},
			},
			Volumes:       []v1.Volume{volume},
			NodeName:      nodeName,
			RestartPolicy: v1.RestartPolicyNever,
		},
	}
	return ctopPod
}

func parseTarget(arg string) (kind CtopKind, target string) {
	splits := strings.SplitN(arg, "/", 2)
	if len(splits) == 1 {
		kind = CtopPod
		target = splits[0]
	} else {
		kind = CtopKind(splits[0])
		target = splits[1]
	}
	return
}
