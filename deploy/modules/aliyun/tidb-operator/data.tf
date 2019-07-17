data "alicloud_zones" "all" {
  network_type = "Vpc"
}

data "alicloud_vswitches" "default" {
  vpc_id = var.vpc_id
}

data "alicloud_instance_types" "default" {
  availability_zone = data.alicloud_zones.all.zones[0]["id"]
  cpu_core_count    = var.default_worker_cpu_core_count
}

# Workaround map to list transformation, see stackoverflow.com/questions/43893295
data "template_file" "vswitch_id" {
  count    = var.vpc_id == "" ? 0 : length(data.alicloud_vswitches.default.vswitches)
  template = data.alicloud_vswitches.default.vswitches[count.index]["id"]
}

# Get cluster bootstrap token
data "external" "token" {
  depends_on = [alicloud_cs_managed_kubernetes.k8s]

  # Terraform use map[string]string to unmarshal the result, transform the json to conform
  program = ["bash", "-c", "aliyun --region ${var.region} cs POST /clusters/${alicloud_cs_managed_kubernetes.k8s.id}/token --body '{\"is_permanently\": true}' | jq \"{token: .token}\""]
}

data "template_file" "local-volume-provisioner" {
  template = file("${path.module}/templates/local-volume-provisioner.yaml.tpl")

  vars = {
    access_key_id     = var.access_key
    access_key_secret = var.secret_key
  }
}
