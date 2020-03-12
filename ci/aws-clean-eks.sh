#!/bin/bash

# Copyright 2020 PingCAP, Inc.
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

#
# aws-k8s-tester cannot clean all resources created when some error happened.
# This script is used to clean resources created by aws-k8s-tester in our CI.
#
# DO NOT USE THIS SCRIPT FOR OTHER USES!
#

function get_stacks() {
    aws cloudformation list-stacks --stack-status-filter CREATE_COMPLETE DELETE_FAILED --query 'StackSummaries[*].StackName' --output text
}

function fix_eks_mng_deletion_issues() {
    local cluster="$1"
    local mng="$2"
    while IFS=$'\n' read -r line; do
        read -r code resourceIds <<< $line
        if [ "$code" == "Ec2SecurityGroupDeletionFailure" ]; then
            echo "info: clear security group '$resourceIds'"
            for eni in $(aws ec2 describe-network-interfaces --filters "Name=group-id,Values=$resourceIds" --query 'NetworkInterfaces[*].NetworkInterfaceId' --output text); do
                echo "info: clear leaked network interfaces '$eni'"
                aws ec2 delete-network-interface --network-interface-id "$eni"
            done
            aws ec2 delete-security-group --group-id $resourceIds
        fi
    done <<< $(aws eks describe-nodegroup --cluster-name "$cluster" --nodegroup-name "$mng" --query 'nodegroup.health.issues' --output json | jq -r '.[].resourceIds |= join(",") | .[] | "\(.code)\t\(.resourceIds)"')
}

function clean_eks() {
    local CLUSTER="$1"
    echo "info: deleting mng stack"
    local regex='^'$CLUSTER'-mng-[0-9]+$'
    local mngStack=
    for stackName in $(get_stacks); do
        if [[ ! "$stackName" =~ $regex ]]; then
            continue
        fi
        mngStack=$stackName
        break
    done
    if [ -n "$mngStack" ]; then
        echo "info: mng stack found '$mngStack', deleting it"
        aws cloudformation delete-stack --stack-name $mngStack
        aws cloudformation wait stack-delete-complete --stack-name $mngStack
        if [ $? -ne 0 ]; then
            echo "error: failed to delete mng stack '$mngStack', delete related resource first"
            for mngName in $(aws eks list-nodegroups --cluster-name jenkins-tidb-operator-e2e2 --query 'nodegroups[*]' --output text); do
                fix_eks_mng_deletion_issues "$CLUSTER" $mngName
            done
            aws cloudformation delete-stack --stack-name $mngStack
            aws cloudformation wait stack-delete-complete --stack-name $mngStack
        fi
    else
        echo "info: mng stack not found, skipped"
    fi

    echo "info: deleting cluster/cluster-role/mng-role/vpc stacks"
    local stacks=(
        $CLUSTER-cluster    
        $CLUSTER-role-cluster
        $CLUSTER-role-mng
        $CLUSTER-vpc
    )
    for stack in ${stacks[@]}; do
        echo "info: deleting stack $stack"
        aws cloudformation delete-stack --stack-name $stack
        aws cloudformation wait stack-delete-complete --stack-name $stack
    done
}

# https://github.com/aws/aws-cli#other-configurable-variables
if [ -n "${AWS_REGION}" ]; then
    export AWS_DEFAULT_REGION=${AWS_REGION:-}
fi

aws sts get-caller-identity
if [ $? -ne 0 ]; then
    echo "error: failed to get caller identity"
    exit 1
fi

for CLUSTER in $@; do
    echo "info: start to clean eks test cluster '$CLUSTER'"
    clean_eks "$CLUSTER"
    if [ $? -eq 0 ]; then
        echo "info: succesfully cleaned the eks test cluster '$CLUSTER'"
    else
        echo "fatal: failed to clean the eks test cluster '$CLUSTER'"
        exit 1
    fi
done
