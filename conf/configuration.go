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

// Package conf contains definition of data type named ConfigStruct that
// represents configuration of Insights Results Aggregator. This package also
// contains function named LoadConfiguration that can be used to load
// configuration from provided configuration file and/or from environment
// variables. Additionally several specific functions named
// GetBrokerConfiguration, GetStorageConfiguration, GetLoggingConfiguration,
// GetCloudWatchConfiguration, and GetServerConfiguration are to be used to
// return specific configuration options.
//
// Generated documentation is available at:
// https://godoc.org/github.com/RedHatInsights/insights-results-aggregator/conf
//
// Documentation in literate-programming-style is available at:
// https://redhatinsights.github.io/insights-results-aggregator/packages/conf/configuration.html
package conf

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/RedHatInsights/insights-operator-utils/logger"
	mapset "github.com/deckarep/golang-set"
	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	configFileEnvVariableName   = "INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE"
	defaultOrgAllowlistFileName = "org_allowlist.csv"
	defaultStorageBackend       = "ocp_recommendations"

	noBrokerConfig = "warning: no broker configurations found in clowder config"
	noSaslConfig   = "warning: SASL configuration is missing"
	noTopicMapping = "warning: no kafka mapping found for topic %s"
	noStorage      = "warning: no storage section in Clowder config"
)

// MetricsConfiguration holds metrics related configuration
type MetricsConfiguration struct {
	Namespace string `mapstructure:"namespace" toml:"namespace"`
}

// StorageBackend contains global storage backend configuration
type StorageBackend struct {
	Use string `mapstructure:"use" toml:"use"`
}

// ConfigStruct is a structure holding the whole service configuration
type ConfigStruct struct {
	Broker     broker.Configuration `mapstructure:"broker" toml:"broker"`
	Server     server.Configuration `mapstructure:"server" toml:"server"`
	Processing struct {
		OrgAllowlistFile string `mapstructure:"org_allowlist_file" toml:"org_allowlist_file"`
	} `mapstructure:"processing"`
	OCPRecommendationsStorage storage.Configuration             `mapstructure:"ocp_recommendations_storage" toml:"ocp_recommendations_storage"`
	DVORecommendationsStorage storage.Configuration             `mapstructure:"dvo_recommendations_storage" toml:"dvo_recommendations_storage"`
	StorageBackend            StorageBackend                    `mapstructure:"storage_backend" toml:"storage_backend"`
	Logging                   logger.LoggingConfiguration       `mapstructure:"logging" toml:"logging"`
	CloudWatch                logger.CloudWatchConfiguration    `mapstructure:"cloudwatch" toml:"cloudwatch"`
	Redis                     storage.RedisConfiguration        `mapstructure:"redis" toml:"redis"`
	Metrics                   MetricsConfiguration              `mapstructure:"metrics" toml:"metrics"`
	SentryLoggingConf         logger.SentryLoggingConfiguration `mapstructure:"sentry" toml:"sentry"`
	KafkaZerologConf          logger.KafkaZerologConfiguration  `mapstructure:"kafka_zerolog" toml:"kafka_zerolog"`
}

// Config has exactly the same structure as *.toml file
var Config ConfigStruct

// LoadConfiguration loads configuration from defaultConfigFile, file set in
// configFileEnvVariableName or from env or from Clowder.
func LoadConfiguration(defaultConfigFile string) error {
	configFile, specified := os.LookupEnv(configFileEnvVariableName)
	if specified {
		// we need to separate the directory name and filename without
		// extension
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
		// viper is not smart enough to understand the structure of
		// config by itself
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

	err = viper.Unmarshal(&Config)
	if err != nil {
		return fmt.Errorf("fatal - can not unmarshal configuration: %s", err)
	}

	if err := updateConfigFromClowder(&Config); err != nil {
		fmt.Println("Error loading clowder configuration")
		return err
	}

	// manage default storage backend
	if Config.StorageBackend.Use == "" {
		// don't use right now as it breaks BDD:
		// fmt.Println("Setting default storage backend")
		Config.StorageBackend.Use = defaultStorageBackend
	}

	// everything's should be ok
	return nil
}

// GetBrokerConfiguration returns broker configuration
func GetBrokerConfiguration() broker.Configuration {
	Config.Broker.OrgAllowlist = getOrganizationAllowlist()

	return Config.Broker
}

func getOrganizationAllowlist() mapset.Set {
	if !Config.Broker.OrgAllowlistEnabled {
		return nil
	}

	if Config.Processing.OrgAllowlistFile == "" {
		Config.Processing.OrgAllowlistFile = defaultOrgAllowlistFileName
	}

	orgAllowlistFileData, err := os.ReadFile(Config.Processing.OrgAllowlistFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Organization allowlist file could not be opened")
	}

	allowlist, err := loadAllowlistFromCSV(bytes.NewBuffer(orgAllowlistFileData))
	if err != nil {
		log.Fatal().Err(err).Msg("Allowlist CSV could not be processed")
	}

	return allowlist
}

// GetStorageBackendConfiguration returns storage backend configuration
func GetStorageBackendConfiguration() StorageBackend {
	return Config.StorageBackend
}

// GetOCPRecommendationsStorageConfiguration returns storage configuration for
// OCP recommendations database
func GetOCPRecommendationsStorageConfiguration() storage.Configuration {
	return Config.OCPRecommendationsStorage
}

// GetDVORecommendationsStorageConfiguration returns storage configuration for
// DVO recommendations database
func GetDVORecommendationsStorageConfiguration() storage.Configuration {
	return Config.DVORecommendationsStorage
}

// GetRedisConfiguration returns Redis storage configuration
func GetRedisConfiguration() storage.RedisConfiguration {
	return Config.Redis
}

// GetLoggingConfiguration returns logging configuration
func GetLoggingConfiguration() logger.LoggingConfiguration {
	return Config.Logging
}

// GetCloudWatchConfiguration returns cloudwatch configuration
func GetCloudWatchConfiguration() logger.CloudWatchConfiguration {
	return Config.CloudWatch
}

// GetSentryLoggingConfiguration returns the sentry log configuration
func GetSentryLoggingConfiguration() logger.SentryLoggingConfiguration {
	return Config.SentryLoggingConf
}

// GetKafkaZerologConfiguration returns the kafkazero log configuration
func GetKafkaZerologConfiguration() logger.KafkaZerologConfiguration {
	return Config.KafkaZerologConf
}

// GetServerConfiguration returns server configuration
func GetServerConfiguration() server.Configuration {
	err := checkIfFileExists(Config.Server.APISpecFile)
	if err != nil {
		log.Fatal().Err(err).Msg("All customer facing APIs MUST serve the current OpenAPI specification")
	}

	return Config.Server
}

// GetMetricsConfiguration returns metrics configuration
func GetMetricsConfiguration() MetricsConfiguration {
	return Config.Metrics
}

// checkIfFileExists returns nil if path doesn't exist or isn't a file,
// otherwise it returns corresponding error
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

// loadAllowlistFromCSV creates a new CSV reader and returns a Set of allowlisted org. IDs
func loadAllowlistFromCSV(r io.Reader) (mapset.Set, error) {
	allowlist := mapset.NewSet()

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
				"organization ID on line %v in allowlist CSV is not numerical. Found value: %v",
				index+1, line[0],
			)
		}

		allowlist.Add(types.OrgID(orgID))
	}

	return allowlist, nil
}

