//
// Run hack/pin-deps.sh to change pinned dependency versions.
//

module github.com/pingcap/tidb-operator

go 1.19

require (
	cloud.google.com/go/storage v1.6.0
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.2
	github.com/Masterminds/semver v1.4.2
	github.com/agiledragon/gomonkey/v2 v2.7.0
	github.com/aws/aws-sdk-go v1.44.72
	github.com/aws/aws-sdk-go-v2 v1.16.11
	github.com/aws/aws-sdk-go-v2/config v1.17.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.53.0
	github.com/aws/smithy-go v1.12.1
	github.com/docker/docker v20.10.2+incompatible
	github.com/dustin/go-humanize v1.0.0
	github.com/emicklei/go-restful v2.16.0+incompatible
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gogo/protobuf v1.3.2
	github.com/google/go-cmp v0.5.8
	github.com/google/gofuzz v1.1.0
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/minio/minio-go/v6 v6.0.55
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/generic-admission-server v1.14.1-0.20210422140326-da96454c926d
	github.com/pingcap/TiProxy/lib v0.0.0-20230201020701-df06ec482c69
	github.com/pingcap/advanced-statefulset/client v1.17.1-0.20230717084314-aebf289f5b33
	github.com/pingcap/errors v0.11.4
	github.com/pingcap/kvproto v0.0.0-20200927054727-1290113160f0
	github.com/pingcap/tidb-operator/pkg/apis v1.6.0-alpha.6
	github.com/pingcap/tidb-operator/pkg/client v1.6.0-alpha.6
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.26.0
	github.com/prometheus/prom2json v1.3.0
	github.com/r3labs/diff/v2 v2.15.1
	github.com/robfig/cron v1.1.0
	github.com/sethvargo/go-password v0.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.5.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.1
	github.com/tikv/pd v2.1.17+incompatible
	go.etcd.io/etcd/client/v3 v3.5.0
	go.uber.org/atomic v1.9.0
	gocloud.dev v0.18.0
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac
	gomodules.xyz/jsonpatch/v2 v2.1.0
	google.golang.org/grpc v1.38.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.17
	k8s.io/apiextensions-apiserver v0.22.17
	k8s.io/apimachinery v0.22.17
	k8s.io/apiserver v0.22.17
	k8s.io/cli-runtime v0.22.17
	k8s.io/client-go v0.22.17
	k8s.io/component-base v0.22.17
	k8s.io/klog/v2 v2.9.0
	k8s.io/kube-aggregator v0.22.17
	k8s.io/kube-scheduler v0.22.17
	k8s.io/kubectl v0.22.17
	k8s.io/kubernetes v1.22.17
	k8s.io/utils v0.0.0-20211116205334-6203023598ed
	mvdan.cc/sh/v3 v3.4.3
	sigs.k8s.io/controller-runtime v0.7.2
)

require (
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible // indirect
	github.com/fvbommel/sortorder v1.0.1 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/opencontainers/runc v1.0.2 // indirect
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca // indirect
	go.etcd.io/etcd/api/v3 v3.5.0 // indirect
	go.opentelemetry.io/contrib v0.20.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.20.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.20.0 // indirect
	go.opentelemetry.io/otel v0.20.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp v0.20.0 // indirect
	go.opentelemetry.io/otel/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk/export/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/trace v0.20.0 // indirect
	go.opentelemetry.io/proto/otlp v0.7.0 // indirect
	go.starlark.net v0.0.0-20200306205701-8dd3e2ee1dd5 // indirect
	k8s.io/component-helpers v0.22.17 // indirect
	k8s.io/kubelet v0.0.0 // indirect
	sigs.k8s.io/kustomize/api v0.8.11 // indirect
	sigs.k8s.io/kustomize/kyaml v0.11.0 // indirect
)

