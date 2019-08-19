resource "null_resource" "wait-tiller-ready" {
  depends_on = [var.kubeconfig_filename, var.wait_on_resource]

  provisioner "local-exec" {
    working_dir = path.cwd
    command     = <<EOS
until helm ls; do
  echo "Wait tiller ready"
  sleep 5
done
EOS
    environment = {
      KUBECONFIG = var.kubeconfig_filename
    }
  }
}

data "helm_repository" "pingcap" {
  depends_on = [null_resource.wait-tiller-ready]
  name = "pingcap"
  url = "https://charts.pingcap.org/"
}

resource "helm_release" "tidb-cluster" {
  depends_on = [null_resource.wait-tiller-ready]

  repository = data.helm_repository.pingcap.name
  chart = "tidb-cluster"
  version = var.tidb_cluster_chart_version
  namespace = var.cluster_name
  name = var.cluster_name
  wait = false

  values = [
    var.base_values,
    var.override_values
  ]

  set {
    name = "pd.image"
    value = "pingcap/pd:${var.cluster_version}"
  }
  set {
    name = "pd.replicas"
    value = var.pd_count
  }
  set {
    name = "pd.nodeSelector.dedicated"
    value = "${var.cluster_name}-pd"
  }
  set {
    name = "pd.tolerations[0].key"
    value = "dedicated"
  }
  set {
    name = "pd.tolerations[0].value"
    value = "${var.cluster_name}-pd"
  }
  set {
    name = "pd.tolerations[0].operator"
    value = "Equal"
  }
  set {
    name = "pd.tolerations[0].effect"
    value = "NoSchedule"
  }
  set {
    name = "tikv.image"
    value = "pingcap/tikv:${var.cluster_version}"
  }
  set {
    name = "tikv.replicas"
    value = var.tikv_count
  }
  set {
    name = "tikv.nodeSelector.dedicated"
    value = "${var.cluster_name}-tikv"
  }
  set {
    name = "tikv.tolerations[0].key"
    value = "dedicated"
  }
  set {
    name = "tikv.tolerations[0].value"
    value = "${var.cluster_name}-tikv"
  }
  set {
    name = "tikv.tolerations[0].operator"
    value = "Equal"
  }
  set {
    name = "tikv.tolerations[0].effect"
    value = "NoSchedule"
  }
  set {
    name = "tidb.image"
    value = "pingcap/tidb:${var.cluster_version}"
  }
  set {
    name = "tidb.replicas"
    value = var.tidb_count
  }
  set {
    name = "tidb.nodeSelector.dedicated"
    value = "${var.cluster_name}-tidb"
  }
  set {
    name = "tidb.tolerations[0].key"
    value = "dedicated"
  }
  set {
    name = "tidb.tolerations[0].value"
    value = "${var.cluster_name}-tidb"
  }
  set {
    name = "tidb.tolerations[0].operator"
    value = "Equal"
  }
  set {
    name = "tidb.tolerations[0].effect"
    value = "NoSchedule"
  }
}

resource "null_resource" "wait-tidb-ready" {
  depends_on = [helm_release.tidb-cluster]

  provisioner "local-exec" {
    working_dir = path.cwd
    command = <<EOS
until kubectl get po -n ${var.cluster_name} -lapp.kubernetes.io/component=tidb | grep Running; do
  echo "Wait TiDB pod running"
  sleep 5
done
EOS
    interpreter = var.local_exec_interpreter
    environment = {
      KUBECONFIG = var.kubeconfig_filename
    }
  }
}
