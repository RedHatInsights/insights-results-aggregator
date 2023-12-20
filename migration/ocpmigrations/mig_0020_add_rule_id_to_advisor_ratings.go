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

package ocpmigrations

import (
	"database/sql"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var mig0020ModifyAdvisorRatingsTable = migration.Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		// common code for renaming column and adding a new one
		_, err := tx.Exec(`
			ALTER TABLE advisor_ratings RENAME COLUMN rule_id TO rule_fqdn;
			ALTER TABLE advisor_ratings ADD COLUMN rule_id VARCHAR NOT NULL DEFAULT '.';
		`)

		if err != nil {
			return err
		}

		// Rename rule_id to rule_fqdn
		_, err = tx.Exec(`
				UPDATE advisor_ratings SET rule_id = CONCAT(rule_fqdn, '|', error_key);
			`)
		return err
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		// Remove the rule_id column
		_, err := tx.Exec(`
				ALTER TABLE advisor_ratings DROP COLUMN IF EXISTS rule_id;
			`)
		if err != nil {
			return err
		}

		// Rename rule_fqdn back to rule_id
		_, err = tx.Exec(`
			ALTER TABLE advisor_ratings RENAME COLUMN rule_fqdn TO rule_id;
		`)
		return err
	},
}
