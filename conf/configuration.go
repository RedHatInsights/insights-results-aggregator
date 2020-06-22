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

package conf

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	mapset "github.com/deckarep/golang-set"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/logger"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	configFileEnvVariableName   = "INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE"
	defaultOrgWhiteListFileName = "org_whitelist.csv"
	defaultContentPath          = "/rules-content"
)

// Config has exactly the same structure as *.toml file
var Config struct {
	Broker     broker.Configuration `mapstructure:"broker" toml:"broker"`
	Server     server.Configuration `mapstructure:"server" toml:"server"`
	Processing struct {
		OrgWhiteListFile string `mapstructure:"org_whitelist_file" toml:"org_whitelist_file"`
	} `mapstructure:"processing"`
	Storage    storage.Configuration          `mapstructure:"storage" toml:"storage"`
	Logging    logger.LoggingConfiguration    `mapstructure:"logging" toml:"logging"`
	CloudWatch logger.CloudWatchConfiguration `mapstructure:"cloudwatch" toml:"cloudwatch"`
}

// LoadConfiguration loads configuration from defaultConfigFile, file set in configFileEnvVariableName or from env
func LoadConfiguration(defaultConfigFile string) error {
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
		fakeTomlConfigWriter := new(bytes.Buffer)

		err := toml.NewEncoder(fakeTomlConfigWriter).Encode(Config)
		if err != nil {
			return err
		}

		fakeTomlConfig := fakeTomlConfigWriter.String()

		viper.SetConfigType("toml")

		err = viper.ReadConfig(strings.NewReader(fakeTomlConfig))
		if err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("fatal error config file: %s", err)
	}

	// override config from env if there's variable in env

	const envPrefix = "INSIGHTS_RESULTS_AGGREGATOR_"

	viper.AutomaticEnv()
	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "__"))

	return viper.Unmarshal(&Config)
}

// GetBrokerConfiguration returns broker configuration
func GetBrokerConfiguration() broker.Configuration {
	Config.Broker.OrgWhitelist = getOrganizationWhitelist()

	return Config.Broker
}

func getOrganizationWhitelist() mapset.Set {
	if !Config.Broker.OrgWhitelistEnabled {
		return nil
	}

	if len(Config.Processing.OrgWhiteListFile) == 0 {
		Config.Processing.OrgWhiteListFile = defaultOrgWhiteListFileName
	}

	orgWhiteListFileData, err := ioutil.ReadFile(Config.Processing.OrgWhiteListFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Organization whitelist file could not be opened")
	}

	whitelist, err := loadWhitelistFromCSV(bytes.NewBuffer(orgWhiteListFileData))
	if err != nil {
		log.Fatal().Err(err).Msg("Whitelist CSV could not be processed")
	}

	return whitelist
}

// GetStorageConfiguration returns storage configuration
func GetStorageConfiguration() storage.Configuration {
	return Config.Storage
}

// GetLoggingConfiguration returns logging configuration
func GetLoggingConfiguration() logger.LoggingConfiguration {
	return Config.Logging
}

// GetCloudWatchConfiguration returns cloudwatch configuration
func GetCloudWatchConfiguration() logger.CloudWatchConfiguration {
	return Config.CloudWatch
}

// GetServerConfiguration returns server configuration
func GetServerConfiguration() server.Configuration {
	err := checkIfFileExists(Config.Server.APISpecFile)
	if err != nil {
		log.Fatal().Err(err).Msg("All customer facing APIs MUST serve the current OpenAPI specification")
	}

	return Config.Server
}

// checkIfFileExists returns nil if path doesn't exist or isn't a file, otherwise it returns corresponding error
func checkIfFileExists(path string) error {
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("OpenAPI spec file path does not exist. Path: '%v'", path)
	} else if err != nil {
		return err
	}

	if fileMode := fileInfo.Mode(); !fileMode.IsRegular() {
		return fmt.Errorf("OpenAPI spec file path is not a file. Path: '%v'", path)
	}

	return nil
}

// loadWhitelistFromCSV creates a new CSV reader and returns a Set of whitelisted org. IDs
func loadWhitelistFromCSV(r io.Reader) (mapset.Set, error) {
	whitelist := mapset.NewSet()

	reader := csv.NewReader(r)

	lines, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV file: %v", err)
	}

	for index, line := range lines {
		if index == 0 {
			continue // skip header
		}

		orgID, err := strconv.ParseUint(line[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf(
				"organization ID on line %v in whitelist CSV is not numerical. Found value: %v",
				index+1, line[0],
			)
		}

		whitelist.Add(types.OrgID(orgID))
	}

	return whitelist, nil
}
