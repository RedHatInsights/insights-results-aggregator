/*
Copyright © 2020 Red Hat, Inc.

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

package main_test

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"

	main "github.com/RedHatInsights/insights-results-aggregator"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

func mustLoadConfiguration(path string) {
	err := main.LoadConfiguration(path)
	if err != nil {
		panic(err)
	}
}

func removeFile(t *testing.T, filename string) {
	err := os.Remove(filename)
	if err != nil {
		t.Fatal(err)
	}
}

// TestLoadConfiguration loads a configuration file for testing
func TestLoadConfiguration(t *testing.T) {
	err := os.Unsetenv("INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE")
	if err != nil {
		t.Fatal(err)
	}

	mustLoadConfiguration("tests/config1")
}

// TestLoadConfigurationEnvVariable tests loading the config. file for testing from an environment variable
func TestLoadConfigurationEnvVariable(t *testing.T) {
	err := os.Setenv("INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE", "tests/config1")
	if err != nil {
		t.Fatal(err)
	}

	mustLoadConfiguration("foobar")
}

// TestLoadingConfigurationFailure tests loading a non-existent configuration file
func TestLoadingConfigurationFailure(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic as expected")
		}
	}()

	err := os.Unsetenv("INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE")
	if err != nil {
		t.Fatal(err)
	}

	mustLoadConfiguration("this does not exist")
}

// TestLoadBrokerConfiguration tests loading the broker configuration sub-tree
func TestLoadBrokerConfiguration(t *testing.T) {
	TestLoadConfiguration(t)

	brokerCfg := main.GetBrokerConfiguration()

	assert.Equal(t, "localhost:9092", brokerCfg.Address)
	assert.Equal(t, "platform.results.ccx", brokerCfg.Topic)
	assert.Equal(t, "aggregator", brokerCfg.Group)
}

// TestLoadServerConfiguration tests loading the server configuration sub-tree
func TestLoadServerConfiguration(t *testing.T) {
	TestLoadConfiguration(t)

	serverCfg := main.GetServerConfiguration()

	assert.Equal(t, ":8080", serverCfg.Address)
	assert.Equal(t, "/api/v1/", serverCfg.APIPrefix)
}

// TestLoadStorageConfiguration tests loading the storage configuration sub-tree
func TestLoadStorageConfiguration(t *testing.T) {
	TestLoadConfiguration(t)

	storageCfg := main.GetStorageConfiguration()

	assert.Equal(t, "sqlite3", storageCfg.Driver)
	assert.Equal(t, "xyzzy", storageCfg.SQLiteDataSource)
}

// TestLoadConfigurationOverrideFromEnv tests overriding configuration by env variables
func TestLoadConfigurationOverrideFromEnv(t *testing.T) {
	os.Clearenv()

	mustLoadConfiguration("tests/config1")

	storageCfg := main.GetStorageConfiguration()
	assert.Equal(t, storage.Configuration{
		Driver:           "sqlite3",
		SQLiteDataSource: "xyzzy",
		PGUsername:       "user",
		PGPassword:       "password",
		PGHost:           "localhost",
		PGPort:           5432,
		PGDBName:         "aggregator",
		PGParams:         "",
	}, storageCfg)

	err := os.Setenv("INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER", "postgres")

	if err != nil {
		t.Fatal(err)
	}

	err = os.Setenv(
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__PG_PASSWORD",
		"some very secret password",
	)

	if err != nil {
		t.Fatal(err)
	}

	storageCfg = main.GetStorageConfiguration()
	assert.Equal(t, storage.Configuration{
		Driver:           "postgres",
		SQLiteDataSource: "xyzzy",
		PGUsername:       "user",
		PGPassword:       "some very secret password",
		PGHost:           "localhost",
		PGPort:           5432,
		PGDBName:         "aggregator",
		PGParams:         "",
	}, storageCfg)
}

// TestLoadOrganizationWhitelist tests if the whitelist CSV file gets loaded properly
func TestLoadOrganizationWhitelist(t *testing.T) {
	expectedWhitelist := mapset.NewSetWith(
		types.OrgID(1),
		types.OrgID(2),
		types.OrgID(3),
	)
	orgWhitelist := main.GetOrganizationWhitelist()
	if equal := orgWhitelist.Equal(expectedWhitelist); !equal {
		t.Errorf(
			"Org whitelist did not load properly. Order of elements does not matter. Expected %v. Got %v",
			expectedWhitelist, orgWhitelist,
		)
	}
}

// TestLoadWhitelistFromCSVExtraParam tests incorrect CSV format
func TestLoadWhitelistFromCSVExtraParam(t *testing.T) {
	extraParamCSV := `OrgID
1,2
3
`
	r := strings.NewReader(extraParamCSV)
	_, err := main.LoadWhitelistFromCSV(r)
	if err == nil {
		t.Errorf("Invalid CSV got loaded")
	}
}

// TestLoadWhitelistFromCSVNonInt tests non-integer ID in CSV
func TestLoadWhitelistFromCSVNonInt(t *testing.T) {
	nonIntIDCSV := `OrgID
str
3
`
	r := strings.NewReader(nonIntIDCSV)
	_, err := main.LoadWhitelistFromCSV(r)
	if err == nil {
		t.Errorf("Non-integer organization ID got parsed")
	}
}

func TestLoadConfigurationFromFile(t *testing.T) {
	config := `[broker]
		address = "localhost:9092"
		topic = "platform.results.ccx"
		group = "aggregator"
		enabled = true

		[server]
		address = ":8080"
		api_prefix = "/api/v1/"
		api_spec_file = "openapi.json"
		debug = true

		[processing]
		org_whitelist = "org_whitelist.csv"

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
	if err != nil {
		t.Fatal(err)
	}

	defer removeFile(t, tmpFilename)

	os.Clearenv()
	mustSetEnv(t, main.ConfigFileEnvVariableName, tmpFilename)
	mustLoadConfiguration("tests/config1")

	brokerCfg := main.GetBrokerConfiguration()

	assert.Equal(t, "localhost:9092", brokerCfg.Address)
	assert.Equal(t, "platform.results.ccx", brokerCfg.Topic)
	assert.Equal(t, "aggregator", brokerCfg.Group)
	assert.Equal(t, true, brokerCfg.Enabled)

	assert.Equal(t, server.Configuration{
		Address:     ":8080",
		APIPrefix:   "/api/v1/",
		APISpecFile: "openapi.json",
		Debug:       true,
	}, main.GetServerConfiguration())

	orgWhiteList := main.GetOrganizationWhitelist()

	assert.True(
		t,
		orgWhiteList.Equal(mapset.NewSetWith(
			types.OrgID(1),
			types.OrgID(2),
			types.OrgID(3),
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
	}, main.GetStorageConfiguration())
}

func GetTmpConfigFile(configData string) (configFile string, err error) {
	tmpFile, err := ioutil.TempFile("/tmp", "tmp_config_*.toml")
	if err != nil {
		return "", err
	}

	defer func() {
		err = tmpFile.Close()
		if err != nil {
			configFile = ""
		}
	}()

	if _, err := tmpFile.Write([]byte(configData)); err != nil {
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

func mustSetEnv(t *testing.T, key, val string) {
	err := os.Setenv(key, val)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfigurationFromEnv(t *testing.T) {
	os.Clearenv()

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__BROKER__ADDRESS", "localhost:9093")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__BROKER__TOPIC", "platform.results.ccx")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__BROKER__GROUP", "aggregator")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__BROKER__ENABLED", "true")

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__SERVER__ADDRESS", ":8080")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__SERVER__API_PREFIX", "/api/v1/")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__SERVER__API_SPEC_FILE", "openapi.json")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__SERVER__DEBUG", "true")

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__PROCESSING__ORG_WHITELIST", "org_whitelist.csv")

	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER", "sqlite3")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE", ":memory:")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__PG_USERNAME", "user")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__PG_PASSWORD", "password")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__PG_HOST", "localhost")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__PG_PORT", "5432")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__PG_DB_NAME", "aggregator")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__PG_PARAMS", "params")
	mustSetEnv(t, "INSIGHTS_RESULTS_AGGREGATOR__STORAGE__LOG_SQL_QUERIES", "true")

	mustLoadConfiguration("/non_existing_path")

	brokerCfg := main.GetBrokerConfiguration()

	assert.Equal(t, "localhost:9093", brokerCfg.Address)
	assert.Equal(t, "platform.results.ccx", brokerCfg.Topic)
	assert.Equal(t, "aggregator", brokerCfg.Group)
	assert.Equal(t, true, brokerCfg.Enabled)

	assert.Equal(t, server.Configuration{
		Address:     ":8080",
		APIPrefix:   "/api/v1/",
		APISpecFile: "openapi.json",
		Debug:       true,
	}, main.GetServerConfiguration())

	orgWhiteList := main.GetOrganizationWhitelist()

	assert.True(
		t,
		orgWhiteList.Equal(mapset.NewSetWith(
			types.OrgID(1),
			types.OrgID(2),
			types.OrgID(3),
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
	}, main.GetStorageConfiguration())
}
