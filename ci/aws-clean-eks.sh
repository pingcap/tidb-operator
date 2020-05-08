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

function delete_security_group() {
    local sgId="$1"
    echo "info: deleting security group '$sgId'"
    for eni in $(aws ec2 describe-network-interfaces --filters "Name=group-id,Values=$sgId" --query 'NetworkInterfaces[*].NetworkInterfaceId' --output text); do
        echo "info: clear leaked network interfaces '$eni'"
        aws ec2 delete-network-interface --network-interface-id "$eni"
    done
    aws ec2 delete-security-group --group-id "$sgId"
    if [ $? -eq 0 ]; then
        echo "info: succesfully deleted security group '$sgId'"
    else
        echo "error: failed to deleted security group '$sgId'"
    fi
}

function delete_vpc() {
    local vpcId="$1"
    echo "info: deleting vpc '$vpcId'"
    for sgId in $(aws ec2 describe-security-groups --filters "Name=vpc-id,Values=$vpcId" --query "SecurityGroups[?GroupName != 'default'].GroupId" --output text); do
        delete_security_group "$sgId"
    done
    aws ec2 delete-vpc --vpc-id "$vpcId"
    if [ $? -eq 0 ]; then
        echo "info: succesfully deleted vpc '$vpcId'"
    else
        echo "error: failed to deleted vpc '$vpcId'"
    fi
}

function fix_eks_mng_deletion_issues() {
    local cluster="$1"
    local mng="$2"
    while IFS=$'\n' read -r line; do
        read -r code resourceIds <<< $line
        if [ "$code" == "Ec2SecurityGroupDeletionFailure" ]; then
            IFS=',' read -ra sgIds <<< "$resourceIds"
            for sgId in ${sgIds[@]}; do
                delete_security_group "$sgId"
            done
        fi
    done <<< $(aws eks describe-nodegroup --cluster-name "$cluster" --nodegroup-name "$mng" --query 'nodegroup.health.issues' --output json | jq -r '.[].resourceIds |= join(",") | .[] | "\(.code)\t\(.resourceIds)"')
}

function clean_eks() {
    local CLUSTER="$1"
    echo "info: deleting mng-sg/mng/cluster/cluster-role/mng-role/vpc stacks"
    local stacks=(
        $CLUSTER-mng-mng-sg
        $CLUSTER-mng
        $CLUSTER-role-mng
        $CLUSTER-cluster
        $CLUSTER-role-cluster
        $CLUSTER-vpc
    )
    for stack in ${stacks[@]}; do
        echo "info: deleting stack $stack"
        aws cloudformation delete-stack --stack-name $stack
        aws cloudformation wait stack-delete-complete --stack-name $stack
        if [ $? -ne 0 ]; then
            echo "error: failed to delete stack '$stack'"
            if [ "$stack" == "$CLUSTER-mng" ]; then
                echo "info: try to fix mng stack '$stack'"
                for mngName in $(aws eks list-nodegroups --cluster-name "$CLUSTER" --query 'nodegroups[*]' --output text); do
                    fix_eks_mng_deletion_issues "$CLUSTER" $mngName
                done
            elif [ "$stack" == "$CLUSTER-vpc" ]; then
                echo "info: try to fix vpc stack '$stack'"
                while IFS=$'\n' read -r sgId; do
                    delete_security_group "$sgId"
                done <<< $(aws cloudformation describe-stacks --stack-name "$stack" --query 'Stacks[*].Outputs[*]' --output json | jq -r '.[] | .[] | select(.OutputKey == "ControlPlaneSecurityGroupID") | .OutputValue')
                while IFS=$'\n' read -r vpcId; do
                    delete_vpc "$vpcId"
                done <<< $(aws cloudformation describe-stacks --stack-name "$stack" --query 'Stacks[*].Outputs[*]' --output json | jq -r '.[] | .[] | select(.OutputKey == "VPCID") | .OutputValue')
            else
                echo "fatal: unable to delete stack $stack"
                exit 1
            fi
            echo "info: try to delete the stack '$stack' again"
            aws cloudformation delete-stack --stack-name $stack
            aws cloudformation wait stack-delete-complete --stack-name $stack
            if [ $? -ne 0 ]; then
                echo "fatal: unable to delete stack $stack"
                exit 1
            fi
        fi
    done
    local keyPairName=$CLUSTER-key-nodes
    echo "info: deleting key pair $keyPairName"
    aws ec2 delete-key-pair --key-name $keyPairName
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
