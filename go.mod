//
// Run hack/pin-deps.sh to change pinned dependency versions.
//

module github.com/pingcap/tidb-operator

go 1.23.2

require (
	cloud.google.com/go/storage v1.30.1
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.16.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.8.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute v1.0.0
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.13
	github.com/BurntSushi/toml v1.4.0
	github.com/Masterminds/semver v1.5.0
	github.com/agiledragon/gomonkey/v2 v2.12.0
	github.com/aws/aws-sdk-go v1.48.14
	github.com/aws/aws-sdk-go-v2 v1.16.11
	github.com/aws/aws-sdk-go-v2/config v1.17.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.53.0
	github.com/aws/smithy-go v1.12.1
	github.com/docker/go-units v0.5.0
	github.com/dustin/go-humanize v1.0.1
	github.com/emicklei/go-restful v2.16.0+incompatible
	github.com/ghodss/yaml v1.0.1-0.20220118164431-d8423dcdf344
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gogo/protobuf v1.3.2
	github.com/google/go-cmp v0.6.0
	github.com/google/gofuzz v1.2.0
	github.com/mholt/archiver/v3 v3.5.1
	github.com/minio/minio-go/v6 v6.0.57
	github.com/ncw/directio v1.0.5
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.27.6
	github.com/openshift/generic-admission-server v1.14.1-0.20230920151207-22a210ddee02
	github.com/pingcap/advanced-statefulset/client v1.17.1-0.20241017064830-af926cc6da0d
	github.com/pingcap/errors v0.11.4
	github.com/pingcap/kvproto v0.0.0-20231122054644-fb0f5c2a0a10
	github.com/pingcap/tidb-operator/pkg/apis v1.6.0
	github.com/pingcap/tidb-operator/pkg/client v1.6.0
	github.com/pingcap/tiproxy/lib v0.0.0-20230907130944-eb5b4b9c9e79
	github.com/prometheus/client_golang v1.17.0
	github.com/prometheus/client_model v0.6.1
	github.com/prometheus/common v0.45.0
	github.com/prometheus/prom2json v1.3.0
	github.com/r3labs/diff/v2 v2.15.1
	github.com/robfig/cron v1.2.0
	github.com/sethvargo/go-password v0.3.1
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.9.0
	github.com/tikv/pd v2.1.17+incompatible
	go.etcd.io/etcd/client/v3 v3.5.16
	gocloud.dev v0.18.0
	golang.org/x/sync v0.9.0
	golang.org/x/time v0.5.0
	gomodules.xyz/jsonpatch/v2 v2.4.0
	google.golang.org/grpc v1.59.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.28.14
	k8s.io/apiextensions-apiserver v0.28.14
	k8s.io/apimachinery v0.28.14
	k8s.io/apiserver v0.28.14
	k8s.io/cli-runtime v0.28.14
	k8s.io/client-go v0.28.14
	k8s.io/component-base v0.28.14
	k8s.io/klog/v2 v2.110.1
	k8s.io/kube-aggregator v0.28.14
	k8s.io/kube-scheduler v0.28.14
	k8s.io/kubectl v0.28.14
	k8s.io/kubelet v0.28.14
	k8s.io/utils v0.0.0-20240921022957-49e7df575cb6
	mvdan.cc/sh/v3 v3.4.3
	sigs.k8s.io/controller-runtime v0.7.2
)

require (
	cloud.google.com/go v0.110.10 // indirect
	cloud.google.com/go/compute v1.23.3 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v1.1.5 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Azure/azure-pipeline-go v0.2.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.28 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.22 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.6 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.2.2 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/andybalholm/brotli v1.0.1 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230305170008-8188dc5388df // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.12.14 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.18 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.16.13 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chai2010/gettext-go v1.0.2 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/dsnet/compress v0.0.2-0.20210315054119-f66993602bf5 // indirect
	github.com/emicklei/go-restful/v3 v3.10.2 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-logr/logr v1.3.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.20.0 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/cel-go v0.16.1 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/google/wire v0.3.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go v2.0.2+incompatible // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/grafana/regexp v0.0.0-20221122212121-6b5c0a4cb7fd // indirect
	github.com/gregjones/httpcache v0.0.0-20180305231024-9cad4c3443a7 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/klauspost/cpuid v1.2.3 // indirect
	github.com/klauspost/pgzip v1.2.5 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-ieproxy v0.0.0-20190610004146-91bb50d98149 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/minio/md5-simd v1.1.0 // indirect
	github.com/minio/sha256-simd v0.1.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/term v0.0.0-20221205130635-1aeaba878587 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170603005431-491d3605edfb // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nwaples/rardecode v1.1.3 // indirect
	github.com/nxadm/tail v1.4.4 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.2 // indirect
	github.com/pingcap/check v0.0.0-20211026125417-57bd13f7b5f0 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/procfs v0.11.1 // indirect
	github.com/prometheus/prometheus v0.49.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/ulikunitz/xz v0.5.9 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	go.etcd.io/etcd/api/v3 v3.5.16 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.16 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.46.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.46.1 // indirect
	go.opentelemetry.io/otel v1.21.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.21.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.21.0 // indirect
	go.opentelemetry.io/otel/metric v1.21.0 // indirect
	go.opentelemetry.io/otel/sdk v1.21.0 // indirect
	go.opentelemetry.io/otel/trace v1.21.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	go.starlark.net v0.0.0-20230525235612-a134d8f9ddca // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.23.0 // indirect
	golang.org/x/crypto v0.27.0 // indirect
	golang.org/x/exp v0.0.0-20231206192017-f3f8817b8deb // indirect
	golang.org/x/net v0.29.0 // indirect
	golang.org/x/oauth2 v0.15.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/term v0.24.0 // indirect
	golang.org/x/text v0.18.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/api v0.153.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20231120223509-83a465c0220f // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20231127180814-3a041ad873d4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231120223509-83a465c0220f // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/kms v0.28.14 // indirect
	k8s.io/kube-openapi v0.0.0-20230717233707-2695361300d9 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.1.2 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kustomize/api v0.13.5-0.20230601165947-6ce0bf390ce3 // indirect
	sigs.k8s.io/kustomize/kyaml v0.14.3-0.20230601165947-6ce0bf390ce3 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.3.0 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace github.com/pingcap/tidb-operator/pkg/apis => ./pkg/apis

replace github.com/pingcap/tidb-operator/pkg/client => ./pkg/client

replace github.com/renstrom/dedent => github.com/lithammer/dedent v1.1.0

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible

// workaround for github.com/advisories/GHSA-25xm-hr59-7c27
// TODO: remove it after upgrading github.com/mholt/archiver greater than v3.5.0
replace github.com/ulikunitz/xz => github.com/ulikunitz/xz v0.5.12
