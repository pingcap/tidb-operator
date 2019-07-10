module "eks" {
  source = "terraform-aws-modules/eks/aws"

  cluster_name       = var.eks_name
  cluster_version    = var.eks_version
  vpc_id             = var.vpc_id
  config_output_path = var.config_output_path
  subnets            = var.subnets

  tags = {
    app = "tidb"
  }

  worker_groups = [
    {
      name                 = "${var.eks_name}-control"
      key_name             = var.ssh_key_name
      instance_type        = var.default_worker_group_instance_type
      public_ip            = false
      asg_desired_capacity = var.default_worker_group_instance_count
      asg_max_size         = var.default_worker_group_instance_count + 2
    },
  ]
}

# kubernetes and helm providers rely on EKS, but terraform provider doesn't support depends_on
# follow this link https://github.com/hashicorp/terraform/issues/2430#issuecomment-370685911
# we have the following hack
resource "local_file" "kubeconfig" {
  depends_on        = [module.eks]
  sensitive_content = module.eks.kubeconfig
  filename          = module.eks.kubeconfig_filename
}

provider "helm" {
  alias    = "initial"
  insecure = true
  # service_account = "tiller"
  install_tiller = false # currently this doesn't work, so we install tiller in the local-exec provisioner. See https://github.com/terraform-providers/terraform-provider-helm/issues/148
  kubernetes {
    config_path = local_file.kubeconfig.filename
  }
}

resource "null_resource" "setup-env" {
  depends_on = [local_file.kubeconfig]

  provisioner "local-exec" {
    working_dir = path.module
    command     = <<EOS
echo "${local_file.kubeconfig.sensitive_content}" > kube_config.yaml
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/${var.operator_version}/manifests/crd.yaml
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/${var.operator_version}/manifests/tiller-rbac.yaml
kubectl apply -f manifests/local-volume-provisioner.yaml
kubectl apply -f manifests/gp2-storageclass.yaml
helm init --service-account tiller --upgrade --wait
until helm ls; do
  echo "Wait tiller ready"
  sleep 5
done
rm kube_config.yaml
EOS
    environment = {
      KUBECONFIG = "kube_config.yaml"
    }
  }
}

data "helm_repository" "pingcap" {
  provider = "helm.initial"
  depends_on = ["null_resource.setup-env"]
  name = "pingcap"
  url = "http://charts.pingcap.org/"
}

resource "helm_release" "tidb-operator" {
  provider = "helm.initial"
  depends_on = ["null_resource.setup-env"]

  repository = data.helm_repository.pingcap.name
  chart = "tidb-operator"
  version = var.operator_version
  namespace = "tidb-admin"
  name = "tidb-operator"
  values = [var.operator_helm_values]
}



