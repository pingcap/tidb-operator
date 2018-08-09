// Copyright 2017 PingCAP, Inc.
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

package simutil

import (
	"github.com/pingcap/pd/pkg/logutil"
	log "github.com/sirupsen/logrus"
)

// Logger is the global logger used for simulator.
var Logger *log.Logger

// InitLogger initializes the Logger with log level.
func InitLogger(level string) {
	Logger = log.New()
	Logger.Level = logutil.StringToLogLevel(level)
}
