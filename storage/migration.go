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
type MigrationVersion int

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
var migrations []Migration

// AddMigration adds a new migration to the list of available migrations.
func AddMigration(m Migration) {
	migrations = append(migrations, m)
}

// InitMigrationInfo creates a table containing migration information in the database.
func InitMigrationInfo(db *DBStorage) error {
	tx, err := db.connection.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("CREATE TABLE migration_info (version INTEGER)")
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("INSERT INTO migration_info(version) VALUES(0)")
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// GetDBVersion reads the current version of the database from the migration info table.
func GetDBVersion(db *DBStorage) (MigrationVersion, error) {
	versionRow := db.connection.QueryRow("SELECT version FROM migration_info")
	var version MigrationVersion = 0
	err := versionRow.Scan(&version)
	return version, err
}

// SetDBVersion attempts to get the database into the specified
// target version using available migration steps.
func SetDBVersion(db *DBStorage, targetVer MigrationVersion) error {
	migrationCount := len(migrations)
	if targetVer < 0 || targetVer > MigrationVersion(migrationCount) {
		return fmt.Errorf("Invalid target version (available version range is 0-%d)", migrationCount)
	}

	// Get current database version.
	currentVer, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	// Current version is unexpectedly high.
	if currentVer < 0 || currentVer > MigrationVersion(migrationCount) {
		return fmt.Errorf("Current version (%d) is outside of available migration boundaries", currentVer)
	}

	// Already at target version.
	if currentVer == targetVer {
		return nil
	}

	// Begin a new transaction.
	tx, err := db.connection.Begin()
	if err != nil {
		return err
	}

	for currentVer < targetVer {
		if err = migrations[currentVer].StepUp(tx); err != nil {
			tx.Rollback()
			return err
		}
		currentVer++
	}

	for currentVer > targetVer {
		if err = migrations[currentVer-1].StepDown(tx); err != nil {
			tx.Rollback()
			return err
		}
		currentVer--
	}

	_, err = tx.Exec("UPDATE migration_info SET version=?", currentVer)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
