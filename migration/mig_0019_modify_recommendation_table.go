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

var mig0019ModifyRecommendationTable = Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		if driver != types.DBDriverPostgres {
			return nil
		}

		// Fix rule_fqdn value for records created in migration 16
		_, err := tx.Exec(`
			UPDATE recommendation
				SET rule_fqdn = REGEXP_REPLACE(rule_fqdn, '\.(?!.*\|)(?!.*\.)', '|');
		`)

		return err
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		return nil
	},
}
