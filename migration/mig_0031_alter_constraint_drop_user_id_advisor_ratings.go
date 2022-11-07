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
// This migration drops the user_id columns from the advisor_ratings table.
// Some tables have the user_id in the PRIMARY KEY, but we should be OK dropping it
// anyway because user_id was actually account_number, so the tables shouldn't have
// duplicate records per rule (or per cluster) per organization.
// Organization IDs were added and populated in previous migrations.

package migration

import (
	"database/sql"
	"fmt"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

type alterConstraintStep struct {
	tableName     string
	oldConstraint string
	newConstraint string
}

var migrationStep = alterConstraintStep{
	tableName:     "advisor_ratings",
	oldConstraint: "(user_id, org_id, rule_fqdn, error_key)",
	newConstraint: "(org_id, rule_fqdn, error_key)",
}

var mig0031AlterConstraintDropUserAdvisorRatings = Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		if driver == types.DBDriverPostgres {

			// user_id is in the primary key, we need to create a new one
			dropPKQuery := fmt.Sprintf(alterTableDropPK, migrationStep.tableName, migrationStep.tableName)
			_, err := tx.Exec(dropPKQuery)
			if err != nil {
				return err
			}

			addPKQuery := fmt.Sprintf(
				alterTableAddPK,
				migrationStep.tableName,
				migrationStep.tableName,
				migrationStep.newConstraint,
			)

			_, err = tx.Exec(addPKQuery)
			if err != nil {
				return err
			}

			dropColumnQuery := fmt.Sprintf(alterTableDropColumnQuery, migrationStep.tableName, userIDColumn)
			_, err = tx.Exec(dropColumnQuery)
			if err != nil {
				return err
			}
		}

		return nil
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		if driver == types.DBDriverPostgres {

			addColumnQuery := fmt.Sprintf(alterTableAddVarcharColumn, migrationStep.tableName, userIDColumn)
			_, err := tx.Exec(addColumnQuery)
			if err != nil {
				return err
			}

			dropPKQuery := fmt.Sprintf(alterTableDropPK, migrationStep.tableName, migrationStep.tableName)
			_, err = tx.Exec(dropPKQuery)
			if err != nil {
				return err
			}

			addPKQuery := fmt.Sprintf(
				alterTableAddPK,
				migrationStep.tableName,
				migrationStep.tableName,
				migrationStep.oldConstraint,
			)
			_, err = tx.Exec(addPKQuery)
			if err != nil {
				return err
			}
		}

		return nil
	},
}
