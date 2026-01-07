// Copyright 2022 Red Hat, Inc
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
//
// This migration changes drops the PK on the rule_disable table because it contains
// user_id and we want to keep old records for informational purposes. Creates a non-unique
// index instead to retain the same performance.

package ocpmigrations

import (
	"database/sql"
	"fmt"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	tableName = "rule_disable"
	pkName    = "rule_disable_pkey"
)

var mig0028AlterRuleDisablePKAndIndex = migration.Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {

		if driver == types.DBDriverPostgres {
			alterQuery := fmt.Sprintf("ALTER TABLE %v DROP CONSTRAINT IF EXISTS %v", tableName, pkName)
			_, err := tx.Exec(alterQuery)
			if err != nil {
				return err
			}

			query := fmt.Sprintf("ALTER TABLE %v ADD CONSTRAINT %v PRIMARY KEY (org_id, rule_id, error_key)", tableName, pkName)
			_, err = tx.Exec(query)
			if err != nil {
				return err
			}
		}

		return nil
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		if driver == types.DBDriverPostgres {
			dropIndexQuery := fmt.Sprintf("ALTER TABLE %v DROP CONSTRAINT IF EXISTS %v", tableName, pkName)
			_, err := tx.Exec(dropIndexQuery)
			if err != nil {
				return err
			}

			addPKQuery := fmt.Sprintf("ALTER TABLE %v ADD CONSTRAINT %v PRIMARY KEY (user_id, org_id, rule_id, error_key)", tableName, pkName)
			_, err = tx.Exec(addPKQuery)
			if err != nil {
				return err
			}
		}

		return nil
	},
}
