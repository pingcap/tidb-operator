resource "local_file" "kubeconfig" {
  depends_on        = [module.tidb-operator]
  sensitive_content = module.tidb-operator.kubeconfig
  filename          = module.tidb-operator.kubeconfig_filename
}

provider "helm" {
  alias          = "k8s"
  insecure       = true
  install_tiller = false
  kubernetes {
    config_path = local_file.kubeconfig.filename
  }
}

# TiDB cluster declaration example
#module "example-cluster" {
#  providers = {
#    helm = "helm.k8s"
#  }
#  source = "./tidb-cluster"
#
#  cluster_name               = "example-cluster"
#  ack                        = module.tidb-operator
#  pd_count                   = 3
#  pd_instance_type           = "ecs.g5.large"
#  tikv_count                 = 3
#  tikv_instance_type         = "ecs.i2.2xlarge"
#  tidb_count                 = 2
#  tidb_instance_type         = "ecs.c5.4xlarge"
#  monitor_instance_type      = "ecs.c5.xlarge"
#  tidb_cluster_chart_version = "v1.0.0-beta.3"
#  override_values            = file("example.yaml")
#}

module "default-cluster" {
  providers = {
    helm = "helm.k8s"
  }
  source = "./tidb-cluster"

  cluster_name               = "my-cluster"
  ack                        = module.tidb-operator
  pd_count                   = 3
  pd_instance_type           = "ecs.g5.large"
  tikv_count                 = 3
  tikv_instance_type         = "ecs.i2.2xlarge"
  tidb_count                 = 2
  tidb_instance_type         = "ecs.c5.4xlarge"
  monitor_instance_type      = "ecs.c5.xlarge"
  tidb_cluster_chart_version = "v1.0.0-beta.3"
  override_values            = file("default-cluster.yaml")
}
