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
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadErrorBody(t *testing.T) {
	cases := []struct {
		desc  string
		input io.Reader
		want  string
	}{
		{
			desc:  "normal",
			input: bytes.NewReader([]byte("ok")),
			want:  "ok",
		},
		{
			desc:  "timeout",
			input: iotest.TimeoutReader(bytes.NewReader([]byte("ok"))),
			want:  "timeout",
		},
	}

	for _, tt := range cases {
		t.Run(tt.desc, func(t *testing.T) {
			err := ReadErrorBody(tt.input)
			require.Error(t, err)
			assert.Equal(t, tt.want, err.Error())
		})
	}
}

func TestDoBodyOK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			// just echo the body
			data, err := io.ReadAll(r.Body)
			if err != nil {
				// different from the below 500
				w.WriteHeader(403)
				return
			}
			w.WriteHeader(200)
			_, err = w.Write(data)
			assert.NoError(t, err)
		case "/server_error":
			w.WriteHeader(500)
			return
		}
	}))

	cases := []struct {
		desc   string
		method func(context.Context, *http.Client, string) ([]byte, error)
		path   string
		want   []byte
		hasErr bool
	}{
		{
			desc:   "GetBodyOK",
			method: GetBodyOK,
			path:   "/ok",
			want:   []byte(""),
		},
		{
			desc: "PutBodyOK",
			method: func(ctx context.Context, c *http.Client, s string) ([]byte, error) {
				return PutBodyOK(ctx, c, s, bytes.NewReader([]byte("ok")))
			},
			path: "/ok",
			want: []byte("ok"),
		},
		{
			desc:   "DeleteBodyOK",
			method: DeleteBodyOK,
			path:   "/ok",
			want:   []byte(""),
		},
		{
			desc: "PostBodyOK",
			method: func(ctx context.Context, cli *http.Client, url string) ([]byte, error) {
				return PostBodyOK(ctx, cli, url, bytes.NewReader([]byte("ok")))
			},
			path: "/ok",
			want: []byte("ok"),
		},
		{
			desc:   "error status code",
			method: GetBodyOK,
			path:   "/server_error",
			want:   nil,
			hasErr: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.desc, func(t *testing.T) {
			cli := ts.Client()
			ctx := context.Background()
			data, err := tt.method(ctx, cli, ts.URL+tt.path)
			if tt.hasErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, data)
		})
	}
}
