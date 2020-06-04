//
// Run hack/pin-deps.sh to change pinned dependency versions.
//

module github.com/pingcap/tidb-operator

go 1.13

require (
	github.com/Azure/go-autorest/autorest/mocks v0.3.0 // indirect
	github.com/BurntSushi/toml v0.3.1
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/Masterminds/semver v1.4.2
	github.com/Microsoft/go-winio v0.4.12 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/aws/aws-sdk-go v1.30.9
	github.com/chai2010/gettext-go v0.0.0-20170215093142-bf70f2a70fb1 // indirect
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/coreos/etcd v3.3.15+incompatible
	github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/elazarl/goproxy v0.0.0-20190421051319-9d40249d3c2f // indirect; indirectload
	github.com/elazarl/goproxy/ext v0.0.0-20190421051319-9d40249d3c2f // indirect
	github.com/emicklei/go-restful v2.9.5+incompatible
	github.com/fatih/color v1.7.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-openapi/loads v0.19.4
	github.com/go-openapi/spec v0.19.3
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/groupcache v0.0.0-20181024230925-c65c006176ff // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/go-cmp v0.3.1
	github.com/gophercloud/gophercloud v0.3.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.1-0.20190118093823-f849b5445de4 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.13.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/jonboulle/clockwork v0.1.0
	github.com/juju/errors v0.0.0-20180806074554-22422dad46e1
	github.com/juju/loggo v0.0.0-20180524022052-584905176618 // indirect
	github.com/juju/testing v0.0.0-20180920084828-472a3e8b2073 // indirect
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/minio/minio-go/v6 v6.0.55
	github.com/mohae/deepcopy v0.0.0-20170603005431-491d3605edfb
	github.com/nwaples/rardecode v1.0.0 // indirect
	github.com/onsi/ginkgo v1.10.3
	github.com/onsi/gomega v1.5.0
	github.com/openshift/generic-admission-server v1.14.0
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	github.com/pierrec/lz4 v2.0.5+incompatible // indirect
	github.com/pingcap/advanced-statefulset/client v1.16.0
	github.com/pingcap/check v0.0.0-20190102082844-67f458068fc8 // indirect
	github.com/pingcap/errors v0.11.0
	github.com/pingcap/kvproto v0.0.0-20191217072959-393e6c0fd4b7
	github.com/pingcap/pd v2.1.17+incompatible
	github.com/pingcap/tidb v2.1.0-beta+incompatible
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/common v0.0.0-20181126121408-4724e9255275
	github.com/prometheus/prometheus v1.8.2
	github.com/robfig/cron v1.1.0
	github.com/sirupsen/logrus v1.5.0
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/tmc/grpc-websocket-proxy v0.0.0-20171017195756-830351dc03c6 // indirect
	github.com/uber-go/atomic v0.0.0-00010101000000-000000000000 // indirect
	github.com/uber/jaeger-client-go v2.19.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.2.0+incompatible // indirect
	github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	github.com/yisaer/crd-validation v0.0.3
	gocloud.dev v0.18.0
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gomodules.xyz/jsonpatch/v2 v2.0.1
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce // indirect
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.0.0
	k8s.io/apiextensions-apiserver v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/apiserver v0.0.0
	k8s.io/cli-runtime v0.0.0
	k8s.io/client-go v0.0.0
	k8s.io/code-generator v0.0.0
	k8s.io/component-base v0.0.0
	k8s.io/klog v1.0.0
	k8s.io/kube-aggregator v0.0.0
	k8s.io/kube-openapi v0.0.0-20190918143330-0270cf2f1c1d
	k8s.io/kubectl v0.0.0
	k8s.io/kubernetes v1.16.0
	k8s.io/utils v0.0.0-20190801114015-581e00157fb1
	sigs.k8s.io/apiserver-builder-alpha v0.0.0-20191113095113-4493943d2568
	sigs.k8s.io/apiserver-builder-alpha/cmd v0.0.0-20191113095113-4493943d2568
	sigs.k8s.io/controller-runtime v0.4.0
)

replace github.com/renstrom/dedent => github.com/lithammer/dedent v1.1.0

replace k8s.io/api => k8s.io/api v0.0.0-20190918155943-95b840bb6a1f

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190918161926-8f644eb6e783

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655

replace k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190918160949-bfa5e2e684ad

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190918162238-f783a3654da8

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90

replace k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190912054826-cd179ad6a269

replace k8s.io/csi-api => k8s.io/csi-api v0.0.0-20190118125032-c4557c74373f

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190918161219-8c8f079fddc3

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20190918162944-7a93a0ddadd8

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20190918162534-de037b596c1e

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20190918162820-3b5c1246eb18

replace k8s.io/kubelet => k8s.io/kubelet v0.0.0-20190918162654-250a1838aa2c

replace k8s.io/metrics => k8s.io/metrics v0.0.0-20190918162108-227c654b2546

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20190918161442-d4c9c65c82af

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.0.0-20190918162410-e45c26d066f2

replace k8s.io/sample-controller => k8s.io/sample-controller v0.0.0-20190918161628-92eb3cb7496c

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190918163234-a9c1f33e9fb9

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190918163108-da9fdfce26bb

replace k8s.io/component-base => k8s.io/component-base v0.0.0-20190918160511-547f6c5d7090

replace k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20190918163402-db86a8c7bb21

replace k8s.io/kubectl => k8s.io/kubectl v0.0.0-20190918164019-21692a0861df

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20190918163543-cfa506e53441

replace k8s.io/node-api => k8s.io/node-api v0.0.0-20190918163711-2299658ad911

replace github.com/uber-go/atomic => go.uber.org/atomic v1.5.0

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible
