module github.com/RedHatInsights/insights-results-aggregator

go 1.23.0

toolchain go1.24.4

require (
	github.com/BurntSushi/toml v1.5.0
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/IBM/sarama v1.45.2
	github.com/RedHatInsights/insights-operator-utils v1.25.15
	github.com/RedHatInsights/insights-results-aggregator-data v1.3.9
	github.com/RedHatInsights/insights-results-types v1.23.5
	github.com/deckarep/golang-set/v2 v2.8.0
	github.com/go-redis/redismock/v9 v9.2.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.10.9
	github.com/prometheus/client_golang v1.22.0
	github.com/prometheus/client_model v0.6.2
	github.com/qustavo/sqlhooks/v2 v2.1.0
	github.com/redhatinsights/app-common-go v1.6.8
	github.com/rs/zerolog v1.34.0
	github.com/spf13/viper v1.20.1
	github.com/stretchr/testify v1.10.0
	github.com/verdverm/frisby v0.0.0-20170604211311-b16556248a9a
	github.com/xdg/scram v1.0.5
	golang.org/x/sync v0.15.0
)

require (
	github.com/RedHatInsights/cloudwatch v0.0.0-20210111105023-1df2bdfe3291 // indirect
	github.com/RedHatInsights/kafka-zerolog v1.0.0 // indirect
	github.com/archdx/zerolog-sentry v1.8.5 // indirect
	github.com/aws/aws-sdk-go v1.55.7 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bitly/go-simplejson v0.5.1 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/eapache/go-resiliency v1.7.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20230731223053-c322873962e3 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/getkin/kin-openapi v0.132.0 // indirect
	github.com/getsentry/sentry-go v0.33.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.3.0 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/h2non/parth v0.0.0-20190131123155-b4df798d6542 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/mozillazg/request v0.8.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oasdiff/yaml v0.0.0-20250309154309-f31be36b4037 // indirect
	github.com/oasdiff/yaml3 v0.0.0-20250309153720-d2182401db90 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/common v0.65.0 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20250401214520-65e299d6c5c9 // indirect
	github.com/redis/go-redis/v9 v9.11.0 // indirect
	github.com/sagikazarmark/locafero v0.9.0 // indirect
	github.com/segmentio/kafka-go v0.4.48 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.14.0 // indirect
	github.com/spf13/cast v1.9.2 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/xdg/stringprep v1.0.3 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/h2non/gock.v1 v1.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
