# TiDB dashboard 
With Grafana v5.x or later, we can use provisioning feature to statically provision datasources and dashboards. No need to use scripts to configure Grafana.

The JSON files in dashboards are copied from [tidb-ansible](https://github.com/pingcap/tidb-ansible/tree/release-3.0/scripts), and need to replace variables in the json file(It was did by python file before).

In grafana, the symbol "{{xx}}" represents a variable, but it conflict with helm grammar, "{{" should be replaced with "{{`{{`}}"

