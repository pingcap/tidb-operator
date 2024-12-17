// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httputil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// DeferClose captures and prints the error from closing (if an error occurs).
// This is designed to be used in a defer statement.
func DeferClose(c io.Closer) {
	if err := c.Close(); err != nil {
		fmt.Printf("Error closing: %v\n", err)
	}
}

// ReadErrorBody in the error case ready the body message.
// But return it as an error (or return an error from reading the body).
func ReadErrorBody(body io.Reader) (err error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	return errors.New(string(bodyBytes))
}

// GetBodyOK returns the body or an error if the response is not okay
func GetBodyOK(ctx context.Context, httpClient *http.Client, apiURL string) ([]byte, error) {
	return DoBodyOK(ctx, httpClient, apiURL, "GET", nil)
}

// PutBodyOK will PUT and returns the body or an error if the response is not okay
func PutBodyOK(ctx context.Context, httpClient *http.Client, apiURL string) ([]byte, error) {
	return DoBodyOK(ctx, httpClient, apiURL, "PUT", nil)
}

// DeleteBodyOK will DELETE and returns the body or an error if the response is not okay
func DeleteBodyOK(ctx context.Context, httpClient *http.Client, apiURL string) ([]byte, error) {
	return DoBodyOK(ctx, httpClient, apiURL, "DELETE", nil)
}

// PostBodyOK will POST and returns the body or an error if the response is not okay
func PostBodyOK(ctx context.Context, httpClient *http.Client, apiURL string, reqBody io.Reader) ([]byte, error) {
	return DoBodyOK(ctx, httpClient, apiURL, "POST", reqBody)
}

// DoBodyOK returns the body or an error if the response is not okay(StatusCode >= 400)
func DoBodyOK(ctx context.Context, httpClient *http.Client, apiURL, method string, reqBody io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, apiURL, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer DeferClose(res.Body)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= http.StatusBadRequest {
		errMsg := fmt.Errorf("error response %v URL %s,body response: %s", res.StatusCode, apiURL, string(body))
		return nil, errMsg
	}
	return body, err
}
