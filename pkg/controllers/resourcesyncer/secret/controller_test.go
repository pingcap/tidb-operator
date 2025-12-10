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

package secret

import (
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
)

type fileData struct {
	isDir bool
	data  []byte
}

func TestReconcile(t *testing.T) {
	cases := []struct {
		desc   string
		labels map[string]string
		files  map[string]*fileData
		objs   []client.Object

		expectedFiles map[string]*fileData

		hasErr bool
	}{
		{
			desc: "empty dir and no secret",
			labels: map[string]string{
				"test": "test",
			},

			objs: []client.Object{},

			expectedFiles: map[string]*fileData{},
		},
		{
			desc: "empty dir",
			labels: map[string]string{
				"test": "test",
			},

			objs: []client.Object{
				fake.FakeObj(
					"aaa",
					fake.ResourceVersion[corev1.Secret]("test"),
					fake.Label[corev1.Secret]("test", "test"),
					secretData("file", []byte(`data`)),
				),
			},

			expectedFiles: map[string]*fileData{
				"aaa": {
					isDir: true,
				},
				"aaa/file": {
					data: []byte(`data`),
				},
				"aaa/resourceVersion": {
					data: []byte(`test`),
				},
			},
		},
		{
			desc: "empty dir with two secrets",
			labels: map[string]string{
				"test": "test",
			},

			objs: []client.Object{
				fake.FakeObj(
					"aaa",
					fake.ResourceVersion[corev1.Secret]("test"),
					fake.Label[corev1.Secret]("test", "test"),
					secretData("file", []byte(`data`)),
				),
				fake.FakeObj(
					"bbb",
					fake.ResourceVersion[corev1.Secret]("testxxx"),
					fake.Label[corev1.Secret]("test", "test"),
					secretData("file", []byte(`dataxxx`)),
				),
			},

			expectedFiles: map[string]*fileData{
				"aaa": {
					isDir: true,
				},
				"aaa/file": {
					data: []byte(`data`),
				},
				"aaa/resourceVersion": {
					data: []byte(`test`),
				},
				"bbb": {
					isDir: true,
				},
				"bbb/file": {
					data: []byte(`dataxxx`),
				},
				"bbb/resourceVersion": {
					data: []byte(`testxxx`),
				},
			},
		},
		{
			desc: "secret file inplace update",
			labels: map[string]string{
				"test": "test",
			},
			files: map[string]*fileData{
				"aaa": {
					isDir: true,
				},
				"aaa/file": {
					data: []byte(`data`),
				},
				"aaa/resourceVersion": {
					data: []byte(`test`),
				},
			},

			objs: []client.Object{
				fake.FakeObj(
					"aaa",
					fake.ResourceVersion[corev1.Secret]("testyyy"),
					fake.Label[corev1.Secret]("test", "test"),
					secretData("file", []byte(`datayyy`)),
				),
			},

			expectedFiles: map[string]*fileData{
				"aaa": {
					isDir: true,
				},
				"aaa/file": {
					data: []byte(`datayyy`),
				},
				"aaa/resourceVersion": {
					data: []byte(`testyyy`),
				},
			},
		},
		{
			desc: "add/del secret file",
			labels: map[string]string{
				"test": "test",
			},
			files: map[string]*fileData{
				"aaa": {
					isDir: true,
				},
				"aaa/file": {
					data: []byte(`data`),
				},
				"aaa/resourceVersion": {
					data: []byte(`test`),
				},
			},

			objs: []client.Object{
				fake.FakeObj(
					"aaa",
					fake.ResourceVersion[corev1.Secret]("testyyy"),
					fake.Label[corev1.Secret]("test", "test"),
					secretData("fileyyy", []byte(`datayyy`)),
				),
			},

			expectedFiles: map[string]*fileData{
				"aaa": {
					isDir: true,
				},
				"aaa/fileyyy": {
					data: []byte(`datayyy`),
				},
				"aaa/resourceVersion": {
					data: []byte(`testyyy`),
				},
			},
		},
		{
			desc: "skip update if secret rv is not changed",
			labels: map[string]string{
				"test": "test",
			},
			files: map[string]*fileData{
				"aaa": {
					isDir: true,
				},
				"aaa/file": {
					data: []byte(`data`),
				},
				"aaa/resourceVersion": {
					data: []byte(`test`),
				},
			},

			objs: []client.Object{
				fake.FakeObj(
					"aaa",
					fake.ResourceVersion[corev1.Secret]("test"),
					fake.Label[corev1.Secret]("test", "test"),
					// change data to test whether update is actually skipped
					// not really happened in real world
					secretData("fileyyy", []byte(`datayyy`)),
				),
			},

			expectedFiles: map[string]*fileData{
				"aaa": {
					isDir: true,
				},
				"aaa/file": {
					data: []byte(`data`),
				},
				"aaa/resourceVersion": {
					data: []byte(`test`),
				},
			},
		},
		{
			desc: "delete secret",
			labels: map[string]string{
				"test": "test",
			},
			files: map[string]*fileData{
				"aaa": {
					isDir: true,
				},
				"aaa/file": {
					data: []byte(`data`),
				},
				"aaa/resourceVersion": {
					data: []byte(`test`),
				},
			},

			objs: []client.Object{},

			expectedFiles: map[string]*fileData{},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			fc := client.NewFakeClient(c.objs...)
			dirFs := setupFs(tt, c.files)
			r := Reconciler{
				Client:    fc,
				Labels:    c.labels,
				BaseDirFS: dirFs,
				cache:     map[string]string{},
			}

			_, err := r.Reconcile(t.Context(), ctrl.Request{})
			if !c.hasErr {
				require.NoError(tt, err)
			}

			actual := map[string]*fileData{}
			err = afero.Walk(dirFs, "", func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if path == "" {
					return nil
				}

				if info.IsDir() {
					actual[path] = &fileData{
						isDir: true,
					}
					return nil
				}
				data, err := afero.ReadFile(dirFs, path)
				require.NoError(tt, err, path)

				actual[path] = &fileData{
					data: data,
				}
				return nil
			})
			require.NoError(tt, err)

			assert.Equal(tt, c.expectedFiles, actual)
		})
	}
}

func setupFs(t *testing.T, files map[string]*fileData) afero.Fs {
	f := afero.NewMemMapFs()
	for path, fileData := range files {
		if fileData.isDir {
			err := f.MkdirAll(path, 0o755)
			require.NoError(t, err)
		}

		dir := filepath.Dir(path)
		err := f.MkdirAll(dir, 0o755)
		require.NoError(t, err)
		err = afero.WriteFile(f, path, fileData.data, 0o644)
		assert.NoError(t, err)
	}

	return f
}

func secretData(key string, val []byte) func(s *corev1.Secret) *corev1.Secret {
	return func(s *corev1.Secret) *corev1.Secret {
		if s.Data == nil {
			s.Data = map[string][]byte{}
		}
		s.Data[key] = val

		return s
	}
}
