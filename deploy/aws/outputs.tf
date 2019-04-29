output "region" {
  description = "AWS region."
  value = "${var.region}"
}

output "eks_version" {
  description = "The Kubernetes server version for the EKS cluster."
  value = "${var.k8s_version}"
}

output "tidb_version" {
  description = "tidb cluster version"
  value = "${var.tidb_version}"
}

output "eks_endpoint" {
  description = "Endpoint for EKS control plane."
  value = "${module.eks.cluster_endpoint}"
}

#output "tidb_dns" {
#  description = "tidb service dns name"
#  value = "${data.kubernetes_service.tidb.load_balancer_ingress.0.hostname}"
#}

output "tidb_dns" {
  description = "tidb service dns name"
  value = "${data.external.tidb_service.result["hostname"]}"
}

output "tidb_port" {
  description = "tidb service port"
  value = "4000"
}

#output "monitor_endpoint" {
#  description = "monitor service endpoint"
#  value = "http://${data.kubernetes_service.monitor.load_balancer_ingress.0.hostname}:3000"
#}

output "monitor_endpoint" {
  description = "monitor service endpoint"
  value = "http://${data.external.monitor_service.result["hostname"]}:3000"
}

output "bastion_ip" {
  description = "Bastion IP address"
  value = "${module.ec2.public_ip}"
}
