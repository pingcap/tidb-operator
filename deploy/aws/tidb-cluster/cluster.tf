#resource "local_file" "kubeconfig_for_cleanup" {
#  content  = var.eks_info.kubeconfig
#  filename = "${var.eks_info.kubeconfig_file}_for_${var.cluster_name}_cleanup"
#}

resource "null_resource" "deploy-cluster" {

  provisioner "local-exec" {
    # EKS writes kube_config to path.cwd/kubeconfig_file
    # Helm values files are managed in path.cwd
    working_dir = path.cwd

    command = <<EOT
until helm ls; do
  echo "Wait tiller ready"
  sleep 5
done
helm upgrade --install ${var.cluster_name} pingcap/tidb-cluster \
--version=${var.tidb_cluster_chart_version} --namespace=${var.cluster_name} \
--set pd.image=pingcap/pd:${var.cluster_version} \
--set pd.replicas=${var.pd_count} \
--set pd.storageClassName=${var.pd_storage_class} \
--set pd.nodeSelector.dedicated=${var.cluster_name}-pd \
--set pd.tolerations[0].key=dedicated \
--set pd.tolerations[0].value=${var.cluster_name}-pd \
--set pd.tolerations[0].operator=Equal \
--set pd.tolerations[0].effect=NoSchedule \
--set tikv.image=pingcap/tikv:${var.cluster_version} \
--set tikv.replicas=${var.tikv_count} \
--set tikv.storageClassName=${var.tikv_storage_class} \
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
--set monitor.persistent=true \
--set monitor.storageClassName=${var.monitor_storage_class} \
-f ${var.override_values} \
--wait
EOT

    interpreter = var.local_exec_interpreter
    environment = {
      KUBECONFIG = var.eks_info.kubeconfig_file
    }
  }

  triggers = {
    values_file_content = file(var.override_values)
    cluster_name = var.cluster_name
    pd_count = var.pd_count
    tikv_count = var.tikv_count
    tidb_count = var.tidb_count
    cluster_version = var.cluster_version
    kubeconfig = var.eks_info.kubeconfig
    override_values = var.override_values
    monitor_storage_class = var.monitor_storage_class
    tikv_storage_class = var.tikv_storage_class
    pd_stroage_class = var.pd_storage_class
  }
}

resource "null_resource" "wait-tidb-ready" {
  depends_on = ["null_resource.deploy-cluster"]

  provisioner "local-exec" {
    working_dir = path.cwd
    command = <<EOS
until kubectl get po -n tidb -lapp.kubernetes.io/component=tidb | grep Running; do
  echo "Wait TiDB pod running"
  sleep 5
done
EOS
    environment = {
      KUBECONFIG = var.eks_info.kubeconfig_file
    }
  }
}


#resource "null_resource" "destroy-cluster" {
#  depends_on = [aws_autoscaling_group.workers, null_resource.deploy-cluster]
#
#  provisioner "local-exec" {
#    working_dir = path.cwd
#    when = "destroy"
#
#    # We cannot specify the destruction order of kubeconfig file and this provsioner,
#    # the workaround here is to re-generate the kubeconfig file locally
#    command = <<EOT
#echo "${var.eks_info.kubeconfig}" > kubeconfig_cleanup_${var.cluster_name}
#kubectl delete -n ${var.cluster_name} svc ${var.cluster_name}-pd
#kubectl delete -n ${var.cluster_name} svc ${var.cluster_name}-grafana
#kubectl get pvc -n ${var.cluster_name} -o jsonpath='{.items[*].spec.volumeName}'|fmt -1 | xargs -I {} kubectl patch pv {} -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
#kubectl delete pvc -n ${var.cluster_name} --all
#rm kubeconfig_cleanup_${var.cluster_name}
#EOT
#
#    interpreter = var.local_exec_interpreter
#    environment = {
#      KUBECONFIG = "kubeconfig_cleanup_${var.cluster_name}"
#    }
#  }
#}
