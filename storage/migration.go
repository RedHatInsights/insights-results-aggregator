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

package storage

import (
	"database/sql"
	"fmt"
)

// MigrationVersion represents a version of the database.
type MigrationVersion uint

// MigrationStep represents an action performed to either increase
// or decrease the migration version of the database.
type MigrationStep func(tx *sql.Tx) error

// Migration type describes a single Migration.
type Migration struct {
	StepUp   MigrationStep
	StepDown MigrationStep
}

// migrations is a list of migrations that, when applied in their order,
// create the most recent version of the database from scratch.
var migrations []Migration = []Migration{}

// GetHighestMigrationVersion returns the highest available migration version.
// The DB version cannot be set to a value higher than this.
// This value is equivalent to the length of the list of available migrations.
func GetHighestMigrationVersion() MigrationVersion {
	return MigrationVersion(len(migrations))
}

// AddMigration adds a new migration to the list of available migrations.
func AddMigration(m Migration) {
	migrations = append(migrations, m)
}

// ClearMigrations clears the list of available migrations.
func ClearMigrations() {
	migrations = []Migration{}
}

// InitMigrationInfo ensures that the migration information table is created.
// If it already exists, no changes will be made to the database.
// Otherwise, a new migration information table will be created and initialized.
func InitMigrationInfo(db *DBStorage) error {
	tx, err := db.connection.Begin()
	if err != nil {
		return err
	}

	// Check if migration info table already exists.
	countResult := tx.QueryRow("SELECT COUNT(*) FROM migration_info")
	var rowCount int
	err = countResult.Scan(&rowCount)
	// If it exists, it must have exactly 1 row.
	// If it doesn't exist, the "no such table" error is expected.
	// Otherwise, there's something wrong.
	if err == nil {
		tx.Rollback()
		if rowCount != 1 {
			return fmt.Errorf("Unexpected number of rows in migration info table (expected: 1, reality: %d)", rowCount)
		}
		return nil
	} else if err.Error() != "no such table: migration_info" {
		tx.Rollback()
		return err
	}

	if err = initInfoTab(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// GetDBVersion reads the current version of the database from the migration info table.
func GetDBVersion(db *DBStorage) (MigrationVersion, error) {
	rows, err := db.connection.Query("SELECT version FROM migration_info")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	// Read the first (and hopefully the only) row in the table.
	if !rows.Next() {
		return 0, fmt.Errorf("Migration info table is empty")
	}

	var version MigrationVersion = 0
	err = rows.Scan(&version)
	if err != nil {
		return 0, err
	}

	// Check if another row is available (it should NOT be).
	if rows.Next() {
		return 0, fmt.Errorf("Migration info table contain multiple rows")
	}

	return version, nil
}

// SetDBVersion attempts to get the database into the specified
// target version using available migration steps.
func SetDBVersion(db *DBStorage, targetVer MigrationVersion) error {
	maxVer := GetHighestMigrationVersion()
	if targetVer > maxVer {
		return fmt.Errorf("Invalid target version (available version range is 0-%d)", maxVer)
	}

	// Get current database version.
	currentVer, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	// Current version is unexpectedly high.
	if currentVer > maxVer {
		return fmt.Errorf("Current version (%d) is outside of available migration boundaries", currentVer)
	}

	return execMigrationStepsInTx(db, currentVer, targetVer)
}

// initInfoTab performs the actual creation and initialization of the migration info table.
// Transaction finalization (rollback/commit) is expected to be done by the calling function.
func initInfoTab(tx *sql.Tx) error {
	_, err := tx.Exec("CREATE TABLE migration_info (version INTEGER NOT NULL)")
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO migration_info(version) VALUES(0)")
	if err != nil {
		return err
	}

	return nil
}

// updateMigrationVersionInDB updates the migration version number in the migration info table.
// This function does NOT rollback in case of an error. The calling function is expected to do that.
func updateMigrationVersionInDB(tx *sql.Tx, newVersion MigrationVersion) error {
	res, err := tx.Exec("UPDATE migration_info SET version=?", newVersion)
	if err != nil {
		return err
	}

	// Check that there is exactly 1 row in the migration info table.
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected != 1 {
		return fmt.Errorf("Unexpected number of affected rows in migration info table (expected: 1, reality: %d)", affected)
	}

	return nil
}

// execMigrationStepsInTx executes the necessary migration steps in a single transaction.
func execMigrationStepsInTx(db *DBStorage, currentVer, targetVer MigrationVersion) error {
	// Already at target version.
	if currentVer == targetVer {
		return nil
	}

	// Begin a new transaction.
	tx, err := db.connection.Begin()
	if err != nil {
		return err
	}

	// Upgrade to target version.
	for currentVer < targetVer {
		if err = migrations[currentVer].StepUp(tx); err != nil {
			tx.Rollback()
			return err
		}
		currentVer++
	}

	// Downgrade to target version.
	for currentVer > targetVer {
		if err = migrations[currentVer-1].StepDown(tx); err != nil {
			tx.Rollback()
			return err
		}
		currentVer--
	}

	if err = updateMigrationVersionInDB(tx, currentVer); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
