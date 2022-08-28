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
// This migration nullifies user_id columns in every single table where such
// a column exists. Previously we used to mistakenly populate this colummn with account_number
// which used to serve the same purpose as org_id (as in one value to represent the whole organization),
// instead of the actual ID of a single user, like we thought we did. In some tables the user_id is not
// really used for functionality, because such tables are used for org-wide functionality, but we want to
// keep it for informational/debugging purposes.
// This migration requires corresponding changes in smart-proxy/aggregator to start populating the user_id
// columns with the proper values.

package migration

import (
	"database/sql"
	"fmt"

	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/rs/zerolog/log"
)

const (
	nullifyUserIDQuery = "UPDATE %v SET user_id = '0'"
)

var tablesToNullify = []string{
	"advisor_ratings",
	"cluster_rule_toggle",
	"cluster_rule_user_feedback",
	"cluster_user_rule_disable_feedback",
	"rule_disable",
}

var mig0029NullifyUserIDColumns = Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {

		if driver == types.DBDriverPostgres {
			for _, table := range tablesToNullify {
				nullifyQuery := fmt.Sprintf(nullifyUserIDQuery, table)

				// exec query to nullify user_id in all rows
				res, err := tx.Exec(nullifyQuery)
				if err != nil {
					return err
				}

				// check number of affected rows
				nOfRows, err := res.RowsAffected()
				if err != nil {
					log.Error().Err(err).Msg("unable to retrieve number of affected rows")
					return err
				}

				log.Info().Msgf("updated %d rows from table %v", nOfRows, table)
			}
		}

		return nil
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		log.Info().Msg("Mig0028 is a one-way ticket. Nothing to be done for StepDown.")
		return nil
	},
}
