// Copyright 2021 Red Hat, Inc
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

package storage

import (
	"database/sql"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// DisabledRule represents a record from rule_cluster_toggle table
type DisabledRule struct {
	ClusterID  types.ClusterName
	RuleID     types.RuleID
	ErrorKey   types.ErrorKey
	Disabled   RuleToggle
	DisabledAt sql.NullTime
	EnabledAt  sql.NullTime
	UpdatedAt  sql.NullTime
}

// ListOfDisabledRules function returns list of all rules disabled from a
// specified account.
func (storage DBStorage) ListOfDisabledRules(userID types.UserID) ([]DisabledRule, error) {
	disabledRules := make([]DisabledRule, 0)
	query := `SELECT
                         cluster_id,
			 rule_id,
			 error_key,
			 disabled_at,
			 updated_at,
			 disabled
	FROM
		cluster_rule_toggle
	WHERE
		user_id = $1 and
		disabled = $2
	`

	// run the query against database
	rows, err := storage.connection.Query(query, userID, RuleToggleDisable)

	// return empty list in case of any error
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	for rows.Next() {
		var disabledRule DisabledRule

		err = rows.Scan(&disabledRule.ClusterID,
			&disabledRule.RuleID,
			&disabledRule.ErrorKey,
			&disabledRule.DisabledAt,
			&disabledRule.UpdatedAt,
			&disabledRule.Disabled)

		if err != nil {
			log.Error().Err(err).Msg("ReadListOfDisabledRules")
			return nil, err
		}

		// append disabled rule read from database to a slice
		disabledRules = append(disabledRules, disabledRule)
	}

	return disabledRules, nil
}
