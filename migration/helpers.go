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

package migration

import (
	"database/sql"
	"strings"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// NewUpdateTableMigration generates a migration which changes tables schema and copies data
// (should work in most of cases like adding a field, altering it and so on)
// Set previousColumns to the list of previous columns if you're changing any columns
func NewUpdateTableMigration(tableName, previousSchema string, previousColumns []string, newSchema string) Migration {
	return Migration{
		StepUp: func(tx *sql.Tx, _ types.DBDriver) error {
			return upgradeTable(tx, tableName, newSchema)
		},
		StepDown: func(tx *sql.Tx, _ types.DBDriver) error {
			return downgradeTable(tx, tableName, previousSchema, previousColumns)
		},
	}
}

func upgradeTable(tx *sql.Tx, tableName, newTableDefinition string) error {
	// disable "G202 (CWE-89): SQL string concatenation"
	// #nosec G202
	_, err := tx.Exec(`ALTER TABLE ` + tableName + ` RENAME TO tmp;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(newTableDefinition)
	if err != nil {
		return err
	}

	// disable "G202 (CWE-89): SQL string concatenation"
	// #nosec G202
	_, err = tx.Exec("INSERT INTO " + tableName + " SELECT * FROM tmp;")
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DROP TABLE tmp;`)
	if err != nil {
		return err
	}

	return nil
}

// downgradeTable downgrades table to oldTableDefinition, useful for sqlite which doesn't support
// most of alter table queries. Set columns to the list of new columns if you're removing any columns
func downgradeTable(tx *sql.Tx, tableName, oldTableDefinition string, columns []string) error {
	// disable "G202 (CWE-89): SQL string concatenation"
	// #nosec G202
	_, err := tx.Exec(`ALTER TABLE ` + tableName + ` RENAME TO tmp;`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(oldTableDefinition)
	if err != nil {
		return err
	}

	columnsStr := "*"
	if len(columns) != 0 {
		columnsStr = strings.Join(columns, ",")
	}

	// disable "G202 (CWE-89): SQL string concatenation"
	// #nosec G202
	_, err = tx.Exec("INSERT INTO " + tableName + " SELECT " + columnsStr + " FROM tmp;")
	if err != nil {
		return err
	}

	_, err = tx.Exec("DROP TABLE tmp;")
	if err != nil {
		return err
	}

	return nil
}
