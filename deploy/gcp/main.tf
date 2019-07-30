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
  credential_path          = "${path.cwd}/credentials"
  kubeconfig               = "${local.credential_path}/kubeconfig_${var.gke_name}"
  tidb_cluster_values_path = "${path.module}/rendered/tidb-cluster-values.yaml"
  vpc_name = var.gke_name
}

module "project-credentials" {
  source = "../modules/gcp/project-credentials"

  path = local.credential_path
  gcloud_project = var.GCP_PROJECT
}

module "vpc" {
  source = "../modules/gcp/vpc"
  create_vpc = var.create_vpc
  gcp_project = var.GCP_PROJECT
  gcp_region = var.GCP_REGION
  vpc_name = local.vpc_name
}

module "tidb-operator" {
  source = "../modules/gcp/tidb-operator"
  gke_name = var.gke_name
  vpc_name = local.vpc_name
  subnetwork_name = var.private_subnet_name
  gcp_project = var.GCP_PROJECT
  gcp_region = var.GCP_REGION
  pd_count = var.pd_count
  tikv_count = var.tikv_count
  tidb_count = var.tidb_count
  monitor_count = var.monitor_count
  kubeconfig_path = local.kubeconfig
  pd_instance_type = var.pd_instance_type
  tikv_instance_type = var.tikv_instance_type
  tidb_instance_type = var.tidb_instance_type
}


resource "google_compute_firewall" "allow_ssh_bastion" {
  name    = "allow-ssh-bastion"
  network = local.vpc_name
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
  network = local.vpc_name
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
  network = local.vpc_name
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

resource "local_file" "tidb-cluster-values" {
  depends_on = [data.template_file.tidb_cluster_values]
  filename = local.tidb_cluster_values_path
  content = data.template_file.tidb_cluster_values.rendered
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
command         = <<EOS
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

