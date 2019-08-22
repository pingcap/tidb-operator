variable "install_prometheus_operator" {
   description = "Whether installing promethes operator"
   default     = true
}

variable "install_kubernetes_monitor" {
   description = "Whether installing kubernetes cluster monitoring"
   default     = true
}

variable "kubeconfig" {
    description = "kubernetes sensitive configuration"
}

variable "filename" {
    description = "kubernetes configuration filename"
}