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

package storage_test

import (
	"database/sql"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

func mockDB(t *testing.T) *storage.DBStorage {
	mockStorage, err := storage.New(storage.Configuration{
		Driver:     "sqlite3",
		DataSource: ":memory:",
	})
	if err != nil {
		t.Fatal(err)
	}
	return mockStorage
}

// TestMigrationInit checks that database migration table initialization succeeds.
func TestMigrationInit(t *testing.T) {
	db := mockDB(t)
	if err := storage.InitMigrationInfo(db); err != nil {
		t.Fatal(err)
	}
}

// TestMigrationGetVersion checks that the initial database migration version is 0.
func TestMigrationGetVersion(t *testing.T) {
	db := mockDB(t)

	if err := storage.InitMigrationInfo(db); err != nil {
		t.Fatal(err)
	}

	version, err := storage.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	}

	if version != 0 {
		t.Fatalf("Unexpected database version -- expected 0; reality: %d", version)
	}
}

// TestMigrationSetVersion checks that it is possible to change
// the database version in both direction (upgrade and downgrade).
func TestMigrationSetVersion(t *testing.T) {
	db := mockDB(t)

	// Initialize migration info table.
	if err := storage.InitMigrationInfo(db); err != nil {
		t.Fatal(err)
	}

	// Create a single migration level for testing purposes.
	{
		stepUp := func(tx *sql.Tx) error {
			_, err := tx.Exec("CREATE TABLE migration_test_table (col INTEGER)")
			return err
		}

		stepDown := func(tx *sql.Tx) error {
			_, err := tx.Exec("DROP TABLE migration_test_table")
			return err
		}

		storage.AddMigration(storage.Migration{StepUp: stepUp, StepDown: stepDown})
	}

	// Step-up from 0 to 1.
	if err := storage.SetDBVersion(db, 1); err != nil {
		t.Fatal(err)
	}

	version, err := storage.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("Unexpected database version -- expected 1; reality: %d", version)
	}

	// Step-down from 1 to 0.
	if err := storage.SetDBVersion(db, 0); err != nil {
		t.Fatal(err)
	}

	version, err = storage.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	}

	if version != 0 {
		t.Fatalf("Unexpected database version -- expected 0; reality: %d", version)
	}
}
