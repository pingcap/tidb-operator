variable "install_prometheus_operator" {
   description = "Whether installing promethes operator"
   default     = true
}

variable "install_kubernetes_monitor" {
   description = "Whether installing kubernetes cluster monitoring"
   default     = true
}

variable "kubeconfig_file" {
  description = "The path that kubeconfig file write to, default to $$${path.module}/kubeconfig if empty."
  default     = ""
}