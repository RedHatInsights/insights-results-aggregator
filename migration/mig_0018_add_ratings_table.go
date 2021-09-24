/*
Copyright © 2021 Red Hat, Inc.
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

var mig0018AddRatingsTable = Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		// Create recommendation table using currently stored rule hits
		_, err := tx.Exec(`
		CREATE TABLE advisor_ratings (
		    user_id VARCHAR NOT NULL,
			org_id VARCHAR NOT NULL,
			rule_id VARCHAR NOT NULL,
			error_key VARCHAR NOT NULL,
			rated_at TIMESTAMP,
			last_updated_at TIMESTAMP,
			rating SMALLINT,
			PRIMARY KEY(user_id, org_id, rule_id, error_key)
		)`)
		return err
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		_, err := tx.Exec(`
			DROP TABLE advisor_ratings;
		`)
		return err
	},
}
