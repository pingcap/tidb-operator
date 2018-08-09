// Copyright 2016 PingCAP, Inc.
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

package api

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/juju/errors"
	"github.com/pingcap/pd/server"
)

func readJSON(r io.ReadCloser, data interface{}) error {
	defer r.Close()

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return errors.Trace(err)
	}
	err = json.Unmarshal(b, data)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func postJSON(url string, data []byte) error {
	resp, err := server.DialClient.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return errors.Trace(err)
	}
	res, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(string(res))
	}
	return nil
}

func doDelete(url string) error {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	res, err := server.DialClient.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()
	return nil
}

func doGet(url string) (*http.Response, error) {
	resp, err := server.DialClient.Get(url)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("http get url %s return code %d", url, resp.StatusCode)
	}
	return resp, nil
}
