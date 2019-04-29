variable "ALICLOUD_ACCESS_KEY" {}
variable "ALICLOUD_SECRET_KEY" {}
variable "ALICLOUD_REGION" {}

provider "alicloud" {
  alias      = "this"
  region     = "${var.ALICLOUD_REGION}"
  access_key = "${var.ALICLOUD_ACCESS_KEY}"
  secret_key = "${var.ALICLOUD_SECRET_KEY}"
}

locals {
  credential_path               = "${path.module}/credentials"
  kubeconfig                    = "${local.credential_path}/kubeconfig_${var.cluster_name}"
  key_file                      = "${local.credential_path}/worker-node-key.pem"
  bastion_key_file              = "${local.credential_path}/bastion-key.pem"
  tidb_cluster_values_path      = "${path.module}/rendered/tidb-cluster-values.yaml"
  local_volume_provisioner_path = "${path.module}/rendered/local-volume-provisioner.yaml"
}

// AliCloud resource requires path existing
resource "null_resource" "prepare-dir" {
  provisioner "local-exec" {
    command = "mkdir -p ${local.credential_path}"
  }
}

module "ack" {
  source  = "./ack"
  version = "1.0.2"

  providers = {
    alicloud = "alicloud.this"
  }

  # TODO: support non-public apiserver
  cluster_name     = "${var.cluster_name}"
  public_apiserver = true
  kubeconfig_file  = "${local.kubeconfig}"
  key_file         = "${local.key_file}"
  vpc_cidr         = "${var.vpc_cidr}"
  k8s_pod_cidr     = "${var.k8s_pod_cidr}"
  k8s_service_cidr = "${var.k8s_service_cidr}"
  vpc_cidr_newbits = "${var.vpc_cidr_newbits}"
  vpc_id           = "${var.vpc_id}"
  group_id         = "${var.group_id}"

  worker_groups = [
    {
      name          = "pd_worker_group"
      instance_type = "${data.alicloud_instance_types.pd.instance_types.0.id}"
      min_size      = "${var.pd_count}"
      max_size      = "${var.pd_count}"
      node_taints   = "dedicated=pd:NoScheduler"
      node_labels   = "dedicated=pd"
      post_userdata = "${file("userdata/pd-userdata.sh")}"
    },
    {
      name          = "tikv_worker_group"
      instance_type = "${data.alicloud_instance_types.tikv.instance_types.0.id}"
      min_size      = "${var.tikv_count}"
      max_size      = "${var.tikv_count}"
      node_taints   = "dedicated=tikv:NoScheduler"
      node_labels   = "dedicated=tikv"
      post_userdata = "${file("userdata/tikv-userdata.sh")}"
    },
    {
      name          = "tidb_worker_group"
      instance_type = "${var.tidb_instance_type != "" ? var.tidb_instance_type : data.alicloud_instance_types.tidb.instance_types.0.id}"
      min_size      = "${var.tidb_count}"
      max_size      = "${var.tidb_count}"
      node_taints   = "dedicated=tidb:NoScheduler"
      node_labels   = "dedicated=tidb"
    },
    {
      name          = "monitor_worker_group"
      instance_type = "${var.monitor_intance_type != "" ? var.monitor_intance_type : data.alicloud_instance_types.monitor.instance_types.0.id}"
      min_size      = 1
      max_size      = 1
    },
  ]
}

// Workaround: ACK does not support customize node RAM role, access key is the only way get local volume provisioner working
// TODO: use STS when upstream get this fixed
resource "local_file" "local-volume-provisioner" {
  depends_on = ["data.template_file.local-volume-provisioner"]
  filename   = "${local.local_volume_provisioner_path}"
  content    = "${data.template_file.local-volume-provisioner.rendered}"
}

resource "local_file" "tidb-cluster-values" {
  depends_on = ["data.template_file.tidb-cluster-values"]
  content    = "${data.template_file.tidb-cluster-values.rendered}"
  filename   = "${local.tidb_cluster_values_path}"
}

resource "null_resource" "setup-env" {
  depends_on = ["module.ack", "local_file.local-volume-provisioner"]

  provisioner "local-exec" {
    command = <<EOS
kubectl apply -f manifests/crd.yaml
kubectl apply -f rendered/local-volume-provisioner.yaml
helm init
until helm ls; do
  echo "Wait tiller ready"
done
helm upgrade --install tidb-operator ${path.module}/charts/tidb-operator --namespace=tidb-admin --set scheduler.kubeSchedulerImageName=gcr.akscn.io/google_containers/hyperkube
EOS

    environment = {
      KUBECONFIG = "${local.kubeconfig}"
    }
  }
}

# Workaround: Terraform cannot specify provider dependency, so we take over kubernetes and helm stuffs,
# But we cannot ouput kubernetes and helm resources in this way.
# TODO: use helm and kubernetes provider when upstream get this fixed
resource "null_resource" "deploy-tidb-cluster" {
  depends_on = ["null_resource.setup-env"]

  provisioner "local-exec" {
    command = <<EOS
helm upgrade --install tidb-cluster ${path.module}/charts/tidb-cluster --namespace=tidb -f ${local.tidb_cluster_values_path}
echo "TiDB cluster setup complete!"
EOS
  }
}
