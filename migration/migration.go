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

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const defaultDBSchema = "public"

// Version represents a version of the database.
type Version uint

// Schema represents the used schema of the database.
type Schema string

// Step represents an action performed to either increase
// or decrease the migration version of the database.
type Step func(tx *sql.Tx, driver types.DBDriver) error

// Migration type describes a single Migration.
type Migration struct {
	StepUp   Step
	StepDown Step
}

// InitInfoTable ensures that the migration information table is created.
// If it already exists, no changes will be made to the database.
// Otherwise, a new migration information table will be created and initialized.
func InitInfoTable(db *sql.DB, schema Schema) error {
	return withTransaction(db, func(tx *sql.Tx) error {
		if schema == "" {
			schema = defaultDBSchema
		}

		// #nosec G201
		_, err := tx.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %v.migration_info (version INTEGER NOT NULL);", schema))
		if err != nil {
			return err
		}

		// INSERT if there's no rows in the table
		// #nosec G201
		_, err = tx.Exec(
			fmt.Sprintf(
				"INSERT INTO %v.migration_info (version) SELECT 0 WHERE NOT EXISTS (SELECT version FROM %v.migration_info);",
				schema,
				schema,
			),
		)
		if err != nil {
			return err
		}

		var rowCount uint

		// #nosec G201
		err = tx.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %v.migration_info;", schema)).Scan(&rowCount)
		if err != nil {
			return err
		}

		if rowCount != 1 {
			return fmt.Errorf("unexpected number of rows in migration info table (expected: 1, reality: %d)", rowCount)
		}

		return nil
	})
}

// GetDBVersion reads the current version of the database from the migration info table.
func GetDBVersion(db *sql.DB, schema Schema) (Version, error) {
	err := validateNumberOfRows(db, schema)
	if err != nil {
		return 0, err
	}

	if schema == "" {
		schema = defaultDBSchema
	}

	// #nosec G201
	query := fmt.Sprintf("SELECT version FROM %v.migration_info;", schema)

	var version Version // version 0 by default
	err = db.QueryRow(query).Scan(&version)
	err = types.ConvertDBError(err, nil)

	return version, err
}

// SetDBVersion attempts to get the database into the specified
// target version using available migration steps.
func SetDBVersion(
	db *sql.DB,
	dbDriver types.DBDriver,
	dbSchema Schema,
	targetVer Version,
	migrations []Migration,
) error {
	if dbSchema == "" {
		dbSchema = defaultDBSchema
	}

	maxVer := Version(len(migrations))
	if targetVer > maxVer {
		return fmt.Errorf("invalid target version (available version range is 0-%d)", maxVer)
	}

	// Get current database version.
	currentVer, err := GetDBVersion(db, dbSchema)
	if err != nil {
		return err
	}

	// Current version is unexpectedly high.
	if currentVer > maxVer {
		return fmt.Errorf("current version (%d) is outside of available migration boundaries", currentVer)
	}

	return execStepsInTx(db, dbDriver, dbSchema, currentVer, targetVer, migrations)
}

// updateVersionInDB updates the migration version number in the migration info table.
// This function does NOT rollback in case of an error. The calling function is expected to do that.
func updateVersionInDB(tx *sql.Tx, schema Schema, newVersion Version) error {
	// #nosec G201
	res, err := tx.Exec(fmt.Sprintf("UPDATE %v.migration_info SET version=$1;", schema), newVersion)
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
func execStepsInTx(
	db *sql.DB,
	dbDriver types.DBDriver,
	dbSchema Schema,
	currentVer,
	targetVer Version,
	migrations []Migration,
) error {
	// Already at target version.
	if currentVer == targetVer {
		return nil
	}

	return withTransaction(db, func(tx *sql.Tx) error {
		// Upgrade to target version.
		for currentVer < targetVer {
			if err := migrations[currentVer].StepUp(tx, dbDriver); err != nil {
				err = types.ConvertDBError(err, nil)
				return err
			}
			currentVer++
		}

		// Downgrade to target version.
		for currentVer > targetVer {
			if err := migrations[currentVer-1].StepDown(tx, dbDriver); err != nil {
				err = types.ConvertDBError(err, nil)
				return err
			}
			currentVer--
		}

		return updateVersionInDB(tx, dbSchema, currentVer)
	})
}

func validateNumberOfRows(db *sql.DB, schema Schema) error {
	numberOfRows, err := getNumberOfRows(db, schema)
	if err != nil {
		return err
	}
	if numberOfRows != 1 {
		return fmt.Errorf("migration info table contain %v rows", numberOfRows)
	}

	return nil
}

func getNumberOfRows(db *sql.DB, schema Schema) (uint, error) {
	var count uint
	// #nosec G201
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %v.migration_info;", schema)).Scan(&count)
	err = types.ConvertDBError(err, nil)
	return count, err
}
