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

package ops

import (
	"bytes"
	"io"
	"net/url"
	"strings"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/tests/pkg/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog"
)

// ExecOptions passed to ExecWithOptions
type ExecOptions struct {
	Command []string

	Namespace     string
	PodName       string
	ContainerName string

	Stdin         io.Reader
	CaptureStdout bool
	CaptureStderr bool
	// If false, whitespace in std{err,out} will be removed.
	PreserveWhitespace bool
}

// ExecWithOptions executes a command in the specified container,
// returning stdout, stderr and error. `options` allowed for
// additional parameters to be passed.
func (cli *ClientOps) ExecWithOptions(options ExecOptions) (string, string, error) {
	klog.Infof("ExecWithOptions %+v", options)

	config, err := client.LoadConfig()
	if err != nil {
		return "", "", err
	}

	const tty = false

	req := cli.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(options.PodName).
		Namespace(options.Namespace).
		SubResource("exec").
		Param("container", options.ContainerName)
	req.VersionedParams(&corev1.PodExecOptions{
		Container: options.ContainerName,
		Command:   options.Command,
		Stdin:     options.Stdin != nil,
		Stdout:    options.CaptureStdout,
		Stderr:    options.CaptureStderr,
		TTY:       tty,
	}, codec)

	var stdout, stderr bytes.Buffer
	err = execute("POST", req.URL(), config, options.Stdin, &stdout, &stderr, tty)

	if options.PreserveWhitespace {
		return stdout.String(), stderr.String(), err
	}
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

func (cli *ClientOps) KillProcess(ns string, pod string, container string, pname string) error {
	_, _, err := cli.ExecWithOptions(ExecOptions{
		Command:       []string{"pkill", pname},
		Namespace:     ns,
		PodName:       pod,
		ContainerName: container,
		CaptureStderr: true,
		CaptureStdout: true,
	})
	return err
}

func execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return errors.Trace(err)
	}
	return errors.Trace(exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	}))
}

var (
	scheme *runtime.Scheme
	codec  runtime.ParameterCodec
)

func init() {
	scheme = runtime.NewScheme()
	corev1.AddToScheme(scheme)
	codec = runtime.NewParameterCodec(scheme)
}
