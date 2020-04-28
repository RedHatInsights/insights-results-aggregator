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
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/RedHatInsights/insights-results-aggregator/conf"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

const sqlite3 = "sqlite3"

// GetMockStorage creates mocked storage based on in-memory Sqlite instance
func GetMockStorage(init bool) (storage.Storage, error) {
	db, err := sql.Open(sqlite3, ":memory:")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return nil, err
	}

	mockStorage := storage.NewFromConnection(db, storage.DBDriverSQLite3)

	// initialize the database by all required tables
	if init {
		err = mockStorage.Init()
		if err != nil {
			return nil, err
		}
	}

	return mockStorage, nil
}

// MustGetMockStorage creates mocked storage based on in-memory Sqlite instance
// produces t.Fatal(err) on error
func MustGetMockStorage(t testing.TB, init bool) storage.Storage {
	mockStorage, err := GetMockStorage(init)
	FailOnError(t, err)

	return mockStorage
}

// MustCloseStorage closes storage and panics if it wasn't successful
func MustCloseStorage(t testing.TB, s storage.Storage) {
	FailOnError(t, s.Close())
}

// MustGetMockStorageWithExpects returns mock db storage
// with a driver "github.com/DATA-DOG/go-sqlmock" which requires you to write expect
// before each query, so first try to use MustGetMockStorage
// don't forget to call MustCloseMockStorageWithExpects
func MustGetMockStorageWithExpects(t *testing.T) (storage.Storage, sqlmock.Sqlmock) {
	return MustGetMockStorageWithExpectsForDriver(t, storage.DBDriverGeneral)
}

// MustGetMockStorageWithExpectsForDriver returns mock db storage
// with specified driver type and
// with a driver "github.com/DATA-DOG/go-sqlmock" which requires you to write expect
// before each query, so first try to use MustGetMockStorage
// don't forget to call MustCloseMockStorageWithExpects
func MustGetMockStorageWithExpectsForDriver(
	t *testing.T, driverType storage.DBDriver,
) (storage.Storage, sqlmock.Sqlmock) {
	db, expects := MustGetMockDBWithExpects(t)

	return storage.NewFromConnection(db, driverType), expects
}

// MustGetMockDBWithExpects returns mock db
// with a driver "github.com/DATA-DOG/go-sqlmock" which requires you to write expect
// before each query, so first try to use MustGetMockStorage
// don't forget to call MustCloseMockDBWithExpects
func MustGetMockDBWithExpects(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, expects, err := sqlmock.New()
	FailOnError(t, err)

	return db, expects
}

// MustCloseMockStorageWithExpects closes mock storage with expects and panics if it wasn't successful
func MustCloseMockStorageWithExpects(
	t *testing.T, mockStorage storage.Storage, expects sqlmock.Sqlmock,
) {
	if err := expects.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expects.ExpectClose()
	FailOnError(t, mockStorage.Close())
}

// MustCloseMockDBWithExpects closes mock db with expects and panics if it wasn't successful
func MustCloseMockDBWithExpects(
	t *testing.T, db *sql.DB, expects sqlmock.Sqlmock,
) {
	if err := expects.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expects.ExpectClose()
	FailOnError(t, db.Close())
}

// MustGetSQLiteFileStorage creates test sqlite storage in file
func MustGetSQLiteFileStorage(b *testing.B) (storage.Storage, func(*testing.B)) {
	const dbFile = "./test.db"

	db, err := sql.Open(sqlite3, dbFile)
	FailOnError(b, err)

	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	FailOnError(b, err)

	sqliteStorage := storage.NewFromConnection(db, storage.DBDriverSQLite3)

	err = sqliteStorage.Init()
	FailOnError(b, err)

	return sqliteStorage, func(b *testing.B) {
		FailOnError(b, os.Remove(dbFile))
	}
}

// MustGetPostgresStorage creates test postgres storage with credentials from config-devel
func MustGetPostgresStorage(b *testing.B) (storage.Storage, func(*testing.B)) {
	err := conf.LoadConfiguration("../config-devel")
	FailOnError(b, err)

	postgresStorage, err := storage.New(conf.GetStorageConfiguration())
	FailOnError(b, err)

	return postgresStorage, func(*testing.B) {}
}
