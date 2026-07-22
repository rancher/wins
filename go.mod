module github.com/rancher/wins

go 1.26.4

// replacements to match embedded system-agent
replace (
	github.com/fsnotify/fsnotify => github.com/fsnotify/fsnotify v1.9.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.68.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.68.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc => go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.43.0
	golang.org/x/oauth2 => golang.org/x/oauth2 v0.36.0
	golang.org/x/sync => golang.org/x/sync v0.22.0
	golang.org/x/sys => golang.org/x/sys v0.47.0
)

replace (
	github.com/docker/cli => github.com/docker/cli v29.6.1+incompatible
	github.com/docker/docker => github.com/docker/docker v28.5.2+incompatible
	inet.af/tcpproxy => github.com/inetaf/tcpproxy v0.0.0-20240214030015-3ce58045626c
	k8s.io/api => k8s.io/api v0.36.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.36.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.36.2
	k8s.io/apiserver => k8s.io/apiserver v0.36.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.36.2
	k8s.io/client-go => k8s.io/client-go v0.36.2
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.36.2
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.36.2
	k8s.io/code-generator => k8s.io/code-generator v0.36.2
	k8s.io/component-base => k8s.io/component-base v0.36.2
	k8s.io/component-helpers => k8s.io/component-helpers v0.36.2
	k8s.io/controller-manager => k8s.io/controller-manager v0.36.2
	k8s.io/cri-api => k8s.io/cri-api v0.36.2
	k8s.io/cri-client => k8s.io/cri-client v0.36.2
	k8s.io/cri-streaming => k8s.io/cri-streaming v0.36.2
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.36.2
	k8s.io/dynamic-resource-allocation => k8s.io/dynamic-resource-allocation v0.36.2
	k8s.io/endpointslice => k8s.io/endpointslice v0.36.2
	k8s.io/externaljwt => k8s.io/externaljwt@ v0.36.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.36.2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.36.2
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.36.2
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.36.2
	k8s.io/kubectl => k8s.io/kubectl v0.36.2
	k8s.io/kubelet => k8s.io/kubelet v0.36.2
	k8s.io/kubernetes => k8s.io/kubernetes v1.36.2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.36.1
	k8s.io/metrics => k8s.io/metrics v0.36.2
	k8s.io/mount-utils => k8s.io/mount-utils v0.36.2
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.36.2
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.36.2
)

require (
	github.com/Microsoft/go-winio v0.6.2
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/magefile/mage v1.16.0
	github.com/mattn/go-colorable v0.1.15
	github.com/pkg/errors v0.9.1
	github.com/rancher/system-agent v0.15.0-rc.5
	github.com/sirupsen/logrus v1.9.4
	github.com/urfave/cli/v2 v2.27.6
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.47.0
	google.golang.org/grpc v1.81.1 // indirect
	k8s.io/api v0.36.2
	sigs.k8s.io/yaml v1.6.0
)

require (
	cel.dev/expr v0.25.1 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.14.3 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.7.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/cli v29.2.0+incompatible // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.7.0 // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/evanphx/json-patch v5.9.11+incompatible // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/cel-go v0.26.0 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-containerregistry v0.20.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus v1.1.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/rancher/lasso v0.2.9 // indirect
	github.com/rancher/permissions v0.0.0-20240924180251-69b0dcb34065 // indirect
	github.com/rancher/rancher/pkg/plan v0.0.0-20260508124826-0b6b24d9811e // indirect
	github.com/rancher/wharfie v0.7.1-0.20251014190711-8cfe84a9efaa // indirect
	github.com/rancher/wrangler/v3 v3.7.0 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/vbatts/tar-split v0.11.3 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	go.etcd.io/etcd/api/v3 v3.6.8 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.6.8 // indirect
	go.etcd.io/etcd/client/v3 v3.6.8 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.65.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.65.0 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.40.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/exp v0.0.0-20251219203646-944ab1f22d93 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/term v0.43.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260406210006-6f92a3bedf2d // indirect
	google.golang.org/protobuf v1.36.12-0.20260120151049-f2248ac996af // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.36.0 // indirect
	k8s.io/apimachinery v0.36.2 // indirect
	k8s.io/apiserver v0.36.2 // indirect
	k8s.io/client-go v0.36.2 // indirect
	k8s.io/cloud-provider v0.34.0 // indirect
	k8s.io/component-base v0.36.2 // indirect
	k8s.io/component-helpers v0.36.2 // indirect
	k8s.io/controller-manager v0.36.2 // indirect
	k8s.io/cri-api v0.36.2 // indirect
	k8s.io/cri-client v0.34.0 // indirect
	k8s.io/klog/v2 v2.140.0 // indirect
	k8s.io/kms v0.36.2 // indirect
	k8s.io/kube-openapi v0.0.0-20260317180543-43fb72c5454a // indirect
	k8s.io/kubelet v0.34.0 // indirect
	k8s.io/kubernetes v1.36.0 // indirect
	k8s.io/streaming v0.36.2 // indirect
	k8s.io/utils v0.0.0-20260210185600-b8788abfbbc2 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.34.0 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.2 // indirect
)
