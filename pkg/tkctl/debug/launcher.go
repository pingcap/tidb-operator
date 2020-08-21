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
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/spf13/cobra"
)

const (
	defaultDockerSocket   = "unix:///var/run/docker.sock"
	dockerContainerPrefix = "docker://"

	CAP_SYS_PTRACE = "SYS_PTRACE"
	CAP_SYS_ADMIN  = "SYS_ADMIN"
)

type IOStreams struct {
	In     io.ReadCloser
	Out    io.WriteCloser
	ErrOut io.WriteCloser
}

// Launcher is responsible for launching debug container
type Launcher struct {
	IOStreams

	targetContainerID string
	image             string
	dockerSocket      string
	ctx               context.Context

	privileged bool

	client *dockerclient.Client
}

// NewLauncher create a launcher instance
func NewLauncher(streams IOStreams) *Launcher {
	return &Launcher{
		dockerSocket: defaultDockerSocket,
		ctx:          context.Background(),

		IOStreams: streams,
	}
}

// NewLauncherCmd create the launcher command
func NewLauncherCmd(streams IOStreams) *cobra.Command {
	launcher := NewLauncher(streams)
	cmd := &cobra.Command{
		Use: "debug-launcher --target-container=CONTAINER_ID --image=IMAGE -- COMMAND",
		RunE: func(c *cobra.Command, args []string) error {
			return launcher.Run(args)
		},
	}
	cmd.Flags().StringVar(&launcher.targetContainerID, "target-container", launcher.targetContainerID,
		"target container id")
	cmd.Flags().StringVar(&launcher.image, "image", launcher.image,
		"debug container image")
	cmd.Flags().StringVar(&launcher.dockerSocket, "docker-socket", launcher.dockerSocket,
		"docker socket to bind")
	cmd.Flags().BoolVar(&launcher.privileged, "privileged", launcher.privileged,
		"whether launch container in privileged mode (full container capabilities)")
	return cmd
}

// Run launches the debug container and attach it.
// We could alternatively just run docker exec in command line, but this brings shell and docker client to the
// image, which is unwanted.
func (l *Launcher) Run(args []string) error {
	client, err := dockerclient.NewClient(l.dockerSocket, "", nil, nil)
	if err != nil {
		return err
	}
	l.client = client
	err = l.pullImage()
	if err != nil {
		return err
	}
	resp, err := l.createContainer(args)
	if err != nil {
		return err
	}
	containerID := resp.ID
	fmt.Fprintf(l.Out, "starting debug container...\n")
	err = l.startContainer(containerID)
	if err != nil {
		return err
	}
	defer l.cleanContainer(containerID)
	err = l.attachToContainer(containerID)
	if err != nil {
		return err
	}
	return nil
}

func (l *Launcher) pullImage() error {
	out, err := l.client.ImagePull(l.ctx, l.image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer out.Close()
	// write pull progress to user
	jsonmessage.DisplayJSONMessagesStream(out, l.Out, 1, true, nil)
	return nil
}

func (l *Launcher) createContainer(command []string) (*container.ContainerCreateCreatedBody, error) {
	if !strings.HasPrefix(l.targetContainerID, dockerContainerPrefix) {
		return nil, fmt.Errorf("Only docker containers are supported now")
	}
	dockerContainerID := l.targetContainerID[len(dockerContainerPrefix):]
	config := &container.Config{
		Entrypoint: strslice.StrSlice(command),
		Image:      l.image,
		Tty:        true,
		OpenStdin:  true,
		StdinOnce:  true,
	}
	hostConfig := &container.HostConfig{
		NetworkMode: container.NetworkMode(containerMode(dockerContainerID)),
		UsernsMode:  container.UsernsMode(containerMode(dockerContainerID)),
		IpcMode:     container.IpcMode(containerMode(dockerContainerID)),
		PidMode:     container.PidMode(containerMode(dockerContainerID)),
		CapAdd:      strslice.StrSlice([]string{CAP_SYS_PTRACE, CAP_SYS_ADMIN}),
		Privileged:  l.privileged,
	}
	body, err := l.client.ContainerCreate(l.ctx, config, hostConfig, nil, "")
	if err != nil {
		return nil, err
	}
	return &body, nil
}

func (l *Launcher) startContainer(id string) error {
	err := l.client.ContainerStart(l.ctx, id, types.ContainerStartOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (l *Launcher) attachToContainer(id string) error {
	opts := types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	}
	resp, err := l.client.ContainerAttach(l.ctx, id, opts)
	if err != nil {
		return err
	}
	streamer := newHijackedIOStreamer(l.IOStreams, resp)
	defer resp.Close()
	return streamer.stream(l.ctx)
}

func (l *Launcher) cleanContainer(id string) error {
	// once attach complete, the debug container is considered to be exited, so its safe to rm --force
	err := l.client.ContainerRemove(l.ctx, id,
		types.ContainerRemoveOptions{
			Force: true,
		})
	if err != nil {
		return err
	}
	return nil
}

func containerMode(id string) string {
	return fmt.Sprintf("container:%s", id)
}
