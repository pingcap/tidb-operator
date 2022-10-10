module github.com/pingcap/tidb-operator/pkg/client

go 1.16

require (
	github.com/pingcap/tidb-operator/pkg/apis v1.4.0-alpha.2
	github.com/pingcap/TiProxy/lib v0.0.0-20221010031757-343299153484 // indirect
	k8s.io/apimachinery v0.19.16
	k8s.io/client-go v0.19.16
)

replace github.com/pingcap/tidb-operator/pkg/apis => ../apis
