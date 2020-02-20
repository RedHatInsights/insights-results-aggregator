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
	"github.com/deckarep/golang-set"
	"github.com/spf13/viper"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var (
	brokerCfg  *viper.Viper
	storageCfg *viper.Viper
	serverCfg  *viper.Viper
)

func loadConfiguration(defaultConfigFile string) {
	configFile, specified := os.LookupEnv("INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE")
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
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s", err))
	}

	// override config from env if there's variable in env

	// workaround because if this issue https://github.com/spf13/viper/issues/507
	storageCfg = viper.Sub("storage")
	brokerCfg = viper.Sub("broker")
	serverCfg = viper.Sub("server")

	const envPrefix = "INSIGHTS_RESULTS_AGGREGATOR__"

	brokerCfg.AutomaticEnv()
	brokerCfg.SetEnvPrefix(envPrefix + "BROKER_")

	storageCfg.AutomaticEnv()
	storageCfg.SetEnvPrefix(envPrefix + "STORAGE_")

	serverCfg.AutomaticEnv()
	serverCfg.SetEnvPrefix(envPrefix + "SERVER_")
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

// getWhitelistFileName retrieves filename of organization whitelist from config file
func getWhitelistFileName() string {
	processingCfg := viper.Sub("processing")
	fileName := processingCfg.GetString("org_whitelist")
	return fileName
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
	fileName := getWhitelistFileName()
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
		PGUsername:       storageCfg.GetString("pg_username"),
		PGPassword:       storageCfg.GetString("pg_password"),
		PGHost:           storageCfg.GetString("pg_host"),
		PGPort:           storageCfg.GetInt("pg_port"),
		PGDBName:         storageCfg.GetString("pg_db_name"),
		PGParams:         storageCfg.GetString("pg_params"),
	}
}

func loadServerConfiguration() server.Configuration {
	return server.Configuration{
		Address:   serverCfg.GetString("address"),
		APIPrefix: serverCfg.GetString("api_prefix"),
		Debug:     serverCfg.GetBool("debug"),
	}
}
