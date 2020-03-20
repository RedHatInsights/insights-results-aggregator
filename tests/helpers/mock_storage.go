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
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

// GetMockStorage creates mocked storage based on in-memory Sqlite instance
func GetMockStorage(init bool) (storage.Storage, error) {
	mockStorage, err := storage.New(storage.Configuration{
		Driver:           "sqlite3",
		SQLiteDataSource: ":memory:",
	})
	if err != nil {
		return nil, err
	}

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
func MustGetMockStorage(t *testing.T, init bool) storage.Storage {
	mockStorage, err := GetMockStorage(init)
	if err != nil {
		t.Fatal(err)
	}

	return mockStorage
}

// MustCloseStorage closes storage and panics if it wasn't successful
func MustCloseStorage(t *testing.T, s storage.Storage) {
	FailOnError(t, s.Close())
}

// MustGetMockStorageWithExpects returns mock db storage
// with a driver "github.com/DATA-DOG/go-sqlmock" which requires you to write expect
// before each query, so first try to use MustGetMockStorage
func MustGetMockStorageWithExpects(t *testing.T) (storage.Storage, sqlmock.Sqlmock) {
	connection, expects, err := sqlmock.New()
	FailOnError(t, err)

	return storage.NewFromConnection(connection, storage.DBDriverGeneral), expects
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
