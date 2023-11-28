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
	"time"

	"github.com/rs/zerolog/log"

	ctypes "github.com/RedHatInsights/insights-results-types"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// DisableRuleSystemWide disables the selected rule for all clusters visible to
// given user
func (storage OCPRecommendationsDBStorage) DisableRuleSystemWide(
	orgID types.OrgID,
	ruleID types.RuleID,
	errorKey types.ErrorKey,
	justification string,
) error {
	now := time.Now()

	const query = `
	INSERT INTO rule_disable(
		org_id, rule_id, error_key, justification, created_at
	)
	VALUES
		($1, $2, $3, $4, $5)
	ON CONFLICT
		(org_id, rule_id, error_key)
	DO UPDATE SET
		justification = $4, created_at = $5
`

	// try to execute the query and check for (any) error
	_, err := storage.connection.Exec(
		query,
		orgID,
		ruleID,
		errorKey,
		justification,
		now,
	)

	if err != nil {
		const msg = "Error during execution SQL exec for system wide rule disable"
		log.Error().Err(err).Msg(msg)
		return err
	}

	return nil
}

// EnableRuleSystemWide enables the selected rule for all clusters visible to
// given user
func (storage OCPRecommendationsDBStorage) EnableRuleSystemWide(
	orgID types.OrgID,
	ruleID types.RuleID,
	errorKey types.ErrorKey,
) error {
	log.Info().Int("org_id", int(orgID)).Msgf("re-enabling rule %v|%v", ruleID, errorKey)

	const query = `DELETE FROM rule_disable
	                WHERE org_id = $1
	                  AND rule_id = $2
	                  AND error_key = $3
	              `

	// try to execute the query and check for (any) error
	_, err := storage.connection.Exec(
		query,
		orgID,
		ruleID,
		errorKey,
	)

	if err != nil {
		const msg = "Error during execution SQL exec for system wide rule enable"
		log.Error().Err(err).Msg(msg)
		return err
	}

	return nil
}

// UpdateDisabledRuleJustification change justification for already disabled rule
func (storage OCPRecommendationsDBStorage) UpdateDisabledRuleJustification(
	orgID types.OrgID,
	ruleID types.RuleID,
	errorKey types.ErrorKey,
	justification string,
) error {
	now := time.Now()

	const query = `UPDATE rule_disable
	                  SET justification = $4, updated_at = $5
	                WHERE org_id = $1
	                  AND rule_id = $2
	                  AND error_key = $3
	              `

	// try to execute the query and check for (any) error
	_, err := storage.connection.Exec(
		query,
		orgID,
		ruleID,
		errorKey,
		justification,
		now,
	)

	if err != nil {
		const msg = "Error during execution SQL exec for system wide rule justification change"
		log.Error().Err(err).Msg(msg)
		return err
	}

	return nil
}

// ReadDisabledRule function returns disabled rule (if disabled) from database
func (storage OCPRecommendationsDBStorage) ReadDisabledRule(
	orgID types.OrgID, ruleID types.RuleID, errorKey types.ErrorKey,
) (ctypes.SystemWideRuleDisable, bool, error) {
	var disabledRule ctypes.SystemWideRuleDisable

	query := `SELECT
			 org_id,
			 rule_id,
			 error_key,
			 justification,
			 created_at,
			 updated_at
		 FROM rule_disable
		WHERE org_id = $1
		  AND rule_id = $2
		  AND error_key = $3
	`

	// run the query against database
	rows, err := storage.connection.Query(query, orgID, ruleID, errorKey)

	// return zero value in case of any error
	if err != nil {
		return disabledRule, false, err
	}
	defer closeRows(rows)

	if rows.Next() {
		err = rows.Scan(
			&disabledRule.OrgID,
			&disabledRule.RuleID,
			&disabledRule.ErrorKey,
			&disabledRule.Justification,
			&disabledRule.CreatedAt,
			&disabledRule.UpdatedAT,
		)

		if err != nil {
			log.Error().Err(err).Msg("Storage.ReadDisabledRule")
			// return zero value in case of any error
			return disabledRule, false, err
		}
		// everything seems ok -> return record read from database
		return disabledRule, true, err
	}

	// no record has been found
	return disabledRule, false, err
}

// ListOfSystemWideDisabledRules function returns list of all rules that have been
// disabled for all clusters by given user
func (storage OCPRecommendationsDBStorage) ListOfSystemWideDisabledRules(
	orgID types.OrgID,
) ([]ctypes.SystemWideRuleDisable, error) {
	disabledRules := make([]ctypes.SystemWideRuleDisable, 0)
	query := `SELECT
			 org_id,
			 rule_id,
			 error_key,
			 justification,
			 created_at,
			 updated_at
		 FROM rule_disable
		WHERE org_id = $1
	`

	// run the query against database
	rows, err := storage.connection.Query(query, orgID)
	// return empty list in case of any error
	if err != nil {
		return disabledRules, err
	}
	defer closeRows(rows)

	for rows.Next() {
		var disabledRule ctypes.SystemWideRuleDisable

		err = rows.Scan(
			&disabledRule.OrgID,
			&disabledRule.RuleID,
			&disabledRule.ErrorKey,
			&disabledRule.Justification,
			&disabledRule.CreatedAt,
			&disabledRule.UpdatedAT,
		)

		if err != nil {
			log.Error().Err(err).Msg("ReadListOfDisabledRules storage error")
			// return partially filled slice + error
			return disabledRules, err
		}

		// append disabled rule read from database to a slice
		disabledRules = append(disabledRules, disabledRule)
	}

	// everything seems ok -> return slice with all the data
	return disabledRules, err
}
