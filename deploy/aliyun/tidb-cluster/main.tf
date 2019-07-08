# kubernetes and helm providers rely on EKS, but terraform provider doesn't support depends_on
# follow this link https://github.com/hashicorp/terraform/issues/2430#issuecomment-370685911
# we have the following hack
resource "local_file" "kubeconfig" {
  depends_on        = [var.ack]
  sensitive_content = var.ack.kubeconfig
  filename          = var.ack.kubeconfig_filename
}

provider "helm" {
  insecure = true
  # service_account = "tiller"
  # install_tiller = true # currently this doesn't work, so we install tiller in the local-exec provisioner. See https://github.com/terraform-providers/terraform-provider-helm/issues/148
  kubernetes {
    config_path = local_file.kubeconfig.filename
  }
}

resource "null_resource" "wait-tiller-ready" {
  depends_on = [var.ack]

  provisioner "local-exec" {
    working_dir = path.cwd
    command     = <<EOS
until helm ls; do
  echo "Wait tiller ready"
done
EOS
    environment = {
      KUBECONFIG = "${local_file.kubeconfig.filename}"
    }
  }
}