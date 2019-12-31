resource "google_container_cluster" "cluster" {
  name           = var.gke_name
  network        = var.vpc_name
  subnetwork     = var.subnetwork_name
  location       = var.location
  node_locations = length(var.node_locations) == 0 ? null : var.node_locations
  project        = var.gcp_project

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

locals {
  # Because `gcloud containers clusters get-credentials` cannot accept location
  # argument, we must use --zone or --region flag according to location's
  # value.
  # This is same as in google terraform provider, see
  # https://github.com/terraform-providers/terraform-provider-google/blob/24c36107e03cbaeb38ae1ebb24de7aa51a0343df/google/resource_container_cluster.go#L956-L960.
  cmd_get_cluster_credentials = length(split("-", var.location)) == 3 ? "gcloud --project ${var.gcp_project} container clusters get-credentials ${google_container_cluster.cluster.name} --zone ${var.location}" : "gcloud --project ${var.gcp_project} container clusters get-credentials ${google_container_cluster.cluster.name} --region ${var.location}"
}

resource "null_resource" "get-credentials" {
  depends_on = [google_container_cluster.cluster]
  triggers = {
    command = local.cmd_get_cluster_credentials
  }
  provisioner "local-exec" {
    command = local.cmd_get_cluster_credentials
    environment = {
      KUBECONFIG = var.kubeconfig_path
    }
  }
}

provider "helm" {
  alias    = "initial"
  insecure = true
  # service_account = "tiller"
  install_tiller = false # currently this doesn't work, so we install tiller in the local-exec provisioner. See https://github.com/terraform-providers/terraform-provider-helm/issues/148
  kubernetes {
    # helm provider loads the file when it's initialized, we must wait for it to be created.
    # However we cannot use resource here, because in refresh phrase, it will
    # not be resolved and argument default value is used. To work around this,
    # we defer initialization by using load_config_file argument.
    # See https://github.com/pingcap/tidb-operator/pull/819#issuecomment-524547459
    config_path = var.kubeconfig_path
    # used to delay helm provisioner initialization in apply phrase
    load_config_file = null_resource.get-credentials.id != "" ? true : null
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
