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
// This migration drops the user_id columns from the
// rule_disable table. This table doesn't have the user_id
// in the constraint(s), so we can remove the column without needing to
// alter it.

package ocpmigrations

import (
	"database/sql"
	"fmt"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	ruleDisableTable = "rule_disable"
)

var mig0030DropRuleDisableUserIDColumn = migration.Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		if driver == types.DBDriverPostgres {
			dropColumnQuery := fmt.Sprintf(alterTableDropColumnQuery, ruleDisableTable, userIDColumn)
			_, err := tx.Exec(dropColumnQuery)
			if err != nil {
				return err
			}
		}

		return nil
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		if driver == types.DBDriverPostgres {
			addColumnQuery := fmt.Sprintf(alterTableAddVarcharColumn, ruleDisableTable, userIDColumn)
			_, err := tx.Exec(addColumnQuery)
			if err != nil {
				return err
			}
		}

		return nil
	},
}
