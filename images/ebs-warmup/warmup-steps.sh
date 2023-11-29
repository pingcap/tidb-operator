#! /usr/bin/env bash

# Copyright 2023 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# See the License for the specific language governing permissions and
# limitations under the License.

set -eu

dev_name_by_mount_point() {
    lsblk -no NAME,MOUNTPOINT -r | awk -v path="$1" '$2 == path {print $1;}'
}

die() {
    echo "$@" >/dev/stderr
    help
    exit 1
}

help() {
    cat <<EOF
Supported flags:
    --block <files_to_be_warmed_up_by_fio>
    --fs <files_to_be_warmed_up_by_filesystem>
    --debug enable \`set -x\`
EOF
}

operation=none
fs_extra=( )
while [ $# -gt 0 ]; do
    case $1 in
        --help | -h) 
            help
            exit 0
            ;;
        --block) operation=fio
            ;;
        --fs) operation=fs
            ;;
        --fs-override-arg) 
            fs_extra=()
            operation=fs_override_arg
            ;;
        --debug) set -x
            ;;
        -*)
            if [ "$operation" != fs_override_arg ]; then
                die "unsupported flag $1"
            fi
            ;&
        *)
            case "$operation" in
                fs_override_arg) 
                    fs_extra+=("$1")
                    ;;
                fio) 
                    echo "spawning wram up task: operation = $operation; file path = $1"
                    device=$(dev_name_by_mount_point "$1")
                    if [ -z "$device" ]; then
                        echo "$1 isn't a mount point, skipping."
                    else 
                        fio --rw=read --bs=256K --iodepth=128 --ioengine=libaio \
                            --numjobs=10 --offset=0% --offset_increment=10% --size=10% \
                            "--name=initialize-$device" \
                            --thread=1 --filename=/dev/"$device" &
                    fi
                    ;;
                fs) 
                    echo "spawning wram up task: operation = $operation; file path = $1, fs_extra = ${fs_extra[*]}"
                    if (( ${#fs_extra[@]} > 0 )); then
                        /warmup "${fs_extra[@]}" --files="$1" -P256 --direct &
                    else
                        /warmup --type=whole --files="$1" -P256 --direct &
                    fi
                    ;;
                *) die "internal error: unsupported operation $1; forgot to call --block or --fs?"
                    ;;
            esac
            ;;
    esac
    shift
done

wait
