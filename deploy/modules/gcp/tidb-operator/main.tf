resource "google_container_cluster" "cluster" {
  name       = var.gke_name
  network    = var.vpc_name
  subnetwork = var.subnetwork_name
  location   = var.gcp_region
  project    = var.gcp_project

  master_auth {
    username = ""
    password = ""

    // due to https://github.com/terraform-providers/terraform-provider-google/issues/3369
    client_certificate_config {
      issue_client_certificate = false
    }
  }

  master_authorized_networks_config {
    cidr_blocks {
      cidr_block = "0.0.0.0/0"
    }
  }

  ip_allocation_policy {
    use_ip_aliases = true
  }

  // See https://kubernetes.io/docs/setup/best-practices/cluster-large/#size-of-master-and-master-components for why initial_node_count is 5. The master node should accommodate up to 100 nodes like this
  remove_default_node_pool = true
  initial_node_count       = 5

  min_master_version = var.gke_version

  lifecycle {
    ignore_changes = [master_auth] // see above linked issue
  }

  maintenance_policy {
    daily_maintenance_window {
      start_time = var.maintenance_window_start_time
    }
  }
}

resource "null_resource" "get-credentials" {
  depends_on = [google_container_cluster.cluster]
  provisioner "local-exec" {
    command = "gcloud --project ${var.gcp_project} container clusters get-credentials ${google_container_cluster.cluster.name} --region ${var.gcp_region}"

    environment = {
      KUBECONFIG = var.kubeconfig_path
    }
  }
}

data "local_file" "kubeconfig" {
  depends_on = [google_container_cluster.cluster, null_resource.get-credentials]
  filename   = var.kubeconfig_path
}

resource "local_file" "kubeconfig" {
  depends_on = [google_container_cluster.cluster, null_resource.get-credentials]
  content    = data.local_file.kubeconfig.content
  filename   = var.kubeconfig_path
}

provider "helm" {
  alias    = "initial"
  insecure = true
  # service_account = "tiller"
  install_tiller = false # currently this doesn't work, so we install tiller in the local-exec provisioner. See https://github.com/terraform-providers/terraform-provider-helm/issues/148
  kubernetes {
    config_path = local_file.kubeconfig.filename
  }
}

resource "null_resource" "setup-env" {
  depends_on = [
    google_container_cluster.cluster,
    null_resource.get-credentials,
    var.tidb_operator_version,
  ]

  provisioner "local-exec" {
    working_dir = path.cwd
    interpreter = ["bash", "-c"]

    command = <<EOS
set -euo pipefail

if ! kubectl get clusterrolebinding cluster-admin-binding 2>/dev/null; then
  kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user $(gcloud config get-value account)
fi

if ! kubectl get serviceaccount -n kube-system tiller 2>/dev/null ; then
  kubectl create serviceaccount --namespace kube-system tiller
fi

kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/${var.tidb_operator_version}/manifests/crd.yaml
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/${var.tidb_operator_version}/manifests/tiller-rbac.yaml
kubectl apply -k manifests/local-ssd
kubectl apply -f manifests/gke/persistent-disk.yaml

helm init --service-account tiller --upgrade --wait
until helm ls; do
  echo "Wait until tiller is ready"
  sleep 5
done
EOS


    environment = {
      KUBECONFIG = var.kubeconfig_path
    }
  }
}

data "helm_repository" "pingcap" {
  provider   = "helm.initial"
  depends_on = [null_resource.setup-env]
  name       = "pingcap"
  url        = "https://charts.pingcap.org/"
}

resource "helm_release" "tidb-operator" {
  provider = "helm.initial"
  depends_on = [
    null_resource.setup-env,
    null_resource.get-credentials,
    data.helm_repository.pingcap,
  ]

  repository = data.helm_repository.pingcap.name
  chart      = "tidb-operator"
  version    = var.tidb_operator_version
  namespace  = "tidb-admin"
  name       = "tidb-operator"
  values     = [var.operator_helm_values]
  wait       = false
}
