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

package version

import (
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"
)

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	gitTreeState = "clean"
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			gitCommit = setting.Value
		case "vcs.time":
			buildDate = setting.Value
		case "vcs.modified":
			gitTreeState = "dirty"
		}
	}
	gitVersion = info.Main.Version
}

var (
	gitVersion   = "v0.0.0-master+$Format:%h$"
	gitCommit    = "$Format:%H$" // sha1 from git, output of $(git rev-parse HEAD)
	gitTreeState = ""            // state of git tree, either "clean" or "dirty"

	buildDate = "1970-01-01T00:00:00Z" // build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
)

// PrintVersionInfo show version info to Stdout.
func PrintVersionInfo() {
	fmt.Printf("TiDB Operator Version: %#v\n", Get())
}

// Info contains versioning information.
type Info struct {
	GitVersion   string `json:"gitVersion"`
	GitCommit    string `json:"gitCommit"`
	GitTreeState string `json:"gitTreeState"`
	BuildDate    string `json:"buildDate"`
	GoVersion    string `json:"goVersion"`
	Compiler     string `json:"compiler"`
	Platform     string `json:"platform"`
}

// String returns info as a JSON string.
func (info Info) String() string {
	jsonBytes, err := json.Marshal(info)
	if err != nil {
		return fmt.Sprintf("Version info marshal failed: %v", err)
	}
	return string(jsonBytes)
}

// KeysAndValues returns the keys and values of the version info.
// This is useful for some logging formats.
func (info *Info) KeysAndValues() []any {
	return []any{
		"gitVersion", info.GitVersion,
		"gitCommit", info.GitCommit,
		"gitTreeState", info.GitTreeState,
		"buildDate", info.BuildDate,
		"goVersion", info.GoVersion,
		"compiler", info.Compiler,
		"platform", info.Platform,
	}
}

// Get returns the overall codebase version. It's for detecting
// what code a binary was built from.
func Get() *Info {
	// these variables typically come from -ldflags settings in the Makefile
	return &Info{
		GitVersion:   gitVersion,
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		BuildDate:    buildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
