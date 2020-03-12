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

package migration_test

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
)

const (
	dbClosedErrorMsg    = "sql: database is closed"
	noSuchTableErrorMsg = "no such table: migration_info"
	stepErrorMsg        = "Migration Step Error"
)

var (
	stepNoopFn = func(tx *sql.Tx) error {
		return nil
	}
	stepErrorFn = func(tx *sql.Tx) error {
		return fmt.Errorf(stepErrorMsg)
	}
	stepRollbackFn = func(tx *sql.Tx) error {
		return tx.Rollback()
	}
	testMigration = migration.Migration{
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

func prepareDB(t *testing.T) *sql.DB {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	return conn
}

func prepareDBAndInfo(t *testing.T) *sql.DB {
	db := prepareDB(t)

	if err := migration.InitInfoTable(db); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}

	return db
}

func prepareDBAndMigrations(t *testing.T) *sql.DB {
	*migration.Migrations = []migration.Migration{testMigration}
	return prepareDBAndInfo(t)
}

// TestMigrationFull tests majority of the migration
// mechanism's functionality, all in one place.
func TestMigrationFull(t *testing.T) {
	// Don't overwrite the migration list, use the real migrations.
	db := prepareDBAndInfo(t)

	maxVer := migration.GetMaxVersion()
	if maxVer == 0 {
		t.Fatal("No migrations available")
	}

	currentVer, err := migration.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	} else if currentVer != 0 {
		t.Fatalf("Unexpected version: %d (expected: %d)", currentVer, 0)
	}

	stepUpAndDown(t, db, maxVer, 0)
}

func stepUpAndDown(t *testing.T, db *sql.DB, upVer, downVer migration.Version) {
	err := migration.SetDBVersion(db, upVer)
	if err != nil {
		t.Fatal(err)
	}

	currentVer, err := migration.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	} else if currentVer != upVer {
		t.Fatalf("Unexpected version: %d (expected: %d)", currentVer, upVer)
	}

	err = migration.SetDBVersion(db, 0)
	if err != nil {
		t.Fatal(err)
	}

	currentVer, err = migration.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	} else if currentVer != downVer {
		t.Fatalf("Unexpected version: %d (expected: %d)", currentVer, downVer)
	}
}

// closeDB closes the connection to DB with check whether the close operation
// was successful or not.
func closeDB(t *testing.T, mockDB *sql.DB) {
	err := mockDB.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// TestMigrationInit checks that database migration table initialization succeeds.
func TestMigrationInit(t *testing.T) {
	db := prepareDB(t)
	defer closeDB(t, db)

	if err := migration.InitInfoTable(db); err != nil {
		t.Fatal(err)
	}

	if _, err := migration.GetDBVersion(db); err != nil {
		t.Fatal(err)
	}
}

// TestMigrationReInit checks that an attempt to re-initialize an already initialized
// migration info table will simply result in a no-op without any error.
func TestMigrationReInit(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer closeDB(t, db)

	if err := migration.InitInfoTable(db); err != nil {
		t.Fatal(err)
	}
}

func TestMigrationInitNotOneRow(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer closeDB(t, db)

	_, err := db.Exec("INSERT INTO migration_info(version) VALUES(10)")
	if err != nil {
		t.Fatal(err)
	}

	const expectedErrStr = "unexpected number of rows in migration info table (expected: 1, reality: 2)"
	if err := migration.InitInfoTable(db); err == nil || err.Error() != expectedErrStr {
		t.Fatal(err)
	}
}

// TestMigrationGetVersion checks that the initial database migration version is 0.
func TestMigrationGetVersion(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer closeDB(t, db)

	version, err := migration.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	}

	if version != 0 {
		t.Fatalf("Unexpected database version (expected: 0, reality: %d)", version)
	}
}

func TestMigrationGetVersionMissingInfoTable(t *testing.T) {
	// Prepare DB without preparing the migration info table.
	db := prepareDB(t)
	defer closeDB(t, db)

	if _, err := migration.GetDBVersion(db); err == nil || err.Error() != noSuchTableErrorMsg {
		t.Fatal(err)
	}
}

func TestMigrationGetVersionMultipleRows(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer closeDB(t, db)

	_, err := db.Exec("INSERT INTO migration_info(version) VALUES(10)")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := migration.GetDBVersion(db); err == nil || err.Error() != "migration info table contain multiple rows" {
		t.Fatal(err)
	}
}

func TestMigrationGetVersionEmptyTable(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer closeDB(t, db)

	_, err := db.Exec("DELETE FROM migration_info")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := migration.GetDBVersion(db); err == nil || err.Error() != "migration info table is empty" {
		t.Fatal(err)
	}
}

func TestMigrationGetVersionInvalidType(t *testing.T) {
	db := prepareDB(t)
	defer closeDB(t, db)

	_, err := db.Exec("CREATE TABLE migration_info ( version TEXT )")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec("INSERT INTO migration_info(version) VALUES('hello world')")
	if err != nil {
		t.Fatal(err)
	}

	const expectedErrStr = `sql: Scan error on column index 0, name "version": ` +
		`converting driver.Value type string ("hello world") to a uint: invalid syntax`
	if _, err := migration.GetDBVersion(db); err == nil || err.Error() != expectedErrStr {
		t.Fatal(err)
	}
}

