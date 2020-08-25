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

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	main "github.com/RedHatInsights/insights-results-aggregator"
	"github.com/RedHatInsights/insights-results-aggregator/conf"
	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

const (
	testsTimeout = 60 * time.Second
)

func mustSetEnv(t testing.TB, key, val string) {
	err := os.Setenv(key, val)
	helpers.FailOnError(t, err)
}

func mustLoadConfiguration(path string) {
	err := conf.LoadConfiguration(path)
	if err != nil {
		panic(err)
	}
}

func setEnvSettings(t testing.TB, settings map[string]string) {
	os.Clearenv()

	for key, val := range settings {
		mustSetEnv(t, key, val)
	}

	mustLoadConfiguration("/non_existing_path")
}

func TestCreateStorage(t *testing.T) {
	os.Clearenv()
	mustLoadConfiguration("tests/config1")

	_, err := main.CreateStorage()
	helpers.FailOnError(t, err)
}

func TestStartService(t *testing.T) {
	// It is necessary to perform migrations for this test
	// because the service won't run on top of an empty DB.
	*main.AutoMigratePtr = true

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		os.Clearenv()

		mustLoadConfiguration("./tests/tests")
		setEnvSettings(t, map[string]string{
			"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER":         "sqlite3",
			"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE": ":memory:",
		})

		go func() {
			main.StartService()
		}()

		time.Sleep(1 * time.Second)
		main.WaitForServiceToStart()
		errCode := main.StopService()
		assert.Equal(t, main.ExitStatusOK, errCode)
	}, testsTimeout)

	*main.AutoMigratePtr = false
}

// TODO: fix with new groups consumer
//func TestStartServiceWithMockBroker(t *testing.T) {
//	const topicName = "topic"
//	*main.AutoMigratePtr = true
//
//	helpers.RunTestWithTimeout(t, func(t *testing.T) {
//		mockBroker := sarama.NewMockBroker(t, 0)
//		defer mockBroker.Close()
//
//		mockBroker.SetHandlerByMap(helpers.GetHandlersMapForMockConsumer(t, mockBroker, topicName))
//
//		setEnvSettings(t, map[string]string{
//			"INSIGHTS_RESULTS_AGGREGATOR__BROKER__ADDRESS": mockBroker.Addr(),
//			"INSIGHTS_RESULTS_AGGREGATOR__BROKER__TOPIC":   topicName,
//			"INSIGHTS_RESULTS_AGGREGATOR__BROKER__ENABLED": "true",
//
//			"INSIGHTS_RESULTS_AGGREGATOR__SERVER__ADDRESS":       ":8080",
//			"INSIGHTS_RESULTS_AGGREGATOR__SERVER__API_PREFIX":    "/api/v1/",
//			"INSIGHTS_RESULTS_AGGREGATOR__SERVER__API_SPEC_FILE": "openapi.json",
//			"INSIGHTS_RESULTS_AGGREGATOR__SERVER__DEBUG":         "true",
//
//			"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__DB_DRIVER":         "sqlite3",
//			"INSIGHTS_RESULTS_AGGREGATOR__STORAGE__SQLITE_DATASOURCE": ":memory:",
//
//			"INSIGHTS_RESULTS_AGGREGATOR__CONTENT__PATH": "./tests/content/ok/",
//		})
//
//		go func() {
//			exitCode := main.StartService()
//			if exitCode != main.ExitStatusOK {
//				panic(fmt.Errorf("StartService exited with a code %v", exitCode))
//			}
//		}()
//
//		main.WaitForServiceToStart()
//		errCode := main.StopService()
//		assert.Equal(t, main.ExitStatusOK, errCode)
//	}, testsTimeout)
//
//	*main.AutoMigratePtr = false
//}

func TestStartService_DBError(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
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

	mockStorage, expects := ira_helpers.MustGetMockStorageWithExpects(t)
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

	*main.AutoMigratePtr = true

	errCode := main.PrepareDB()
	assert.Equal(t, main.ExitStatusOK, errCode)

	*main.AutoMigratePtr = false
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

	*main.AutoMigratePtr = true

	errCode := main.StartService()
	assert.Equal(t, main.ExitStatusConsumerError+main.ExitStatusServerError, errCode)

	*main.AutoMigratePtr = false
}

// TestPrintVersionInfo is dummy ATM - we'll check versions etc. in integration tests
func TestPrintVersionInfo(t *testing.T) {
	main.PrintVersionInfo()
}

// TestPrintHelp checks that printing help returns OK exit code.
func TestPrintHelp(t *testing.T) {
	assert.Equal(t, main.ExitStatusOK, main.PrintHelp())
}

// TestPrintConfig checks that printing configuration info returns OK exit code.
func TestPrintConfig(t *testing.T) {
	assert.Equal(t, main.ExitStatusOK, main.PrintConfig())
}

// TestPrintEnv checks that printing environment variables returns OK exit code.
func TestPrintEnv(t *testing.T) {
	assert.Equal(t, main.ExitStatusOK, main.PrintEnv())
}

