# Copyright 2024 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
STUB = r'''#!/usr/bin/env python3
import json, os, pathlib, sys
name = pathlib.Path(sys.argv[0]).name
args = sys.argv[1:]
with open(os.environ['CALLS'], 'a') as f:
    f.write(json.dumps([name, args]) + '\n')
if os.environ.get('FAIL') == name + ':' + args[0]:
    sys.exit(2)
if name == 'kind':
    if args[:2] == ['get', 'nodes']:
        print('\n'.join(json.loads(os.environ['NODES'])))
    elif args[:2] == ['get', 'kubeconfig']:
        if os.environ.get('FAIL') == 'kind:kubeconfig': sys.exit(2)
        print('test kubeconfig')
    elif args[:2] == ['load', 'docker-image']:
        sys.exit('ctr: content digest for another platform not found')
    elif args[:2] == ['load', 'image-archive']:
        assert pathlib.Path(args[2]).is_file()
    else:
        sys.exit('unexpected kind call')
elif name == 'kubectl':
    node = args[args.index('node') + 1]
    print(json.loads(os.environ['NODES'])[node], end='')
elif name == 'docker':
    if args[:3] == ['image', 'save', '--help']:
        if os.environ['SAVE_FLAG'] == '1': print('--platform string')
    elif args[0] == 'info':
        print(os.environ['DRIVER_STATUS'])
    elif args[0] == 'version':
        print(os.environ['SERVER_API'])
    elif args[0] == 'pull':
        pass
    elif args[:2] == ['image', 'save']:
        if os.environ.get('FAIL') == 'save': sys.exit(2)
        if 'io.containerd.snapshotter.v1' in os.environ['DRIVER_STATUS'] and not any(a.startswith('--platform') for a in args):
            sys.exit('unfiltered archive')
        pathlib.Path(args[args.index('--output') + 1]).write_text('archive')
    else:
        sys.exit('unexpected docker call')
'''


class ImagePreparationTest(unittest.TestCase):
    def run_prepare(self, nodes=None, api='1.54', containerd=True, save_flag=True,
                    fail='', forced_api=''):
        with tempfile.TemporaryDirectory(prefix='image preparation ') as directory:
            root = Path(directory)
            binaries, tmp = root / 'bin', root / 'tmp'
            binaries.mkdir()
            tmp.mkdir()
            for name in ['docker', 'kind', 'kubectl']:
                path = binaries / name
                path.write_text(STUB)
                path.chmod(0o755)
            env = os.environ.copy()
            env.update(PATH=str(binaries) + os.pathsep + env['PATH'],
                       CALLS=str(root / 'calls'), TMPDIR=str(tmp),
                       NODES=json.dumps(nodes if nodes is not None else {'node': 'linux/amd64'}),
                       SERVER_API=api, SAVE_FLAG=str(int(save_flag)), FAIL=fail,
                       DRIVER_STATUS='[["driver-type","io.containerd.snapshotter.v1"]]' if containerd else '[["Backing Filesystem","extfs"]]',
                       V_KIND='kind', V_KUBECTL='kubectl', V_KIND_CLUSTER='test',
                       V_TIDB_CLUSTER_VERSION='v8.5.2', V_TIDB_CLUSTER_VERSION_PREV='v8.5.0')
            env.pop('DOCKER_API_VERSION', None)
            if forced_api:
                env['DOCKER_API_VERSION'] = forced_api
            # A conditional caller disables implicit errexit inside shell functions.
            result = subprocess.run([shutil.which('bash'), '-c',
                                     'vars::loaded() { :; }; source "$1"; '
                                     'if image:prepare; then exit 0; else exit 1; fi',
                                     'test', str(ROOT / 'hack/lib/image.sh')],
                                    env=env, text=True, capture_output=True)
            calls = [json.loads(line) for line in (root / 'calls').read_text().splitlines()]
            self.assertEqual(list(tmp.iterdir()), [], 'temporary files leaked')
            return result, calls

    def test_containerd_archive_is_platform_scoped(self):
        result, calls = self.run_prepare()
        self.assertEqual(result.returncode, 0, result.stderr)
        saves = [args for tool, args in calls if tool == 'docker' and args[:2] == ['image', 'save'] and '--help' not in args]
        self.assertEqual(len(saves), 8)
        self.assertTrue(all('--platform=linux/amd64' in args for args in saves))
        self.assertFalse(any(args[:2] == ['load', 'docker-image'] for tool, args in calls))

    def test_target_platforms_are_grouped(self):
        result, calls = self.run_prepare({'amd': 'linux/amd64', 'arm-one': 'linux/arm64', 'arm-two': 'linux/arm64'})
        self.assertEqual(result.returncode, 0, result.stderr)
        loads = [args for tool, args in calls if tool == 'kind' and args[:2] == ['load', 'image-archive']]
        self.assertEqual(len(loads), 16)
        groups = [args[args.index('--nodes') + 1] for args in loads]
        self.assertEqual(groups.count('amd'), 8)
        self.assertEqual(groups.count('arm-one,arm-two'), 8)

    def test_legacy_classic_store(self):
        result, calls = self.run_prepare({'arm': 'linux/arm64'}, api='1.47', containerd=False)
        self.assertEqual(result.returncode, 0, result.stderr)
        saves = [args for tool, args in calls if tool == 'docker' and args[:2] == ['image', 'save'] and '--help' not in args]
        self.assertEqual(len(saves), 8)
        self.assertTrue(all(not any(a.startswith('--platform') for a in args) for args in saves))
        pulls = [args for tool, args in calls if tool == 'docker' and args[0] == 'pull']
        self.assertTrue(all(args[args.index('--platform') + 1] == 'linux/arm64' for args in pulls))

    def test_unsupported_containerd_api(self):
        for kwargs in [dict(api='1.47'), dict(save_flag=False), dict(forced_api='1.47')]:
            with self.subTest(kwargs=kwargs):
                result, calls = self.run_prepare(**kwargs)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('1.48', result.stderr)
                self.assertFalse(any(tool == 'docker' and args[0] == 'pull' for tool, args in calls))

    def test_failures_propagate_and_clean_up(self):
        for fail in ['kind:get', 'kind:kubeconfig', 'kubectl:--kubeconfig', 'docker:info', 'docker:version', 'docker:pull', 'save', 'kind:load']:
            with self.subTest(fail=fail):
                result, _ = self.run_prepare(fail=fail)
                self.assertNotEqual(result.returncode, 0)

    def test_invalid_node_platforms(self):
        for nodes in [{}, {'node': '/'}, {'node': 'windows/amd64'}]:
            with self.subTest(nodes=nodes):
                result, calls = self.run_prepare(nodes)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(any(tool == 'docker' and args[0] == 'pull' for tool, args in calls))


if __name__ == '__main__':
    unittest.main()
