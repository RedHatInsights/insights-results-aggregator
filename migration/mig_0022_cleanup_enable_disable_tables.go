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

package migration

import (
	"database/sql"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var mig0022CleanupEnableDisableTables = Migration{
	StepUp: func(tx *sql.Tx, _ types.DBDriver) error {
		// delete from cluster_rule_toggle rows, where rule_id doesn't end in .report
		_, err := tx.Exec(`
			DELETE FROM cluster_rule_toggle WHERE rule_id NOT LIKE '%.report'
		`)

		if err != nil {
			return err
		}

		// delete from cluster_user_rule_disable_feedback rows, where rule_id doesn't end in .report
		_, err = tx.Exec(`
			DELETE FROM cluster_user_rule_disable_feedback WHERE rule_id NOT LIKE '%.report'
		`)

		if err != nil {
			return err
		}

		return err
	},

	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		// this is a one way operation, test thoroughly :)
		return nil
	},
}
