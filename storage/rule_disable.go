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
func (storage DBStorage) DisableRuleSystemWide(
	orgID types.OrgID, userID types.UserID,
	ruleID types.RuleID, errorKey types.ErrorKey,
	justification string) error {

	now := time.Now()

	const query = `INSERT INTO rule_disable(
	                   org_id, user_id, rule_id, error_key, justification, created_at
	               )
	               VALUES ($1, $2, $3, $4, $5, $6)
	              `

	// try to execute the query and check for (any) error
	_, err := storage.connection.Exec(
		query,
		orgID,
		userID,
		ruleID,
		errorKey,
		justification,
		now)

	if err != nil {
		const msg = "Error during execution SQL exec for system wide rule disable"
		log.Error().Err(err).Msg(msg)
		return err
	}

	return nil
}

// EnableRuleSystemWide enables the selected rule for all clusters visible to
// given user
func (storage DBStorage) EnableRuleSystemWide(
	orgID types.OrgID, userID types.UserID,
	ruleID types.RuleID, errorKey types.ErrorKey) error {

	const query = `DELETE FROM rule_disable
	                WHERE org_id = $1
	                  AND user_id = $2
	                  AND rule_id = $3
	                  AND error_key = $4
	              `

	// try to execute the query and check for (any) error
	_, err := storage.connection.Exec(
		query,
		orgID,
		userID,
		ruleID,
		errorKey)

	if err != nil {
		const msg = "Error during execution SQL exec for system wide rule enable"
		log.Error().Err(err).Msg(msg)
		return err
	}

	return nil
}

// UpdateDisabledRuleJustification change justification for already disabled rule
func (storage DBStorage) UpdateDisabledRuleJustification(
	orgID types.OrgID, userID types.UserID,
	ruleID types.RuleID, errorKey types.ErrorKey,
	justification string) error {

	now := time.Now()

	const query = `UPDATE rule_disable
	                  SET justification = $5, updated_at = $6
	                WHERE org_id = $1
	                  AND user_id = $2
	                  AND rule_id = $3
	                  AND error_key = $4
	              `

	// try to execute the query and check for (any) error
	_, err := storage.connection.Exec(
		query,
		orgID,
		userID,
		ruleID,
		errorKey,
		justification,
		now)

	if err != nil {
		const msg = "Error during execution SQL exec for system wide rule justification change"
		log.Error().Err(err).Msg(msg)
		return err
	}

	return nil
}

// ReadDisabledRule function returns disabled rule (if disabled) from database
func (storage DBStorage) ReadDisabledRule(
	orgID types.OrgID, userID types.UserID,
	ruleID types.RuleID, errorKey types.ErrorKey) (ctypes.SystemWideRuleDisable, bool, error) {
	var disabledRule ctypes.SystemWideRuleDisable

	query := `SELECT
			 org_id,
			 user_id,
			 rule_id,
			 error_key,
			 justification,
			 created_at,
			 updated_at
		 FROM rule_disable
		WHERE org_id = $1
		  AND user_id = $2
		  AND rule_id = $3
		  AND error_key = $4
	`

	// run the query against database
	rows, err := storage.connection.Query(query, orgID, userID, ruleID, errorKey)

	// return zero value in case of any error
	if err != nil {
		return disabledRule, false, err
	}
	defer closeRows(rows)

	if rows.Next() {
		err = rows.Scan(&disabledRule.OrgID,
			&disabledRule.UserID,
			&disabledRule.RuleID,
			&disabledRule.ErrorKey,
			&disabledRule.Justification,
			&disabledRule.CreatedAt,
			&disabledRule.UpdatedAT)

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
func (storage DBStorage) ListOfSystemWideDisabledRules(
	orgID types.OrgID, userID types.UserID) ([]ctypes.SystemWideRuleDisable, error) {
	disabledRules := make([]ctypes.SystemWideRuleDisable, 0)
	query := `SELECT
			 org_id,
			 user_id,
			 rule_id,
			 error_key,
			 justification,
			 created_at,
			 updated_at
		 FROM rule_disable
		WHERE org_id = $1
		  AND user_id = $2
	`

	// run the query against database
	rows, err := storage.connection.Query(query, orgID, userID)

	// return empty list in case of any error
	if err != nil {
		return disabledRules, err
	}
	defer closeRows(rows)

	for rows.Next() {
		var disabledRule ctypes.SystemWideRuleDisable

		err = rows.Scan(&disabledRule.OrgID,
			&disabledRule.UserID,
			&disabledRule.RuleID,
			&disabledRule.ErrorKey,
			&disabledRule.Justification,
			&disabledRule.CreatedAt,
			&disabledRule.UpdatedAT)

		if err != nil {
			log.Error().Err(err).Msg("ReadListOfDisabledRules")
			// return partially filled slice + error
			return disabledRules, err
		}

		// append disabled rule read from database to a slice
		disabledRules = append(disabledRules, disabledRule)
	}

	// everything seems ok -> return slice with all the data
	return disabledRules, err
}
