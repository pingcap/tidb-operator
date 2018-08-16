---
title: Deploy TiDB to Kubernetes on Google Cloud
summary: Tutorial for deploying TiDB on Google Cloud using Kubernetes.
category: operations
---

# Deploy TiDB, a distributed MySQL compatible database, to Kubernetes on Google Cloud

Lets use Google Cloud Kubernetes to have a straight-forward and reliable install of a TiDB, a distributed MySQL compatible database.


## Deploying Kubernetes on Google

TiDB can be deployed onto any Kubernetes cluster: you can bring up a three node cluster however you see fit and skip this section. But here we will bring up a cluster on Google Cloud designed for TiDB.
Google provides a free small VM called Google Cloud Shell that you can run this tutorial from.
Just click the button:

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.png)](https://console.cloud.google.com/cloudshell/open?git_repo=https://github.com/pingcap/tidb-operator)
<!--
[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.png)](https://console.cloud.google.com/cloudshell/open?git_repo=https://github.com/pingcap/tidb-operator&tutorial=docs/google-kubernetes-tutorial.md)
-->

If you have any issues with this button, you can download this file as markdown and open it.

	git clone https://github.com/pingcap/tidb-operator
	cd tidb-operator
	teachme docs/google-kubernetes-tutorial.md


### Bring up the Kubernetes cluster 

At the end of this cluster creation step, we will have a Kubernetes cluster with kubectl authorized to connect to it.
You can [use terraform]() or other tools to achieve this.

For more detailed information, please review the [Quickstart](https://cloud.google.com/kubernetes-engine/docs/quickstart) instructions for setting up a Kubernetes cluster.

First create a project that this demo will be ran in.

	gcloud projects create tidb-demo

Before you can create a cluster, you must [enable Kubernetes for your project in the console.](https://console.cloud.google.com/projectselector/kubernetes?_ga=2.78459869.-833158988.1529036412)

Now set your gcloud to use this project:

	gcloud config set project tidb-demo

Set it to use a [zone](https://cloud.google.com/compute/docs/regions-zones/)

	gcloud config set compute/zone us-west1-a

Now create the kubernetes cluster.

	gcloud container clusters create tidb

This could take more than a minute to complete. Now is a good time to do some stretches and refill your beverage.

When the cluster is completed, default gcloud to use it.

	gcloud config set container/cluster tidb


## Running TiDB

Now we have a cluster! Verify that kubectl can connect to it and that it has three machines running.

	kubectl get nodes

We can install TiDB with helm charts. Maske sure [helm is installed](https://github.com/helm/helm#install) on your platform.

We can get the TiDB helm charts from the source repository.

	git clone https://github.com/pingcap/tidb-operator
	cd tidb-operator
	git checkout gregwebs/kube-tutorial

Helm will need a couple of permissions to work properly.

	kubectl create serviceaccount tiller --namespace kube-system
	kubectl apply -f manifests/tiller-rbac.yaml
	helm init --service-account tiller --upgrade

Now we can run the TiDB operator and the TiDB cluster

	kubectl apply -f ./manifests/crd.yaml
	kubectl apply -f manifests/gke-storage.yml
	helm install charts/tidb-operator -n tidb-operator --namespace=tidb-admin

We can watch the operator come up with

	watch kubectl get pods --namespace tidb-admin -o wide

Now with a single command we can bring-up a full TiDB cluster.

	helm install charts/tidb-cluster -n tidb --namespace=tidb

Now we can watch our cluster come up

	watch kubectl get pods --namespace tidb -o wide

Now lets connect to our MySQL database. This will connect from within the Kubernetes cluster.

	kubectl run -n tidb mysql-client --rm -i --tty --image mysql -- mysql -P 4000 -u root -h $(kubectl get svc demo-cluster-tidb -n tidb --output json | jq -r '.spec.clusterIP')

Now you are up and running with a distribute MySQL database!
