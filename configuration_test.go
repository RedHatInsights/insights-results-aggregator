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
	"github.com/RedHatInsights/insights-results-aggregator"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/deckarep/golang-set"
	"os"
	"strings"
	"testing"
)

// TestLoadConfiguration loads a configuration file for testing
func TestLoadConfiguration(t *testing.T) {
	err := os.Unsetenv("INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE")
	if err != nil {
		t.Fatal(err)
	}
	main.LoadConfiguration("tests/config1")
}

// TestLoadConfigurationEnvVariable tests loading the config. file for testing from an environemnt variable
func TestLoadConfigurationEnvVariable(t *testing.T) {
	err := os.Setenv("INSIGHTS_RESULTS_AGGREGATOR_CONFIG_FILE", "tests/config1")
	if err != nil {
		t.Fatal(err)
	}
	main.LoadConfiguration("foobar")
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
	main.LoadConfiguration("this does not exist")
}

// TestLoadBrokerConfiguration tests loading the broker configuration sub-tree
func TestLoadBrokerConfiguration(t *testing.T) {
	TestLoadConfiguration(t)
	brokerCfg := main.LoadBrokerConfiguration()
	if brokerCfg.Address != "localhost:9092" {
		t.Fatal("Improper broker address", brokerCfg.Address)
	}
	if brokerCfg.Topic != "platform.results.ccx" {
		t.Fatal("Improper broker topic", brokerCfg.Topic)
	}
	if brokerCfg.Group != "aggregator" {
		t.Fatal("Improper broker group", brokerCfg.Group)
	}
}

// TestLoadServerConfiguration tests loading the server configuration sub-tree
func TestLoadServerConfiguration(t *testing.T) {
	TestLoadConfiguration(t)
	serverCfg := main.LoadServerConfiguration()
	if serverCfg.Address != ":8080" {
		t.Fatal("Improper server address", serverCfg.Address)
	}
	if serverCfg.APIPrefix != "/api/v1/" {
		t.Fatal("Improper server API prefix", serverCfg.APIPrefix)
	}
}

// TestLoadStorageConfiguration tests loading the storage configuration sub-tree
func TestLoadStorageConfiguration(t *testing.T) {
	TestLoadConfiguration(t)
	storageCfg := main.LoadStorageConfiguration()
	if storageCfg.Driver != "sqlite3" {
		t.Fatal("Improper DB driver name", storageCfg.Driver)
	}
	if storageCfg.DataSource != "xyzzy" {
		t.Fatal("Improper DB data source name", storageCfg.DataSource)
	}
}

// TestLoadOrganizationWhitelist tests if the whitelist CSV file gets loaded properly
func TestLoadOrganizationWhitelist(t *testing.T) {
	expectedWhitelist := mapset.NewSetWith(
		types.OrgID(1),
		types.OrgID(2),
		types.OrgID(3),
	)
	orgWhitelist := main.LoadOrganizationWhitelist()
	if equal := orgWhitelist.Equal(expectedWhitelist); !equal {
		t.Errorf("Org whitelist did not load properly. Order of elements does not matter. Expected %v. Got %v", expectedWhitelist, orgWhitelist)
	}
}

func TestCreateReaderFromFile(t *testing.T) {
	_, err := main.CreateReaderFromFile("nonexistent.file")
	if err == nil {
		t.Error("Nonexistent file was opened.")
	}
}

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
