module "demo-cluster" {
  source                                 = "./tidb-cluster"
  eks_info = module.eks.eks_info
  subnets = split(
    ",",
    var.create_vpc ? join(",", module.vpc.private_subnets) : join(",", var.subnets),
  )

  cluster_name                  = "demo-cluster"
  cluster_version               = "v3.0.0-rc.2"
  ssh_key_name                  = module.key-pair.key_name
  pd_count                      = 1
  pd_instance_type              = "t2.xlarge"
  tikv_count                    = 1
  tikv_instance_type            = "t2.xlarge"
  tidb_count                    = 1
  tidb_instance_type            = "t2.xlarge"
  monitor_instance_type         = "t2.xlarge"
  monitor_storage_size          = "100Gi"
  monitor_enable_anonymous_user = true
  override_values               = "values/default.yaml"
}

module "test-cluster" {
  source                                 = "./tidb-cluster"
  eks_info = module.eks.eks_info
  subnets = split(
    ",",
    var.create_vpc ? join(",", module.vpc.private_subnets) : join(",", var.subnets),
  )

  cluster_name                  = "test-cluster"
  cluster_version               = "v3.0.0-rc.1"
  ssh_key_name                  = module.key-pair.key_name
  pd_count                      = 1
  pd_instance_type              = "t2.xlarge"
  tikv_count                    = 1
  tikv_instance_type            = "t2.xlarge"
  tidb_count                    = 1
  tidb_instance_type            = "t2.xlarge"
  monitor_instance_type         = "t2.xlarge"
  monitor_storage_size          = "100Gi"
  monitor_enable_anonymous_user = true
  override_values               = "values/default.yaml"
}

module "prod-cluster" {
  source                                 = "./tidb-cluster"
  eks_info = module.eks.eks_info
  subnets = ["subnet-0043bd7c0ce42020b"]
  # subnets = split(
  #   ",",
  #   var.create_vpc ? join(",", module.vpc.private_subnets) : join(",", var.subnets),
  # )

  cluster_name                  = "prod-cluster"
  cluster_version               = "v3.0.0-rc.1"
  ssh_key_name                  = module.key-pair.key_name
  pd_count                      = 1
  pd_instance_type              = "t2.xlarge"
  tikv_count                    = 3
  tikv_instance_type            = "t2.xlarge"
  tidb_count                    = 1
  tidb_instance_type            = "t2.xlarge"
  monitor_instance_type         = "t2.xlarge"
  monitor_storage_size          = "100Gi"
  monitor_enable_anonymous_user = true
  override_values               = "values/default.yaml"
}
