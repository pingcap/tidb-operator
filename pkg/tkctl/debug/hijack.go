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
	"runtime"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/docker/pkg/term"
	"github.com/pingcap/tidb-operator/pkg/tkctl/streams"
	"github.com/sirupsen/logrus"
)

type Streams struct {
	In  *streams.In
	Out *streams.Out
	Err io.Writer
}

// A hijackedIOStreamer handles copying input to and output from streams to the connection.
// hijack logic is modified from docker-cli (docker/cli/command/container/hijack.go),
// which properly handled tty and control sequence
type hijackedIOStreamer struct {
	streams      Streams
	inputStream  io.ReadCloser
	outputStream io.Writer
	errorStream  io.Writer

	resp types.HijackedResponse

	tty bool
}

func newHijackedIOStreamer(ioStreams IOStreams, resp types.HijackedResponse) *hijackedIOStreamer {
	return &hijackedIOStreamer{
		streams: Streams{
			In:  streams.NewIn(ioStreams.In),
			Out: streams.NewOut(ioStreams.Out),
			Err: ioStreams.ErrOut,
		},

		inputStream:  ioStreams.In,
		outputStream: ioStreams.Out,
		errorStream:  ioStreams.ErrOut,

		resp: resp,

		tty: true,
	}
}

// stream handles setting up the IO and then begins streaming stdin/stdout
// to/from the hijacked connection, blocking until it is either done reading
// output, the user inputs the detach key sequence when in TTY mode, or when
// the given context is cancelled.
func (h *hijackedIOStreamer) stream(ctx context.Context) error {
	restoreInput, err := h.setupInput()
	if err != nil {
		return fmt.Errorf("unable to setup input stream: %s", err)
	}

	defer restoreInput()

	outputDone := h.beginOutputStream(restoreInput)
	inputDone, detached := h.beginInputStream(restoreInput)

	select {
	case err := <-outputDone:
		return err
	case <-inputDone:
		// Input stream has closed.
		if h.outputStream != nil || h.errorStream != nil {
			// Wait for output to complete streaming.
			select {
			case err := <-outputDone:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	case err := <-detached:
		// Got a detach key sequence.
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *hijackedIOStreamer) setupInput() (restore func(), err error) {
	if h.inputStream == nil || !h.tty {
		// No need to setup input TTY.
		// The restore func is a nop.
		return func() {}, nil
	}

	if err := setRawTerminal(h.streams); err != nil {
		return nil, fmt.Errorf("unable to set IO streams as raw terminal: %s", err)
	}

	// Use sync.Once so we may call restore multiple times but ensure we
	// only restore the terminal once.
	var restoreOnce sync.Once
	restore = func() {
		restoreOnce.Do(func() {
			restoreTerminal(h.streams, h.inputStream)
		})
	}

	h.inputStream = ioutils.NewReadCloserWrapper(h.inputStream, h.inputStream.Close)

	return restore, nil
}

func (h *hijackedIOStreamer) beginOutputStream(restoreInput func()) <-chan error {
	if h.outputStream == nil && h.errorStream == nil {
		// There is no need to copy output.
		return nil
	}

	outputDone := make(chan error)
	go func() {
		var err error

		// When TTY is ON, use regular copy
		if h.outputStream != nil && h.tty {
			_, err = io.Copy(h.outputStream, h.resp.Reader)
			// We should restore the terminal as soon as possible
			// once the connection ends so any following print
			// messages will be in normal type.
			restoreInput()
		} else {
			_, err = stdcopy.StdCopy(h.outputStream, h.errorStream, h.resp.Reader)
		}

		logrus.Debug("[hijack] End of stdout")

		if err != nil {
			logrus.Debugf("Error receiveStdout: %s", err)
		}

		outputDone <- err
	}()

	return outputDone
}

func (h *hijackedIOStreamer) beginInputStream(restoreInput func()) (doneC <-chan struct{}, detachedC <-chan error) {
	inputDone := make(chan struct{})
	detached := make(chan error)

	go func() {
		if h.inputStream != nil {
			_, err := io.Copy(h.resp.Conn, h.inputStream)
			// We should restore the terminal as soon as possible
			// once the connection ends so any following print
			// messages will be in normal type.
			restoreInput()

			logrus.Debug("[hijack] End of stdin")

			if _, ok := err.(term.EscapeError); ok {
				detached <- err
				return
			}

			if err != nil {
				// This error will also occur on the receive
				// side (from stdout) where it will be
				// propagated back to the caller.
				logrus.Debugf("Error sendStdin: %s", err)
			}
		}

		if err := h.resp.CloseWrite(); err != nil {
			logrus.Debugf("Couldn't send EOF: %s", err)
		}

		close(inputDone)
	}()

	return inputDone, detached
}

func setRawTerminal(streams Streams) error {
	if err := streams.In.SetRawTerminal(); err != nil {
		return err
	}
	return streams.Out.SetRawTerminal()
}

func restoreTerminal(streams Streams, in io.Closer) error {
	streams.In.RestoreTerminal()
	streams.Out.RestoreTerminal()
	if in != nil && runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return in.Close()
	}
	return nil
}
