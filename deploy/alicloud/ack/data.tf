data "alicloud_zones" "all" {
  network_type = "Vpc"
}

data "alicloud_vswitches" "default" {
  vpc_id = "${var.vpc_id}"
}

// Get cluster bootstrap token
data "external" "token" {
  depends_on = ["alicloud_cs_managed_kuberentes.this"]

  program = ["aliyun", "cs", "POST", "/clusters/${alicloud_cs_managed_kuberentes.this.id}/token", "--body", "'{\"is_permanently\": true}'"]
}

data "template_file" "userdata" {
  template = "${file("${path.module}/templates/user_data.sh.tpl")}"
  count    = "${length(var.worker_groups)}"

  vars {
    pre_userdata   = "${lookup(var.worker_groups[count.index], "pre_userdata", var.group_default["pre_userdata"])}"
    post_userdata  = "${lookup(var.worker_groups[count.index], "post_userdata", var.group_default["post_userdata"])}"
    open_api_token = "${data.external.token.token}"
  }
}
