module github.com/pingcap/tidb-operator/pkg/apis

go 1.16

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/Masterminds/semver v1.4.2
	github.com/aws/aws-sdk-go v1.44.72 // indirect
	github.com/go-openapi/spec v0.19.3
	github.com/google/go-cmp v0.5.8
	github.com/google/gofuzz v1.1.0
	github.com/mohae/deepcopy v0.0.0-20170603005431-491d3605edfb
	github.com/onsi/gomega v1.10.2
	github.com/pingcap/TiProxy/lib v0.0.0-20221024031838-544b0cc89dd9
	github.com/pingcap/errors v0.11.4
	github.com/prometheus/common v0.26.0
	github.com/prometheus/prometheus v1.8.2
	k8s.io/api v0.19.16
	k8s.io/apiextensions-apiserver v0.19.16
	k8s.io/apimachinery v0.19.16
	k8s.io/klog/v2 v2.3.0
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	k8s.io/utils v0.0.0-20200912215256-4140de9c8800
	sigs.k8s.io/yaml v1.2.0
)
