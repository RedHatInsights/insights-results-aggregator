// Copyright 2026 Red Hat, Inc
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
// This migration removes invalid rows from cluster_rule_toggle where error_key
// is empty. Such rows were created before error_key was added to the primary key
// and populated for existing rules.

package ocpmigrations

import (
	"database/sql"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/rs/zerolog/log"
)

var mig0032CleanupClusterRuleToggleEmptyErrorKey = migration.Migration{
	StepUp: func(tx *sql.Tx, _ types.DBDriver) error {
		res, err := tx.Exec(`
			DELETE FROM cluster_rule_toggle WHERE error_key = ''
		`)
		if err != nil {
			return err
		}

		deletedRows, err := res.RowsAffected()
		if err != nil {
			log.Error().Err(err).Msg("unable to retrieve number of deleted rows")
			return err
		}

		log.Info().Msgf("deleted %d rows from table cluster_rule_toggle with empty error_key", deletedRows)

		return nil
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		log.Info().Msg("Mig0032 is a one-way ticket. Nothing to be done for StepDown.")
		return nil
	},
}
