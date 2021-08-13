module github.com/pingcap/tidb-operator/pkg/client

go 1.13

require (
	github.com/pingcap/tidb-operator/pkg/apis v0.0.0
	k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
	k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90
)

replace github.com/pingcap/tidb-operator/pkg/apis => ../apis
