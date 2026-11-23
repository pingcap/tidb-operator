// Copyright 2026 PingCAP, Inc.
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

package util

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/pkg/backup/s3auth"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestParseS3BROptions(t *testing.T) {
	for _, options := range [][]string{
		nil,
		{"--s3.role-arn="},
		{"--s3.external-id=external"},
		{"--s3.role-arn=first", "--s3.role-arn", "second", "--s3.external-id", ""},
		{"--s3.role-arn", "--s3.external-id=consumed-as-role"},
		{"--", "--s3.role-arn=ignored"},
		{"--checksum", "--s3.role-arn=role", "--filter", "table.*", "--s3.region=region"},
		{"--filter", "--s3.role-arn=consumed-as-filter"},
		{"--s3.role-arn"},
		{"--s3.external-id"},
		{"-f", "--s3.role-arn=consumed-as-filter"},
		{"--s3.endpoint=first", "--s3.endpoint", ""},
	} {
		t.Run("options", func(t *testing.T) {
			// Define surrounding BR bool/string flags too, so parity checks do
			// not accidentally repeat the production unknown-flag behavior.
			flags := pflag.NewFlagSet("BR", pflag.ContinueOnError)
			role := flags.String("s3.role-arn", "", "")
			external := flags.String("s3.external-id", "", "")
			region := flags.String("s3.region", "", "")
			endpoint := flags.String("s3.endpoint", "", "")
			flags.Bool("checksum", false, "")
			flags.StringArrayP("filter", "f", nil, "")
			err := flags.Parse(options)
			actual, parseErr := parseS3BROptions(options)
			if err != nil {
				require.Error(t, parseErr)
				return
			}
			require.NoError(t, parseErr)
			require.Equal(t, *role, actual.roleARN)
			require.Equal(t, *external, actual.externalID)
			require.Equal(t, *region, actual.region)
			require.Equal(t, *endpoint, actual.endpoint)
		})
	}
}

func TestStorageBackendUsesS3AssumeRole(t *testing.T) {
	f := s3auth.Install(t)
	f.ExpectedExternalID = "external-id"
	f.Objects["prefix/backupmeta"] = []byte("metadata")
	provider := v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
		Bucket: "bucket", Prefix: "prefix", Region: "us-east-1", Endpoint: "https://unused.unit.test", Provider: "aws",
	}}
	backend, err := NewStorageBackendWithBROptions(provider, &StorageCredential{}, []string{
		"--s3.role-arn=" + s3auth.RoleARN, "--s3.external-id", f.ExpectedExternalID,
		"--s3.region=us-west-2", "--s3.endpoint=" + s3auth.Endpoint,
	})
	require.NoError(t, err)
	defer backend.Close()
	data, err := backend.ReadAll(context.Background(), "backupmeta")
	require.NoError(t, err)
	require.Equal(t, "metadata", string(data))
	require.Equal(t, "us-east-1", provider.S3.Region)
	require.Equal(t, "https://unused.unit.test", provider.S3.Endpoint)
	for _, request := range f.Requests {
		require.Equal(t, "us-west-2", request.Region)
		if request.Service == "s3" {
			require.Equal(t, s3auth.RoleKey, request.AccessKey)
		}
		if request.Service == "sts" {
			require.NotEqual(t, "s3.unit.test", request.Host)
		}
	}
}

func TestSTSRegionAndEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name, specRegion, regionOption, envRegion, defaultRegion, endpointMode string
		wantSTSHost, wantSTSRegion, wantS3Region                               string
	}{
		{name: "spec region", specRegion: "us-west-2", envRegion: "ap-southeast-1", wantSTSHost: "sts.us-west-2.amazonaws.com", wantSTSRegion: "us-west-2", wantS3Region: "us-west-2"},
		{name: "BR option wins", specRegion: "us-west-2", regionOption: "--s3.region=ap-southeast-1", wantSTSHost: "sts.ap-southeast-1.amazonaws.com", wantSTSRegion: "ap-southeast-1", wantS3Region: "ap-southeast-1"},
		{name: "legacy", specRegion: "us-west-2", endpointMode: "LeGaCy", wantSTSHost: "sts.amazonaws.com", wantSTSRegion: "us-east-1", wantS3Region: "us-west-2"},
		{name: "environment only affects STS", envRegion: "ap-southeast-1", defaultRegion: "us-west-2", wantSTSHost: "sts.ap-southeast-1.amazonaws.com", wantSTSRegion: "ap-southeast-1", wantS3Region: "us-east-1"},
		{name: "default environment region", defaultRegion: "us-west-2", wantSTSHost: "sts.us-west-2.amazonaws.com", wantSTSRegion: "us-west-2", wantS3Region: "us-east-1"},
		{name: "no region", wantSTSHost: "sts.us-east-1.amazonaws.com", wantSTSRegion: "us-east-1", wantS3Region: "us-east-1"},
		{name: "empty BR option uses spec for STS", specRegion: "us-west-2", regionOption: "--s3.region=", wantSTSHost: "sts.us-west-2.amazonaws.com", wantSTSRegion: "us-west-2", wantS3Region: "us-east-1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := s3auth.Install(t)
			t.Setenv("AWS_REGION", tc.envRegion)
			t.Setenv("AWS_DEFAULT_REGION", tc.defaultRegion)
			t.Setenv("AWS_STS_REGIONAL_ENDPOINTS", tc.endpointMode)
			f.Objects["prefix/backupmeta"] = []byte("metadata")
			provider := v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
				Bucket: "bucket", Prefix: "prefix", Region: tc.specRegion, Endpoint: s3auth.Endpoint, Provider: "aws",
			}}
			options := []string{"--s3.role-arn=" + s3auth.RoleARN}
			if tc.regionOption != "" {
				options = append(options, tc.regionOption)
			}
			backend, err := NewStorageBackendWithBROptions(provider, nil, options)
			require.NoError(t, err)
			defer backend.Close()
			_, err = backend.ReadAll(context.Background(), "backupmeta")
			require.NoError(t, err)
			require.Len(t, f.Requests, 2)
			require.Equal(t, "sts", f.Requests[0].Service)
			require.Equal(t, tc.wantSTSHost, f.Requests[0].Host)
			require.Equal(t, tc.wantSTSRegion, f.Requests[0].Region)
			require.Equal(t, "s3", f.Requests[1].Service)
			require.Equal(t, tc.wantS3Region, f.Requests[1].Region)
		})
	}
}

