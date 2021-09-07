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

var mig0016AddRecommendationsTable = Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		// Create recommendation table using currently stored rule hits
		if driver != types.DBDriverPostgres {
			_, err := tx.Exec(`
			CREATE TABLE recommendation
				AS SELECT
					cluster_id,
					rule_fqdn,
					error_key
				FROM rule_hit;
			`)
			if err != nil {
				return err
			}
			// stop here if sqLite
			return nil
		}

		_, err := tx.Exec(`
			CREATE TABLE recommendation
				AS SELECT
					cluster_id,
					REGEXP_REPLACE(rule_fqdn, 'report$', error_key) AS rule_fqdn,
					error_key
				FROM rule_hit;
		`)

		if err != nil {
			return err
		}

		//Add the primary_key to the new table
		_, err = tx.Exec(`
			ALTER TABLE recommendation
				ADD CONSTRAINT recommendation_pk
					PRIMARY KEY (cluster_id, rule_fqdn, error_key);`)
		return err
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		_, err := tx.Exec(`
			DROP TABLE recommendation
		`)
		return err
	},
}
