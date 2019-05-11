variable "GCP_CREDENTIALS_PATH" {}
variable "GCP_REGION" {}
variable "GCP_PROJECT" {}

provider "google" {
  credentials = "${file("${var.GCP_CREDENTIALS_PATH}")}"
  region = "${var.GCP_REGION}"
}

locals {
  credential_path = "${path.module}/credentials"
  kubeconfig                    = "${local.credential_path}/kubeconfig_${var.cluster_name}"
  key_file                      = "${local.credential_path}/${var.cluster_name}-node-key.pem"
  bastion_key_file              = "${local.credential_path}/${var.cluster_name}-bastion-key.pem"
}

resource "null_resource" "prepare-dir" {
  provisioner "local-exec" {
    command = "mkdir -p ${local.credential_path}"
  }
}

resource "google_compute_network" "vpc_network" {
  name = "vpc-network"
  auto_create_subnetworks = false
  project = "${var.GCP_PROJECT}"
}

resource "google_compute_subnetwork" "private_subnet" {
  ip_cidr_range = "172.31.252.0/22"
  name = "private-subnet"
  network = "${google_compute_network.vpc_network.self_link}"
  project = "${var.GCP_PROJECT}"
  secondary_ip_range {
    ip_cidr_range = "172.30.0.0/16"
    range_name = "pods-${var.GCP_REGION}"
  }
  secondary_ip_range {
    ip_cidr_range = "172.31.224.0/20"
    range_name = "services-${var.GCP_REGION}"
  }
}

resource "google_compute_subnetwork" "public_subnet" {
  ip_cidr_range = "172.29.252.0/22"
  name = "public-subnet"
  network = "${google_compute_network.vpc_network.self_link}"
  project = "${var.GCP_PROJECT}"
}

resource "google_container_cluster" "cluster" {
  name = "the-cluster" // turn this into var
  network = "${google_compute_network.vpc_network.self_link}"
  subnetwork = "${google_compute_subnetwork.private_subnet.self_link}"
  location = "${var.GCP_REGION}"
  project = "${var.GCP_PROJECT}"

  private_cluster_config {
    enable_private_endpoint = false
    enable_private_nodes = true
    master_ipv4_cidr_block = "172.31.64.0/28"
  }

  ip_allocation_policy {
    use_ip_aliases = true
  }

  remove_default_node_pool = true
  initial_node_count = 1
}

resource "google_container_node_pool" "pd_pool" {
  project = "${var.GCP_PROJECT}"
  cluster = "${google_container_cluster.cluster.name}"
  location = "${google_container_cluster.cluster.location}"
  name = "pd-pool"
  initial_node_count = "1"

  node_config {
    machine_type = "n1-standard-1"
    local_ssd_count = 1
  }

}

resource "google_container_node_pool" "tikv_pool" {
  project = "${var.GCP_PROJECT}"
  cluster = "${google_container_cluster.cluster.name}"
  location = "${google_container_cluster.cluster.location}"
  name = "tikv-pool"
  initial_node_count = "1"

  node_config {
    machine_type = "n1-standard-1"
    local_ssd_count = 1
  }

}

resource "google_container_node_pool" "tidb_pool" {
  project = "${var.GCP_PROJECT}"
  cluster = "${google_container_cluster.cluster.name}"
  location = "${google_container_cluster.cluster.location}"
  name = "tidb-pool"
  initial_node_count = "1"

  node_config {
    machine_type = "n1-standard-1"
  }

}

resource "google_compute_firewall" "allow_ssh_bastion" {
  name = "allow-ssh-bastion"
  network = "${google_compute_network.vpc_network.self_link}"
  project = "${var.GCP_PROJECT}"

  allow {
    protocol = "tcp"
    ports = ["22"]
  }
  source_ranges = ["0.0.0.0/0"]
  target_tags = ["bastion"]
}

resource "google_compute_instance" "bastion" {
  project = "${var.GCP_PROJECT}"
  zone = "${var.GCP_REGION}-a"
  machine_type = "f1-micro"
  name = "bastion"
  "boot_disk" {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-1804-lts"
    }
  }
  "network_interface" {
    subnetwork = "${google_compute_subnetwork.public_subnet.self_link}"
    access_config {}
  }
  tags = ["bastion"]

  metadata_startup_script = "sudo apt-get install -y mysql-client && curl -s https://packagecloud.io/install/repositories/akopytov/sysbench/script.rpm.sh | bash && sudo apt-get -y install sysbench"
}