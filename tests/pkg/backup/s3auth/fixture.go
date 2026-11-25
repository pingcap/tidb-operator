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

const irsaAccessKeyPrefix = "irsa-access-key-"

// Request records identifiers, never secret keys or real tokens.
type Request struct {
	Service, Method, Host, AccessKey, RoleARN, ExternalID, Region string
	Action, WebIdentityToken                                      string
}

// Fixture accepts S3 access only with STS-issued credentials unless AllowSource is set.
// Install must not be used from parallel tests because the AWS SDK reads http.DefaultClient.
type Fixture struct {
	mu                                                   sync.Mutex
	Objects                                              map[string][]byte
	Requests                                             []Request
	AllowSource, DenyRole, DenyAssumeRole                bool
	ExpectedExternalID                                   string
	ExpectedSourceKey, ExpectedSourceToken               string
	ExpectedWebIdentityRoleARN, ExpectedWebIdentityToken string
	DenyWebIdentity                                      bool
	WebIdentityExpiry                                    time.Duration
	webIdentitySeq                                       int
	webIdentityKey, webIdentitySessionToken              string
}

// Install redirects every AWS request to this transport; unexpected hosts fail locally.
func Install(t testing.TB) *Fixture {
	t.Helper()
	for name, value := range map[string]string{
		"AWS_ACCESS_KEY_ID": SourceKey, "AWS_SECRET_ACCESS_KEY": "source-secret-key",
		"AWS_SESSION_TOKEN": SourceToken, "AWS_REGION": "us-east-1", "AWS_DEFAULT_REGION": "",
		"AWS_SHARED_CREDENTIALS_FILE": filepath.Join(t.TempDir(), "missing"),
		"AWS_CONFIG_FILE":             filepath.Join(t.TempDir(), "missing"),
		"AWS_SDK_LOAD_CONFIG":         "0", "AWS_PROFILE": "", "AWS_DEFAULT_PROFILE": "",
		"AWS_ROLE_ARN": "", "AWS_WEB_IDENTITY_TOKEN_FILE": "",
		"AWS_EC2_METADATA_DISABLED": "true", "AWS_STS_REGIONAL_ENDPOINTS": "",
	} {
		t.Setenv(name, value)
	}
	f := &Fixture{Objects: make(map[string][]byte), ExpectedSourceKey: SourceKey, ExpectedSourceToken: SourceToken}
	previous := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: f}
	t.Cleanup(func() { http.DefaultClient = previous })
	return f
}

// RoundTrip implements STS AssumeRole and AssumeRoleWithWebIdentity, and S3 Head/Get/List/Delete without network I/O.
func (f *Fixture) RoundTrip(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	authorization := req.Header.Get("Authorization")
	sessionToken := req.Header.Get("X-Amz-Security-Token")
	accessKey, region := "", ""
	if _, credential, ok := strings.Cut(authorization, "Credential="); ok {
		parts := strings.Split(credential, "/")
		accessKey = parts[0]
		if len(parts) > 2 {
			region = parts[2]
		}
	}
	record := Request{Method: req.Method, Host: req.URL.Host, AccessKey: accessKey, Region: region}
	irsaMode := f.ExpectedWebIdentityRoleARN != "" || f.webIdentitySeq > 0
	if strings.HasPrefix(req.URL.Host, "sts.") {
		record.Service = "sts"
		if err := req.ParseForm(); err != nil {
			return nil, err
		}
		record.RoleARN, record.ExternalID = req.Form.Get("RoleArn"), req.Form.Get("ExternalId")
		record.Action = req.Form.Get("Action")
		if record.Action == "AssumeRoleWithWebIdentity" {
			return f.assumeRoleWithWebIdentity(req, record, authorization)
		}
		f.Requests = append(f.Requests, record)
		if record.Action != "AssumeRole" {
			return nil, fmt.Errorf("unexpected STS action")
		}
		key, token := f.ExpectedSourceKey, f.ExpectedSourceToken
		// Default sources in IRSA mode and any explicit IRSA keys must use the latest session.
		if strings.HasPrefix(key, irsaAccessKeyPrefix) || (irsaMode && (key == "" || key == SourceKey)) {
			key, token = f.webIdentityKey, f.webIdentitySessionToken
		}
		if key == "" || accessKey != key || sessionToken != token {
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
	allowedSource := accessKey != "" && accessKey == f.ExpectedSourceKey
	if irsaMode {
		allowedSource = accessKey != "" && accessKey == f.webIdentityKey && sessionToken == f.webIdentitySessionToken
	}
	authorized := accessKey == RoleKey && !f.DenyRole || allowedSource && f.AllowSource
	if !authorized {
		return response(req, 403, `<Error><Code>AccessDenied</Code><Message>storage identity denied</Message></Error>`), nil
	}
	if accessKey == RoleKey && sessionToken != "role-session-token" {
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

func (f *Fixture) assumeRoleWithWebIdentity(req *http.Request, record Request, authorization string) (*http.Response, error) {
	token := req.Form.Get("WebIdentityToken")
	switch token {
	case "irsa-jwt-1", "irsa-jwt-2":
		record.WebIdentityToken = token
	}
	f.Requests = append(f.Requests, record)
	if authorization != "" {
		return nil, fmt.Errorf("STS web identity request was signed")
	}
	if f.DenyWebIdentity || (f.ExpectedWebIdentityRoleARN != "" && record.RoleARN != f.ExpectedWebIdentityRoleARN) ||
		token == "" || (f.ExpectedWebIdentityToken != "" && token != f.ExpectedWebIdentityToken) {
		return response(req, 403, `<ErrorResponse><Error><Code>AccessDenied</Code><Message>web identity denied</Message></Error></ErrorResponse>`), nil
	}
	expiry := f.WebIdentityExpiry
	if expiry == 0 {
		expiry = time.Hour
	}
	f.webIdentitySeq++
	f.webIdentityKey = irsaAccessKeyPrefix + strconv.Itoa(f.webIdentitySeq)
	f.webIdentitySessionToken = fmt.Sprintf("irsa-session-token-%d", f.webIdentitySeq)
	body := fmt.Sprintf(`<AssumeRoleWithWebIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
	<AssumeRoleWithWebIdentityResult>
		<Credentials><AccessKeyId>%s</AccessKeyId><SecretAccessKey>irsa-secret-key</SecretAccessKey><SessionToken>%s</SessionToken><Expiration>%s</Expiration></Credentials>
		<SubjectFromWebIdentityToken>system:serviceaccount:backup:backup-manager</SubjectFromWebIdentityToken>
		<AssumedRoleUser><AssumedRoleId>id:backup-manager</AssumedRoleId><Arn>arn:aws:sts::111122223333:assumed-role/backup-manager/session</Arn></AssumedRoleUser>
	</AssumeRoleWithWebIdentityResult>
	<ResponseMetadata><RequestId>test</RequestId></ResponseMetadata>
</AssumeRoleWithWebIdentityResponse>`, f.webIdentityKey, f.webIdentitySessionToken, time.Now().Add(expiry).UTC().Format(time.RFC3339))
	return response(req, 200, body), nil
}

func response(req *http.Request, status int, body string) *http.Response {
	result := &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/xml"}, "Content-Length": {strconv.Itoa(len(body))}},
		Body: io.NopCloser(strings.NewReader(body)), ContentLength: int64(len(body)), Request: req}
	if req.Method == http.MethodHead {
		result.Body = io.NopCloser(strings.NewReader(""))
	}
	return result
}