// TestGetDBForMigrations checks that the function ensures the existence of
// the migration_info table and that the SQL DB connection works correctly.
func TestGetDBForMigrations(t *testing.T) {
	db, dbConn, exitCode := main.GetDBForMigrations()
	assert.Equal(t, main.ExitStatusOK, exitCode)
	defer ira_helpers.MustCloseStorage(t, db)

	row := dbConn.QueryRow("SELECT version FROM migration_info;")
	var version migration.Version
	err := row.Scan(&version)
	assert.NoError(t, err, "unable to read version from migration info table")
}

// TestPrintMigrationInfo checks that printing migration info exits with OK code.
func TestPrintMigrationInfo(t *testing.T) {
	db, dbConn, exitCode := main.GetDBForMigrations()
	assert.Equal(t, exitCode, main.ExitStatusOK)
	defer ira_helpers.MustCloseStorage(t, db)

	exitCode = main.PrintMigrationInfo(dbConn)
	assert.Equal(t, main.ExitStatusOK, exitCode)
}

// TestPrintMigrationInfoClosedDB checks that printing migration info with
// a closed DB connection results in a migration error exit code.
func TestPrintMigrationInfoClosedDB(t *testing.T) {
	db, dbConn, exitCode := main.GetDBForMigrations()
	assert.Equal(t, exitCode, main.ExitStatusOK)
	// Close DB connection immediately.
	ira_helpers.MustCloseStorage(t, db)

	exitCode = main.PrintMigrationInfo(dbConn)
	assert.Equal(t, main.ExitStatusMigrationError, exitCode)
}

// TestSetMigrationVersionZero checks that it is possible to set migration version to 0.
func TestSetMigrationVersionZero(t *testing.T) {
	db, dbConn, exitCode := main.GetDBForMigrations()
	assert.Equal(t, exitCode, main.ExitStatusOK)
	defer ira_helpers.MustCloseStorage(t, db)

	exitCode = main.SetMigrationVersion(dbConn, db.GetDBDriverType(), "0")
	assert.Equal(t, main.ExitStatusOK, exitCode)

	version, err := migration.GetDBVersion(dbConn)
	assert.NoError(t, err, "unable to get migration version")

	assert.Equal(t, migration.Version(0), version)
}

// TestSetMigrationVersionZero checks that it is to upgrade DB to the latest migration.
func TestSetMigrationVersionLatest(t *testing.T) {
	db, dbConn, exitCode := main.GetDBForMigrations()
	assert.Equal(t, exitCode, main.ExitStatusOK)
	defer ira_helpers.MustCloseStorage(t, db)

	exitCode = main.SetMigrationVersion(dbConn, db.GetDBDriverType(), "latest")
	assert.Equal(t, main.ExitStatusOK, exitCode)

	version, err := migration.GetDBVersion(dbConn)
	assert.NoError(t, err, "unable to get migration version")

	assert.Equal(t, migration.GetMaxVersion(), version)
}

// TestSetMigrationVersionClosedDB checks that setting the migration version
// with a closed DB connection results in a migration error exit code.
func TestSetMigrationVersionClosedDB(t *testing.T) {
	db, dbConn, exitCode := main.GetDBForMigrations()
	assert.Equal(t, exitCode, main.ExitStatusOK)
	// Close DB connection immediately.
	ira_helpers.MustCloseStorage(t, db)

	exitCode = main.SetMigrationVersion(dbConn, db.GetDBDriverType(), "0")
	assert.Equal(t, main.ExitStatusMigrationError, exitCode)
}

// TestSetMigrationVersionInvalid checks that when supplied an invalid version
// argument, the set version function exits with a migration error code.
func TestSetMigrationVersionInvalid(t *testing.T) {
	db, dbConn, exitCode := main.GetDBForMigrations()
	assert.Equal(t, exitCode, main.ExitStatusOK)
	// Close DB connection immediately.
	ira_helpers.MustCloseStorage(t, db)

	exitCode = main.SetMigrationVersion(dbConn, db.GetDBDriverType(), "")
	assert.Equal(t, main.ExitStatusMigrationError, exitCode)
}

// TestPerformMigrationsPrint checks that the command for
// printing migration info exits with the OK exit code.
func TestPerformMigrationsPrint(t *testing.T) {
	oldArgs := os.Args

	os.Args = []string{os.Args[0], "migrations"}
	exitCode := main.PerformMigrations()
	assert.Equal(t, main.ExitStatusOK, exitCode)

	os.Args = oldArgs
}

// TestPerformMigrationsPrint checks that the command for
// setting migration version exits with the OK exit code.
func TestPerformMigrationsSet(t *testing.T) {
	oldArgs := os.Args

	os.Args = []string{os.Args[0], "migrations", "0"}
	exitCode := main.PerformMigrations()
	assert.Equal(t, main.ExitStatusOK, exitCode)

	os.Args = oldArgs
}

// TestPerformMigrationsPrint checks that supplying too many arguments
// to the migration sub-commands results in the migration error exit code.
func TestPerformMigrationsTooManyArgs(t *testing.T) {
	oldArgs := os.Args

	os.Args = []string{os.Args[0], "migrations", "hello", "world"}
	exitCode := main.PerformMigrations()
	assert.Equal(t, main.ExitStatusMigrationError, exitCode)

	os.Args = oldArgs
}
