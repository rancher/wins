module github.com/rancher/wins

go 1.21

replace (
	github.com/docker/cli => github.com/docker/cli v20.10.16+incompatible
	github.com/docker/docker => github.com/docker/docker v20.10.24+incompatible
	github.com/google/go-cmp => github.com/google/go-cmp v0.5.7
	github.com/klauspost/compress => github.com/klauspost/compress v1.15.4
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.46.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.46.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc => go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.23.1
	golang.org/x/oauth2 => golang.org/x/oauth2 v0.0.0-20220524215830-622c5d57e401
	golang.org/x/sync => golang.org/x/sync v0.0.0-20220513210516-0976fa681c29
	k8s.io/api => k8s.io/api v0.27.10
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.27.10
	k8s.io/apimachinery => k8s.io/apimachinery v0.27.10
	k8s.io/apiserver => k8s.io/apiserver v0.27.10
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.27.10
	k8s.io/client-go => github.com/rancher/client-go v1.27.4-rancher1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.27.10
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.27.10
	k8s.io/code-generator => k8s.io/code-generator v0.27.10
	k8s.io/component-base => k8s.io/component-base v0.27.10
	k8s.io/component-helpers => k8s.io/component-helpers v0.27.10
	k8s.io/controller-manager => k8s.io/controller-manager v0.27.10
	k8s.io/cri-api => k8s.io/cri-api v0.27.10
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.27.10
	k8s.io/dynamic-resource-allocation => k8s.io/dynamic-resource-allocation v0.27.10
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.27.10
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.27.10
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.27.10
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.27.10
	k8s.io/kubectl => k8s.io/kubectl v0.27.10
	k8s.io/kubelet => k8s.io/kubelet v0.27.10
	k8s.io/kubernetes => k8s.io/kubernetes v1.27.10
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.27.10
	k8s.io/metrics => k8s.io/metrics v0.27.10
	k8s.io/mount-utils => k8s.io/mount-utils v0.27.10
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.27.10
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.27.10

)

require (
	github.com/Microsoft/go-winio v0.6.1
	github.com/Microsoft/hcsshim v0.8.25
	github.com/buger/jsonparser v1.1.1
	github.com/fsnotify/fsnotify v1.6.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/gogo/protobuf v1.3.2
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/magefile/mage v1.13.0
	github.com/mattn/go-colorable v0.1.13
	github.com/pkg/errors v0.9.1
	github.com/rancher/remotedialer v0.2.6-0.20201012155453-8b1b7bb7d05f
	github.com/rancher/system-agent v0.3.7
	github.com/sirupsen/logrus v1.9.3
	github.com/urfave/cli/v2 v2.25.7
	golang.org/x/sync v0.5.0
	golang.org/x/sys v0.16.0
	google.golang.org/grpc v1.61.0
	inet.af/tcpproxy v0.0.0-20200125044825-b6bb9b5b8252
)

require (
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v1.4.10 // indirect
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/containerd/cgroups v1.0.1 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.14.3 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.4.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/cli v24.0.0+incompatible // indirect
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/docker/docker v24.0.0+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.7.0 // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.1 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/cel-go v0.12.7 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/go-containerregistry v0.16.1 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.4.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.5 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/onsi/ginkgo/v2 v2.11.0 // indirect
	github.com/onsi/gomega v1.27.10 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc3 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/rancher/lasso v0.0.0-20230830164424-d684fdeb6f29 // indirect
	github.com/rancher/wharfie v0.6.2 // indirect
	github.com/rancher/wrangler v1.1.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spf13/cobra v1.7.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/urfave/cli v1.22.12 // indirect
	github.com/vbatts/tar-split v0.11.3 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	go.etcd.io/etcd/api/v3 v3.5.7 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.7 // indirect
	go.etcd.io/etcd/client/v3 v3.5.7 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.35.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.35.1 // indirect
	go.opentelemetry.io/otel v1.23.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.23.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.10.0 // indirect
	go.opentelemetry.io/otel/metric v1.23.1 // indirect
	go.opentelemetry.io/otel/sdk v1.23.1 // indirect
	go.opentelemetry.io/otel/trace v1.23.1 // indirect
	go.opentelemetry.io/proto/otlp v1.1.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.19.0 // indirect
	golang.org/x/crypto v0.16.0 // indirect
	golang.org/x/mod v0.14.0 // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/oauth2 v0.15.0 // indirect
	golang.org/x/term v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.16.1 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20231212172506-995d672761c0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240102182953-50ed04b92917 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240102182953-50ed04b92917 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.27.10 // indirect
	k8s.io/apimachinery v0.27.10 // indirect
	k8s.io/apiserver v0.27.10 // indirect
	k8s.io/client-go v0.27.10 // indirect
	k8s.io/cloud-provider v0.0.0 // indirect
	k8s.io/component-base v0.27.10 // indirect
	k8s.io/controller-manager v0.27.10 // indirect
	k8s.io/klog/v2 v2.90.1 // indirect
	k8s.io/kms v0.27.10 // indirect
	k8s.io/kube-openapi v0.0.0-20230501164219-8b0f38b5fd1f // indirect
	k8s.io/kubelet v0.24.2 // indirect
	k8s.io/kubernetes v1.27.10 // indirect
	k8s.io/utils v0.0.0-20230209194617-a36077c30491 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.1.2 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
