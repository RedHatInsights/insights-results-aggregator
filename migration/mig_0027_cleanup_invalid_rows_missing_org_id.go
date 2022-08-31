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
//
// This migration deletes invalid rows from tables that were modified in mig0026 (added
// and populated org_id columns based on `report` table) -- cluster_rule_toggle,
// cluster_rule_user_feedback and cluster_user_rule_disable_feedback. If the newly populated
// org_id columns are at the default value (0), that means we don't have that cluster in the
// report table (could happen, we sometimes don't check if a cluster exists), therefore they're
// invalid rows.

package migration

import (
	"database/sql"
	"fmt"

	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/rs/zerolog/log"
)

const (
	deleteEmptyOrgIDQuery = "DELETE FROM %v WHERE org_id = '0'"
)

var tablesToDeleteFrom = []string{
	"cluster_rule_toggle",
	"cluster_rule_user_feedback",
	"cluster_user_rule_disable_feedback",
}

var mig0027CleanupInvalidRowsMissingOrgID = Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {

		if driver == types.DBDriverPostgres {
			for _, table := range tablesToDeleteFrom {
				deleteQuery := fmt.Sprintf(deleteEmptyOrgIDQuery, table)

				// execute delete
				res, err := tx.Exec(deleteQuery)
				if err != nil {
					return err
				}

				// check number of affected (deleted) rows
				deletedRows, err := res.RowsAffected()
				if err != nil {
					log.Error().Err(err).Msg("unable to retrieve number of deleted rows")
					return err
				}

				log.Info().Msgf("deleted %d rows from table %v", deletedRows, table)
			}
		}

		return nil
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		log.Info().Msg("Mig0027 is a one-way ticket. Nothing to be done for StepDown.")
		return nil
	},
}
