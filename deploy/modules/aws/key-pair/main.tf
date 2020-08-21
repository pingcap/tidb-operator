provider "aws" {
  region = var.region
}

locals {
  public_key_filename  = "${var.path}/${var.name}.pub"
  private_key_filename = "${var.path}/${var.name}.pem"
}

resource "tls_private_key" "generated" {
  algorithm = "RSA"
}

resource "aws_key_pair" "generated" {
  key_name   = var.name
  public_key = tls_private_key.generated.public_key_openssh

  lifecycle {
    ignore_changes = [key_name]
  }
}

resource "local_file" "public_key_openssh" {
  count    = var.path != "" ? 1 : 0
  content  = tls_private_key.generated.public_key_openssh
  filename = local.public_key_filename
}

resource "local_file" "private_key_pem" {
  count    = var.path != "" ? 1 : 0
  content  = tls_private_key.generated.private_key_pem
  filename = local.private_key_filename
}

resource "null_resource" "chmod" {
  count      = var.path != "" ? 1 : 0
  depends_on = [local_file.private_key_pem]

  triggers = {
    key = tls_private_key.generated.private_key_pem
  }

  provisioner "local-exec" {
    command = "chmod 600 ${local.private_key_filename}"
  }
}
