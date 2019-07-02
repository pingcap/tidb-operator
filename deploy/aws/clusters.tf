# TiDB cluster declaration example
#module "example-cluster" {
#  source   = "./tidb-cluster"
#  eks_info = local.default_eks
#  subnets = local.default_subnets
#
#  # NOTE: cluster_name cannot be changed after creation
#  cluster_name                  = "demo-cluster"
#  cluster_version               = "v3.0.0"
#  ssh_key_name                  = module.key-pair.key_name
#  pd_count                      = 1
#  pd_instance_type              = "t2.xlarge"
#  tikv_count                    = 1
#  tikv_instance_type            = "t2.xlarge"
#  tidb_count                    = 1
#  tidb_instance_type            = "t2.xlarge"
#  monitor_instance_type         = "t2.xlarge"
#  # yaml file that passed to helm to customize the release
#  override_values               = "values/default.yaml"
#}


module "default-cluster" {
  source   = "./tidb-cluster"
  eks_info = module.eks
  subnets  = local.default_subnets

  cluster_name          = var.default_cluster_name
  cluster_version       = var.default_cluster_version
  ssh_key_name          = module.key-pair.key_name
  pd_count              = var.default_cluster_pd_count
  pd_instance_type      = var.default_cluster_pd_instance_type
  tikv_count            = var.default_cluster_tikv_count
  tikv_instance_type    = var.default_cluster_tidb_instance_type
  tidb_count            = var.default_cluster_tidb_count
  tidb_instance_type    = var.default_cluster_tidb_instance_type
  monitor_instance_type = var.default_cluster_monitor_instance_type
  override_values       = "values/default.yaml"
}
