module github.com/rancher/wins

go 1.22.0

toolchain go1.22.8

replace (
	github.com/docker/cli => github.com/docker/cli v27.2.1+incompatible
	github.com/docker/docker => github.com/docker/docker v27.2.1+incompatible
	github.com/google/go-cmp => github.com/google/go-cmp v0.6.0
	github.com/klauspost/compress => github.com/klauspost/compress v1.17.9
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.53.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.53.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc => go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0
	golang.org/x/oauth2 => golang.org/x/oauth2 v0.23.0
	golang.org/x/sync => golang.org/x/sync v0.8.0
	k8s.io/api => k8s.io/api v0.29.7
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.29.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.29.7
	k8s.io/apiserver => k8s.io/apiserver v0.29.7
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.29.7
	k8s.io/client-go => github.com/rancher/client-go v1.29.3-rancher1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.29.7
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.29.7
	k8s.io/code-generator => k8s.io/code-generator v0.29.7
	k8s.io/component-base => k8s.io/component-base v0.29.7
	k8s.io/component-helpers => k8s.io/component-helpers v0.29.7
	k8s.io/controller-manager => k8s.io/controller-manager v0.29.7
	k8s.io/cri-api => k8s.io/cri-api v0.29.7
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.29.7
	k8s.io/dynamic-resource-allocation => k8s.io/dynamic-resource-allocation v0.29.7
	k8s.io/endpointslice => k8s.io/endpointslice v1.29.7
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.29.7
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.29.7
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.29.7
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.29.7
	k8s.io/kubectl => k8s.io/kubectl v0.29.7
	k8s.io/kubelet => k8s.io/kubelet v0.29.7
	k8s.io/kubernetes => k8s.io/kubernetes v1.29.7
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.29.7
	k8s.io/metrics => k8s.io/metrics v0.29.7
	k8s.io/mount-utils => k8s.io/mount-utils v0.29.7
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.29.7
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.29.7

)

require (
	github.com/Microsoft/go-winio v0.6.2
	github.com/Microsoft/hcsshim v0.8.25
	github.com/buger/jsonparser v1.1.1
	github.com/fsnotify/fsnotify v1.7.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/gogo/protobuf v1.3.2
	github.com/gorilla/websocket v1.5.3
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/magefile/mage v1.15.0
	github.com/mattn/go-colorable v0.1.13
	github.com/pkg/errors v0.9.1
	github.com/rancher/remotedialer v0.4.1
	github.com/rancher/system-agent v0.3.10
	github.com/sirupsen/logrus v1.9.3
	github.com/urfave/cli/v2 v2.27.4
	golang.org/x/sync v0.8.0
	golang.org/x/sys v0.25.0
	google.golang.org/grpc v1.66.2
	inet.af/tcpproxy v0.0.0-20231102063150-2862066fc2a9
	k8s.io/api v0.30.0
)

require (
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230305170008-8188dc5388df // indirect
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/cgroups v1.1.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.14.3 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/cli v27.1.1+incompatible // indirect
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.7.0 // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/cel-go v0.17.8 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/go-containerregistry v0.20.2 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/prometheus/client_golang v1.16.0 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.44.0 // indirect
	github.com/prometheus/procfs v0.10.1 // indirect
	github.com/rancher/lasso v0.0.0-20230830164424-d684fdeb6f29 // indirect
	github.com/rancher/permissions v0.0.0-20240924180251-69b0dcb34065 // indirect
	github.com/rancher/wharfie v0.6.7 // indirect
	github.com/rancher/wrangler v1.1.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spf13/cobra v1.7.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/urfave/cli v1.22.15 // indirect
	github.com/vbatts/tar-split v0.11.3 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	go.etcd.io/etcd/api/v3 v3.5.10 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.10 // indirect
	go.etcd.io/etcd/client/v3 v3.5.10 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.42.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.53.0 // indirect
	go.opentelemetry.io/otel v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.19.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/otel/sdk v1.28.0 // indirect
	go.opentelemetry.io/otel/trace v1.28.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/crypto v0.25.0 // indirect
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d // indirect
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/oauth2 v0.21.0 // indirect
	golang.org/x/term v0.22.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240701130421-f6361c86f094 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240701130421-f6361c86f094 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.25.4 // indirect
	k8s.io/apimachinery v0.30.0 // indirect
	k8s.io/apiserver v0.29.7 // indirect
	k8s.io/client-go v0.30.0 // indirect
	k8s.io/cloud-provider v0.0.0 // indirect
	k8s.io/component-base v0.29.7 // indirect
	k8s.io/controller-manager v0.29.7 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kms v0.29.7 // indirect
	k8s.io/kube-openapi v0.0.0-20240228011516-70dd3763d340 // indirect
	k8s.io/kubelet v0.24.2 // indirect
	k8s.io/kubernetes v1.30.0 // indirect
	k8s.io/utils v0.0.0-20240502163921-fe8a2dddb1d0 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.29.0 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)
