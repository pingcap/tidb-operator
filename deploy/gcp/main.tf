variable "GCP_CREDENTIALS_PATH" {
}

variable "GCP_REGION" {
}

variable "GCP_PROJECT" {
}

provider "google" {
  credentials = file(var.GCP_CREDENTIALS_PATH)
  region      = var.GCP_REGION
  project     = var.GCP_PROJECT
}

// required for taints on node pools
provider "google-beta" {
  credentials = file(var.GCP_CREDENTIALS_PATH)
  region      = var.GCP_REGION
  project     = var.GCP_PROJECT
}

locals {
  credential_path          = "${path.module}/credentials"
  kubeconfig               = "${local.credential_path}/kubeconfig_${var.cluster_name}"
  tidb_cluster_values_path = "${path.module}/rendered/tidb-cluster-values.yaml"
}

resource "null_resource" "prepare-dir" {
  provisioner "local-exec" {
    command = "mkdir -p ${local.credential_path}"
  }
}

resource "null_resource" "set-gcloud-project" {
  provisioner "local-exec" {
    command = "gcloud config set project ${var.GCP_PROJECT}"
  }
}

resource "google_compute_network" "vpc_network" {
  name                    = "vpc-network"
  auto_create_subnetworks = false
  project                 = var.GCP_PROJECT
}

resource "google_compute_subnetwork" "private_subnet" {
  ip_cidr_range = "172.31.252.0/22"
  name          = "private-subnet"
  network       = google_compute_network.vpc_network.name
  project       = var.GCP_PROJECT

  secondary_ip_range {
    ip_cidr_range = "172.30.0.0/16"
    range_name    = "pods-${var.GCP_REGION}"
  }

  secondary_ip_range {
    ip_cidr_range = "172.31.224.0/20"
    range_name    = "services-${var.GCP_REGION}"
  }

  lifecycle {
    ignore_changes = [secondary_ip_range]
  }
}

resource "google_compute_subnetwork" "public_subnet" {
  ip_cidr_range = "172.29.252.0/22"
  name          = "public-subnet"
  network       = google_compute_network.vpc_network.name
  project       = var.GCP_PROJECT
}

resource "google_container_cluster" "cluster" {
  name       = var.cluster_name
  network    = google_compute_network.vpc_network.name
  subnetwork = google_compute_subnetwork.private_subnet.name
  location   = var.GCP_REGION
  project    = var.GCP_PROJECT

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

  remove_default_node_pool = true
  initial_node_count       = 1

  min_master_version = "latest"

  lifecycle {
    ignore_changes = [master_auth] // see above linked issue
  }
}

