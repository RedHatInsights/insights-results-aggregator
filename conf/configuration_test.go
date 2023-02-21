/*
Copyright © 2020, 2021, 2022 Red Hat, Inc.

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

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER", "sqlite3")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE", ":memory:")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__PG_USERNAME", "user")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__PG_PASSWORD", "password")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__PG_HOST", "localhost")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__PG_PORT", "5432")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__PG_DB_NAME", "aggregator")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__PG_PARAMS", "params")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__LOG_SQL_QUERIES", "true")

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__CONTENT__PATH", "/rules-content")
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
func TestLoadConfiguration(t *testing.T) {
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

// TestLoadStorageConfiguration tests loading the storage configuration sub-tree
func TestLoadStorageConfiguration(t *testing.T) {
	TestLoadConfiguration(t)

	storageCfg := conf.GetStorageConfiguration()

	assert.Equal(t, "sqlite3", storageCfg.Driver)
	assert.Equal(t, ":memory:", storageCfg.SQLiteDataSource)
}

// TestLoadConfigurationOverrideFromEnv tests overriding configuration by env variables
func TestLoadConfigurationOverrideFromEnv(t *testing.T) {
	os.Clearenv()

	const configPath = "../tests/config1"

	mustLoadConfiguration(configPath)

	storageCfg := conf.GetStorageConfiguration()
	assert.Equal(t, storage.Configuration{
		Driver:           "sqlite3",
		SQLiteDataSource: ":memory:",
		PGUsername:       "user",
		PGPassword:       "password",
		PGHost:           "localhost",
		PGPort:           5432,
		PGDBName:         "aggregator",
		PGParams:         "",
	}, storageCfg)

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER", "postgres")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__PG_PASSWORD", "some very secret password")

	mustLoadConfiguration(configPath)

	storageCfg = conf.GetStorageConfiguration()
	assert.Equal(t, storage.Configuration{
		Driver:           "postgres",
		SQLiteDataSource: ":memory:",
		PGUsername:       "user",
		PGPassword:       "some very secret password",
		PGHost:           "localhost",
		PGPort:           5432,
		PGDBName:         "aggregator",
		PGParams:         "",
	}, storageCfg)
}

// TestLoadOrganizationAllowlist tests if the allowlist CSV file gets loaded properly
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

		[content]
		path = "/rules-content"

		[processing]
		org_allowlist_file = "org_allowlist.csv"

		[server]
		address = ":8080"
		api_prefix = "/api/v1/"
		api_spec_file = "openapi.json"
		debug = true

		[storage]
		db_driver = "sqlite3"
		sqlite_datasource = ":memory:"
		pg_username = "user"
		pg_password = "password"
		pg_host = "localhost"
		pg_port = 5432
		pg_db_name = "aggregator"
		pg_params = "params"
		log_sql_queries = true
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
		Driver:           "sqlite3",
		SQLiteDataSource: ":memory:",
		LogSQLQueries:    true,
		PGUsername:       "user",
		PGPassword:       "password",
		PGHost:           "localhost",
		PGPort:           5432,
		PGDBName:         "aggregator",
		PGParams:         "params",
	}, conf.GetStorageConfiguration())
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
		Driver:           "sqlite3",
		SQLiteDataSource: ":memory:",
		LogSQLQueries:    true,
		PGUsername:       "user",
		PGPassword:       "password",
		PGHost:           "localhost",
		PGPort:           5432,
		PGDBName:         "aggregator",
		PGParams:         "params",
	}, conf.GetStorageConfiguration())
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
	storageCfg := conf.GetStorageConfiguration()

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
