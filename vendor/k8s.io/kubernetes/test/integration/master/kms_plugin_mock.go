// +build !windows

/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package master

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"os"

	"golang.org/x/sys/unix"
	"google.golang.org/grpc"

	kmsapi "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
)

const (
	kmsAPIVersion = "v1beta1"
	sockFile      = "/tmp/kms-provider.sock"
	unixProtocol  = "unix"
)

// base64Plugin gRPC sever for a mock KMS provider.
// Uses base64 to simulate encrypt and decrypt.
type base64Plugin struct {
	grpcServer *grpc.Server
	listener   net.Listener

	// Allow users of the plugin to sense requests that were passed to KMS.
	encryptRequest chan *kmsapi.EncryptRequest
	decryptRequest chan *kmsapi.DecryptRequest
}

func NewBase64Plugin() (*base64Plugin, error) {
	if err := cleanSockFile(); err != nil {
		return nil, err
	}

	listener, err := net.Listen(unixProtocol, sockFile)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on the unix socket, error: %v", err)
	}

	server := grpc.NewServer()

	result := &base64Plugin{
		grpcServer:     server,
		listener:       listener,
		encryptRequest: make(chan *kmsapi.EncryptRequest, 1),
		decryptRequest: make(chan *kmsapi.DecryptRequest, 1),
	}

	kmsapi.RegisterKeyManagementServiceServer(server, result)

	return result, nil
}

func (s *base64Plugin) cleanUp() {
	s.grpcServer.Stop()
	s.listener.Close()
	cleanSockFile()
}

var testProviderAPIVersion = kmsAPIVersion

func (s *base64Plugin) Version(ctx context.Context, request *kmsapi.VersionRequest) (*kmsapi.VersionResponse, error) {
	return &kmsapi.VersionResponse{Version: testProviderAPIVersion, RuntimeName: "testKMS", RuntimeVersion: "0.0.1"}, nil
}

func (s *base64Plugin) Decrypt(ctx context.Context, request *kmsapi.DecryptRequest) (*kmsapi.DecryptResponse, error) {
	s.decryptRequest <- request
	buf := make([]byte, base64.StdEncoding.DecodedLen(len(request.Cipher)))
	n, err := base64.StdEncoding.Decode(buf, request.Cipher)
	if err != nil {
		return nil, err
	}

	return &kmsapi.DecryptResponse{Plain: buf[:n]}, nil
}

func (s *base64Plugin) Encrypt(ctx context.Context, request *kmsapi.EncryptRequest) (*kmsapi.EncryptResponse, error) {
	s.encryptRequest <- request

	buf := make([]byte, base64.StdEncoding.EncodedLen(len(request.Plain)))
	base64.StdEncoding.Encode(buf, request.Plain)

	return &kmsapi.EncryptResponse{Cipher: buf}, nil
}

func cleanSockFile() error {
	err := unix.Unlink(sockFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete the socket file, error: %v", err)
	}
	return nil
}
