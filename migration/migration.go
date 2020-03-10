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

// Package migration contains an implementation of a simple database migration
// mechanism that allows semi-automatic transitions between various database
// versions as well as building the latest version of the database from
// scratch.
//
// Please look into README.md with further instructions how to use it.
package migration

import (
	"database/sql"
	"fmt"
	"strings"
)

// Version represents a version of the database.
type Version uint

// Step represents an action performed to either increase
// or decrease the migration version of the database.
type Step func(tx *sql.Tx) error

// Migration type describes a single Migration.
type Migration struct {
	StepUp   Step
	StepDown Step
}

// migrations is a list of migrations that, when applied in their order,
// create the most recent version of the database from scratch.
var migrations = []Migration{
	mig1,
	mig3,
}

// GetMaxVersion returns the highest available migration version.
// The DB version cannot be set to a value higher than this.
// This value is equivalent to the length of the list of available migrations.
func GetMaxVersion() Version {
	return Version(len(migrations))
}

// InitInfoTable ensures that the migration information table is created.
// If it already exists, no changes will be made to the database.
// Otherwise, a new migration information table will be created and initialized.
func InitInfoTable(db *sql.DB) error {
	// Check if migration info table already exists.
	countResult := db.QueryRow("SELECT COUNT(*) FROM migration_info")
	var rowCount int
	err := countResult.Scan(&rowCount)
	// If it exists, it must have exactly 1 row.
	// If it doesn't exist, the "no such table" error is expected.
	// Otherwise, there's something wrong.
	if err == nil {
		if rowCount != 1 {
			return fmt.Errorf("unexpected number of rows in migration info table (expected: 1, reality: %d)", rowCount)
		}
		return nil
	} else if !strings.Contains(err.Error(), "migration_info") {
		// An error not related to the nonexistence of the migration_info table.
		return err
	}

	return initInfoTab(db)
}

// GetDBVersion reads the current version of the database from the migration info table.
func GetDBVersion(db *sql.DB) (Version, error) {
	rows, err := db.Query("SELECT version FROM migration_info")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	// Read the first (and hopefully the only) row in the table.
	if !rows.Next() {
		return 0, fmt.Errorf("migration info table is empty")
	}

	var version Version = 0
	err = rows.Scan(&version)
	if err != nil {
		return 0, err
	}

	// Check if another row is available (it should NOT be).
	if rows.Next() {
		return 0, fmt.Errorf("migration info table contain multiple rows")
	}

	return version, nil
}

// SetDBVersion attempts to get the database into the specified
// target version using available migration steps.
func SetDBVersion(db *sql.DB, targetVer Version) error {
	maxVer := GetMaxVersion()
	if targetVer > maxVer {
		return fmt.Errorf("invalid target version (available version range is 0-%d)", maxVer)
	}

	// Get current database version.
	currentVer, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	// Current version is unexpectedly high.
	if currentVer > maxVer {
		return fmt.Errorf("current version (%d) is outside of available migration boundaries", currentVer)
	}

	return execStepsInTx(db, currentVer, targetVer)
}

// initInfoTab performs the actual creation and initialization of the migration info table.
func initInfoTab(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("CREATE TABLE migration_info (version INTEGER NOT NULL)")
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Exec("INSERT INTO migration_info(version) VALUES(0)")
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// updateVersionInDB updates the migration version number in the migration info table.
// This function does NOT rollback in case of an error. The calling function is expected to do that.
func updateVersionInDB(tx *sql.Tx, newVersion Version) error {
	res, err := tx.Exec("UPDATE migration_info SET version=$1", newVersion)
	if err != nil {
		return err
	}

	// Check that there is exactly 1 row in the migration info table.
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected != 1 {
		return fmt.Errorf("unexpected number of affected rows in migration info table (expected: 1, reality: %d)", affected)
	}

	return nil
}

// execStepsInTx executes the necessary migration steps in a single transaction.
func execStepsInTx(db *sql.DB, currentVer, targetVer Version) error {
	// Already at target version.
	if currentVer == targetVer {
		return nil
	}

	// Begin a new transaction.
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Upgrade to target version.
	for currentVer < targetVer {
		if err = migrations[currentVer].StepUp(tx); err != nil {
			_ = tx.Rollback()
			return err
		}
		currentVer++
	}

	// Downgrade to target version.
	for currentVer > targetVer {
		if err = migrations[currentVer-1].StepDown(tx); err != nil {
			_ = tx.Rollback()
			return err
		}
		currentVer--
	}

	if err = updateVersionInDB(tx, currentVer); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
