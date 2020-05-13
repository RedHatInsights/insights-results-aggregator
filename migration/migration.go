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

// Version represents a version of the database.
type Version uint

// Step represents an action performed to either increase
// or decrease the migration version of the database.
type Step func(tx *sql.Tx, driver types.DBDriver) error

// Migration type describes a single Migration.
type Migration struct {
	StepUp   Step
	StepDown Step
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
	return withTransaction(db, func(tx *sql.Tx) error {
		_, err := tx.Exec("CREATE TABLE IF NOT EXISTS migration_info (version INTEGER NOT NULL);")
		if err != nil {
			return err
		}

		// INSERT if there's no rows in the table
		_, err = tx.Exec(`
			INSERT INTO migration_info (version) SELECT 0 WHERE NOT EXISTS (SELECT version FROM migration_info);
		`)
		if err != nil {
			return err
		}

		var rowCount uint
		err = tx.QueryRow("SELECT COUNT(*) FROM migration_info").Scan(&rowCount)
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
func GetDBVersion(db *sql.DB) (Version, error) {
	err := validateNumberOfRows(db)
	if err != nil {
		return 0, err
	}

	var version Version = 0
	err = db.QueryRow("SELECT version FROM migration_info").Scan(&version)
	err = types.ConvertDBError(err)

	return version, err
}

// SetDBVersion attempts to get the database into the specified
// target version using available migration steps.
func SetDBVersion(db *sql.DB, dbDriver types.DBDriver, targetVer Version) error {
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

	return execStepsInTx(db, dbDriver, currentVer, targetVer)
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
func execStepsInTx(db *sql.DB, dbDriver types.DBDriver, currentVer, targetVer Version) error {
	// Already at target version.
	if currentVer == targetVer {
		return nil
	}

	return withTransaction(db, func(tx *sql.Tx) error {
		// Upgrade to target version.
		for currentVer < targetVer {
			if err := migrations[currentVer].StepUp(tx, dbDriver); err != nil {
				err = types.ConvertDBError(err)
				return err
			}
			currentVer++
		}

		// Downgrade to target version.
		for currentVer > targetVer {
			if err := migrations[currentVer-1].StepDown(tx, dbDriver); err != nil {
				err = types.ConvertDBError(err)
				return err
			}
			currentVer--
		}

		if err := updateVersionInDB(tx, currentVer); err != nil {
			return err
		}

		return nil
	})
}

func validateNumberOfRows(db *sql.DB) error {
	numberOfRows, err := getNumberOfRows(db)
	if err != nil {
		return err
	}
	if numberOfRows != 1 {
		return fmt.Errorf("migration info table contain %v rows", numberOfRows)
	}

	return nil
}

func getNumberOfRows(db *sql.DB) (uint, error) {
	var count uint
	err := db.QueryRow("SELECT COUNT(*) FROM migration_info;").Scan(&count)
	err = types.ConvertDBError(err)
	return count, err
}
