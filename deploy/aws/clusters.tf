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
# module example-cluster {
#   source = "../modules/aws/tidb-cluster"

#   eks = local.eks
#   subnets = local.subnets
#   region  = var.region
#   cluster_name    = "example"

#   ssh_key_name                  = module.key-pair.key_name
#   pd_count                      = 1
#   pd_instance_type              = "c5.large"
#   tikv_count                    = 1
#   tikv_instance_type            = "c5d.large"
#   tidb_count                    = 1
#   tidb_instance_type            = "c4.large"
#   monitor_instance_type         = "c5.large"
#   create_tidb_cluster_release   = false
# }

module "default-cluster" {
  providers = {
    helm = helm.eks
  }
  source  = "../modules/aws/tidb-cluster"
  eks     = local.eks
  subnets = local.subnets
  region  = var.region

  cluster_name                = var.default_cluster_name
  cluster_version             = var.default_cluster_version
  ssh_key_name                = module.key-pair.key_name
  pd_count                    = var.default_cluster_pd_count
  pd_instance_type            = var.default_cluster_pd_instance_type
  tikv_count                  = var.default_cluster_tikv_count
  tikv_instance_type          = var.default_cluster_tikv_instance_type
  tidb_count                  = var.default_cluster_tidb_count
  tidb_instance_type          = var.default_cluster_tidb_instance_type
  monitor_instance_type       = var.default_cluster_monitor_instance_type
  create_tidb_cluster_release = var.create_tidb_cluster_release
  create_tiflash_node_pool    = var.create_tiflash_node_pool
  create_cdc_node_pool        = var.create_cdc_node_pool
  tiflash_count               = var.cluster_tiflash_count
  cdc_count                   = var.cluster_cdc_count
  cdc_instance_type           = var.cluster_cdc_instance_type
  tiflash_instance_type       = var.cluster_tiflash_instance_type
}
