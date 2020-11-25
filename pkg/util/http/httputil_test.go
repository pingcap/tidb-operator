// Copyright 2020 PingCAP, Inc.
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

package httputil

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/iotest"

	. "github.com/onsi/gomega"
)

func TestReadErrorBody(t *testing.T) {
	g := NewGomegaWithT(t)
	var reader io.Reader
	var err error

	// test read body
	reader = bytes.NewReader([]byte("ok"))
	err = ReadErrorBody(reader)
	g.Expect(err.Error()).Should(Equal("ok"))

	// test failed to read all from reader
	reader = iotest.TimeoutReader(bytes.NewReader([]byte("ok")))
	err = ReadErrorBody(reader)
	g.Expect(err).Should(Equal(iotest.ErrTimeout))
}

func TestDoBodyOK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			// just echo the body
			data, err := ioutil.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(403)
			}
			_, err = w.Write(data)
			if err != nil {
				w.WriteHeader(403)
			}
			w.WriteHeader(200)
		case "/server_error":
			w.WriteHeader(500)
			return
		}
	}))

	g := NewGomegaWithT(t)
	cli := ts.Client()

	// test normal
	reqBody := bytes.NewReader([]byte("ok"))
	data, err := DoBodyOK(cli, ts.URL+"/ok", "GET", reqBody)
	g.Expect(err).Should(BeNil())
	g.Expect(data).Should(Equal([]byte("ok")))

	// test error status code
	_, err = DoBodyOK(cli, ts.URL+"/server_error", "GET", reqBody)
	g.Expect(err).ShouldNot(BeNil())

	// test GetBodyOK
	data, err = GetBodyOK(cli, ts.URL+"/ok")
	g.Expect(err).Should(BeNil())
	g.Expect(data).Should(Equal([]byte("")))

	// test PutBodyOK
	data, err = PutBodyOK(cli, ts.URL+"/ok")
	g.Expect(err).Should(BeNil())
	g.Expect(data).Should(Equal([]byte("")))

	// test DeleteBodyOK
	data, err = DeleteBodyOK(cli, ts.URL+"/ok")
	g.Expect(err).Should(BeNil())
	g.Expect(data).Should(Equal([]byte("")))

	// test PostBodyOK
	data, err = PostBodyOK(cli, ts.URL+"/ok", bytes.NewReader([]byte("ok")))
	g.Expect(err).Should(BeNil())
	g.Expect(data).Should(Equal([]byte("ok")))
}
