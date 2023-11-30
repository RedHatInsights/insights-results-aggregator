/*
Copyright © 2020, 2021, 2022 Red Hat, Inc.

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

	_, _, err := main.CreateStorage()
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
			"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__DB_DRIVER":         "sqlite3",
			"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__SQLITE_DATASOURCE": ":memory:",
		})

		go func() {
			main.StartService()
		}()

		errCode := main.StopService()
		assert.Equal(t, main.ExitStatusOK, errCode)
	}, testsTimeout)

	*main.AutoMigratePtr = false
}

func TestStartService_DBError(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		buf := new(bytes.Buffer)
		log.Logger = zerolog.New(buf)

		setEnvSettings(t, map[string]string{
			"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__DB_DRIVER":         "sqlite3",
			"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__SQLITE_DATASOURCE": "/non/existing/path",
		})

		exitCode := main.StartService()
		assert.Equal(t, main.ExitStatusPrepareDbError, exitCode)
		assert.Contains(t, buf.String(), "unable to open database file: no such file or directory")
	}, testsTimeout)
}

func TestCreateStorage_BadDriver(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__DB_DRIVER":         "non-existing-driver",
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__SQLITE_DATASOURCE": "/non/existing/path",
	})

	_, _, err := main.CreateStorage()
	assert.EqualError(t, err, "driver non-existing-driver is not supported")
}

func TestCloseStorage_Error(t *testing.T) {
	const errStr = "close error"

	buf := new(bytes.Buffer)
	log.Logger = zerolog.New(buf)

	mockStorage, expects := ira_helpers.MustGetMockStorageWithExpects(t)
	expects.ExpectClose().WillReturnError(fmt.Errorf(errStr))

	main.CloseStorage(mockStorage.(*storage.OCPRecommendationsDBStorage))

	assert.Contains(t, buf.String(), errStr)
}

func TestPrepareDB_DBError(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__DB_DRIVER":         "non-existing-driver",
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__SQLITE_DATASOURCE": "/non/existing/path",
	})

	errCode := main.PrepareDB()
	assert.Equal(t, main.ExitStatusPrepareDbError, errCode)
}

func TestPrepareDB(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__DB_DRIVER":         "sqlite3",
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__SQLITE_DATASOURCE": ":memory:",

		"INSIGHTS_RESULTS_AGGREGATOR__CONTENT__PATH": "./tests/content/ok/",
	})

	*main.AutoMigratePtr = true

	errCode := main.PrepareDB()
	assert.Equal(t, main.ExitStatusOK, errCode)

	*main.AutoMigratePtr = false
}

func TestPrepareDB_NoRulesDirectory(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__DB_DRIVER":         "sqlite3",
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__SQLITE_DATASOURCE": ":memory:",

		"INSIGHTS_RESULTS_AGGREGATOR__CONTENT__PATH": "/non-existing-path",
	})

	errCode := main.PrepareDB()
	assert.Equal(t, main.ExitStatusPrepareDbError, errCode)
}

func TestPrepareDB_BadRules(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__DB_DRIVER":         "sqlite3",
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__SQLITE_DATASOURCE": ":memory:",

		"INSIGHTS_RESULTS_AGGREGATOR__CONTENT__PATH": "./tests/content/bad_metadata_status/",
	})

	errCode := main.PrepareDB()
	assert.Equal(t, main.ExitStatusPrepareDbError, errCode)
}

func TestStartConsumer_DBError(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__DB_DRIVER":         "non-existing-driver",
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__SQLITE_DATASOURCE": "bad-data-source",
	})

	err := main.StartConsumer(conf.GetBrokerConfiguration())
	assert.EqualError(t, err, "driver non-existing-driver is not supported")
}

func TestStartConsumer_BadBrokerAddress(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__DB_DRIVER":         "sqlite3",
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__SQLITE_DATASOURCE": ":memory:",

		"INSIGHTS_RESULTS_AGGREGATOR__BROKER__ADDRESS": "non-existing-host:999999",
		"INSIGHTS_RESULTS_AGGREGATOR__BROKER__ENABLED": "true",
	})

	err := main.StartConsumer(conf.GetBrokerConfiguration())
	assert.EqualError(
		t, err, "kafka: client has run out of available brokers to talk to (Is your cluster reachable?)",
	)
}

func TestStartServer_DBError(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__DB_DRIVER":         "non-existing-driver",
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__SQLITE_DATASOURCE": "bad-data-source",
	})

	err := main.StartServer()
	assert.EqualError(t, err, "driver non-existing-driver is not supported")
}

func TestStartServer_BadServerAddress(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__DB_DRIVER":         "sqlite3",
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__SQLITE_DATASOURCE": ":memory:",

		"INSIGHTS_RESULTS_AGGREGATOR__SERVER__ADDRESS":       "localhost:999999",
		"INSIGHTS_RESULTS_AGGREGATOR__SERVER__API_SPEC_FILE": "openapi.json",
	})

	err := main.StartServer()
	assert.EqualError(t, err, "listen tcp: address 999999: invalid port")
}

func TestStartService_BadBrokerAndServerAddress(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__DB_DRIVER":         "sqlite3",
		"INSIGHTS_RESULTS_AGGREGATOR__OCP_RECOMMENDATIONS_STORAGE__SQLITE_DATASOURCE": ":memory:",

		"INSIGHTS_RESULTS_AGGREGATOR__BROKER__ADDRESS": "non-existing-host:1",
		"INSIGHTS_RESULTS_AGGREGATOR__BROKER__ENABLED": "true",

		"INSIGHTS_RESULTS_AGGREGATOR__SERVER__ADDRESS":       "non-existing-host:1",
		"INSIGHTS_RESULTS_AGGREGATOR__SERVER__API_SPEC_FILE": "openapi.json",

		"INSIGHTS_RESULTS_AGGREGATOR__CONTENT__PATH": "./tests/content/ok/",
	})

	*main.AutoMigratePtr = true

	errCode := main.StartService()
	assert.Equal(t, main.ExitStatusError, errCode)

	*main.AutoMigratePtr = false
}

// TestPrintVersionInfo is dummy ATM - we'll check versions etc. in integration tests
func TestPrintVersionInfo(_ *testing.T) {
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

// TestFillInInfoParams test the behaviour of function fillInInfoParams
func TestFillInInfoParams(t *testing.T) {
	// map to be used by this unit test
	m := make(map[string]string)

	// preliminary test if Go Universe is still ok
	assert.Empty(t, m, "Map should be empty at the beginning")

	// try to fill-in all info params
	main.FillInInfoParams(m)

	// preliminary test if Go Universe is still ok
	assert.Len(t, m, 5, "Map should contains exactly five items")

	// does the map contain all expected keys?
	assert.Contains(t, m, "BuildVersion")
	assert.Contains(t, m, "BuildTime")
	assert.Contains(t, m, "BuildBranch")
	assert.Contains(t, m, "BuildCommit")
	assert.Contains(t, m, "UtilsVersion")
}
