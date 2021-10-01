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

package migration

import (
	"database/sql"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var mig0019ModifyRecommendationRuleFQDN = Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		if driver != types.DBDriverPostgres {
			// Add rule_id column
			_, err := tx.Exec(`
				ALTER TABLE recommendation ADD COLUMN rule_id VARCHAR NOT NULL DEFAULT '.';
				UPDATE recommendation SET rule_id = rule_fqdn + '|' + error_key;
			`)
			if err != nil {
				return err
			}

			// Add the created_at column
			_, err = tx.Exec(`
				ALTER TABLE recommendation ADD COLUMN created_at TIMESTAMP WITHOUT TIME ZONE;
			`)
			return err
		}

		// Recreate table to fix rule_fqdn value for records created in migration 16
		// The regex expression has two parts separated by a logical or `|`:
		// (\.(?!.*\|)(?!.*\.|\|).*) finds the last dot and all the characters that follows it
		// (\|.*) finds the '|' and all the characters that follow it
		// Both patterns are replaced by an empty string, so we are left with only the rule's
		// component ID in the `rule_fqdn` column
		_, err := tx.Exec(`
			UPDATE recommendation
				SET rule_fqdn = REGEXP_REPLACE(rule_fqdn, '(\.(?!.*\|)(?!.*\.|\|).*)|(\|.*)', '');
			`)

		if err != nil {
			return err
		}

		// Add the rule_id column, with a little trick to ensure future inserts do not allow it to be empty
		// Postgres doesn't allow using other columns in default value, and inserting a simple '.', we can
		// avoid using triggers to fill the column and save some time
		_, err = tx.Exec(`
			ALTER TABLE recommendation
			    ADD COLUMN rule_id VARCHAR NOT NULL DEFAULT '.';
			UPDATE recommendation
				SET rule_id = CONCAT(rule_fqdn, '|', error_key);
		`)
		if err != nil {
			return err
		}

		//Add the created_at column with current UTC time as value
		_, err = tx.Exec(`
			ALTER TABLE recommendation
				ADD COLUMN created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'utc');
		`)
		return err
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		if driver == types.DBDriverPostgres {
			//Remove the created_at column
			_, err := tx.Exec(`
			ALTER TABLE recommendation DROP COLUMN IF EXISTS created_at;
			`)

			return err
		}
		return nil
	},
}
