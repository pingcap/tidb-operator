region             = "westus2"
resource_group     = "tidb-k8s"
availability_zones = [1,2,3]

pd_instance_type   = "standard_f16s_v2"
pd_count           = 3
tikv_instance_type = "standard_e16ds_v4"
tikv_count         = 3
tidb_instance_type = "standard_f16s_v2"
tidb_count         = 3
