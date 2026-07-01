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
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tsoapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsHealthy(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		c := &client{
			url: "http://tso",
			httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, tsoHealthPrefix, r.URL.Path)
				return response(http.StatusOK, `"ok"`), nil
			})},
		}

		healthy, err := c.IsHealthy(context.Background())
		require.NoError(t, err)
		assert.True(t, healthy)
	})

	t.Run("unexpected body", func(t *testing.T) {
		c := &client{
			url: "http://tso",
			httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return response(http.StatusOK, `"not ok"`), nil
			})},
		}

		healthy, err := c.IsHealthy(context.Background())
		require.Error(t, err)
		assert.False(t, healthy)
	})

	t.Run("non 2xx", func(t *testing.T) {
		c := &client{
			url: "http://tso",
			httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return response(http.StatusServiceUnavailable, "unavailable"), nil
			})},
		}

		healthy, err := c.IsHealthy(context.Background())
		require.Error(t, err)
		assert.False(t, healthy)
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func response(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}