resource "google_container_node_pool" "pd_pool" {
  // The monitor pool is where tiller must first be deployed to.
  depends_on         = [google_container_node_pool.monitor_pool]
  provider           = google-beta
  project            = var.GCP_PROJECT
  cluster            = google_container_cluster.cluster.name
  location           = google_container_cluster.cluster.location
  name               = "pd-pool"
  initial_node_count = var.pd_count

  management {
    auto_repair = false
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
  project            = var.GCP_PROJECT
  cluster            = google_container_cluster.cluster.name
  location           = google_container_cluster.cluster.location
  name               = "tikv-pool"
  initial_node_count = var.tikv_count

  management {
    auto_repair = false
    auto_upgrade = false
  }

  node_config {
    machine_type    = var.tikv_instance_type
    image_type      = "UBUNTU"
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
  // The pool order is monitor -> tikv -> pd -> tidb
  depends_on         = [google_container_node_pool.pd_pool]
  provider           = google-beta
  project            = var.GCP_PROJECT
  cluster            = google_container_cluster.cluster.name
  location           = google_container_cluster.cluster.location
  name               = "tidb-pool"
  initial_node_count = var.tidb_count

  management {
    auto_repair = false
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
  project            = var.GCP_PROJECT
  cluster            = google_container_cluster.cluster.name
  location           = google_container_cluster.cluster.location
  name               = "monitor-pool"
  initial_node_count = var.monitor_count

  management {
    auto_repair = false
    auto_upgrade = false
  }

  node_config {
    machine_type = var.monitor_instance_type
    tags         = ["monitor"]
    oauth_scopes = ["storage-ro", "logging-write", "monitoring"]
  }
}

resource "google_compute_firewall" "allow_ssh_bastion" {
  name    = "allow-ssh-bastion"
  network = google_compute_network.vpc_network.self_link
  project = var.GCP_PROJECT

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["bastion"]
}

resource "google_compute_firewall" "allow_mysql_from_bastion" {
  name    = "allow-mysql-from-bastion"
  network = google_compute_network.vpc_network.self_link
  project = var.GCP_PROJECT

  allow {
    protocol = "tcp"
    ports    = ["4000"]
  }

  source_tags = ["bastion"]
  target_tags = ["tidb"]
}

resource "google_compute_firewall" "allow_ssh_from_bastion" {
  name    = "allow-ssh-from-bastion"
  network = google_compute_network.vpc_network.self_link
  project = var.GCP_PROJECT

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_tags = ["bastion"]
  target_tags = ["tidb", "tikv", "pd", "monitor"]
}

resource "google_compute_instance" "bastion" {
  project      = var.GCP_PROJECT
  zone         = data.google_compute_zones.available.names[0]
  machine_type = var.bastion_instance_type
  name         = "bastion"

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-1804-lts"
    }
  }

  network_interface {
    subnetwork = google_compute_subnetwork.public_subnet.self_link
    access_config {
    }
  }

  tags = ["bastion"]

  metadata_startup_script = "sudo apt-get install -y mysql-client && curl -s https://packagecloud.io/install/repositories/akopytov/sysbench/script.deb.sh | bash && sudo apt-get -y install sysbench"
}

resource "null_resource" "get-credentials" {
  provisioner "local-exec" {
    command = "gcloud container clusters get-credentials ${google_container_cluster.cluster.name} --region ${var.GCP_REGION}"

    environment = {
      KUBECONFIG = local.kubeconfig
    }
  }

  provisioner "local-exec" {
    when = destroy

    command = <<EOS
kubectl get pvc -n tidb -o jsonpath='{.items[*].spec.volumeName}'|fmt -1 | xargs -I {} kubectl patch pv {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
EOS


    environment = {
      KUBECONFIG = local.kubeconfig
    }
  }
}

resource "local_file" "tidb-cluster-values" {
  depends_on = [data.template_file.tidb_cluster_values]
  filename = local.tidb_cluster_values_path
  content = data.template_file.tidb_cluster_values.rendered
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

kubectl apply -f manifests/crd.yaml
kubectl apply -k manifests/local-ssd
kubectl apply -f manifests/gke/persistent-disk.yaml
kubectl apply -f manifests/tiller-rbac.yaml

helm init --service-account tiller --upgrade
until helm ls; do
  echo "Wait until tiller is ready"
  sleep 1
done
helm upgrade --install tidb-operator --namespace tidb-admin ${path.module}/charts/tidb-operator --set operatorImage=${var.tidb_operator_registry}/tidb-operator:${var.tidb_operator_version}
EOS


    environment = {
      KUBECONFIG = local.kubeconfig
    }
  }
}

resource "null_resource" "deploy-tidb-cluster" {
  depends_on = [
    null_resource.setup-env,
    local_file.tidb-cluster-values,
    google_container_node_pool.pd_pool,
  ]

  triggers = {
    values = data.template_file.tidb_cluster_values.rendered
  }

  provisioner "local-exec" {
    interpreter = ["bash", "-c"]
command = <<EOS
set -euo pipefail

helm upgrade --install tidb-cluster ${path.module}/charts/tidb-cluster --namespace=tidb -f ${local.tidb_cluster_values_path}
until kubectl get po -n tidb -lapp.kubernetes.io/component=tidb | grep Running; do
  echo "Wait for TiDB pod running"
  sleep 5
done

until kubectl get svc -n tidb tidb-cluster-tidb -o json | jq '.status.loadBalancer.ingress[0]' | grep ip; do
  echo "Wait for TiDB internal loadbalancer IP"
  sleep 5
done
EOS

    environment = {
      KUBECONFIG = local.kubeconfig
    }
  }
}

