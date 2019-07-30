provider "helm" {
  alias = "gke"
  insecure = true
  install_tiller = false
  kubernetes {
    config_path = local.kubeconfig
  }
}