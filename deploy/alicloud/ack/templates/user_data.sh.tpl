#!/bin/bash -xe

# Pre userdata code
${pre_userdata}

# Bootstrap node and join the k8s cluster
curl http://aliacs-k8s-${region}.oss-${region}-internal.aliyuncs.com/public/pkg/run/attach/1.12.6-aliyun.1/attach_node.sh | bash -s -- --ess "true" --openapi-token "${open_api_token}" %{ if node_labels != "" }--labels ${node_labels}%{ endif } %{ if node_taints != "" }--taints ${node_taints}%{ endif }

# Post userdata code
${post_userdata}
