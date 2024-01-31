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

const postgres = "postgres"

// MustGetMockStorageWithExpects returns mock db storage
// with a driver "github.com/DATA-DOG/go-sqlmock" which requires you to write expect
// before each query, so first try to use MustGetPostgresStorage
// don't forget to call MustCloseMockStorageWithExpects
func MustGetMockStorageWithExpects(t *testing.T) (storage.OCPRecommendationsStorage, sqlmock.Sqlmock) {
	return MustGetMockStorageWithExpectsForDriver(t, types.DBDriverGeneral)
}

// MustGetMockStorageWithExpectsForDriver returns mock db storage
// with specified driver type and
// with a driver "github.com/DATA-DOG/go-sqlmock" which requires you to write expect
// before each query, so first try to use MustGetPostgresStorage
// don't forget to call MustCloseMockStorageWithExpects
func MustGetMockStorageWithExpectsForDriver(
	t *testing.T, driverType types.DBDriver,
) (storage.OCPRecommendationsStorage, sqlmock.Sqlmock) {
	db, expects := MustGetMockDBWithExpects(t)

	return storage.NewOCPRecommendationsFromConnection(db, driverType), expects
}

// MustGetMockDBWithExpects returns mock db
// with a driver "github.com/DATA-DOG/go-sqlmock" which requires you to write expect
// before each query, so first try to use MustGetPostgresStorage
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

// MustGetPostgresStorage creates test postgres storage with credentials from config-devel
func MustGetPostgresStorage(tb testing.TB, init bool) (storage.OCPRecommendationsStorage, func()) {
	err := conf.LoadConfiguration("../config-devel")
	helpers.FailOnError(tb, err)

	storageConf := &conf.Config.OCPRecommendationsStorage

	dbConn := getPostgresConnection(tb, storageConf)

	postgresStorage, err := storage.NewOCPRecommendationsStorage(*storageConf)
	helpers.FailOnError(tb, err)

	return postgresStorage, func() {
		postgresCloser(tb, dbConn, postgresStorage, storageConf.PGDBName)
	}
}

// MustGetPostgresStorageDVO creates test postgres storage with credentials from config-devel for DVO storage
func MustGetPostgresStorageDVO(tb testing.TB, init bool) (storage.DVORecommendationsStorage, func()) {
	err := conf.LoadConfiguration("../config-devel")
	helpers.FailOnError(tb, err)

	// set StorageBackend.Use to DVO
	conf.Config.StorageBackend.Use = types.DVORecommendationsStorage

	storageConf := &conf.Config.DVORecommendationsStorage

	dbConn := getPostgresConnection(tb, storageConf)

	postgresStorage, err := storage.NewDVORecommendationsStorage(*storageConf)
	helpers.FailOnError(tb, err)

	return postgresStorage, func() {
		postgresCloser(tb, dbConn, postgresStorage, storageConf.PGDBName)
	}
}

func initPostgresDB(tb testing.TB, storage storage.Storage, initAndMigrate bool) {
	helpers.FailOnError(tb, storage.GetConnection().Ping())

	if initAndMigrate {
		helpers.FailOnError(tb, storage.MigrateToLatest())
		helpers.FailOnError(tb, storage.Init())
	}
}

func getPostgresConnection(tb testing.TB, config *storage.Configuration) *sql.DB {
	dbAdminPassword := os.Getenv("INSIGHTS_RESULTS_AGGREGATOR__TESTS_DB_ADMIN_PASS")

	// force postgres and replace db name with test one
	config.Driver = postgres
	config.PGDBName += "_test_db_" + strings.ReplaceAll(uuid.New().String(), "-", "_")
	config.PGPassword = dbAdminPassword

	connString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s sslmode=disable",
		config.PGHost, config.PGPort, config.PGUsername, config.PGPassword,
	)

	adminConn, err := sql.Open(config.Driver, connString)
	helpers.FailOnError(tb, err)

	query := "CREATE DATABASE " + config.PGDBName + ";"
	_, err = adminConn.Exec(query)
	helpers.FailOnError(tb, err)

	return adminConn
}

func postgresCloser(tb testing.TB, conn *sql.DB, storage storage.Storage, dbName string) {
	MustCloseStorage(tb, storage)

	_, err := conn.Exec("DROP DATABASE " + dbName)
	helpers.FailOnError(tb, err)

	helpers.FailOnError(tb, storage.Close())
}

// MustCloseStorage closes the storage and calls t.Fatal on error
func MustCloseStorage(tb testing.TB, s storage.Storage) {
	helpers.FailOnError(tb, s.Close())
}

// PrepareDB prepares mock OCPRecommendationsDBStorage
func PrepareDB(t *testing.T) (*storage.OCPRecommendationsDBStorage, func()) {
	mockStorage, closer := MustGetPostgresStorage(t, false)
	dbStorage := mockStorage.(*storage.OCPRecommendationsDBStorage)

	return dbStorage, closer
}

// PrepareDVODB prepares mock DVORecommendationsDBStorage
func PrepareDVODB(t *testing.T) (*storage.DVORecommendationsDBStorage, func()) {
	mockStorage, closer := MustGetPostgresStorageDVO(t, false)
	dbStorage := mockStorage.(*storage.DVORecommendationsDBStorage)

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
