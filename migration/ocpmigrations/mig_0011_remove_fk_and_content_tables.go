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

/*
   migration11 removes foreign keys to rules and rule error keys, and then removes the
   rule and rule_error_key tables because it won't be stored in database anymore
*/

import (
	"database/sql"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var migrationClusterRuleUserFeedback = migration.Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		var err error
		if driver == types.DBDriverPostgres {
			_, err = tx.Exec(`
				ALTER TABLE cluster_rule_user_feedback DROP
					CONSTRAINT cluster_rule_user_feedback_rule_id_fkey
				`)

		} else {
			err = migration.UpgradeTable(
				tx,
				clusterRuleUserFeedbackTable,
				`
				CREATE TABLE cluster_rule_user_feedback (
					cluster_id VARCHAR NOT NULL,
					rule_id VARCHAR NOT NULL,
					user_id VARCHAR NOT NULL,
					message VARCHAR NOT NULL,
					user_vote SMALLINT NOT NULL,
					added_at TIMESTAMP NOT NULL,
					updated_at TIMESTAMP NOT NULL,

					PRIMARY KEY(cluster_id, rule_id, user_id),
					CONSTRAINT cluster_rule_user_feedback_cluster_id_fkey
					FOREIGN KEY (cluster_id) REFERENCES report(cluster) ON DELETE CASCADE
)
`)
		}
		return err
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		_, err := tx.Exec(`
				ALTER TABLE cluster_rule_user_feedback
					ADD CONSTRAINT cluster_rule_user_feedback_rule_id_fkey FOREIGN KEY(rule_id) REFERENCES rule(module) ON DELETE CASCADE
				`)
		return err
	},
}

var migrationContentTables = migration.Migration{
	StepUp: func(tx *sql.Tx, _ types.DBDriver) error {
		if _, err := tx.Exec("DROP TABLE rule_error_key"); err != nil {
			return err
		}

		_, err := tx.Exec("DROP TABLE rule")
		return err
	},

	StepDown: func(tx *sql.Tx, _ types.DBDriver) error {
		// create table rule
		if _, err := tx.Exec(`
			CREATE TABLE rule (
				module VARCHAR PRIMARY KEY,
				name VARCHAR NOT NULL,
				summary VARCHAR NOT NULL,
				reason VARCHAR NOT NULL,
				resolution VARCHAR NOT NULL,
				more_info VARCHAR NOT NULL
			)`); err != nil {
			return err
		}

		_, err := tx.Exec(`
			CREATE TABLE rule_error_key (
				error_key VARCHAR NOT NULL,
				rule_module VARCHAR NOT NULL REFERENCES rule(module) ON DELETE CASCADE,
				condition VARCHAR NOT NULL,
				description VARCHAR NOT NULL,
				impact INTEGER NOT NULL,
				likelihood INTEGER NOT NULL,
				publish_date TIMESTAMP NOT NULL,
				active BOOLEAN NOT NULL,
				generic VARCHAR NOT NULL,
				tags VARCHAR NOT NULL DEFAULT '',
				PRIMARY KEY("error_key", "rule_module")
			)`)
		return err
	},
}

var mig0011RemoveFKAndContentTables = migration.Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		if err := migrationClusterRuleUserFeedback.StepUp(tx, driver); err != nil {
			return err
		}
		return migrationContentTables.StepUp(tx, driver)
	},

	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		if err := migrationContentTables.StepDown(tx, driver); err != nil {
			return err
		}
		return migrationClusterRuleUserFeedback.StepDown(tx, driver)
	},
}
