locals {
  public_key_filename  = "${var.path}/${var.name}.pub"
  private_key_filename = "${var.path}/${var.name}.pem"
}

resource "tls_private_key" "generated" {
  algorithm = "RSA"
}

resource "azurerm_ssh_public_key" "generated" {
  name                = var.name
  location            = var.region
  resource_group_name = var.resource_group
  public_key          = tls_private_key.generated.public_key_openssh
}

resource "local_file" "public_key_openssh" {
  count    = var.path != "" ? 1 : 0
  content  = tls_private_key.generated.public_key_openssh
  filename = local.public_key_filename
}

resource "local_file" "private_key_pem" {
  count               = var.path != "" ? 1 : 0
  content             = tls_private_key.generated.private_key_pem
  filename            = local.private_key_filename
  file_permission     = "0600"
}
