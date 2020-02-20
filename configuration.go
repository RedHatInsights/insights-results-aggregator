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
	"fmt"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
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
	return broker.Configuration{
		Address: brokerCfg.GetString("address"),
		Topic:   brokerCfg.GetString("topic"),
		Group:   brokerCfg.GetString("group"),
		Enabled: brokerCfg.GetBool("enabled"),
	}
}

func loadStorageConfiguration() storage.Configuration {
	return storage.Configuration{
		Driver:     storageCfg.GetString("driver"),
		DataSource: storageCfg.GetString("datasource"),
	}
}

func loadServerConfiguration() server.Configuration {
	return server.Configuration{
		Address:   serverCfg.GetString("address"),
		APIPrefix: serverCfg.GetString("api_prefix"),
	}
}
