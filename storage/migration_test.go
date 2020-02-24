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
	"fmt"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

const (
	stepErrorMsg = "Migration Step Error"
)

var (
	stepNoopFn = func(tx *sql.Tx) error {
		return nil
	}
	stepErrorFn = func(tx *sql.Tx) error {
		return fmt.Errorf(stepErrorMsg)
	}
	testMigration = storage.Migration{
		StepUp: func(tx *sql.Tx) error {
			_, err := tx.Exec("CREATE TABLE migration_test_table (col INTEGER)")
			return err
		},
		StepDown: func(tx *sql.Tx) error {
			_, err := tx.Exec("DROP TABLE migration_test_table")
			return err
		},
	}
)

func prepareDB(t *testing.T) *storage.DBStorage {
	mockStorage, err := storage.New(storage.Configuration{
		Driver:           "sqlite3",
		SQLiteDataSource: ":memory:",
	})
	if err != nil {
		t.Fatal(err)
	}
	return mockStorage
}

func prepareDBAndMigrations(t *testing.T) *storage.DBStorage {
	storage.ClearMigrations()
	storage.AddMigration(testMigration)
	if storage.GetHighestMigrationVersion() != 1 {
		t.Fatal("Unable to prepare list of migrations")
	}

	db := prepareDB(t)

	if err := storage.InitMigrationInfo(db); err != nil {
		db.Close()
		t.Fatal(err)
	}

	return db
}

// TestMigrationClearMigrations checks that after clearing
// the migration list, the highest available migration version is 0.
func TestMigrationClearMigrations(t *testing.T) {
	storage.ClearMigrations()
	if ver := storage.GetHighestMigrationVersion(); ver != 0 {
		t.Fatalf("Unexpected highest migration version after clearing migration list (expected: 0, reality: %d)", ver)
	}
}

// TestMigrationAddMigration checks that adding a migration to the migration list works.
func TestMigrationAddMigration(t *testing.T) {
	storage.ClearMigrations()

	storage.AddMigration(testMigration)
	if ver := storage.GetHighestMigrationVersion(); ver != 1 {
		t.Fatalf("Unexpected highest migration version after adding first migration (expected: 1, reality: %d)", ver)
	}

	storage.AddMigration(testMigration)
	if ver := storage.GetHighestMigrationVersion(); ver != 2 {
		t.Fatalf("Unexpected highest migration version after adding second migration (expected: 2, reality: %d)", ver)
	}
}

// TestMigrationInit checks that database migration table initialization succeeds.
func TestMigrationInit(t *testing.T) {
	db := prepareDB(t)
	defer db.Close()

	if err := storage.InitMigrationInfo(db); err != nil {
		t.Fatal(err)
	}
}

// TestMigrationReInit checks that an attempt to re-initialize an already initialized
// migration info table will simply result in a no-op without any error.
func TestMigrationReInit(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer db.Close()

	if err := storage.InitMigrationInfo(db); err != nil {
		t.Fatal(err)
	}
}

// TestMigrationGetVersion checks that the initial database migration version is 0.
func TestMigrationGetVersion(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer db.Close()

	version, err := storage.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	}

	if version != 0 {
		t.Fatalf("Unexpected database version (expected: 0, reality: %d)", version)
	}
}

// TestMigrationSetVersion checks that it is possible to change
// the database version in both direction (upgrade and downgrade).
func TestMigrationSetVersion(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer db.Close()

	// Step-up from 0 to 1.
	if err := storage.SetDBVersion(db, 1); err != nil {
		t.Fatal(err)
	}

	version, err := storage.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("Unexpected database version (expected: 1, reality: %d)", version)
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
		t.Fatalf("Unexpected database version (expected: 0, reality: %d)", version)
	}
}

func TestMigrationNoInfoTable(t *testing.T) {
	db := prepareDB(t)
	defer db.Close()

	// Intentionally missing info table initialization here.

	_, err := storage.GetDBVersion(db)
	if err == nil || err.Error() != "no such table: migration_info" {
		t.Fatal("Migration info table should be missing when not initialized")
	}
}

func TestMigrationSetVersionSame(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer db.Close()

	// Step-up from 0 to 1.
	if err := storage.SetDBVersion(db, 1); err != nil {
		t.Fatal(err)
	}

	// Set version to.
	if err := storage.SetDBVersion(db, 1); err != nil {
		t.Fatal(err)
	}

	version, err := storage.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("Unexpected database version (expected: 1, reality: %d)", version)
	}
}

func TestMigrationSetVersionTargetTooHigh(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer db.Close()

	// Step-up from 0 to 2 (impossible -- only 1 migration is available).
	if err := storage.SetDBVersion(db, 2); err.Error() != "Invalid target version (available version range is 0-1)" {
		t.Fatal(err)
	}
}

// TestMigrationSetVersionUpError checks that an error during a step-up is correctly handled.
func TestMigrationSetVersionUpError(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer db.Close()

	storage.AddMigration(storage.Migration{
		StepUp:   stepErrorFn,
		StepDown: stepNoopFn,
	})

	if err := storage.SetDBVersion(db, 2); err.Error() != stepErrorMsg {
		t.Fatal(err)
	}
}

// TestMigrationSetVersionDownError checks that an error during a step-down is correctly handled.
func TestMigrationSetVersionDownError(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer db.Close()

	storage.AddMigration(storage.Migration{
		StepUp:   stepNoopFn,
		StepDown: stepErrorFn,
	})

	// First we need to step-up before we can step-down.
	if err := storage.SetDBVersion(db, 2); err != nil {
		t.Fatal(err)
	}

	if err := storage.SetDBVersion(db, 1); err == nil || err.Error() != stepErrorMsg {
		t.Fatal(err)
	}
}

// TestMigrationSetVersionCurrentTooHighError makes sure that if the current DB version
// is outside of the available migration range, it is reported as an error.
func TestMigrationSetVersionCurrentTooHighError(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer db.Close()

	if err := storage.SetDBVersion(db, 1); err != nil {
		t.Fatal(err)
	}

	storage.ClearMigrations()

	if err := storage.SetDBVersion(db, 0); err == nil || err.Error() != "Current version (1) is outside of available migration boundaries" {
		t.Fatal(err)
	}
}
