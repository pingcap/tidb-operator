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
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/spf13/pflag"
)

type s3BROptions struct {
	roleARN     string
	externalID  string
	region      string
	endpoint    string
	regionSet   bool
	endpointSet bool
}

// NewStorageBackendWithBROptions applies BR's S3 assume-role settings to storage
// operations performed by backup-manager itself. Explicit roles never fall back
// to the source credentials when assuming the role or accessing S3 fails.
func NewStorageBackendWithBROptions(provider v1alpha1.StorageProvider, source *StorageCredential, options []string) (*StorageBackend, error) {
	if GetStorageType(provider) != v1alpha1.BackupStorageTypeS3 {
		return NewStorageBackend(provider, source)
	}
	settings, err := parseS3BROptions(options)
	if err != nil {
		return nil, err
	}
	// The caller's provider can be shared with BR argument construction.
	s3Provider := *provider.S3
	if settings.regionSet {
		s3Provider.Region = settings.region
	}
	if settings.endpointSet {
		s3Provider.Endpoint = settings.endpoint
	}
	provider.S3 = &s3Provider
	if settings.roleARN != "" {
		region := makeS3Config(provider.S3, true).region
		config := aws.NewConfig().WithRegion(region)
		if source != nil && source.awsCred != nil {
			config.WithCredentials(source.awsCred)
		}
		// Keep the S3 endpoint out of this session: STS must use its own endpoint.
		sess, err := session.NewSession(config)
		if err != nil {
			return nil, fmt.Errorf("create S3 assume-role session: %w", err)
		}
		roleCredentials := stscreds.NewCredentials(sess, settings.roleARN, func(p *stscreds.AssumeRoleProvider) {
			if settings.externalID != "" {
				p.ExternalID = aws.String(settings.externalID)
			}
		})
		source = &StorageCredential{awsCred: roleCredentials}
	}
	return NewStorageBackend(provider, source)
}

func parseS3BROptions(options []string) (s3BROptions, error) {
	var result s3BROptions
	flags := pflag.NewFlagSet("S3 storage authentication", pflag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.ParseErrorsWhitelist.UnknownFlags = true
	flags.StringVar(&result.roleARN, "s3.role-arn", "", "")
	flags.StringVar(&result.externalID, "s3.external-id", "", "")
	flags.StringVar(&result.region, "s3.region", "", "")
	flags.StringVar(&result.endpoint, "s3.endpoint", "", "")
	// A filter is a string value even if it starts with "--". BR validates
	// filter expressions; consume it here rather than misreading it as a role.
	flags.StringArrayP("filter", "f", nil, "")
	if err := flags.Parse(options); err != nil {
		// pflag errors can contain option values. Never include raw BR options.
		return s3BROptions{}, fmt.Errorf("invalid BR S3 role, external-id, region or endpoint options")
	}
	result.regionSet = flags.Changed("s3.region")
	result.endpointSet = flags.Changed("s3.endpoint")
	// Match BR: an empty role disables AssumeRole, and external-id by itself
	// does not introduce a new validation error.
	return result, nil
}
