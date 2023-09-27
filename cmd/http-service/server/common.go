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
	"fmt"
	"os"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/pingcap/tidb-operator/http-service/pbgen/api"
)

const (
	HeaderKeyKubernetesID = "kubernetes-id"

	// if we return an error in the gRPC service, the HTTP response body will be something like
	// `{"code":2, "message":"k8s client not found", "details":[]}`.
	// so we choose to return a nil error and set the HTTP status code in the response header.
	ResponseKeyStatusCode = "x-http-code"

	// we use the `cluster-id` as the namespace of the TidbCluster,
	// and use the same name for all TidbClusters.
	tidbClusterName = "db"

	memoryStorageUnit = "Gi"

	// the name of the environment variable that indicates whether the program is running in a local environment.
	// if it is running in a local environment:
	// - remove CPU and memory requests so that the pods can be scheduled on a small local machine.
	localRunEnvName = "LOCAL_RUN"
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

func convertResourceRequirements(res *api.Resource) (corev1.ResourceRequirements, error) {
	cpu, err := resource.ParseQuantity(strconv.Itoa(int(res.Cpu)))
	if err != nil {
		return corev1.ResourceRequirements{}, err
	}
	memory, err := resource.ParseQuantity(fmt.Sprintf("%d%s", res.Memory, memoryStorageUnit))
	if err != nil {
		return corev1.ResourceRequirements{}, err
	}

	ret := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    cpu,
			corev1.ResourceMemory: memory,
		},
		Limits: corev1.ResourceList{ // == Requests
			corev1.ResourceCPU:    cpu,
			corev1.ResourceMemory: memory,
		},
	}

	if res.Storage != nil && *res.Storage > 0 {
		storage, err := resource.ParseQuantity(fmt.Sprintf("%d%s", *res.Storage, memoryStorageUnit))
		if err != nil {
			return corev1.ResourceRequirements{}, err
		}
		ret.Requests[corev1.ResourceStorage] = storage
		ret.Limits[corev1.ResourceStorage] = storage
	}

	if os.Getenv(localRunEnvName) != "" {
		// remove CPU and memory requests so that the pods can be scheduled on a small local machine
		ret.Requests[corev1.ResourceCPU] = resource.Quantity{}
		ret.Requests[corev1.ResourceMemory] = resource.Quantity{}
	}

	return ret, nil
}