func TestS3RoleDoesNotFallBack(t *testing.T) {
	for _, denySTS := range []bool{true, false} {
		t.Run("denied", func(t *testing.T) {
			f := s3auth.Install(t)
			f.AllowSource, f.DenyRole, f.DenyAssumeRole = true, true, denySTS
			f.Objects["prefix/backupmeta"] = []byte("metadata")
			backend, err := NewStorageBackendWithBROptions(v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
				Bucket: "bucket", Prefix: "prefix", Region: "us-east-1", Endpoint: s3auth.Endpoint, Provider: "aws",
			}}, nil, []string{"--s3.role-arn=" + s3auth.RoleARN})
			require.NoError(t, err)
			defer backend.Close()
			_, err = backend.ReadAll(context.Background(), "backupmeta")
			require.Error(t, err)
			for _, request := range f.Requests {
				if request.Service == "s3" {
					require.Equal(t, s3auth.RoleKey, request.AccessKey)
				}
			}
		})
	}
}

func TestS3NoRoleKeepsSourceCredentials(t *testing.T) {
	f := s3auth.Install(t)
	f.AllowSource = true
	f.Objects["prefix/backupmeta"] = []byte("metadata")
	backend, err := NewStorageBackendWithBROptions(v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
		Bucket: "bucket", Prefix: "prefix", Endpoint: s3auth.Endpoint, Provider: "aws",
	}}, &StorageCredential{awsCred: credentials.NewStaticCredentials(s3auth.SourceKey, "secret", s3auth.SourceToken)}, []string{"--s3.external-id=unused"})
	require.NoError(t, err)
	defer backend.Close()
	_, err = backend.ReadAll(context.Background(), "backupmeta")
	require.NoError(t, err)
	for _, request := range f.Requests {
		require.Equal(t, "s3", request.Service)
		require.Equal(t, s3auth.SourceKey, request.AccessKey)
	}
}

func TestS3RoleCredentialsRefresh(t *testing.T) {
	f := s3auth.Install(t)
	f.ExpectedSourceKey, f.ExpectedSourceToken = "explicit-source-key", "explicit-source-token"
	f.Objects["prefix/backupmeta"] = []byte("metadata")
	backend, err := NewStorageBackendWithBROptions(v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
		Bucket: "bucket", Prefix: "prefix", Endpoint: s3auth.Endpoint, Provider: "aws",
	}}, &StorageCredential{awsCred: credentials.NewStaticCredentials(f.ExpectedSourceKey, "secret", f.ExpectedSourceToken)}, []string{"--s3.role-arn=" + s3auth.RoleARN})
	require.NoError(t, err)
	defer backend.Close()
	_, err = backend.ReadAll(context.Background(), "backupmeta")
	require.NoError(t, err)
	client, ok := backend.AsS3()
	require.True(t, ok)
	client.Config.Credentials.Expire()
	_, err = backend.ReadAll(context.Background(), "backupmeta")
	require.NoError(t, err)
	assumeRoleCount := 0
	for _, request := range f.Requests {
		if request.Service == "sts" {
			assumeRoleCount++
		}
	}
	require.Equal(t, 2, assumeRoleCount)
}

func TestS3RoleMetadataRead(t *testing.T) {
	f := s3auth.Install(t)
	f.Objects["prefix/backupmeta"] = []byte("metadata")
	backend, err := NewStorageBackendWithBROptions(v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
		Bucket: "bucket", Prefix: "prefix", Endpoint: s3auth.Endpoint, Provider: "aws",
	}}, nil, []string{"--s3.role-arn=" + s3auth.RoleARN})
	require.NoError(t, err)
	defer backend.Close()
	data, err := backend.ReadAll(context.Background(), "backupmeta")
	require.NoError(t, err)
	require.Equal(t, "metadata", string(data))
}

func TestNonS3IgnoresS3RoleOptions(t *testing.T) {
	backend, err := NewStorageBackendWithBROptions(v1alpha1.StorageProvider{Local: &v1alpha1.LocalStorageProvider{VolumeMount: corev1.VolumeMount{MountPath: t.TempDir()}}}, nil, []string{"--s3.role-arn"})
	require.NoError(t, err)
	require.NoError(t, backend.Close())
}

func TestS3RoleParseErrorDoesNotExposeOptions(t *testing.T) {
	_, err := parseS3BROptions([]string{"--s3.external-id=secret-marker", "--s3.role-arn"})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "secret-marker")
}
