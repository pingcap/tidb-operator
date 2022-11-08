module github.com/pingcap/tidb-operator/pkg/client

go 1.16

require (
	github.com/aws/aws-sdk-go v1.44.72 // indirect
	github.com/pingcap/tidb-operator/pkg/apis v1.4.0-beta.1
	github.com/stretchr/testify v1.8.0 // indirect
	k8s.io/apimachinery v0.19.16
	k8s.io/client-go v0.19.16
)

replace github.com/pingcap/tidb-operator/pkg/apis => ../apis
