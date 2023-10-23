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
	"errors"

	"github.com/pingcap/tidb-operator/http-service/pbgen/api"
)

func (s *ClusterServer) CreateBackup(ctx context.Context, req *api.CreateBackupReq) (*api.CreateBackupResp, error) {
	return nil, errors.New("CreateBackup not implemented")
}

func (s *ClusterServer) CreateRestore(ctx context.Context, req *api.CreateRestoreReq) (*api.CreateRestoreResp, error) {
	return nil, errors.New("CreateRestore not implemented")
}

func (s *ClusterServer) GetBackup(ctx context.Context, req *api.GetBackupReq) (*api.GetBackupResp, error) {
	return nil, errors.New("GetBackup not implemented")
}

func (s *ClusterServer) GetRestore(ctx context.Context, req *api.GetRestoreReq) (*api.GetRestoreResp, error) {
	return nil, errors.New("GetRestore not implemented")
}

func (s *ClusterServer) StopBackup(ctx context.Context, req *api.StopBackupReq) (*api.StopBackupResp, error) {
	return nil, errors.New("StopBackup not implemented")
}

func (s *ClusterServer) StopRestore(ctx context.Context, req *api.StopRestoreReq) (*api.StopRestoreResp, error) {
	return nil, errors.New("StopRestore not implemented")
}

func (s *ClusterServer) DeleteBackup(ctx context.Context, req *api.DeleteBackupReq) (*api.DeleteBackupResp, error) {
	return nil, errors.New("DeleteBackup not implemented")
}
