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

package node

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/ssh"
)

var (
	awsNodeInitCmd = `
sudo bash -c '
test -d /mnt/disks || mkdir -p /mnt/disks
df -h /mnt/disks
if mountpoint /mnt/disks &>/dev/null; then
    echo "info: /mnt/disks is a mountpoint"
else
    echo "info: /mnt/disks is not a mountpoint, creating local volumes on the rootfs"
fi
cd /mnt/disks
for ((i = 1; i <= 32; i++)) {
    if [ ! -d vol$i ]; then
        mkdir vol$i
    fi
    if ! mountpoint vol$i &>/dev/null; then
        mount --bind vol$i vol$i
    fi
}
echo "info: increase max open files for containers"
if ! grep -qF "OPTIONS" /etc/sysconfig/docker; then
    echo 'OPTIONS="--default-ulimit nofile=1024000:1024000"' >> /etc/sysconfig/docker
fi
systemctl restart docker
'
`
)

func InitNode(node *v1.Node) error {
	var initNodeCmd string
	if framework.TestContext.Provider == "aws" {
		initNodeCmd = awsNodeInitCmd
	} else {
		return nil
	}
	return ssh.IssueSSHCommand(initNodeCmd, framework.TestContext.Provider, node)
}
