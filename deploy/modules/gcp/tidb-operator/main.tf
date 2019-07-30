resource "google_container_cluster" "cluster" {
  name = var.gke_name
  network = var.vpc_name
  subnetwork = var.subnetwork_name
  location = var.gcp_region
  project = var.gcp_project

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

  // see https://github.com/terraform-providers/terraform-provider-google/issues/3385 for why initial_node_count is sum of node counts
  remove_default_node_pool = true
  initial_node_count       = var.pd_count + var.tikv_count + var.tidb_count + var.monitor_count

  min_master_version = var.gke_version

  lifecycle {
    ignore_changes = [master_auth] // see above linked issue
  }

  maintenance_policy {
    daily_maintenance_window {
      start_time = "01:00"
    }
  }
}

resource "google_container_node_pool" "pd_pool" {
  // The monitor pool is where tiller must first be deployed to.
  depends_on         = [google_container_node_pool.monitor_pool]
  provider           = google-beta
  project            = var.gcp_project
  cluster            = google_container_cluster.cluster.name
  location           = google_container_cluster.cluster.location
  name               = "pd-pool"
  initial_node_count = var.pd_count

  management {
    auto_repair  = false
    auto_upgrade = false
  }

  node_config {
    machine_type    = var.pd_instance_type
    local_ssd_count = 0

    taint {
      effect = "NO_SCHEDULE"
      key    = "dedicated"
      value  = "pd"
    }

    labels = {
      dedicated = "pd"
    }

    tags         = ["pd"]
    oauth_scopes = ["storage-ro", "logging-write", "monitoring"]
  }
}

resource "google_container_node_pool" "tikv_pool" {
  provider           = google-beta
  project            = var.gcp_project
  cluster            = google_container_cluster.cluster.name
  location           = google_container_cluster.cluster.location
  name               = "tikv-pool"
  initial_node_count = var.tikv_count

  management {
    auto_repair  = false
    auto_upgrade = false
  }

  node_config {
    machine_type = var.tikv_instance_type
    image_type   = "UBUNTU"
    // This value cannot be changed (instead a new node pool is needed)
    // 1 SSD is 375 GiB
    local_ssd_count = 1

    taint {
      effect = "NO_SCHEDULE"
      key    = "dedicated"
      value  = "tikv"
    }

    labels = {
      dedicated = "tikv"
    }

    tags         = ["tikv"]
    oauth_scopes = ["storage-ro", "logging-write", "monitoring"]
  }
}

resource "google_container_node_pool" "tidb_pool" {
  // The pool order is tikv -> monitor -> pd -> tidb
  depends_on         = [google_container_node_pool.pd_pool]
  provider           = google-beta
  project            = var.gcp_project
  cluster            = google_container_cluster.cluster.name
  location           = google_container_cluster.cluster.location
  name               = "tidb-pool"
  initial_node_count = var.tidb_count

  management {
    auto_repair  = false
    auto_upgrade = false
  }

  node_config {
    machine_type = var.tidb_instance_type

    taint {
      effect = "NO_SCHEDULE"
      key    = "dedicated"
      value  = "tidb"
    }

    labels = {
      dedicated = "tidb"
    }

    tags         = ["tidb"]
    oauth_scopes = ["storage-ro", "logging-write", "monitoring"]
  }
}

resource "google_container_node_pool" "monitor_pool" {
  // Setup local SSD on TiKV nodes first (this can take some time)
  // Create the monitor pool next because that is where tiller will be deployed to
  depends_on         = [google_container_node_pool.tikv_pool]
  project            = var.gcp_project
  cluster            = google_container_cluster.cluster.name
  location           = google_container_cluster.cluster.location
  name               = "monitor-pool"
  initial_node_count = var.monitor_count

  management {
    auto_repair  = false
    auto_upgrade = false
  }

  node_config {
    machine_type = var.monitor_instance_type
    tags         = ["monitor"]
    oauth_scopes = ["storage-ro", "logging-write", "monitoring"]
  }
}

resource "null_resource" "get-credentials" {
  depends_on = [google_container_cluster.cluster]
  provisioner "local-exec" {
    command = "gcloud container clusters get-credentials ${google_container_cluster.cluster.name} --region ${var.gcp_region}"

    environment = {
      KUBECONFIG = var.kubeconfig_path
    }
  }

  provisioner "local-exec" {
    when = destroy

    command = <<EOS
kubectl get pvc -n tidb -o jsonpath='{.items[*].spec.volumeName}'|fmt -1 | xargs -I {} kubectl patch pv {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
EOS


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
    config_path = var.kubeconfig_path
  }
}

resource "null_resource" "setup-env" {
  depends_on = [
    google_container_cluster.cluster,
    null_resource.get-credentials,
    var.tidb_operator_registry,
    var.tidb_operator_version,
  ]

  provisioner "local-exec" {
    working_dir = path.module
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
helm upgrade --install tidb-operator --namespace tidb-admin ${path.module}/charts/tidb-operator --set operatorImage=${var.tidb_operator_registry}/tidb-operator:${var.tidb_operator_version}
EOS


    environment = {
      KUBECONFIG = var.kubeconfig_path
    }
  }
}

data "helm_repository" "pingcap" {
  provider = "helm.initial"
  depends_on = ["null_resource.setup-env"]
  name = "pingcap"
  url = "http://charts.pingcap.org/"
}

resource "helm_release" "tidb-operator" {
  provider = "helm.initial"
  depends_on = [null_resource.setup-env, null_resource.get-credentials]

  repository = data.helm_repository.pingcap.name
  chart = "tidb-operator"
  version = var.tidb_operator_version
  namespace = "tidb-admin"
  name = "tidb-operator"
  values = [var.operator_helm_values]
}



