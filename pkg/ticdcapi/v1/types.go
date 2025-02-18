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

package ticdcapi

type CaptureStatus struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	IsOwner bool   `json:"is_owner"`
}

type captureInfo struct {
	ID            string `json:"id"`
	IsOwner       bool   `json:"is_owner"`
	AdvertiseAddr string `json:"address"`
}

// drainCaptureRequest is request for manual `DrainCapture`
type drainCaptureRequest struct {
	CaptureID string `json:"capture_id"`
}

// drainCaptureResp is response for manual `DrainCapture`
type drainCaptureResp struct {
	CurrentTableCount int `json:"current_table_count"`
}
