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

/*
	migration14 removes user_id from primary key
	This is not really compatible for the StepDown, as in the new solution, there will be null user_id values
	which can't be used in a primary key...
*/

var mig0014ModifyClusterRuleToggleAlter = Migration{
	StepUp: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`
			ALTER TABLE cluster_rule_toggle DROP CONSTRAINT cluster_rule_toggle_pkey,
				ADD CONSTRAINT cluster_rule_toggle_pkey PRIMARY KEY (cluster_id, rule_id);
			ALTER TABLE cluster_rule_toggle ALTER COLUMN user_id DROP NOT NULL;
		`)
		return err
	},
	StepDown: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`
			ALTER TABLE cluster_rule_toggle DROP CONSTRAINT cluster_rule_toggle_pkey,
				ADD CONSTRAINT cluster_rule_toggle_pkey PRIMARY KEY (cluster_id, rule_id, user_id);
			ALTER TABLE cluster_rule_toggle ALTER COLUMN user_id SET NOT NULL;
		`)
		return err
	},
}

var mig0014ModifyClusterRuleToggleGeneral = NewUpdateTableMigration(
	clusterRuleToggleTable,
	`
		CREATE TABLE cluster_rule_toggle (
			cluster_id VARCHAR NOT NULL,
			rule_id VARCHAR NOT NULL,
			user_id VARCHAR NOT NULL,
			disabled SMALLINT NOT NULL,
			disabled_at TIMESTAMP NULL,
			enabled_at TIMESTAMP NULL,
			updated_at TIMESTAMP NOT NULL,

			CHECK (disabled >= 0 AND disabled <= 1),
			PRIMARY KEY(cluster_id, rule_id, user_id)
		);
	`,
	nil,
	`
		CREATE TABLE cluster_rule_toggle (
			cluster_id VARCHAR NOT NULL,
			rule_id VARCHAR NOT NULL,
			user_id VARCHAR NULL,
			disabled SMALLINT NOT NULL,
			disabled_at TIMESTAMP NULL,
			enabled_at TIMESTAMP NULL,
			updated_at TIMESTAMP NOT NULL,

			CHECK (disabled >= 0 AND disabled <= 1),
			PRIMARY KEY(cluster_id, rule_id)
		)
	`,
)

var mig0014ModifyClusterRuleToggle = Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		if driver == types.DBDriverPostgres {
			return mig0014ModifyClusterRuleToggleAlter.StepUp(tx, driver)
		}

		return mig0014ModifyClusterRuleToggleGeneral.StepUp(tx, driver)
	},

	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		if driver == types.DBDriverPostgres {
			return mig0014ModifyClusterRuleToggleAlter.StepDown(tx, driver)
		}

		return mig0014ModifyClusterRuleToggleGeneral.StepDown(tx, driver)
	},
}