require (
	cloud.google.com/go v0.54.0 // indirect
	github.com/Azure/azure-pipeline-go v0.2.1 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.18 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.13 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.1 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/GoogleCloudPlatform/k8s-cloud-provider v0.0.0-20200415212048-7901bc822317 // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/Microsoft/go-winio v0.4.15 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.12.14 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.18 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.16.13 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/chai2010/gettext-go v0.0.0-20170215093142-bf70f2a70fb1 // indirect
	github.com/containerd/containerd v1.4.4 // indirect
	github.com/coreos/go-semver v0.3.0
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dimchansky/utfbom v1.1.0 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/elazarl/goproxy v0.0.0-20190421051319-9d40249d3c2f // indirect; indirectload
	github.com/evanphx/json-patch v4.11.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/google/wire v0.3.0 // indirect
	github.com/googleapis/gax-go v2.0.2+incompatible // indirect
	github.com/googleapis/gax-go/v2 v2.0.5 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/imdario/mergo v0.3.10 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/mattn/go-ieproxy v0.0.0-20190610004146-91bb50d98149 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/minio/sha256-simd v0.1.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/moby/term v0.0.0-20210610120745-9d4ed1856297 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/mohae/deepcopy v0.0.0-20170603005431-491d3605edfb // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nwaples/rardecode v1.0.0 // indirect
	github.com/nxadm/tail v1.4.4 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pierrec/lz4 v2.0.5+incompatible // indirect
	github.com/pingcap/check v0.0.0-20190102082844-67f458068fc8 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/prometheus/prometheus v1.8.2 // indirect
	github.com/russross/blackfriday v1.5.2 // indirect
	github.com/ulikunitz/xz v0.5.6 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.5 // indirect
	go.opencensus.io v0.22.3 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.23.0 // indirect
	golang.org/x/crypto v0.0.0-20220314234659-1baeb1ce4c0b // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/term v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	golang.org/x/tools v0.1.12 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/api v0.20.0 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c // indirect
	google.golang.org/protobuf v1.26.0 // indirect
	gopkg.in/gcfg.v1 v1.2.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.51.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	honnef.co/go/tools v0.0.1-2020.1.3 // indirect
	k8s.io/cloud-provider v0.22.17 // indirect
	k8s.io/csi-translation-lib v0.22.17 // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/kube-openapi v0.0.0-20211109043538-20434351676c // indirect
	k8s.io/legacy-cloud-providers v0.0.0 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.30 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace github.com/pingcap/tidb-operator/pkg/apis => ./pkg/apis

replace github.com/pingcap/tidb-operator/pkg/client => ./pkg/client

replace github.com/renstrom/dedent => github.com/lithammer/dedent v1.1.0

replace k8s.io/api => k8s.io/api v0.22.17

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.22.17

replace k8s.io/apimachinery => k8s.io/apimachinery v0.22.17

replace k8s.io/apiserver => k8s.io/apiserver v0.22.17

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.22.17

replace k8s.io/client-go => k8s.io/client-go v0.22.17

replace k8s.io/code-generator => k8s.io/code-generator v0.22.17

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.22.17

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.22.17

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.22.17

replace k8s.io/kubelet => k8s.io/kubelet v0.22.17

replace k8s.io/metrics => k8s.io/metrics v0.22.17

replace k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.22.17

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.22.17

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.22.17

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.22.17

replace k8s.io/component-base => k8s.io/component-base v0.22.17

replace k8s.io/cri-api => k8s.io/cri-api v0.22.17

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.22.17

replace k8s.io/kubectl => k8s.io/kubectl v0.22.17

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.22.17

replace github.com/uber-go/atomic => go.uber.org/atomic v1.5.0

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible

replace github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.11.1

replace k8s.io/controller-manager => k8s.io/controller-manager v0.22.17

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.22.17

replace k8s.io/component-helpers => k8s.io/component-helpers v0.22.17

replace k8s.io/mount-utils => k8s.io/mount-utils v0.22.17

// workaround for github.com/advisories/GHSA-25xm-hr59-7c27
// TODO: remove it after upgrading github.com/mholt/archiver greater than v3.5.0
replace github.com/ulikunitz/xz => github.com/ulikunitz/xz v0.5.8
