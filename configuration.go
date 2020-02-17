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
	"github.com/spf13/viper"
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
}

func loadBrokerConfiguration() broker.Configuration {
	orgWhitelist := loadOrganizationWhitelist()
	brokerCfg := viper.Sub("broker")
	return broker.Configuration{
		Address:      brokerCfg.GetString("address"),
		Topic:        brokerCfg.GetString("topic"),
		Group:        brokerCfg.GetString("group"),
		Enabled:      brokerCfg.GetBool("enabled"),
		OrgWhitelist: orgWhitelist,
	}
}

func loadOrganizationWhitelist() []types.OrgID {
	var whitelist []types.OrgID

	processingCfg := viper.Sub("processing")
	fileName := processingCfg.GetString("org_whitelist")

	csvFile, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Error opening %v. Error: %v", fileName, err)
	}

	reader := csv.NewReader(bufio.NewReader(csvFile))

	lines, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Error reading CSV file: %v", err)
	}

	for index, line := range lines {
		if index == 0 {
			continue // skip header
		}

		orgID, err := strconv.Atoi(line[0]) // single record per line
		if err != nil {
			log.Fatalf("Organization ID on line %v in whitelist CSV is not numerical. Found value: %v", index+1, line[0])
		}

		whitelist = append(whitelist, types.OrgID(orgID))
	}
	return whitelist
}

func loadStorageConfiguration() storage.Configuration {
	storageCfg := viper.Sub("storage")
	return storage.Configuration{
		Driver:     storageCfg.GetString("driver"),
		DataSource: storageCfg.GetString("datasource"),
	}
}

func loadServerConfiguration() server.Configuration {
	serverCfg := viper.Sub("server")
	return server.Configuration{
		Address:   serverCfg.GetString("address"),
		APIPrefix: serverCfg.GetString("api_prefix"),
	}
}
