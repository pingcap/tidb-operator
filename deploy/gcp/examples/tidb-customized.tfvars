pd_instance_type   = "n1-standard-2"
tikv_instance_type = "n1-highmem-4"
tidb_instance_type = "n1-standard-8"

# specify tidb version
tidb_version = "3.0.5"

# override tidb cluster values
override_values = <<EOF
pd:
  hostNetwork: true
tikv:
  resources:
     requests:
       cpu: "1"
       memory: 1Gi
       storage: 10Gi
  hostNetwork: true
tidb:
  resources:
    requests:
      cpu: "1"
      memory: 1Gi
  hostNetwork: true
EOF
