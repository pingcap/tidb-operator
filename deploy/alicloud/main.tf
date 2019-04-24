provider "alicloud" {
  access_key = "${var.alicloud_access_key}"
  secret_key = "${var.alicloud.secret.key}"
  region     = "${var.region}"
}

module "kubernetes" {
  source  = "aliyun/kubernetes/alicloud"
  version = "1.0.0"
}
