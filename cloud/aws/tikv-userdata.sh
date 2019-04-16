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

# format and mount nvme disk
if grep nvme0n1 /etc/fstab; then
    echo "disk already mounted"
else
    mkfs -t ext4 /dev/nvme0n1
    mkdir -p /mnt/disks/ssd1
    cat <<EOF >> /etc/fstab
/dev/nvme0n1 /mnt/disks/ssd1 ext4 defaults,nofail,noatime,nodelalloc 0 2
EOF
    mount -a
fi
