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
// This migration drops the user_id columns from all tables.
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

const (
	userIDColumn = "user_id"
)

// tables that don't have user_id in the PK/constraint
var tablesWithoutConstraint = []string{
	clusterRuleToggleTable,
	ruleDisableTable,
}

// tables that have user_id in the PK/constraint, needs to alter the constraint first
var tablesWithConstraint = []alterConstraintStep{
	{
		tableName:     advisorRatingsTable,
		oldConstraint: "(user_id, org_id, rule_fqdn, error_key)",
		newConstraint: "(org_id, rule_fqdn, error_key)",
	},
	{
		tableName:     clusterRuleUserFeedbackTable,
		oldConstraint: "(cluster_id, rule_id, user_id, error_key)",
		newConstraint: "(cluster_id, org_id, rule_id, error_key)",
	},
	{
		tableName:     clusterUserRuleDisableFeedbackTable,
		oldConstraint: "(cluster_id, user_id, rule_id, error_key)",
		newConstraint: "(cluster_id, org_id, rule_id, error_key)",
	},
}

var mig0029DropUserIDColumns = Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		if driver == types.DBDriverPostgres {
			// tables that don't have user_id in a constraint can be modified simply
			for _, table := range tablesWithoutConstraint {
				dropColumnQuery := fmt.Sprintf("ALTER TABLE %v DROP COLUMN IF EXISTS %v", table, userIDColumn)
				_, err := tx.Exec(dropColumnQuery)
				if err != nil {
					return err
				}
			}

			// tables with user_id in constraint are slightly more complicated, we need to create a new PK
			for _, step := range tablesWithConstraint {
				alterQuery := fmt.Sprintf("ALTER TABLE %v DROP CONSTRAINT IF EXISTS %v_pkey", step.tableName, step.tableName)
				_, err := tx.Exec(alterQuery)
				if err != nil {
					return err
				}

				query := fmt.Sprintf(
					"ALTER TABLE %v ADD CONSTRAINT %v_pkey PRIMARY KEY %v",
					step.tableName,
					step.tableName,
					step.newConstraint,
				)

				_, err = tx.Exec(query)
				if err != nil {
					return err
				}

				dropColumnQuery := fmt.Sprintf("ALTER TABLE %v DROP COLUMN IF EXISTS %v", step.tableName, userIDColumn)
				_, err = tx.Exec(dropColumnQuery)
				if err != nil {
					return err
				}
			}
		}

		return nil
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		if driver == types.DBDriverPostgres {
			// tables that don't have user_id in a constraint can be modified simply
			for _, table := range tablesWithoutConstraint {
				addColumnQuery := fmt.Sprintf("ALTER TABLE %v ADD COLUMN %v VARCHAR NOT NULL DEFAULT '-1'", table, userIDColumn)
				_, err := tx.Exec(addColumnQuery)
				if err != nil {
					return err
				}
			}

			// tables with user_id in constraint are slightly more complicated, we need to re-create the old PK
			for _, step := range tablesWithConstraint {
				addColumnQuery := fmt.Sprintf("ALTER TABLE %v ADD COLUMN %v VARCHAR NOT NULL DEFAULT '-1'", step.tableName, userIDColumn)
				_, err := tx.Exec(addColumnQuery)
				if err != nil {
					return err
				}

				alterQuery := fmt.Sprintf("ALTER TABLE %v DROP CONSTRAINT IF EXISTS %v_pkey", step.tableName, step.tableName)
				_, err = tx.Exec(alterQuery)
				if err != nil {
					return err
				}

				query := fmt.Sprintf(
					"ALTER TABLE %v ADD CONSTRAINT %v_pkey PRIMARY KEY %v",
					step.tableName,
					step.tableName,
					step.oldConstraint,
				)
				_, err = tx.Exec(query)
				if err != nil {
					return err
				}
			}
		}

		return nil
	},
}
