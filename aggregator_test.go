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
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	main "github.com/RedHatInsights/insights-results-aggregator"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

const (
	testsTimeout = 60 * time.Second
)

func setEnvSettings(t *testing.T, settings map[string]string) {
	os.Clearenv()

	for key, val := range settings {
		mustSetEnv(t, key, val)
	}

	mustLoadConfiguration("/non_existing_path")
}

func TestCreateStorage(t *testing.T) {
	TestLoadConfiguration(t)

	_, err := main.CreateStorage()
	helpers.FailOnError(t, err)
}

func TestStartService(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		os.Clearenv()

		mustLoadConfiguration("./tests/tests")

		go func() {
			main.StartService()
		}()

		main.WaitForServiceToStart()
		errCode := main.StopService()
		assert.Equal(t, 0, errCode)
	}, testsTimeout)
}

func TestStartServiceWithMockBroker(t *testing.T) {
	const topicName = "topic"

	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		mockBroker := sarama.NewMockBroker(t, 0)
		defer mockBroker.Close()

		mockBroker.SetHandlerByMap(helpers.GetHandlersMapForMockConsumer(t, mockBroker, topicName))

		setEnvSettings(t, map[string]string{
			"INSIGHTS_RESULTS_AGGREGATOR__BROKER__ADDRESS": mockBroker.Addr(),
			"INSIGHTS_RESULTS_AGGREGATOR__BROKER__TOPIC":   topicName,
			"INSIGHTS_RESULTS_AGGREGATOR__BROKER__ENABLED": "true",

			"INSIGHTS_RESULTS_AGGREGATOR__SERVER__ADDRESS":       ":8080",
			"INSIGHTS_RESULTS_AGGREGATOR__SERVER__API_PREFIX":    "/api/v1/",
			"INSIGHTS_RESULTS_AGGREGATOR__SERVER__API_SPEC_FILE": "openapi.json",
			"INSIGHTS_RESULTS_AGGREGATOR__SERVER__DEBUG":         "true",

			"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER":         "sqlite3",
			"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE": ":memory:",

			"INSIGHTS_RESULTS_AGGREGATOR__CONTENT__PATH": "./tests/content/ok/",
		})

		go func() {
			exitCode := main.StartService()
			if exitCode != 0 {
				panic(fmt.Errorf("StartService exited with a code %v", exitCode))
			}
		}()

		main.WaitForServiceToStart()
		errCode := main.StopService()
		assert.Equal(t, 0, errCode)
	}, testsTimeout)
}

func TestStartService_DBError(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		buf := new(bytes.Buffer)
		log.Logger = zerolog.New(buf)

		setEnvSettings(t, map[string]string{
			"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER":         "sqlite3",
			"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE": "/non/existing/path",
		})

		exitCode := main.StartService()
		assert.Equal(t, main.ExitStatusPrepareDbError, exitCode)
		assert.Contains(t, buf.String(), "unable to open database file: no such file or directory")
	}, testsTimeout)
}

func TestCreateStorage_BadDriver(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER":         "non-existing-driver",
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE": "/non/existing/path",
	})

	_, err := main.CreateStorage()
	assert.EqualError(t, err, "driver non-existing-driver is not supported")
}

func TestCloseStorage_Error(t *testing.T) {
	const errStr = "close error"

	buf := new(bytes.Buffer)
	log.Logger = zerolog.New(buf)

	mockStorage, expects := helpers.MustGetMockStorageWithExpects(t)
	expects.ExpectClose().WillReturnError(fmt.Errorf(errStr))

	main.CloseStorage(mockStorage.(*storage.DBStorage))

	assert.Contains(t, buf.String(), errStr)
}

func TestPrepareDB_DBError(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER":         "non-existing-driver",
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE": "/non/existing/path",
	})

	errCode := main.PrepareDB()
	assert.Equal(t, main.ExitStatusPrepareDbError, errCode)
}

func TestPrepareDB(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER":         "sqlite3",
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE": ":memory:",

		"INSIGHTS_RESULTS_AGGREGATOR__CONTENT__PATH": "./tests/content/ok/",
	})

	errCode := main.PrepareDB()
	assert.Equal(t, main.ExitStatusOK, errCode)
}

func TestPrepareDB_NoRulesDirectory(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER":         "sqlite3",
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE": ":memory:",

		"INSIGHTS_RESULTS_AGGREGATOR__CONTENT__PATH": "/non-existing-path",
	})

	errCode := main.PrepareDB()
	assert.Equal(t, main.ExitStatusPrepareDbError, errCode)
}

func TestPrepareDB_BadRules(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER":         "sqlite3",
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE": ":memory:",

		"INSIGHTS_RESULTS_AGGREGATOR__CONTENT__PATH": "./tests/content/bad_metadata_status/",
	})

	errCode := main.PrepareDB()
	assert.Equal(t, main.ExitStatusPrepareDbError, errCode)
}

func TestStartConsumer_DBError(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER":         "non-existing-driver",
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE": "bad-data-source",
	})

	errCode := main.StartConsumer()
	assert.Equal(t, main.ExitStatusConsumerError, errCode)
}

func TestStartConsumer_BadBrokerAddress(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER":         "sqlite3",
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE": ":memory:",

		"INSIGHTS_RESULTS_AGGREGATOR__BROKER__ADDRESS": "non-existing-host:1",
		"INSIGHTS_RESULTS_AGGREGATOR__BROKER__ENABLED": "true",
	})

	errCode := main.StartConsumer()
	assert.Equal(t, main.ExitStatusConsumerError, errCode)
}

func TestStartServer_DBError(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER":         "non-existing-driver",
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE": "bad-data-source",
	})

	errCode := main.StartServer()
	assert.Equal(t, main.ExitStatusServerError, errCode)
}

func TestStartServer_BadServerAddress(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER":         "sqlite3",
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE": ":memory:",

		"INSIGHTS_RESULTS_AGGREGATOR__SERVER__ADDRESS":       "non-existing-host:1",
		"INSIGHTS_RESULTS_AGGREGATOR__SERVER__API_SPEC_FILE": "openapi.json",
	})

	errCode := main.StartServer()
	assert.Equal(t, main.ExitStatusServerError, errCode)
}

func TestStartService_BadBrokerAndServerAddress(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER":         "sqlite3",
		"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE": ":memory:",

		"INSIGHTS_RESULTS_AGGREGATOR__BROKER__ADDRESS": "non-existing-host:1",
		"INSIGHTS_RESULTS_AGGREGATOR__BROKER__ENABLED": "true",

		"INSIGHTS_RESULTS_AGGREGATOR__SERVER__ADDRESS":       "non-existing-host:1",
		"INSIGHTS_RESULTS_AGGREGATOR__SERVER__API_SPEC_FILE": "openapi.json",

		"INSIGHTS_RESULTS_AGGREGATOR__CONTENT__PATH": "./tests/content/ok/",
	})

	errCode := main.StartService()
	assert.Equal(t, main.ExitStatusConsumerError+main.ExitStatusServerError, errCode)
}
