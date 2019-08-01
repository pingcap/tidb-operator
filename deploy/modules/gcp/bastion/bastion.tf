resource "google_compute_firewall" "allow_ssh_bastion" {
  name    = "allow-ssh-bastion"
  network = var.vpc_name
  project = var.gcp_project

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["bastion"]
}

resource "google_compute_firewall" "allow_mysql_from_bastion" {
  name    = "allow-mysql-from-bastion"
  network = var.vpc_name
  project = var.gcp_project

  allow {
    protocol = "tcp"
    ports    = ["4000"]
  }

  source_tags = ["bastion"]
  target_tags = ["tidb"]
}

resource "google_compute_firewall" "allow_ssh_from_bastion" {
  name    = "allow-ssh-from-bastion"
  network = var.vpc_name
  project = var.gcp_project

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_tags = ["bastion"]
  target_tags = ["tidb", "tikv", "pd", "monitor"]
}

resource "google_compute_instance" "bastion" {
  project      = var.gcp_project
  zone         = data.google_compute_zones.available.names[0]
  machine_type = var.bastion_instance_type
  name         = "bastion"

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-1804-lts"
    }
  }

  network_interface {
    subnetwork = var.public_subnet_name
  }

  tags = ["bastion"]

  metadata_startup_script = "sudo apt-get install -y mysql-client && curl -s https://packagecloud.io/install/repositories/akopytov/sysbench/script.deb.sh | bash && sudo apt-get -y install sysbench"
}

