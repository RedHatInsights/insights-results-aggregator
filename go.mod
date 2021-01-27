module github.com/RedHatInsights/insights-results-aggregator

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/DATA-DOG/go-sqlmock v1.4.1
	github.com/RedHatInsights/cloudwatch v0.0.0-20210111105023-1df2bdfe3291
	github.com/RedHatInsights/insights-content-service v0.0.0-20201009081018-083923779f00
	github.com/RedHatInsights/insights-operator-utils v1.6.9
	github.com/RedHatInsights/insights-results-aggregator-data v0.0.0-20201109115720-126bd0348556
	github.com/RedHatInsights/insights-results-aggregator-utils v0.0.0-20200616074815-67f30b0e724d // indirect
	github.com/RedHatInsights/insights-results-smart-proxy v0.0.0-20200619163313-7d5e376de430 // indirect
	github.com/Shopify/sarama v1.27.1
	github.com/aws/aws-sdk-go v1.35.7
	github.com/deckarep/golang-set v1.7.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gchaincl/sqlhooks v1.3.0
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.8.0
	github.com/lib/pq v1.8.0
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/rs/zerolog v1.20.0
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	github.com/verdverm/frisby v0.0.0-20170604211311-b16556248a9a
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/h2non/gock.v1 v1.0.15
)
