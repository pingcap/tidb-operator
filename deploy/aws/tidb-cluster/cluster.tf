resource "local_file" "kubeconfig" {
    content     = var.eks_info.kubeconfig
    filename = "${path.module}/kubeconfig_${var.cluster_name}.yaml"
}

resource "null_resource" "deploy-cluster" {
  depends_on = [local_file.kubeconfig]

  provisioner "local-exec" {
    working_dir = path.module

    command = <<EOT
helm upgrade --install ${var.cluster_name} pingcap/tidb-cluster \
--version=${var.operator_version} --namespace=${var.cluster_name} \
--set enableConfigMapRollout=true \
--set pd.image=pingcap/pd:${var.cluster_version} \
--set pd.replicas=${var.pd_count} \
--set pd.storageClassName=ebs-gp2 \
--set pd.nodeSelector.dedicated=${var.cluster_name}-pd \
--set pd.tolerations[0].key=dedicated \
--set pd.tolerations[0].value=${var.cluster_name}-pd \
--set pd.tolerations[0].operator=Equal \
--set pd.tolerations[0].effect=NoSchedule \
--set tikv.image=pingcap/tikv:${var.cluster_version} \
--set tikv.replicas=${var.tikv_count} \
--set tikv.storageClassName=ebs-gp2 \
--set tikv.nodeSelector.dedicated=${var.cluster_name}-tikv \
--set tikv.tolerations[0].key=dedicated \
--set tikv.tolerations[0].value=${var.cluster_name}-tikv \
--set tikv.tolerations[0].operator=Equal \
--set tikv.tolerations[0].effect=NoSchedule \
--set tidb.image=pingcap/tidb:${var.cluster_version} \
--set tidb.replicas=${var.tidb_count} \
--set tidb.nodeSelector.dedicated=${var.cluster_name}-tidb \
--set tidb.tolerations[0].key=dedicated \
--set tidb.tolerations[0].value=${var.cluster_name}-tidb \
--set tidb.tolerations[0].operator=Equal \
--set tidb.tolerations[0].effect=NoSchedule \
--set tidb.service.type=LoadBalancer \
--set tidb.service.annotations."service\.beta\.kubernetes\.io/aws-load-balancer-internal"="0.0.0.0/0" \
--set tidb.service.annotations."service\.beta\.kubernetes\.io/aws-load-balancer-type"=nlb \
--set monitor.grafana.service.type=LoadBalancer \
--set monitor.persistent=true \
--set monitor.storageClassName=ebs-gp2 \
--set monitor.storage=${var.monitor_storage_size} \
--set monitor.grafana.config.GF_AUTH_ANONYMOUS_ENABLED=${var.monitor_enable_anonymous_user} \
-f ${var.override_values} \
--wait
EOT

    interpreter = var.local_exec_interpreter
    environment = {
      KUBECONFIG = "kubeconfig_${var.cluster_name}.yaml"
    }
  }

  triggers = {
    # timestamp = timestamp()
    cluster_name = var.cluster_name
    operator_version = var.operator_version
    pd_count = var.pd_count
    tikv_count = var.tikv_count
    tidb_count = var.tidb_count
    monitor_storage_size = var.monitor_storage_size
    monitor_enable_anonymous_user = var.monitor_enable_anonymous_user
    kubeconfig = var.eks_info.kubeconfig
  }
}

# resource "null_resource" "destroy-cluster" {
#   depends_on = [local_file.kubeconfig]

#   provisioner "local-exec" {
#     when = "destroy"
#     command = <<EOT
# helm delete ${var.cluster_name} --purge
# kubectl get pvc -n ${var.cluster_name} -o jsonpath='{.items[*].spec.volumeName}'|fmt -1 | xargs -I {} kubectl patch pv {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
# kubectl delete pvc -n ${var.cluster_name} --all
# EOT

#     interpreter = var.local_exec_interpreter
#     environment = {
#       KUBECONFIG = "kubeconfig_${var.cluster_name}.yaml"
#     }
#   }
# }
