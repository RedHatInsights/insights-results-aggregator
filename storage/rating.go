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
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// RateOnRule function stores the vote (rating) from a given user to a rule+error key
func (storage *OCPRecommendationsDBStorage) RateOnRule(
	orgID types.OrgID,
	ruleFqdn types.RuleID,
	errorKey types.ErrorKey,
	rating types.UserVote,
) error {
	query := `
		INSERT INTO advisor_ratings
		(org_id, rule_fqdn, error_key, rated_at, last_updated_at, rating, rule_id)
		VALUES
		($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (org_id, rule_fqdn, error_key) DO UPDATE SET
		last_updated_at = $5, rating = $6
	`
	statement, err := storage.connection.Prepare(query)
	if err != nil {
		log.Error().Err(err).Msg("RateOnRule Unable to prepare statement")
	}

	defer func() {
		err := statement.Close()
		if err != nil {
			log.Error().Err(err).Msg(closeStatementError)
		}
	}()

	now := time.Now()
	ruleID := string(ruleFqdn) + "|" + string(errorKey)
	_, err = statement.Exec(orgID, ruleFqdn, errorKey, now, now, rating, ruleID)
	err = types.ConvertDBError(err, nil)
	if err != nil {
		log.Error().Err(err).Msg("RateOnRule")
		return err
	}

	metrics.RatingOnRules.Inc()

	return nil
}

// GetRuleRating retrieves rating for given rule and user
func (storage *OCPRecommendationsDBStorage) GetRuleRating(
	orgID types.OrgID,
	ruleSelector types.RuleSelector,
) (
	ruleRating types.RuleRating,
	err error,
) {
	err = storage.connection.QueryRow(
		`SELECT rule_id, rating
		FROM advisor_ratings
		WHERE org_id = $1 AND rule_id = $2`,
		orgID, ruleSelector,
	).Scan(
		&ruleRating.Rule,
		&ruleRating.Rating,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			err = &types.ItemNotFoundError{
				ItemID: fmt.Sprintf("%v/%v/rating", orgID, ruleSelector),
			}
		}
	}

	return
}
