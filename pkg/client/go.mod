module github.com/pingcap/tidb-operator/pkg/client

go 1.13

require (
	github.com/pingcap/tidb-operator/pkg/apis v0.0.0
	k8s.io/apimachinery v0.16.15
	k8s.io/client-go v0.16.15
)

replace github.com/pingcap/tidb-operator/pkg/apis => ../apis
