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

var mig0012CreateClusterUserRuleDisableFeedback = migration.Migration{
	StepUp: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`
			CREATE TABLE cluster_user_rule_disable_feedback (
				cluster_id VARCHAR NOT NULL,
				user_id VARCHAR NOT NULL,
				rule_id VARCHAR NOT NULL,
				message VARCHAR NOT NULL,
				added_at TIMESTAMP NOT NULL,
				updated_at TIMESTAMP NOT NULL,

				PRIMARY KEY(cluster_id, user_id, rule_id)
			)`)
		return err
	},
	StepDown: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`DROP TABLE cluster_user_rule_disable_feedback`)
		return err
	},
}
