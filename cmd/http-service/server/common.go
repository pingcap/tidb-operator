// Copyright 2023 PingCAP, Inc.
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

package server

import (
	"context"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	HeaderKeyKubernetesID = "kubernetes-id"

	// if we return an error in the gRPC service, the HTTP response body will be something like
	// `{"code":2, "message":"k8s client not found", "details":[]}`.
	// so we choose to return a nil error and set the HTTP status code in the response header.
	ResponseKeyStatusCode = "x-http-code"
)

func getKubernetesID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md.Get(HeaderKeyKubernetesID)) == 0 {
		return ""
	}
	return md.Get(HeaderKeyKubernetesID)[0]
}

func setResponseStatusCodes(ctx context.Context, statusCode int) {
	_ = grpc.SetHeader(ctx, metadata.Pairs(ResponseKeyStatusCode, strconv.Itoa(statusCode)))
}
