// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helpers

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/google/uuid"

	"github.com/RedHatInsights/insights-results-aggregator/conf"
	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const sqlite3 = "sqlite3"
const postgres = "postgres"

// MustGetMockStorage creates mocked storage based on in-memory Sqlite instance by default
// or on postgresql with config taken from config-devel.toml
// if env variable INSIGHTS_RESULTS_AGGREGATOR__TESTS_DB is set to "postgres"
// INSIGHTS_RESULTS_AGGREGATOR__TESTS_DB_ADMIN_PASS is set to db admin's password
// produces t.Fatal(err) on error
func MustGetMockStorage(tb testing.TB, init bool) (storage.OCPRecommendationsStorage, func()) {
	return MustGetPostgresStorage(tb, init)
}

// MustGetMockStorageWithExpects returns mock db storage
// with a driver "github.com/DATA-DOG/go-sqlmock" which requires you to write expect
// before each query, so first try to use MustGetMockStorage
// don't forget to call MustCloseMockStorageWithExpects
func MustGetMockStorageWithExpects(t *testing.T) (storage.OCPRecommendationsStorage, sqlmock.Sqlmock) {
	return MustGetMockStorageWithExpectsForDriver(t, types.DBDriverGeneral)
}

// MustGetMockStorageWithExpectsForDriver returns mock db storage
// with specified driver type and
// with a driver "github.com/DATA-DOG/go-sqlmock" which requires you to write expect
// before each query, so first try to use MustGetMockStorage
// don't forget to call MustCloseMockStorageWithExpects
func MustGetMockStorageWithExpectsForDriver(
	t *testing.T, driverType types.DBDriver,
) (storage.OCPRecommendationsStorage, sqlmock.Sqlmock) {
	db, expects := MustGetMockDBWithExpects(t)

	return storage.NewOCPRecommendationsFromConnection(db, driverType), expects
}

// MustGetMockDBWithExpects returns mock db
// with a driver "github.com/DATA-DOG/go-sqlmock" which requires you to write expect
// before each query, so first try to use MustGetMockStorage
// don't forget to call MustCloseMockDBWithExpects
func MustGetMockDBWithExpects(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, expects, err := sqlmock.New()
	helpers.FailOnError(t, err)

	return db, expects
}

// MustCloseMockStorageWithExpects closes mock storage with expects and panics if it wasn't successful
func MustCloseMockStorageWithExpects(
	t *testing.T, mockStorage storage.OCPRecommendationsStorage, expects sqlmock.Sqlmock,
) {
	if err := expects.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expects.ExpectClose()
	helpers.FailOnError(t, mockStorage.Close())
}

// MustCloseMockDBWithExpects closes mock db with expects and panics if it wasn't successful
func MustCloseMockDBWithExpects(
	t *testing.T, db *sql.DB, expects sqlmock.Sqlmock,
) {
	if err := expects.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expects.ExpectClose()
	helpers.FailOnError(t, db.Close())
}

// MustGetSQLiteMemoryStorage creates test sqlite storage in file
func MustGetSQLiteMemoryStorage(tb testing.TB, init bool) (storage.OCPRecommendationsStorage, func()) {
	sqliteStorage := mustGetSqliteStorage(tb, ":memory:", init)

	return sqliteStorage, func() {
		MustCloseStorage(tb, sqliteStorage)
	}
}

// MustGetSQLiteFileStorage creates test sqlite storage in file
func MustGetSQLiteFileStorage(tb testing.TB, init bool) (storage.OCPRecommendationsStorage, func()) {
	dbFilename := fmt.Sprintf("/tmp/insights-results-aggregator.test.%v.db", uuid.New().String())

	sqliteStorage := mustGetSqliteStorage(tb, dbFilename, init)

	return sqliteStorage, func() {
		MustCloseStorage(tb, sqliteStorage)
		helpers.FailOnError(tb, os.Remove(dbFilename))
	}
}

func mustGetSqliteStorage(tb testing.TB, datasource string, init bool) storage.OCPRecommendationsStorage {
	db, err := sql.Open(sqlite3, datasource)
	helpers.FailOnError(tb, err)

	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	helpers.FailOnError(tb, err)

	sqliteStorage := storage.NewOCPRecommendationsFromConnection(db, types.DBDriverSQLite3)

	if init {
		helpers.FailOnError(tb, sqliteStorage.MigrateToLatest())
		helpers.FailOnError(tb, sqliteStorage.Init())
	}

	return sqliteStorage
}

// MustGetPostgresStorage creates test postgres storage with credentials from config-devel
func MustGetPostgresStorage(tb testing.TB, init bool) (storage.OCPRecommendationsStorage, func()) {
	dbAdminPassword := os.Getenv("INSIGHTS_RESULTS_AGGREGATOR__TESTS_DB_ADMIN_PASS")

	err := conf.LoadConfiguration("../config-devel")
	helpers.FailOnError(tb, err)

	// force postgres and replace db name with test one
	storageConf := &conf.Config.OCPRecommendationsStorage
	storageConf.Driver = postgres
	storageConf.PGDBName += "_test_db_" + strings.ReplaceAll(uuid.New().String(), "-", "_")
	storageConf.PGPassword = dbAdminPassword
	storageConf.PGUsername = postgres

	connString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s sslmode=disable",
		storageConf.PGHost, storageConf.PGPort, storageConf.PGUsername, storageConf.PGPassword,
	)

	adminConn, err := sql.Open(storageConf.Driver, connString)
	helpers.FailOnError(tb, err)

	query := "CREATE DATABASE " + storageConf.PGDBName + ";"
	_, err = adminConn.Exec(query)
	helpers.FailOnError(tb, err)

	postgresStorage, err := storage.NewOCPRecommendationsStorage(*storageConf)

	helpers.FailOnError(tb, err)
	helpers.FailOnError(tb, postgresStorage.GetConnection().Ping())

	if init {
		helpers.FailOnError(tb, postgresStorage.MigrateToLatest())
		helpers.FailOnError(tb, postgresStorage.Init())
	}

	return postgresStorage, func() {
		MustCloseStorage(tb, postgresStorage)

		_, err := adminConn.Exec("DROP DATABASE " + conf.Config.OCPRecommendationsStorage.PGDBName)
		helpers.FailOnError(tb, err)

		helpers.FailOnError(tb, adminConn.Close())
	}
}

// MustCloseStorage closes the storage and calls t.Fatal on error
func MustCloseStorage(tb testing.TB, s storage.Storage) {
	helpers.FailOnError(tb, s.Close())
}

// PrepareDB prepares mock OCPRecommendationsDBStorage
func PrepareDB(t *testing.T) (*storage.OCPRecommendationsDBStorage, func()) {
	mockStorage, closer := MustGetMockStorage(t, false)
	dbStorage := mockStorage.(*storage.OCPRecommendationsDBStorage)

	return dbStorage, closer
}

// PrepareDBAndInfo prepares mock OCPRecommendationsDBStorage and info table
func PrepareDBAndInfo(t *testing.T) (
	*sql.DB,
	types.DBDriver,
	migration.Schema,
	func(),
) {
	storage, closer := PrepareDB(t)

	dbConn := storage.GetConnection()
	dbSchema := storage.GetDBSchema()

	if err := migration.InitInfoTable(dbConn, dbSchema); err != nil {
		closer()
		t.Fatal(err)
	}

	return dbConn, storage.GetDBDriverType(), dbSchema, closer
}
