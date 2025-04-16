// Copyright 2025 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.
package backup

import (
	"encoding/json"
	"fmt"
	"mime"
	"time"

	"github.com/juju/errors"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
)

// RFC3339Time is a wrapper of `time.Time` that marshals to a RFC3339
// JSON string when encoding to / decoding from json.
type RFC3339Time time.Time

const (
	SeverityError  = "ERROR"
	SeverityManual = "MANUAL"
)

type StreamBackupError struct {
	// the unix epoch time (in millisecs) of the time the error reported.
	HappenAt uint64 `protobuf:"varint,1,opt,name=happen_at,json=happenAt,proto3" json:"happen_at,omitempty"`
	// the unified error code of the error.
	ErrorCode string `protobuf:"bytes,2,opt,name=error_code,json=errorCode,proto3" json:"error_code,omitempty"`
	// the user-friendly error message.
	ErrorMessage string `protobuf:"bytes,3,opt,name=error_message,json=errorMessage,proto3" json:"error_message,omitempty"`
	// the store id of who issues the error.
	StoreId uint64 `protobuf:"varint,4,opt,name=store_id,json=storeId,proto3" json:"store_id,omitempty"`
}

// PauseV2Info is extra information attached to a paused task.
type PauseV2Info struct {
	// Severity is the severity of this pause.
	// `SeverityError`: The task encounters some fatal errors and have to be paused.
	// `SeverityManual`: The task was paused by a normal operation.
	Severity string `json:"severity"`
	// OperatorHostName is the hostname that pauses the task.
	OperatorHostName string `json:"operation_hostname"`
	// OperatorPID is the pid of the operator process.
	OperatorPID int `json:"operation_pid"`
	// OperationTime is the time when the task was paused.
	OperationTime time.Time `json:"operation_time"`
	// PayloadType is the mime type of the payload.
	// For now, only the two types are supported:
	// - application/x-protobuf?messagetype=brpb.StreamBackupError
	// - text/plain
	PayloadType string `json:"payload_type"`
	// Payload is the payload attached to the pause.
	Payload []byte `json:"payload"`
}

func NewPauseV2Info(kv *pdapi.KeyValue) (*PauseV2Info, error) {
	rawPauseV2 := kv.Value
	if len(rawPauseV2) == 0 {
		return nil, errors.Errorf("pause payload is empty in %s", kv.Key)
	}
	var pauseV2 PauseV2Info
	err := json.Unmarshal(rawPauseV2, &pauseV2)
	if err != nil {
		return nil, errors.Annotatef(err, "failed to unmarshal pause payload for kv pair %s", kv.Key)
	}
	return &pauseV2, nil
}

func (p *PauseV2Info) ParseError() (string, error) {
	m, param, err := mime.ParseMediaType(p.PayloadType)
	if err != nil {
		return "", errors.Annotatef(err, "%s isn't a valid mime type", p.PayloadType)
	}

	switch m {
	case "text/plain":
		// Note: consider the charset?
		return string(p.Payload), nil
	case "application/x-protobuf":
		msgType, ok := param["messagetype"]
		if !ok {
			return "", errors.Errorf("x-protobuf didn't specified msgType (%s)", p.PayloadType)
		}
		if msgType != "brpb.StreamBackupError" {
			return "", errors.Errorf("only type brpb.StreamBackupError is supported (%s)", p.PayloadType)
		}
		var msg StreamBackupError
		err := json.Unmarshal(p.Payload, &msg)
		if err != nil {
			return "", errors.Annotatef(err, "failed to unmarshal the payload")
		}
		return fmt.Sprintf("Paused by error(store %d): %s", msg.StoreId, msg.ErrorMessage), nil
	default:
		return "", errors.Errorf("unsupported payload type %s", m)
	}
}