// TestMigrationSetVersion checks that it is possible to change
// the database version in both direction (upgrade and downgrade).
func TestMigrationSetVersion(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer closeDB(t, db)

	// Step-up from 0 to 1.
	if err := migration.SetDBVersion(db, 1); err != nil {
		t.Fatal(err)
	}

	version, err := migration.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("Unexpected database version (expected: 1, reality: %d)", version)
	}

	// Step-down from 1 to 0.
	if err := migration.SetDBVersion(db, 0); err != nil {
		t.Fatal(err)
	}

	version, err = migration.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	}

	if version != 0 {
		t.Fatalf("Unexpected database version (expected: 0, reality: %d)", version)
	}
}

func TestMigrationNoInfoTable(t *testing.T) {
	db := prepareDB(t)
	defer closeDB(t, db)

	// Intentionally missing info table initialization here.

	_, err := migration.GetDBVersion(db)
	if err == nil || err.Error() != noSuchTableErrorMsg {
		t.Fatal("Migration info table should be missing when not initialized")
	}
}

func TestMigrationSetVersionSame(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer closeDB(t, db)

	// Step-up from 0 to 1.
	if err := migration.SetDBVersion(db, 1); err != nil {
		t.Fatal(err)
	}

	// Set version to.
	if err := migration.SetDBVersion(db, 1); err != nil {
		t.Fatal(err)
	}

	version, err := migration.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("Unexpected database version (expected: 1, reality: %d)", version)
	}
}

func TestMigrationSetVersionTargetTooHigh(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer closeDB(t, db)

	// Step-up from 0 to 2 (impossible -- only 1 migration is available).
	if err := migration.SetDBVersion(db, 2); err.Error() != "invalid target version (available version range is 0-1)" {
		t.Fatal(err)
	}
}

// TestMigrationSetVersionUpError checks that an error during a step-up is correctly handled.
func TestMigrationSetVersionUpError(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer closeDB(t, db)

	*migration.Migrations = []migration.Migration{
		{
			StepUp:   stepErrorFn,
			StepDown: stepNoopFn,
		},
	}

	if err := migration.SetDBVersion(db, 1); err == nil || err.Error() != stepErrorMsg {
		t.Fatal(err)
	}
}

// TestMigrationSetVersionDownError checks that an error during a step-down is correctly handled.
func TestMigrationSetVersionDownError(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer closeDB(t, db)

	*migration.Migrations = []migration.Migration{
		{
			StepUp:   stepNoopFn,
			StepDown: stepErrorFn,
		},
	}

	// First we need to step-up before we can step-down.
	if err := migration.SetDBVersion(db, 1); err != nil {
		t.Fatal(err)
	}

	if err := migration.SetDBVersion(db, 0); err == nil || err.Error() != stepErrorMsg {
		t.Fatal(err)
	}
}

// TestMigrationSetVersionCurrentTooHighError makes sure that if the current DB version
// is outside of the available migration range, it is reported as an error.
func TestMigrationSetVersionCurrentTooHighError(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer closeDB(t, db)

	if _, err := db.Exec("UPDATE migration_info SET version=10"); err != nil {
		t.Fatal(err)
	}

	const expectedErrStr = "current version (10) is outside of available migration boundaries"
	if err := migration.SetDBVersion(db, 0); err == nil || err.Error() != expectedErrStr {
		t.Fatal(err)
	}
}

func TestMigrationInitClosedDB(t *testing.T) {
	db := prepareDB(t)
	// Intentionally no `defer` here.
	closeDB(t, db)

	if err := migration.InitInfoTable(db); err == nil || err.Error() != dbClosedErrorMsg {
		t.Fatal(err)
	}
}

func TestMigrationGetVersionClosedDB(t *testing.T) {
	db := prepareDBAndMigrations(t)
	// Intentionally no `defer` here.
	closeDB(t, db)

	if _, err := migration.GetDBVersion(db); err == nil || err.Error() != dbClosedErrorMsg {
		t.Fatal(err)
	}
}

func TestMigrationSetVersionClosedDB(t *testing.T) {
	db := prepareDBAndMigrations(t)
	// Intentionally no `defer` here.
	closeDB(t, db)

	if err := migration.SetDBVersion(db, 0); err == nil || err.Error() != dbClosedErrorMsg {
		t.Fatal(err)
	}
}

func TestMigrationInitRollbackStep(t *testing.T) {
	db := prepareDBAndMigrations(t)
	defer closeDB(t, db)

	*migration.Migrations = []migration.Migration{{
		StepUp:   stepRollbackFn,
		StepDown: stepNoopFn,
	}}

	const expectedErrStr = "sql: transaction has already been committed or rolled back"
	if err := migration.SetDBVersion(db, 1); err == nil || err.Error() != expectedErrStr {
		t.Fatal(err)
	}
}
