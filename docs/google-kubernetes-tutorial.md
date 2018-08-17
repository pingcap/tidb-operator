---
title: Deploy TiDB to Kubernetes on Google Cloud
summary: Tutorial for deploying TiDB on Google Cloud using Kubernetes.
category: operations
---

# Deploy TiDB, a distributed MySQL compatible database, to Kubernetes on Google Cloud

Lets use Google Cloud Kubernetes to have a straight-forward and reliable install of a TiDB, a distributed MySQL compatible database.  This tutorial takes you through these steps:

- Creating a new Google Cloud Project (optional)
- Launching a new 3 node Kubernetes cluster (optional)
- Installing the Helm package manager for Kubernetes
- Deploying TiDB to your Kubernetes cluster
- Connecting to TiDB
- Shutting down down the Kubernetes cluster (optional)


## Deploying Kubernetes on Google

TiDB can be deployed onto any Kubernetes cluster: you can bring up a three node cluster however you see fit or look for our documentation on deploying to other systems.
But here we will bring up a Kubernetes cluster on Google Cloud designed for TiDB.
Google provides a free small VM called Google Cloud Shell that you can run this tutorial from.
Just click the button:

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.png)](https://console.cloud.google.com/cloudshell/open?git_repo=https://github.com/pingcap/tidb-operator)
<!--
[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.png)](https://console.cloud.google.com/cloudshell/open?git_repo=https://github.com/pingcap/tidb-operator&tutorial=docs/google-kubernetes-tutorial.md)
-->

If you have any issues with this button, you can download this file as markdown and open it.

```sh
git clone https://github.com/pingcap/tidb-operator
cd tidb-operator
teachme docs/google-kubernetes-tutorial.md
```

Alternatively, you can run this from your laptop. You just need to [setup the gcloud tool first](https://cloud.google.com/sdk/docs/quickstarts).


## Create a new GCP Project

First create a project that this demo will be ran in:

	gcloud projects create MY_NEW_PROJECT_NAME

Before you can create a cluster, you must [enable Kubernetes for your project in the console.](https://console.cloud.google.com/projectselector/kubernetes?_ga=2.78459869.-833158988.1529036412)

Now set your gcloud to use this project:

	gcloud config set project MY_NEW_PROJECT_NAME


### Launch a three node Kubernetes cluster

At the end of this cluster creation step, we will have a Kubernetes cluster with kubectl authorized to connect to it.

For more detailed information, please review the [Quickstart](https://cloud.google.com/kubernetes-engine/docs/quickstart) instructions for setting up a Kubernetes cluster.

Set gcloud to use a [zone](https://cloud.google.com/compute/docs/regions-zones/):

	gcloud config set compute/zone us-west1-a

Now create the kubernetes cluster:

	gcloud container clusters create tidb

This could take more than a minute to complete. Now is a good time to do some stretches and refill your beverage.

When the Kubernetes cluster is completed, default gcloud to use it:

	gcloud config set container/cluster tidb

Now we have a Kubernetes cluster! Verify that kubectl can connect to it and that it has three machines running.

	kubectl get nodes


## Installing the Helm package manager for Kubernetes

We can install TiDB with helm charts. The following script [installs helm](https://github.com/helm/helm#install) and then moves it into a local bin directory since system installs do not persist with Google Cloud Shell:

```sh
curl https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get > get_helm.sh
chmod 700 get_helm.sh && ./get_helm.sh

mkdir -p ~/bin
cp /usr/local/bin/helm ~/bin
echo 'PATH="$PATH:$HOME/bin"' >> ~/.bashrc
```

Helm will need a couple of permissions to work properly:

``` sh
kubectl create serviceaccount tiller --namespace kube-system
kubectl apply -f manifests/tiller-rbac.yaml
helm init --service-account tiller --upgrade
```


## Deploy TiDB to your Kubernetes cluster

The TiDB helm charts are included in this repository.  Assuming you are still in the `tidb-operator` working directory, installing can be performed with:

```sh
kubectl apply -f ./manifests/crd.yaml
kubectl apply -f manifests/gke-storage.yml
helm install charts/tidb-operator -n tidb-admin --namespace=tidb-admin
```

We can watch the operator come up with:

	watch kubectl get pods --namespace tidb-admin -o wide

Now with a single command we can bring-up a full TiDB cluster:

	helm install charts/tidb-cluster -n tidb --namespace=tidb

Now we can watch our TiDB cluster come up:

	watch kubectl get pods --namespace tidb -o wide

## Connecting to TiDB

Now lets connect to our MySQL database. This will connect from within the Kubernetes cluster:

	kubectl run -n tidb mysql-client --rm -i --tty --image mysql -- mysql -P 4000 -u root -h $(kubectl get svc demo-tidb -n tidb --output json | jq -r '.spec.clusterIP')

Now you are up and running with a distributed MySQL database!

## Shutting down the Kubernetes Cluster

Once you have finished experimenting, you can delete the Kubernetes cluster with:

	gcloud container clusters delete tidb
