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

// copyright 2019 pingcap, inc.
//
// licensed under the apache license, version 2.0 (the "license");
// you may not use this file except in compliance with the license.
// you may obtain a copy of the license at
//
//     http://www.apache.org/licenses/license-2.0
//
// unless required by applicable law or agreed to in writing, software
// distributed under the license is distributed on an "as is" basis,
// see the license for the specific language governing permissions and
// limitations under the license.

package api

import (
	"encoding/json"
	"net/http"

	"github.com/pingcap/errors"
	"k8s.io/klog"
)

// Response defines a new response struct for http
type Response struct {
	Action     string      `json:"action"`
	StatusCode int         `json:"status_code"`
	Message    string      `json:"message,omitempty"`
	Payload    interface{} `json:"payload,omitempty"`
}

func newResponse(action string) *Response {
	return &Response{Action: action, StatusCode: http.StatusOK}
}

func (r *Response) statusCode(code int) *Response {
	r.StatusCode = code
	return r
}

func (r *Response) message(msg string) *Response {
	r.Message = msg
	return r
}

func (r *Response) payload(payload interface{}) *Response {
	r.Payload = payload
	return r
}

// ExtractResponse extract response from api
func ExtractResponse(data []byte) ([]byte, error) {
	respData := &Response{}
	if err := json.Unmarshal(data, respData); err != nil {
		return nil, errors.AddStack(err)
	}

	if respData.StatusCode != http.StatusOK {
		d, err := json.Marshal(respData.Payload)
		if err != nil {
			klog.Errorf("marshal data failed %v", d)
		}

		return d, errors.New(respData.Message)
	}

	return json.Marshal(respData.Payload)
}
