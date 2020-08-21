#!/bin/sh
# set system ulimits
cat <<EOF > /etc/security/limits.d/99-tidb.conf
root        soft        nofile        1000000
root        hard        nofile        1000000
root        soft        core          unlimited
root        soft        stack         10240
EOF
# config docker ulimits
cp /usr/lib/systemd/system/docker.service /etc/systemd/system/docker.service
sed -i 's/LimitNOFILE=infinity/LimitNOFILE=1048576/' /etc/systemd/system/docker.service
sed -i 's/LimitNPROC=infinity/LimitNPROC=1048576/' /etc/systemd/system/docker.service
systemctl daemon-reload
systemctl restart docker
