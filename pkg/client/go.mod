module github.com/pingcap/tidb-operator/pkg/client

go 1.13

require (
	github.com/pingcap/tidb-operator/pkg/apis v0.0.0
	k8s.io/apimachinery v0.19.14
	k8s.io/client-go v0.19.14
)

replace github.com/pingcap/tidb-operator/pkg/apis => ../apis
