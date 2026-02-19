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
// This migration adds a new column "org_id" to cluster_rule_toggle,
// cluster_rule_user_feedback and cluster_user_rule_disable_feedback tables,
// then it populates the new column based on the report table, where we have
// information about cluster_id + org_id, so we don't have to use the org_id
// populator from c.r.c. team.

package ocpmigrations

import (
	"database/sql"
	"fmt"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	addOrgIDQuery    = "ALTER TABLE %v ADD COLUMN org_id VARCHAR NOT NULL DEFAULT '0'"
	updateOrgIDQuery = "UPDATE %v as t SET org_id = report.org_id FROM report WHERE report.cluster = t.cluster_id"
	dropOrgIDQuery   = "ALTER TABLE %v DROP COLUMN IF EXISTS org_id"
)

var tablesToUpdate = []string{
	"cluster_rule_toggle",
	"cluster_rule_user_feedback",
	"cluster_user_rule_disable_feedback",
}

var mig0026AddAndPopulateOrgIDColumns = migration.Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {

		for _, table := range tablesToUpdate {
			alterQuery := fmt.Sprintf(addOrgIDQuery, table)
			updateQuery := fmt.Sprintf(updateOrgIDQuery, table) // #nosec G201 -- table is from constant tablesToUpdate

			// add org_id column
			_, err := tx.Exec(alterQuery)
			if err != nil {
				return err
			}

			if driver == types.DBDriverPostgres {
				// update org_id from report table
				_, err = tx.Exec(updateQuery)
				if err != nil {
					return err
				}
			}
		}

		return nil
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {

		for _, table := range tablesToUpdate {
			dropQuery := fmt.Sprintf(dropOrgIDQuery, table)

			// Remove the columns
			_, err := tx.Exec(dropQuery)
			if err != nil {
				return err
			}
		}

		return nil
	},
}
