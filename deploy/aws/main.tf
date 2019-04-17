provider "aws" {
  region = "${var.region}"
}

provider "kubernetes" {
  config_path = "${path.module}/credentials/kubeconfig_my-cluster"
}

provider "helm" {
  insecure = true
  kubernetes {
    config_path = "${path.module}/credentials/kubeconfig_my-cluster"
  }
}

resource "aws_key_pair" "k8s_node" {
  key_name_prefix = "${var.cluster_name}"
  public_key = "${file("~/.ssh/id_rsa.pub")}"
}

# module "vpc" {
#   source = "terraform-aws-modules/vpc/aws"
#   version = "1.60.0"
#   name = "${var.cluster_name}"
#   cidr = "${var.vpc_cidr}"
#   create_vpc = "${var.create_vpc}"
#   azs = ["${data.aws_availability_zones.available.names[0]}", "${data.aws_availability_zones.available.names[1]}", "${data.aws_availability_zones.available.names[2]}"]
#   private_subnets = "${var.private_subnets}"
#   public_subnets = "${var.public_subnets}"
#   enable_nat_gateway = true
#   single_nat_gateway = true

#   # The following tags are required for ELB
#   private_subnet_tags = {
#     "kubernetes.io/cluster/${var.cluster_name}" = "shared"
#   }
#   public_subnet_tags = {
#     "kubernetes.io/cluster/${var.cluster_name}" = "shared"
#   }
#   vpc_tags = {
#     "kubernetes.io/cluster/${var.cluster_name}" = "shared"
#   }
# }

module "eks" {
  source = "terraform-aws-modules/eks/aws"
  version = "2.3.1"
  cluster_name = "${var.cluster_name}"
  cluster_version = "${var.k8s_version}"
  config_output_path = "credentials/"
  # subnets = ["${module.vpc.private_subnets}"]
  # vpc_id = "${module.vpc.vpc_id}"
  subnets = "${var.subnets}"
  vpc_id = "${var.vpc_id}"

  # instance types: https://aws.amazon.com/ec2/instance-types/
  # instance prices: https://aws.amazon.com/ec2/pricing/on-demand/

  worker_groups = [
    {
      # pd
      name = "pd_worker_group"
      key_name = "${aws_key_pair.k8s_node.key_name}"
      # WARNING: if you change instance type, you must also modify the corresponding disk mounting in pd-userdata.sh script
      # instance_type = "c5d.xlarge" # 4c, 8G, 100G NVMe SSD
      instance_type = "m5d.xlarge" # 4c, 16G, 150G NVMe SSD
      root_volume_size = "50" # rest NVMe disk for PD data
      public_ip = false
      kubelet_extra_args = "--register-with-taints=dedicated=pd:NoSchedule --node-labels=dedicated=pd"
      asg_desired_capacity = "${var.pd_count}"
      asg_max_size  = "${var.pd_count + 2}"
      additional_userdata = "${file("pd-userdata.sh")}"
    },
    { # tikv
      name = "tikv_worker_group"
      key_name = "${aws_key_pair.k8s_node.key_name}"
      # WARNING: if you change instance type, you must also modify the corresponding disk mounting in tikv-userdata.sh script
      instance_type = "i3.2xlarge" # 8c, 61G, 1.9T NVMe SSD
      root_volume_type = "gp2"
      root_volume_size = "100"
      public_ip = false
      kubelet_extra_args = "--register-with-taints=dedicated=tikv:NoSchedule --node-labels=dedicated=tikv"
      asg_desired_capacity = "${var.tikv_count}"
      asg_max_size = "${var.tikv_count + 2}"
      additional_userdata = "${file("tikv-userdata.sh")}"
    },
    { # tidb
      name = "tidb_worker_group"
      key_name = "${aws_key_pair.k8s_node.key_name}"
      instance_type = "c4.4xlarge" # 16c, 30G
      root_volume_type = "gp2"
      root_volume_size = "100"
      public_ip = false
      kubelet_extra_args = "--register-with-taints=dedicated=tidb:NoSchedule --node-labels=dedicated=tidb"
      asg_desired_capacity = "${var.tidb_count}"
      asg_max_size = "${var.tidb_count + 2}"
    },
    { # monitor
      name = "monitor_worker_group"
      key_name = "${aws_key_pair.k8s_node.key_name}"
      instance_type = "c5.xlarge" # 4c, 8G
      root_volume_type = "gp2"
      root_volume_size = "100"
      public_ip = false
      asg_desired_capacity = 1
      asg_max_size = 3
    }
  ]

  worker_group_count = "4"

  tags = {
    app = "tidb"
  }
}

resource "null_resource" "setup-env" {
  depends_on = ["module.eks"]

  provisioner "local-exec" {
    working_dir = "${path.module}"
    command = <<EOS
kubectl apply -f manifests/crd.yaml
kubectl apply -f manifests/local-volume-provisioner.yaml
kubectl apply -f manifests/gp2-storageclass.yaml
kubectl apply -f manifests/tiller-rbac.yaml
helm init --service-account tiller --upgrade --wait
EOS
    environment = {
      KUBECONFIG = "${path.module}/credentials/kubeconfig_my-cluster"
    }
  }
}

resource "helm_release" "tidb-operator" {
  depends_on = ["null_resource.setup-env"]
  name = "tidb-operator"
  namespace = "tidb-admin"
  chart = "${path.module}/charts/tidb-operator"
}

resource "helm_release" "tidb-cluster" {
  depends_on = ["helm_release.tidb-operator"]
  name = "tidb-cluster"
  namespace = "tidb"
  chart = "${path.module}/charts/tidb-cluster"
  values = [
    "${data.template_file.tidb_cluster_values.rendered}"
  ]
}
