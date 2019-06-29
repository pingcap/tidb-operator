data "aws_ami" "eks_worker" {
  filter {
    name   = "name"
    values = ["amazon-eks-node-${var.eks_info.version}-${var.worker_ami_name_filter}"]
  }

  most_recent = true

  # Owner ID of AWS EKS team
  owners = ["602401143452"]
}

data "template_file" "userdata" {
  template = file("${path.module}/templates/userdata.sh.tpl")
  count = local.worker_group_count

  vars = {
    cluster_name = var.eks_info.name
    endpoint = var.eks_info.endpoint
    cluster_auth_base64 = var.eks_info.ca
    pre_userdata = lookup(
      local.tidb_cluster_worker_groups[count.index],
      "pre_userdata",
      local.workers_group_defaults["pre_userdata"],
    )
    additional_userdata = lookup(
      local.tidb_cluster_worker_groups[count.index],
      "additional_userdata",
      local.workers_group_defaults["additional_userdata"],
    )
    bootstrap_extra_args = lookup(
      local.tidb_cluster_worker_groups[count.index],
      "bootstrap_extra_args",
      local.workers_group_defaults["bootstrap_extra_args"],
    )
    kubelet_extra_args = lookup(
      local.tidb_cluster_worker_groups[count.index],
      "kubelet_extra_args",
      local.workers_group_defaults["kubelet_extra_args"],
    )
  }
}

data "template_file" "launch_template_userdata" {
  template = file("${path.module}/templates/userdata.sh.tpl")
  count = var.worker_group_launch_template_count

  vars = {
    cluster_name = var.eks_info.name
    endpoint = var.eks_info.endpoint
    cluster_auth_base64 = var.eks_info.ca
    pre_userdata = lookup(
      var.worker_groups_launch_template[count.index],
      "pre_userdata",
      local.workers_group_launch_template_defaults["pre_userdata"],
    )
    additional_userdata = lookup(
      var.worker_groups_launch_template[count.index],
      "additional_userdata",
      local.workers_group_launch_template_defaults["additional_userdata"],
    )
    bootstrap_extra_args = lookup(
      var.worker_groups_launch_template[count.index],
      "bootstrap_extra_args",
      local.workers_group_launch_template_defaults["bootstrap_extra_args"],
    )
    kubelet_extra_args = lookup(
      var.worker_groups_launch_template[count.index],
      "kubelet_extra_args",
      local.workers_group_launch_template_defaults["kubelet_extra_args"],
    )
  }
}

data "external" "tidb_elb" {
  depends_on = [null_resource.deploy-cluster]
  working_dir = path.cwd
  program    = ["bash", "-c", "kubectl --kubeconfig ${var.eks_info.kubeconfig_file} get svc -n ${var.cluster_name} ${var.cluster_name}-tidb -o json | jq '.status.loadBalancer.ingress[0]'"]
}

data "external" "monitor_elb" {
  depends_on = [null_resource.deploy-cluster]
  working_dir = path.cwd
  program    = ["bash", "-c", "kubectl --kubeconfig ${var.eks_info.kubeconfig_file} get svc -n ${var.cluster_name} ${var.cluster_name}-grafana -o json | jq '.status.loadBalancer.ingress[0]'"]
}
