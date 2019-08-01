//data "external" "tidb_ilb_ip" {
//  depends_on = [null_resource.deploy-tidb-cluster]
//  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n tidb tidb-cluster-tidb -o json | jq '.status.loadBalancer.ingress[0]'"]
//}
//
//data "external" "monitor_ilb_ip" {
//  depends_on = [null_resource.deploy-tidb-cluster]
//  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n tidb tidb-cluster-grafana -o json | jq '.status.loadBalancer.ingress[0]'"]
//}
//
//data "external" "tidb_port" {
//  depends_on = [null_resource.deploy-tidb-cluster]
//  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n tidb tidb-cluster-tidb -o json | jq '.spec.ports | .[] | select( .name == \"mysql-client\") | {port: .port|tostring}'"]
//}
//
//data "external" "monitor_port" {
//  depends_on = [null_resource.deploy-tidb-cluster]
//  program    = ["bash", "-c", "kubectl --kubeconfig ${local.kubeconfig} get svc -n tidb tidb-cluster-grafana -o json | jq '.spec.ports | .[] | select( .name == \"grafana\") | {port: .port|tostring}'"]
//}
//
