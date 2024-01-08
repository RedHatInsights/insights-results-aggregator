/*
Copyright © 2020, 2021, 2022, 2023 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package conf_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/logger"
	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	mapset "github.com/deckarep/golang-set"
	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/conf"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

func mustLoadConfiguration(path string) {
	err := conf.LoadConfiguration(path)
	if err != nil {
		panic(err)
	}
}

func removeFile(t *testing.T, filename string) {
	err := os.Remove(filename)
	helpers.FailOnError(t, err)
}

func setEnvVariables(t *testing.T) {
	os.Clearenv()

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__BROKER__ADDRESS", "localhost:9093")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__BROKER__TOPIC", "platform.results.ccx")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__BROKER__GROUP", "aggregator")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__BROKER__ENABLED", "true")

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__SERVER__ADDRESS", ":8080")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__SERVER__API_PREFIX", "/api/v1/")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__SERVER__API_SPEC_FILE", "openapi.json")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__SERVER__DEBUG", "true")

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__PROCESSING__ORG_ALLOWLIST", "org_allowlist.csv")

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__DB_DRIVER", "postgres")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__PG_USERNAME", "user")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__PG_PASSWORD", "password")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__PG_HOST", "localhost")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__PG_PORT", "5432")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__PG_DB_NAME", "aggregator")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__PG_PARAMS", "params")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__LOG_SQL_QUERIES", "true")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__TYPE", "sql")

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__DVO_RECOMMENDATIONS_STORAGE__DB_DRIVER", "postgres")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__DVO_RECOMMENDATIONS_STORAGE__PG_USERNAME", "user")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__DVO_RECOMMENDATIONS_STORAGE__PG_PASSWORD", "password")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__DVO_RECOMMENDATIONS_STORAGE__PG_HOST", "localhost")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__DVO_RECOMMENDATIONS_STORAGE__PG_PORT", "5432")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__DVO_RECOMMENDATIONS_STORAGE__PG_DB_NAME", "aggregator")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__DVO_RECOMMENDATIONS_STORAGE__PG_PARAMS", "params")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__DVO_RECOMMENDATIONS_STORAGE__LOG_SQL_QUERIES", "true")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__DVO_RECOMMENDATIONS_STORAGE__TYPE", "sql")

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__REDIS__ENDPOINT", "default-redis-endpoint")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__REDIS__DATABASE", "42")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__REDIS__TIMEOUT_SECONDS", "0")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__REDIS__PASSWORD", "top secret")

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__SENTRY__DSN", "test.example.com")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__SENTRY__ENVIRONMENT", "test")
}

func mustSetEnv(t *testing.T, key, val string) {
	err := os.Setenv(key, val)
	helpers.FailOnError(t, err)
}

func GetTmpConfigFile(configData string) (string, error) {
	tmpFile, err := os.CreateTemp("/tmp", "tmp_config_*.toml")
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.WriteString(configData); err != nil {
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

// TestLoadConfiguration loads a configuration file for testing
func TestLoadConfiguration(_ *testing.T) {
	os.Clearenv()

	mustLoadConfiguration("tests/config1")
}

// TestLoadConfigurationEnvVariable tests loading the config. file for testing from an environment variable
func TestLoadConfigurationEnvVariable(t *testing.T) {
	os.Clearenv()

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE", "../tests/config1")

	mustLoadConfiguration("foobar")
}

// TestLoadingConfigurationFailure tests loading a non-existent configuration file
func TestLoadingConfigurationFailure(t *testing.T) {
	os.Clearenv()

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE", "non existing file")

	err := conf.LoadConfiguration("")
	assert.Contains(t, err.Error(), `fatal error config file: Config File "non existing file" Not Found in`)
}

// TestLoadBrokerConfiguration tests loading the broker configuration sub-tree
func TestLoadBrokerConfiguration(t *testing.T) {
	helpers.FailOnError(t, os.Chdir(".."))
	TestLoadConfiguration(t)
	expectedTimeout, _ := time.ParseDuration("30s")

	brokerCfg := conf.GetBrokerConfiguration()

	assert.Equal(t, "localhost:29092", brokerCfg.Address)
	assert.Equal(t, "platform.results.ccx", brokerCfg.Topic)
	assert.Equal(t, "aggregator", brokerCfg.Group)
	assert.Equal(t, expectedTimeout, brokerCfg.Timeout)
}

// TestLoadServerConfiguration tests loading the server configuration sub-tree
func TestLoadServerConfiguration(t *testing.T) {
	TestLoadConfiguration(t)

	serverCfg := conf.GetServerConfiguration()

	assert.Equal(t, ":8080", serverCfg.Address)
	assert.Equal(t, "/api/v1/", serverCfg.APIPrefix)
}

// TestLoadStorageBackendConfigurationChangedFromEnvVar tests loading the
// storage backend configuration subtree
func TestLoadStorageBackendConfigurationChangedFromEnvVar(t *testing.T) {
	os.Clearenv()

	const configPath = "../tests/config1"

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE_BACKEND__USE", "dvo_recommendations")

	mustLoadConfiguration(configPath)

	storageCfg := conf.GetStorageBackendConfiguration()
	assert.Equal(t, "dvo_recommendations", storageCfg.Use)
}

// TestLoadStorageBackendConfigurationChangedFromEnvVar tests loading the
// storage backend configuration subtree doesn't change from empty
func TestLoadStorageBackendConfigurationNotChangedWhenEmpty(t *testing.T) {
	os.Clearenv()

	const configPath = "../tests/config1"

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE_BACKEND__USE", "")
	mustLoadConfiguration(configPath)

	storageCfg := conf.GetStorageBackendConfiguration()
	assert.Equal(t, "", storageCfg.Use)
}

// TestLoadOCPRecommendationsStorageConfiguration tests loading the OCP
// recommendations storage configuration subtree
func TestLoadOCPRecommendationsStorageConfiguration(t *testing.T) {
	TestLoadConfiguration(t)

	storageCfg := conf.GetOCPRecommendationsStorageConfiguration()

	assert.Equal(t, "postgres", storageCfg.Driver)
	assert.Equal(t, "sql", storageCfg.Type)
}

// TestLoadDVORecommendationsStorageConfiguration tests loading the DVO
// recommendations storage configuration subtree
func TestLoadDVORecommendationsStorageConfiguration(t *testing.T) {
	TestLoadConfiguration(t)

	storageCfg := conf.GetDVORecommendationsStorageConfiguration()

	assert.Equal(t, "postgres", storageCfg.Driver)
	assert.Equal(t, "user", storageCfg.PGUsername)
	assert.Equal(t, "password", storageCfg.PGPassword)
	assert.Equal(t, "sql", storageCfg.Type)
}

// TestLoadRedisConfiguration tests loading the Redis configuration subtree
func TestLoadRedisConfiguration(t *testing.T) {
	TestLoadConfiguration(t)

	redisCfg := conf.GetRedisConfiguration()

	assert.Equal(t, "localhost:6379", redisCfg.RedisEndpoint)
	assert.Equal(t, 0, redisCfg.RedisDatabase)
	assert.Equal(t, 30, redisCfg.RedisTimeoutSeconds)
	assert.Equal(t, "", redisCfg.RedisPassword)
}

// TestLoadConfigurationOverrideFromEnv1 tests overriding configuration by env variables
func TestLoadConfigurationOverrideFromEnv1(t *testing.T) {
	os.Clearenv()

	const configPath = "../tests/config1"

	mustLoadConfiguration(configPath)

	storageCfg := conf.GetOCPRecommendationsStorageConfiguration()
	assert.Equal(t, storage.Configuration{
		Driver:     "postgres",
		PGUsername: "user",
		PGPassword: "password",
		PGHost:     "localhost",
		PGPort:     5432,
		PGDBName:   "aggregator",
		PGParams:   "",
		Type:       "sql",
	}, storageCfg)

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__PG_PASSWORD", "some very secret password")

	mustLoadConfiguration(configPath)

	storageCfg = conf.GetOCPRecommendationsStorageConfiguration()
	assert.Equal(t, storage.Configuration{
		Driver:     "postgres",
		PGUsername: "user",
		PGPassword: "some very secret password",
		PGHost:     "localhost",
		PGPort:     5432,
		PGDBName:   "aggregator",
		PGParams:   "",
		Type:       "sql",
	}, storageCfg)
}

// TestLoadConfigurationOverrideFromEnv2 tests overriding configuration by env variables
func TestLoadConfigurationOverrideFromEnv2(t *testing.T) {
	os.Clearenv()

	const configPath = "../tests/config1"

	mustLoadConfiguration(configPath)

	storageCfg := conf.GetDVORecommendationsStorageConfiguration()
	assert.Equal(t, storage.Configuration{
		Driver:     "postgres",
		PGUsername: "user",
		PGPassword: "password",
		PGHost:     "localhost",
		PGPort:     5432,
		PGDBName:   "aggregator",
		PGParams:   "",
		Type:       "sql",
	}, storageCfg)

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__DVO_RECOMMENDATIONS_STORAGE__PG_PASSWORD", "some very secret password")

	mustLoadConfiguration(configPath)

	storageCfg = conf.GetDVORecommendationsStorageConfiguration()
	assert.Equal(t, storage.Configuration{
		Driver:     "postgres",
		PGUsername: "user",
		PGPassword: "some very secret password",
		PGHost:     "localhost",
		PGPort:     5432,
		PGDBName:   "aggregator",
		PGParams:   "",
		Type:       "sql",
	}, storageCfg)
}

// TestLoadOrganizationAllowlist tests if the allow-list CSV file gets loaded properly
func TestLoadOrganizationAllowlist(t *testing.T) {
	expectedAllowlist := mapset.NewSetWith(
		types.OrgID(1),
		types.OrgID(2),
		types.OrgID(3),
		types.OrgID(11789772),
		types.OrgID(656485),
	)

	orgAllowlist := conf.GetOrganizationAllowlist()
	if equal := orgAllowlist.Equal(expectedAllowlist); !equal {
		t.Errorf(
			"Org allowlist did not load properly. Order of elements does not matter. Expected %v. Got %v",
			expectedAllowlist, orgAllowlist,
		)
	}
}

// TestLoadAllowlistFromCSVExtraParam tests incorrect CSV format
func TestLoadAllowlistFromCSVExtraParam(t *testing.T) {
	extraParamCSV := `OrgID
1,2
3
`
	r := strings.NewReader(extraParamCSV)
	_, err := conf.LoadAllowlistFromCSV(r)
	assert.EqualError(t, err, "error reading CSV file: record on line 2: wrong number of fields")
}

// TestLoadAllowlistFromCSVNonInt tests non-integer ID in CSV
func TestLoadAllowlistFromCSVNonInt(t *testing.T) {
	nonIntIDCSV := `OrgID
str
3
`
	r := strings.NewReader(nonIntIDCSV)
	_, err := conf.LoadAllowlistFromCSV(r)
	assert.EqualError(t, err, "organization ID on line 2 in allowlist CSV is not numerical. Found value: str")
}

func TestLoadConfigurationFromFile(t *testing.T) {
	config := `[broker]
		address = "localhost:29092"
		topic = "platform.results.ccx"
		group = "aggregator"
		enabled = true
		enable_org_allowlist = true

		[processing]
		org_allowlist_file = "org_allowlist.csv"

		[server]
		address = ":8080"
		api_prefix = "/api/v1/"
		api_spec_file = "openapi.json"
		debug = true

		[ocp_recommendations_storage]
		db_driver = "postgres"
		pg_username = "user"
		pg_password = "password"
		pg_host = "localhost"
		pg_port = 5432
		pg_db_name = "aggregator"
		pg_params = "params"
		log_sql_queries = true
		type = "sql"

		[redis]
		database = 0
		endpoint = "localhost:6379"
		password = ""
		timeout_seconds = 30

		[sentry]
		dsn = "test.example2.com"
		environment = "test2"
	`

	tmpFilename, err := GetTmpConfigFile(config)
	helpers.FailOnError(t, err)

	defer removeFile(t, tmpFilename)

	os.Clearenv()
	mustSetEnv(t, conf.ConfigFileEnvVariableName, tmpFilename)
	mustLoadConfiguration("../tests/config1")

	brokerCfg := conf.GetBrokerConfiguration()

	assert.Equal(t, "localhost:29092", brokerCfg.Address)
	assert.Equal(t, "platform.results.ccx", brokerCfg.Topic)
	assert.Equal(t, "aggregator", brokerCfg.Group)
	assert.Equal(t, true, brokerCfg.Enabled)

	assert.Equal(t, server.Configuration{
		Address:                      ":8080",
		APIPrefix:                    "/api/v1/",
		APISpecFile:                  "openapi.json",
		AuthType:                     "xrh",
		Auth:                         false,
		Debug:                        true,
		MaximumFeedbackMessageLength: 255,
		OrgOverviewLimitHours:        2,
	}, conf.GetServerConfiguration())

	orgAllowlist := conf.GetOrganizationAllowlist()

	assert.True(
		t,
		orgAllowlist.Equal(mapset.NewSetWith(
			types.OrgID(1),
			types.OrgID(2),
			types.OrgID(3),
			types.OrgID(11789772),
			types.OrgID(656485),
		)),
		"organization_white_list is wrong",
	)

	assert.Equal(t, storage.Configuration{
		Driver:        "postgres",
		LogSQLQueries: true,
		PGUsername:    "user",
		PGPassword:    "password",
		PGHost:        "localhost",
		PGPort:        5432,
		PGDBName:      "aggregator",
		PGParams:      "params",
		Type:          "sql",
	}, conf.GetOCPRecommendationsStorageConfiguration())

	assert.Equal(t, storage.RedisConfiguration{
		RedisEndpoint:       "localhost:6379",
		RedisDatabase:       0,
		RedisTimeoutSeconds: 30,
		RedisPassword:       "",
	}, conf.GetRedisConfiguration())

	assert.Equal(t, logger.SentryLoggingConfiguration{
		SentryDSN:         "test.example2.com",
		SentryEnvironment: "test2",
	}, conf.GetSentryLoggingConfiguration())
}

func TestLoadConfigurationFromEnv(t *testing.T) {
	setEnvVariables(t)

	mustLoadConfiguration("/non_existing_path")

	brokerCfg := conf.GetBrokerConfiguration()

	assert.Equal(t, "localhost:9093", brokerCfg.Address)
	assert.Equal(t, "platform.results.ccx", brokerCfg.Topic)
	assert.Equal(t, "aggregator", brokerCfg.Group)
	assert.Equal(t, true, brokerCfg.Enabled)

	assert.Equal(t, server.Configuration{
		Address:                      ":8080",
		APIPrefix:                    "/api/v1/",
		APISpecFile:                  "openapi.json",
		AuthType:                     "xrh",
		Auth:                         false,
		Debug:                        true,
		MaximumFeedbackMessageLength: 255,
		OrgOverviewLimitHours:        2,
	}, conf.GetServerConfiguration())

	orgAllowlist := conf.GetOrganizationAllowlist()

	assert.True(
		t,
		orgAllowlist.Equal(mapset.NewSetWith(
			types.OrgID(1),
			types.OrgID(2),
			types.OrgID(3),
			types.OrgID(11789772),
			types.OrgID(656485),
		)),
		"organization_white_list is wrong",
	)

	assert.Equal(t, storage.Configuration{
		Driver:        "postgres",
		LogSQLQueries: true,
		PGUsername:    "user",
		PGPassword:    "password",
		PGHost:        "localhost",
		PGPort:        5432,
		PGDBName:      "aggregator",
		PGParams:      "params",
		Type:          "sql",
	}, conf.GetOCPRecommendationsStorageConfiguration())

	assert.Equal(t, storage.RedisConfiguration{
		RedisEndpoint:       "default-redis-endpoint",
		RedisDatabase:       42,
		RedisTimeoutSeconds: 0,
		RedisPassword:       "top secret",
	}, conf.GetRedisConfiguration())

	assert.Equal(t, logger.SentryLoggingConfiguration{
		SentryDSN:         "test.example.com",
		SentryEnvironment: "test",
	}, conf.GetSentryLoggingConfiguration())
}

func TestGetLoggingConfigurationDefault(t *testing.T) {
	setEnvVariables(t)

	mustLoadConfiguration("/non_existing_path")

	assert.Equal(t, logger.LoggingConfiguration{
		Debug:                      false,
		LogLevel:                   "",
		LoggingToCloudWatchEnabled: false,
	},
		conf.GetLoggingConfiguration())
}

func TestGetLoggingConfigurationFromEnv(t *testing.T) {
	setEnvVariables(t)
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__LOGGING__DEBUG", "true")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__LOGGING__LOG_LEVEL", "info")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__LOGGING__LOGGING_TO_CLOUD_WATCH_ENABLED", "true")

	mustLoadConfiguration("/non_existing_path")

	assert.Equal(t, logger.LoggingConfiguration{
		Debug:                      true,
		LogLevel:                   "info",
		LoggingToCloudWatchEnabled: true,
	},
		conf.GetLoggingConfiguration())
}

func TestGetCloudWatchConfigurationDefault(t *testing.T) {
	mustLoadConfiguration("/non_existing_path")

	assert.Equal(t, logger.CloudWatchConfiguration{
		AWSAccessID:             "",
		AWSSecretKey:            "",
		AWSSessionToken:         "",
		AWSRegion:               "",
		LogGroup:                "",
		StreamName:              "",
		CreateStreamIfNotExists: false,
		Debug:                   false,
	}, conf.GetCloudWatchConfiguration())
}

func TestGetMetricsConfiguration(t *testing.T) {
	helpers.FailOnError(t, os.Chdir(".."))
	TestLoadConfiguration(t)

	metricsCfg := conf.GetMetricsConfiguration()
	assert.Equal(t, "aggregator", metricsCfg.Namespace)
}

// TestLoadConfigurationFromEnvVariableClowderEnabled tests loading the config
// file for testing from an environment variable. Clowder config is enabled in
// this case.
func TestLoadConfigurationFromEnvVariableClowderEnabled(t *testing.T) {
	var testDB = "test_db"
	os.Clearenv()

	// explicit database and broker config
	clowder.LoadedConfig = &clowder.AppConfig{
		Database: &clowder.DatabaseConfig{
			Name: testDB,
		},
	}

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE", "../tests/config1")
	mustSetEnv(t, "ACG_CONFIG", "tests/clowder_config.json")

	err := conf.LoadConfiguration("config")
	assert.NoError(t, err, "Failed loading configuration file")

	// Set the org allow list to avoid loading a non-existing file
	conf.Config.Broker.OrgAllowlistEnabled = false

	// retrieve broker config
	brokerCfg := conf.GetBrokerConfiguration()
	storageCfg := conf.GetOCPRecommendationsStorageConfiguration()

	// check
	assert.Equal(t, "localhost:29092", brokerCfg.Address, "Broker doesn't match")
	assert.Equal(t, "platform.results.ccx", brokerCfg.Topic, "Topic doesn't match")
	assert.Equal(t, testDB, storageCfg.PGDBName)
}

// TestClowderConfigForKafka tests loading the config file for testing from an
// environment variable. Clowder config is enabled in this case, checking the Kafka
// configuration.
func TestClowderConfigForKafka(t *testing.T) {
	os.Clearenv()

	var hostname = "kafka"
	var port = 9092
	var topicName = "platform.results.ccx"
	var newTopicName = "new-topic-name"

	// explicit database and broker config
	clowder.LoadedConfig = &clowder.AppConfig{
		Kafka: &clowder.KafkaConfig{
			Brokers: []clowder.BrokerConfig{
				{
					Hostname: hostname,
					Port:     &port,
				},
			},
		},
	}

	clowder.KafkaTopics = make(map[string]clowder.TopicConfig)
	clowder.KafkaTopics[topicName] = clowder.TopicConfig{
		Name:          newTopicName,
		RequestedName: topicName,
	}

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE", "../tests/config1")
	mustSetEnv(t, "ACG_CONFIG", "tests/clowder_config.json")

	err := conf.LoadConfiguration("config")
	assert.NoError(t, err, "Failed loading configuration file")

	conf.Config.Broker.OrgAllowlistEnabled = false

	brokerCfg := conf.GetBrokerConfiguration()
	assert.Equal(t, fmt.Sprintf("%s:%d", hostname, port), brokerCfg.Address)
	assert.Equal(t, newTopicName, conf.Config.Broker.Topic)
}

// TestClowderConfigForStorage tests loading the config file for testing from an
// environment variable. Clowder config is enabled in this case, checking the database
// configuration.
func TestClowderConfigForStorage(t *testing.T) {
	os.Clearenv()

	var name = "db"
	var hostname = "hostname"
	var port = 8888
	var username = "username"
	var password = "password"

	// explicit database and broker config
	clowder.LoadedConfig = &clowder.AppConfig{
		Database: &clowder.DatabaseConfig{
			Name:     name,
			Hostname: hostname,
			Port:     port,
			Username: username,
			Password: password,
		},
	}

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE", "../tests/config1")
	mustSetEnv(t, "ACG_CONFIG", "tests/clowder_config.json")

	err := conf.LoadConfiguration("config")
	assert.NoError(t, err, "Failed loading configuration file")

	ocpStorageConf := conf.GetOCPRecommendationsStorageConfiguration()
	assert.Equal(t, name, ocpStorageConf.PGDBName)
	assert.Equal(t, hostname, ocpStorageConf.PGHost)
	assert.Equal(t, port, ocpStorageConf.PGPort)
	assert.Equal(t, username, ocpStorageConf.PGUsername)
	assert.Equal(t, password, ocpStorageConf.PGPassword)
	// rest of config outside of clowder must be loaded correctly
	assert.Equal(t, "postgres", ocpStorageConf.Driver)
	assert.Equal(t, "sql", ocpStorageConf.Type)

	// same config loaded for DVO storage in envs using clowder (stage/prod)
	dvoStorageConf := conf.GetDVORecommendationsStorageConfiguration()
	assert.Equal(t, name, dvoStorageConf.PGDBName)
	assert.Equal(t, hostname, dvoStorageConf.PGHost)
	assert.Equal(t, port, dvoStorageConf.PGPort)
	assert.Equal(t, username, dvoStorageConf.PGUsername)
	assert.Equal(t, password, dvoStorageConf.PGPassword)
	// rest of config outside of clowder must be loaded correctly
	assert.Equal(t, "postgres", dvoStorageConf.Driver)
	assert.Equal(t, "sql", dvoStorageConf.Type)
}