// updateConfigFromClowder updates the current config with the values defined in clowder
func updateConfigFromClowder(c *ConfigStruct) error {
	if !clowder.IsClowderEnabled() {
		fmt.Println("Clowder is disabled")
		return nil
	}

	// can not use Zerolog at this moment!
	fmt.Println("Clowder is enabled")
	if clowder.LoadedConfig.Kafka == nil {
		fmt.Println("No Kafka configuration available in Clowder, using default one")
	} else {
		if len(clowder.LoadedConfig.Kafka.Brokers) > 0 {
			broker := clowder.LoadedConfig.Kafka.Brokers[0]
			// port can be empty in clowder, so taking it into account
			if broker.Port != nil {
				c.Broker.Address = fmt.Sprintf("%s:%d", broker.Hostname, *broker.Port)
			} else {
				c.Broker.Address = broker.Hostname
			}

			// SSL config
			if broker.Authtype != nil {
				if broker.Sasl != nil {
					c.Broker.SaslUsername = *broker.Sasl.Username
					c.Broker.SaslPassword = *broker.Sasl.Password
					c.Broker.SaslMechanism = *broker.Sasl.SaslMechanism
					c.Broker.SecurityProtocol = *broker.SecurityProtocol
					if caPath, err := clowder.LoadedConfig.KafkaCa(broker); err == nil {
						c.Broker.CertPath = caPath
					}
				} else {
					fmt.Println(noSaslConfig)
				}
			}
		} else {
			fmt.Println(noBrokerConfig)
		}

		if err := updateTopicsMapping(c); err != nil {
			fmt.Println("warning: an error occurred when applying new topics")
		}
	}

	// get DB configuration from clowder
	if clowder.LoadedConfig.Database != nil {
		storageConfig := storage.Configuration{
			PGDBName:   clowder.LoadedConfig.Database.Name,
			PGHost:     clowder.LoadedConfig.Database.Hostname,
			PGPort:     clowder.LoadedConfig.Database.Port,
			PGUsername: clowder.LoadedConfig.Database.Username,
			PGPassword: clowder.LoadedConfig.Database.Password,
		}

		// we're currently using the same storage in all Clowderized envs (stage/prod)
		c.OCPRecommendationsStorage = storageConfig
		c.DVORecommendationsStorage = storageConfig
	} else {
		fmt.Println(noStorage)
	}

	return nil
}

func updateTopicsMapping(c *ConfigStruct) error {
	// Updating topics from clowder mapping if available
	if topicCfg, ok := clowder.KafkaTopics[c.Broker.Topic]; ok {
		c.Broker.Topic = topicCfg.Name
	} else {
		fmt.Printf("warning: no kafka mapping found for topic %s", c.Broker.Topic)
	}

	if topicCfg, ok := clowder.KafkaTopics[c.Broker.DeadLetterQueueTopic]; ok {
		c.Broker.DeadLetterQueueTopic = topicCfg.Name
	} else {
		fmt.Printf(noTopicMapping, c.Broker.DeadLetterQueueTopic)
	}

	if topicCfg, ok := clowder.KafkaTopics[c.Broker.PayloadTrackerTopic]; ok {
		c.Broker.PayloadTrackerTopic = topicCfg.Name
	} else {
		fmt.Printf(noTopicMapping, c.Broker.PayloadTrackerTopic)
	}

	return nil
}
