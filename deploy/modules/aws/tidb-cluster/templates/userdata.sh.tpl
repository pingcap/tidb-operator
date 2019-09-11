#!/bin/bash -xe

# Allow user supplied pre userdata code
${pre_userdata}

# set ulimits
cat <<EOF > /etc/security/limits.d/99-tidb.conf
root        soft        nofile        1000000
root        hard        nofile        1000000
root        soft        core          unlimited
root        soft        stack         10240
EOF
# config docker ulimit
cp /usr/lib/systemd/system/docker.service /etc/systemd/system/docker.service
sed -i 's/LimitNOFILE=infinity/LimitNOFILE=1048576/' /etc/systemd/system/docker.service
sed -i 's/LimitNPROC=infinity/LimitNPROC=1048576/' /etc/systemd/system/docker.service
systemctl daemon-reload
systemctl restart docker

# Bootstrap and join the cluster
/etc/eks/bootstrap.sh --b64-cluster-ca "${cluster_auth_base64}" --apiserver-endpoint "${endpoint}" ${bootstrap_extra_args} --kubelet-extra-args "${kubelet_extra_args}" "${cluster_name}"

# Allow user supplied userdata code
${additional_userdata}
