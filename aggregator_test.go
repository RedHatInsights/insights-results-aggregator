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
	"bytes"
	"github.com/spf13/viper"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

func TestStartStorageConnection(t *testing.T) {
	TestLoadConfiguration(t)
	_, err := main.StartStorageConnection()
	if err != nil {
		t.Fatal("Cannot create storage object", err)
	}
}

func TestStartService(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		config := []byte(`
			[broker]
			address = "localhost:9093"
			topic = "ccx.ocp.results"
			group = "aggregator"
			enabled = false

			[server]
			address = ":1234"
			api_prefix = "/api/v1/"
			debug = true

			[processing]
			org_whitelist = "org_whitelist.csv"

			[metrics]
			enabled = true

			[logging]

			[storage]
			db_driver = "sqlite3"
			sqlite_datasource = ":memory:"
			pg_username = ""
			pg_password = ""
			pg_host = ""
			pg_port = 0
			pg_db_name = ""
			pg_params = ""
		`)

		viper.SetConfigType("toml")
		viper.ReadConfig(bytes.NewBuffer(config))

		go func() {
			main.StartService()
		}()

		main.WaitForServiceToStart()
		main.StopService()
	}, 5*time.Second)
}
