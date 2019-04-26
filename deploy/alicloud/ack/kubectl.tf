resource "null_resource" "setup-kubeconfig" {
  dependes_on = ["alicloud_cs_managed_kuberentes.this"]

  provisioner "local-exec" {
    working_dir = "${path.module}"
    command     = "aliyun cs GET /k8s/c186d078ab7ec424fae8b972f44164a4e/user_config | jq -r '.config' > ~/kubeconfig_${var.cluster_name}"

    environment = {
      KUBECONFIG = "${path.module}/kubeconfig_${var.cluster_name}"
    }
  }
}
