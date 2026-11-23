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

package s3auth

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	Endpoint    = "https://s3.unit.test"
	RoleARN     = "arn:aws:iam::123456789012:role/backup"
	SourceKey   = "source-access-key"
	RoleKey     = "role-access-key"
	SourceToken = "source-session-token" // #nosec G101 -- synthetic test token
)

// Request records identifiers, never secret keys or tokens.
type Request struct {
	Service, Method, Host, AccessKey, RoleARN, ExternalID, Region string
}

// Fixture accepts S3 access only with STS-issued credentials unless AllowSource is set.
// Install must not be used from parallel tests because the AWS SDK reads http.DefaultClient.
type Fixture struct {
	mu                                     sync.Mutex
	Objects                                map[string][]byte
	Requests                               []Request
	AllowSource, DenyRole, DenyAssumeRole  bool
	ExpectedExternalID                     string
	ExpectedSourceKey, ExpectedSourceToken string
}

// Install redirects every AWS request to this transport; unexpected hosts fail locally.
func Install(t testing.TB) *Fixture {
	t.Helper()
	for name, value := range map[string]string{
		"AWS_ACCESS_KEY_ID": SourceKey, "AWS_SECRET_ACCESS_KEY": "source-secret-key",
		"AWS_SESSION_TOKEN": SourceToken, "AWS_REGION": "us-east-1",
		"AWS_SHARED_CREDENTIALS_FILE": filepath.Join(t.TempDir(), "missing"),
		"AWS_CONFIG_FILE":             filepath.Join(t.TempDir(), "missing"),
		"AWS_SDK_LOAD_CONFIG":         "0", "AWS_PROFILE": "", "AWS_DEFAULT_PROFILE": "",
		"AWS_ROLE_ARN": "", "AWS_WEB_IDENTITY_TOKEN_FILE": "",
		"AWS_EC2_METADATA_DISABLED": "true", "AWS_STS_REGIONAL_ENDPOINTS": "regional",
	} {
		t.Setenv(name, value)
	}
	f := &Fixture{Objects: make(map[string][]byte), ExpectedSourceKey: SourceKey, ExpectedSourceToken: SourceToken}
	previous := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: f}
	t.Cleanup(func() { http.DefaultClient = previous })
	return f
}

// RoundTrip implements STS AssumeRole, S3 Head/Get/List/Delete without network I/O.
func (f *Fixture) RoundTrip(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	accessKey, region := "", ""
	if _, credential, ok := strings.Cut(req.Header.Get("Authorization"), "Credential="); ok {
		parts := strings.Split(credential, "/")
		accessKey = parts[0]
		if len(parts) > 2 {
			region = parts[2]
		}
	}
	record := Request{Method: req.Method, Host: req.URL.Host, AccessKey: accessKey, Region: region}
	if strings.HasPrefix(req.URL.Host, "sts.") {
		record.Service = "sts"
		if err := req.ParseForm(); err != nil {
			return nil, err
		}
		record.RoleARN, record.ExternalID = req.Form.Get("RoleArn"), req.Form.Get("ExternalId")
		f.Requests = append(f.Requests, record)
		if accessKey != f.ExpectedSourceKey || req.Header.Get("X-Amz-Security-Token") != f.ExpectedSourceToken {
			return nil, fmt.Errorf("STS did not receive source session credentials")
		}
		if f.DenyAssumeRole || record.RoleARN != RoleARN || record.ExternalID != f.ExpectedExternalID {
			return response(req, 403, `<ErrorResponse><Error><Code>AccessDenied</Code><Message>role denied</Message></Error></ErrorResponse>`), nil
		}
		body := fmt.Sprintf(`<AssumeRoleResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/"><AssumeRoleResult><Credentials><AccessKeyId>%s</AccessKeyId><SecretAccessKey>role-secret-key</SecretAccessKey><SessionToken>role-session-token</SessionToken><Expiration>%s</Expiration></Credentials><AssumedRoleUser><AssumedRoleId>id:backup</AssumedRoleId><Arn>arn:aws:sts::123456789012:assumed-role/backup/session</Arn></AssumedRoleUser></AssumeRoleResult><ResponseMetadata><RequestId>test</RequestId></ResponseMetadata></AssumeRoleResponse>`, RoleKey, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
		return response(req, 200, body), nil
	}
	if req.URL.Host != "s3.unit.test" {
		return nil, fmt.Errorf("unexpected AWS endpoint host %s", req.URL.Host)
	}
	record.Service = "s3"
	f.Requests = append(f.Requests, record)
	authorized := accessKey == RoleKey && !f.DenyRole || accessKey == f.ExpectedSourceKey && f.AllowSource
	if !authorized {
		return response(req, 403, `<Error><Code>AccessDenied</Code><Message>storage identity denied</Message></Error>`), nil
	}
	if accessKey == RoleKey && req.Header.Get("X-Amz-Security-Token") != "role-session-token" {
		return nil, fmt.Errorf("S3 did not receive role session token")
	}
	key := strings.TrimPrefix(req.URL.Path, "/bucket/")
	switch req.Method {
	case http.MethodHead, http.MethodGet:
		if req.URL.Query().Has("list-type") || req.URL.Path == "/bucket" || req.URL.Path == "/bucket/" {
			keys := make([]string, 0, len(f.Objects))
			for name := range f.Objects {
				if strings.HasPrefix(name, req.URL.Query().Get("prefix")) {
					keys = append(keys, name)
				}
			}
			sort.Strings(keys)
			body := `<ListBucketResult><Name>bucket</Name><IsTruncated>false</IsTruncated>`
			for _, name := range keys {
				body += fmt.Sprintf(`<Contents><Key>%s</Key><Size>%d</Size><LastModified>2026-01-01T00:00:00Z</LastModified></Contents>`, name, len(f.Objects[name]))
			}
			return response(req, 200, body+`</ListBucketResult>`), nil
		}
		body, exists := f.Objects[key]
		if !exists {
			return response(req, 404, `<Error><Code>NoSuchKey</Code></Error>`), nil
		}
		return response(req, 200, string(body)), nil
	case http.MethodDelete:
		delete(f.Objects, key)
		return response(req, 204, ""), nil
	case http.MethodPost:
		if req.URL.Query().Has("delete") {
			var request struct {
				Objects []struct{ Key string } `xml:"Object"`
			}
			if err := xml.NewDecoder(req.Body).Decode(&request); err != nil {
				return nil, err
			}
			body := `<DeleteResult>`
			for _, object := range request.Objects {
				delete(f.Objects, object.Key)
				body += `<Deleted><Key>` + object.Key + `</Key></Deleted>`
			}
			return response(req, 200, body+`</DeleteResult>`), nil
		}
	}
	return nil, fmt.Errorf("unexpected S3 operation %s", req.Method)
}

func response(req *http.Request, status int, body string) *http.Response {
	result := &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/xml"}, "Content-Length": {strconv.Itoa(len(body))}},
		Body: io.NopCloser(strings.NewReader(body)), ContentLength: int64(len(body)), Request: req}
	if req.Method == http.MethodHead {
		result.Body = io.NopCloser(strings.NewReader(""))
	}
	return result
}
