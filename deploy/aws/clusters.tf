resource "local_file" "kubeconfig" {
  depends_on        = [module.tidb-operator.eks]
  sensitive_content = module.tidb-operator.eks.kubeconfig
  filename          = module.tidb-operator.eks.kubeconfig_filename
}

# The helm provider for TiDB clusters must be configured in the top level, otherwise removing clusters will failed due to
# the helm provider configuration is removed too.
provider "helm" {
  alias    = "eks"
  insecure = true
  # service_account = "tiller"
  install_tiller = false # currently this doesn't work, so we install tiller in the local-exec provisioner. See https://github.com/terraform-providers/terraform-provider-helm/issues/148
  kubernetes {
    config_path = local_file.kubeconfig.filename
  }
}

# TiDB cluster declaration example
#module "example-cluster" {
#  source   = "./tidb-cluster"
#  eks_info = local.default_eks
#  subnets = local.default_subnets
#
#  # NOTE: cluster_name cannot be changed after creation
#  cluster_name                  = "demo-cluster"
#  cluster_version               = "v3.0.5"
#  ssh_key_name                  = module.key-pair.key_name
#  pd_count                      = 1
#  pd_instance_type              = "t2.xlarge"
#  tikv_count                    = 1
#  tikv_instance_type            = "t2.xlarge"
#  tidb_count                    = 1
#  tidb_instance_type            = "t2.xlarge"
#  monitor_instance_type         = "t2.xlarge"
#  # yaml file that passed to helm to customize the release
#  override_values               = file("values/example.yaml")
#}

module "default-cluster" {
  providers = {
    helm = "helm.eks"
  }
  source  = "../modules/aws/tidb-cluster"
  eks     = local.eks
  subnets = local.subnets
  region  = var.region

  cluster_name          = var.default_cluster_name
  cluster_version       = var.default_cluster_version
  ssh_key_name          = module.key-pair.key_name
  pd_count              = var.default_cluster_pd_count
  pd_instance_type      = var.default_cluster_pd_instance_type
  tikv_count            = var.default_cluster_tikv_count
  tikv_instance_type    = var.default_cluster_tikv_instance_type
  tidb_count            = var.default_cluster_tidb_count
  tidb_instance_type    = var.default_cluster_tidb_instance_type
  monitor_instance_type = var.default_cluster_monitor_instance_type
  override_values       = file("default-cluster.yaml")
}
