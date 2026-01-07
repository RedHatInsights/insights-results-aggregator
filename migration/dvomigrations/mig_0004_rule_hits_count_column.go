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

package dvomigrations

import (
	"database/sql"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var mig0004AddRuleHitsCount = migration.Migration{
	StepUp: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`
			ALTER TABLE dvo.dvo_report ADD COLUMN rule_hits_count JSONB DEFAULT '{}';
		`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			COMMENT ON COLUMN dvo.dvo_report.rule_hits_count IS 'JSON containing rule IDs and the number of hits for each rule';
		`)

		return err
	},
	StepDown: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`ALTER TABLE dvo.dvo_report DROP COLUMN IF EXISTS rule_hits_count;`)
		return err
	},
}
