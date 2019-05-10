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
  ip_cidr_range = "10.0.1.0/24"
  name = "private-subnet"
  network = "${google_compute_network.vpc_network.self_link}"
  project = "${var.GCP_PROJECT}"
}

resource "google_compute_subnetwork" "public_subnet" {
  ip_cidr_range = "10.0.4.0/24"
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
    master_ipv4_cidr_block = "172.16.0.0/28"
  }

  ip_allocation_policy {
    use_ip_aliases = true
  }

//  remove_default_node_pool = true
  initial_node_count = 1
}

//resource "google_container_node_pool" "node_pool" {
//  cluster = "${google_container_cluster.cluster.name}"
//  location = "${google_container_cluster.cluster.location}"
//  name = "the-node-pool"
//  initial_node_count = "3"
//
//  node_config {
//    machine_type = "n1-standard-1"
//  }
//
//}

resource "tls_private_key" "bastion" {
  algorithm = "RSA"
  rsa_bits = 4096
}

resource "google_compute_project_metadata_item" "ssh-key" {
  key = "ssh-key"
  value = "${tls_private_key.bastion.public_key_openssh}"
}

resource "google_compute_instance" "bastion" {
  machine_type = "f1-micro"
  name = "bastion"
  "boot_disk" {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-1804-lts"
    }
  }
  "network_interface" {
    subnetwork = "${google_compute_subnetwork.public_subnet.self_link}"
  }

  metadata {
    ssh-key = "${}"
  }
}