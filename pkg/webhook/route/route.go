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

package route

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/pingcap/tidb-operator/pkg/webhook/statefulset"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	"k8s.io/api/admission/v1beta1"
	glog "k8s.io/klog"
)

// admitFunc is the type we use for all of our validators
type admitFunc func(v1beta1.AdmissionReview) *v1beta1.AdmissionResponse

// marshal responseAdmissionReview and send back
func marshalAndWrite(response v1beta1.AdmissionReview, w http.ResponseWriter) {

	respBytes, err := json.Marshal(response)
	if err != nil {
		glog.Errorf("%v", err)
	}
	if _, err := w.Write(respBytes); err != nil {
		glog.Errorf("%v", err)
	}

}

// serve handles the http portion of a request prior to handing to an admit
// function
func serve(w http.ResponseWriter, r *http.Request, admit admitFunc) {
	var body []byte
	var contentType string
	responseAdmissionReview := v1beta1.AdmissionReview{}
	requestedAdmissionReview := v1beta1.AdmissionReview{}
	deserializer := util.GetCodec()

	// The AdmissionReview that will be returned
	if r.Body == nil {
		err := errors.New("requeset body is nil")
		responseAdmissionReview.Response = util.ARFail(err)
		marshalAndWrite(responseAdmissionReview, w)
		return
	}

	data, err := ioutil.ReadAll(r.Body)

	if err != nil {
		responseAdmissionReview.Response = util.ARFail(err)
		marshalAndWrite(responseAdmissionReview, w)
		return
	}

	body = data

	// verify the content type is accurate
	contentType = r.Header.Get("Content-Type")
	if contentType != "application/json" {
		err := errors.New("expect application/json")
		responseAdmissionReview.Response = util.ARFail(err)
		marshalAndWrite(responseAdmissionReview, w)
		return
	}

	// The AdmissionReview that was sent to the webhook
	if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
		responseAdmissionReview.Response = util.ARFail(err)
	} else {
		// pass to admitFunc
		responseAdmissionReview.Response = admit(requestedAdmissionReview)
	}

	// Return the same UID
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID

	marshalAndWrite(responseAdmissionReview, w)

}

func ServeStatefulSets(w http.ResponseWriter, r *http.Request) {
	serve(w, r, statefulset.AdmitStatefulSets)
}
