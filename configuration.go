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

// Implementation of configuration loading for aggregator
package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/spf13/viper"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	configFileEnvVariableName = "INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE"
	emptyConfig               = `
		[broker]
		[server]
		[processing]
		[metrics]
		[logging]
		[storage]
	`
)

var (
	brokerCfg     *viper.Viper
	storageCfg    *viper.Viper
	serverCfg     *viper.Viper
	processingCfg *viper.Viper
	metricsCfg    *viper.Viper
	loggingCfg    *viper.Viper
)

func loadConfiguration(defaultConfigFile string) {
	configFile, specified := os.LookupEnv(configFileEnvVariableName)
	if specified {
		// we need to separate the directory name and filename without extension
		directory, basename := filepath.Split(configFile)
		file := strings.TrimSuffix(basename, filepath.Ext(basename))
		// parse the configuration
		viper.SetConfigName(file)
		viper.AddConfigPath(directory)
	} else {
		// parse the configuration
		viper.SetConfigName(defaultConfigFile)
		viper.AddConfigPath(".")
	}

	err := viper.ReadInConfig()
	if _, isNotFoundError := err.(viper.ConfigFileNotFoundError); !specified && isNotFoundError {
		// viper is not smart enough to understand the structure of config by itself
		viper.SetConfigType("toml")
		err := viper.ReadConfig(strings.NewReader(emptyConfig))
		if err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	// override config from env if there's variable in env

	// workaround because if this issue https://github.com/spf13/viper/issues/507
	storageCfg = viper.Sub("storage")
	brokerCfg = viper.Sub("broker")
	serverCfg = viper.Sub("server")
	processingCfg = viper.Sub("processing")
	metricsCfg = viper.Sub("metrics")
	loggingCfg = viper.Sub("logging")

	const envPrefix = "INSIGHTS_RESULTS_AGGREGATOR__"

	brokerCfg.AutomaticEnv()
	brokerCfg.SetEnvPrefix(envPrefix + "BROKER_")

	storageCfg.AutomaticEnv()
	storageCfg.SetEnvPrefix(envPrefix + "STORAGE_")

	serverCfg.AutomaticEnv()
	serverCfg.SetEnvPrefix(envPrefix + "SERVER_")

	processingCfg.AutomaticEnv()
	processingCfg.SetEnvPrefix(envPrefix + "PROCESSING_")

	metricsCfg.AutomaticEnv()
	metricsCfg.SetEnvPrefix(envPrefix + "METRICS_")

	loggingCfg.AutomaticEnv()
	loggingCfg.SetEnvPrefix(envPrefix + "LOGGING_")
}

func loadBrokerConfiguration() broker.Configuration {
	orgWhitelist := loadOrganizationWhitelist()

	return broker.Configuration{
		Address:      brokerCfg.GetString("address"),
		Topic:        brokerCfg.GetString("topic"),
		Group:        brokerCfg.GetString("group"),
		Enabled:      brokerCfg.GetBool("enabled"),
		OrgWhitelist: orgWhitelist,
	}
}

// createReaderFromFile creates a io.Reader from the given file
func createReaderFromFile(fileName string) (io.Reader, error) {
	csvFile, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("Error opening %v. Error: %v", fileName, err)
	}
	reader := bufio.NewReader(csvFile)
	return reader, nil
}

// loadWhitelistFromCSV creates a new CSV reader and returns a Set of whitelisted org. IDs
func loadWhitelistFromCSV(r io.Reader) (mapset.Set, error) {
	whitelist := mapset.NewSet()

	reader := csv.NewReader(r)

	lines, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("Error reading CSV file: %v", err)
	}

	for index, line := range lines {
		if index == 0 {
			continue // skip header
		}

		orgID, err := strconv.Atoi(line[0]) // single record per line
		if err != nil {
			return nil, fmt.Errorf("Organization ID on line %v in whitelist CSV is not numerical. Found value: %v", index+1, line[0])
		}

		whitelist.Add(types.OrgID(orgID))
	}
	return whitelist, nil
}

func loadOrganizationWhitelist() mapset.Set {
	fileName := processingCfg.GetString("org_whitelist")
	contentReader, err := createReaderFromFile(fileName)
	if err != nil {
		log.Fatalf("Organization whitelist file could not be opened. Error: %v", err)
	}
	whitelist, err := loadWhitelistFromCSV(contentReader)
	if err != nil {
		log.Fatalf("Whitelist CSV could not be processed. Error: %v", err)
	}
	return whitelist
}

func loadStorageConfiguration() storage.Configuration {
	return storage.Configuration{
		Driver:           storageCfg.GetString("db_driver"),
		SQLiteDataSource: storageCfg.GetString("sqlite_datasource"),
		LogSQLQueries:    storageCfg.GetBool("log_sql_queries"),
		PGUsername:       storageCfg.GetString("pg_username"),
		PGPassword:       storageCfg.GetString("pg_password"),
		PGHost:           storageCfg.GetString("pg_host"),
		PGPort:           storageCfg.GetInt("pg_port"),
		PGDBName:         storageCfg.GetString("pg_db_name"),
		PGParams:         storageCfg.GetString("pg_params"),
	}
}

// getAPISpecFile retrieves the filename of OpenAPI specifications file
func getAPISpecFile(serverCfg *viper.Viper) (string, error) {
	specFile := serverCfg.GetString("api_spec_file")

	fileInfo, err := os.Stat(specFile)
	if os.IsNotExist(err) {
		return specFile, fmt.Errorf("OpenAPI spec file path does not exist. Path: %v", specFile)
	}
	if fileMode := fileInfo.Mode(); !fileMode.IsRegular() {
		return specFile, fmt.Errorf("OpenAPI spec file path is not a file. Path: %v", specFile)
	}

	return specFile, nil
}

func loadServerConfiguration() server.Configuration {
	apiSpecFile, err := getAPISpecFile(serverCfg)
	if err != nil {
		log.Fatalf("All customer facing APIs MUST serve the current OpenAPI specification. Error: %s", err)
	}
	return server.Configuration{
		Address:     serverCfg.GetString("address"),
		APIPrefix:   serverCfg.GetString("api_prefix"),
		APISpecFile: apiSpecFile,
		Debug:       serverCfg.GetBool("debug"),
	}
}
