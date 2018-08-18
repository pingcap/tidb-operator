---
title: Deploy TiDB to Kubernetes on Google Cloud
summary: Tutorial for deploying TiDB on Google Cloud using Kubernetes.
category: operations
---

# Deploy TiDB, a distributed MySQL compatible database, to Kubernetes on Google Cloud

## Introduction

This tutorial takes you through these steps:

- Creating a new Google Cloud Project (optional)
- Launching a new 3 node Kubernetes cluster (optional)
- Installing the Helm package manager for Kubernetes
- Deploying TiDB to your Kubernetes cluster
- Connecting to TiDB
- Shutting down down the Kubernetes cluster (optional)

## Select a Project

Please select a project before proceeding.  The approximate cost for running the compute resources in this tutorial is 10 cents/hour.

<walkthrough-project-billing-setup key="project-id">
</walkthrough-project-billing-setup>

## Enable API Acess

This project will require use of the Compute and Container APIs.  Please enable them before proceeding:

<walkthrough-enable-apis apis="container.googleapis.com,compute.googleapis.com">
</walkthrough-enable-apis>

## Configure gcloud Defaults

This step defaults gcloud to your prefered project and [zone](https://cloud.google.com/compute/docs/regions-zones/), which will simplify the commands used for the rest of this tutorial:

	gcloud config set project {{project-id}} &&
	gcloud config set compute/zone us-west1-a

## Launch a three node Kubernetes cluster

It's now time to launch a 3 node kubernetes cluster! The following command launches a 3 node cluster of `n1-standard-1` machines.

It will take a few minutes to complete:

	gcloud container clusters create tidb

Once the cluster has launched, set it to be the default:

	gcloud config set container/cluster tidb

The last step is to verify that `kubectl` can connect to the cluster, and all three machines are running:

	kubectl get nodes

If you see `Ready` for all nodes, congratulations!  You've setup your first Kubernetes cluster.

## Installing Helm

Helm is the package manager for Kubernetes, and is what allows us to install all of the distributed components of TiDB in a single step.  Helm requires both a server-side and a client-side component to be installed.

Because Cloud Shell system packages do not persist, we also copy the `helm` command to a local directory.

Download the `helm` installer:

	curl https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get > get_helm.sh

Install `helm`:

	chmod 700 get_helm.sh && ./get_helm.sh

Copy `helm` to a local directory:

	mkdir -p ~/bin &&
	cp /usr/local/bin/helm ~/bin &&
	echo 'PATH="$PATH:$HOME/bin"' >> ~/.bashrc

Helm will also need a couple of permissions to work properly:

	kubectl create serviceaccount tiller --namespace kube-system &&
	kubectl apply -f ./manifests/tiller-rbac.yaml &&
	helm init --service-account tiller --upgrade

It takes a minute for helm to initialize its server component (tiller):

	watch "kubectl get pods --namespace kube-system | grep tiller"

When you see `Running`, it's time to proceed to the next step!

## Deploy TiDB Operator

The first TiDB component we are going to install is the TiDB Operator, using a Helm Chart.  The TiDB Operator is the management system that works with Kubernetes to bootstrap your TiDB cluster and keep it running. This step assumes you are in the `tidb-operator` working directory:

	kubectl apply -f ./manifests/crd.yaml &&
	kubectl apply -f ./manifests/gke-storage.yml &&
	helm install ./charts/tidb-operator -n tidb-admin --namespace=tidb-admin

We can watch the operator come up with:

	watch kubectl get pods --namespace tidb-admin -o wide

If you see `Running`, the next step is to launch a TiDB cluster!

## Deploy your first TiDB Cluster

Now with a single command we can bring-up a full TiDB cluster:

	helm install ./charts/tidb-cluster -n tidb --namespace=tidb

It will take a few minutes to launch.  You can monitor the progress with:

	watch kubectl get pods --namespace tidb -o wide

When you see all pods `Running`, it's time to proceed forward!

## Connecting to TiDB

There can be a small delay between the pod being up and running, and the service being available.  When you see `demo-tidb` appear, the service is ready to connect to:

	watch "kubectl get svc -n tidb"

You can connect to the clustered service within the Kubernetes cluster:

	kubectl run -n tidb mysql-client --rm -i --tty --image mysql -- mysql -P 4000 -u root -h $(kubectl get svc demo-tidb -n tidb --output json | jq -r '.spec.clusterIP')

For debugging purposes, you can also establish a tunnel between an individual TiDB pod and your Google Cloud Shell.  For example:

	kubectl -n tidb port-forward demo-tidb-0 4000:4000 &

From your Google shell:

	sudo apt-get install -y mysql-client &&
	mysql -h 127.0.0.1 -u root -P 4000

Congratulations, you are now up and running with a distributed MySQL database!

## Shutting down the Kubernetes Cluster

Once you have finished experimenting, you can delete the Kubernetes cluster with:

	gcloud container clusters delete tidb
