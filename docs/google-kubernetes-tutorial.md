---
title: Deploy TiDB, a distributed MySQL compatible database, to Kubernetes on Google Cloud
summary: Tutorial for deploying TiDB on Google Cloud using Kubernetes.
category: operations
---

# Deploy TiDB, a distributed MySQL compatible database, to Kubernetes on Google Cloud

## Introduction

This tutorial is designed to be [run in Google Cloud Shell](https://console.cloud.google.com/cloudshell/open?git_repo=https://github.com/pingcap/tidb-operator&tutorial=docs/google-kubernetes-tutorial.md). It takes you through these steps:

- Launch a new 3-node Kubernetes cluster (optional)
- Install the Helm package manager for Kubernetes
- Deploy the TiDB Operator
- Deploy your first TiDB cluster
- Connect to the TiDB cluster
- Scale out the TiDB cluster
- Shut down the Kubernetes cluster (optional)

## Select a project

This tutorial launches a 3-node Kubernetes cluster of `n1-standard-1` machines. Pricing information can be [found here](https://cloud.google.com/compute/pricing).

Please select a project before proceeding:

<walkthrough-project-billing-setup key="project-id">
</walkthrough-project-billing-setup>

## Enable API access

This tutorial will require use of the Compute and Container APIs. Please enable them before proceeding:

<walkthrough-enable-apis apis="container.googleapis.com,compute.googleapis.com">
</walkthrough-enable-apis>

## Configure gcloud defaults

This step defaults gcloud to your preferred project and [zone](https://cloud.google.com/compute/docs/regions-zones/), which will simplify the commands used for the rest of this tutorial:

	gcloud config set project {{project-id}} &&
	gcloud config set compute/zone us-west1-a

## Launch a 3-node Kubernetes cluster

It's now time to launch a 3-node kubernetes cluster! The following command launches a 3-node cluster of `n1-standard-1` machines.

It will take a few minutes to complete:

	gcloud container clusters create tidb

Once the cluster has launched, set it to be the default:

	gcloud config set container/cluster tidb

The last step is to verify that `kubectl` can connect to the cluster, and all three machines are running:

	kubectl get nodes

If you see `Ready` for all nodes, congratulations! You've setup your first Kubernetes cluster.

## Install Helm

Helm is the package manager for Kubernetes, and is what allows us to install all of the distributed components of TiDB in a single step. Helm requires both a server-side and a client-side component to be installed.

Install `helm`:

	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get | bash

Copy `helm` to your `$HOME` directory so that it will persist after the Cloud Shell reaches its idle timeout:

	mkdir -p ~/bin &&
	cp /usr/local/bin/helm ~/bin &&
	echo 'PATH="$PATH:$HOME/bin"' >> ~/.bashrc

Helm will also need a couple of permissions to work properly:

	kubectl apply -f ./manifests/tiller-rbac.yaml &&
	helm init --service-account tiller --upgrade

It takes a minute for helm to initialize `tiller`, its server component:

	watch "kubectl get pods --namespace kube-system | grep tiller"

When you see `Running`, it's time to hit `Control + C` and proceed to the next step!

## Deploy TiDB Operator

The first TiDB component we are going to install is the TiDB Operator, using a Helm Chart. TiDB Operator is the management system that works with Kubernetes to bootstrap your TiDB cluster and keep it running. This step assumes you are in the `tidb-operator` working directory:

	kubectl apply -f ./manifests/crd.yaml &&
	kubectl apply -f ./manifests/gke-storage.yml &&
	helm install ./charts/tidb-operator -n tidb-admin --namespace=tidb-admin

We can watch the operator come up with:

	watch kubectl get pods --namespace tidb-admin -o wide

When you see tidb-scheduler and tidb-controller-manager are `Running`, `Control + C` and proceed to launch a TiDB cluster!

## Deploy your first TiDB cluster

Now with a single command we can bring-up a full TiDB cluster:

	helm install ./charts/tidb-cluster -n demo --namespace=tidb --set pd.storageClassName=pd-ssd,tikv.storageClassName=pd-ssd

It will take a few minutes to launch. You can monitor the progress with:

	watch kubectl get pods --namespace tidb -o wide

The TiDB cluster includes 2 TiDB pods, 3 TiKV pods, and 3 PD pods. When you see all pods `Running`, it's time to `Control + C` and proceed forward!

## Connect to the TiDB cluster

There can be a small delay between the pod being up and running, and the service being available. You can watch list services available with:

	watch "kubectl get svc -n tidb"

When you see `demo-tidb` appear, you can `Control + C`. The service is ready to connect to!

To connect to TiDB within the Kubernetes cluster, you can establish a tunnel between the TiDB service and your Cloud Shell. This is recommended only for debugging purposes, because the tunnel will not automatically be transferred if your Cloud Shell restarts. To establish a tunnel:

	kubectl -n tidb port-forward svc/demo-tidb 4000:4000 &>/tmp/port-forward.log &

From your Cloud Shell:

	sudo apt-get install -y mysql-client &&
	mysql -h 127.0.0.1 -u root -P 4000

Try out a MySQL command inside your MySQL terminal:

	select tidb_version();

If you did not specify a password in helm, set one now:

	SET PASSWORD FOR 'root'@'%' = '<change-to-your-password>';

_Note that, this command contains some special characters which cannot be
auto-populated in the google cloud shell tutorial: you need to copy and paste it into your console manually._

Congratulations, you are now up and running with a distributed TiDB database compatible with MySQL!

## Scale out the TiDB cluster

With a single command we can easily scale out the TiDB cluster. To scale out TiKV:

	helm upgrade demo charts/tidb-cluster --set pd.storageClassName=pd-ssd,tikv.storageClassName=pd-ssd,tikv.replicas=5

Now the number of TiKV pods is increased from the default 3 to 5. You can check it with:

	kubectl get po -n tidb

## Destroy the TiDB cluster

When the TiDB cluster is not needed, you can delete it with the following command:

	helm delete demo --purge

The above commands only delete the running pods, the data is persistent. If you do not need the data anymore, you should run the following commands to clean the data and the dynamically created persistent disks:

	kubectl delete pvc -n tidb -l app.kubernetes.io/instance=demo,app.kubernetes.io/managed-by=tidb-operator &&
	kubectl get pv -l app.kubernetes.io/namespace=tidb,app.kubernetes.io/managed-by=tidb-operator,app.kubernetes.io/instance=demo -o name | xargs -I {} kubectl patch {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'

## Shut down the Kubernetes cluster

Once you have finished experimenting, you can delete the Kubernetes cluster with:

	gcloud container clusters delete tidb
