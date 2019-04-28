#!/bin/bash -xe

# Pre userdata code
${pre_userdata}

# Bootstrap node and join the k8s cluster
curl http://aliacs-k8s-cn-hangzhou.oss-cn-hangzhou-internal.aliyuncs.com/public/pkg/run/attach/1.12.6-aliyun.1/attach_node.sh | bash -s -- --openapi-token ${open_api_token} ---labels ${node_labels} --taints ${node_taints}

# Post userdata code
${post_userdata}
