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

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// RateOnRule function stores the vote (rating) from a given user to a rule+error key
func (storage *DBStorage) RateOnRule(
	userID types.UserID,
	orgID types.OrgID,
	ruleID types.RuleID,
	errorKey types.ErrorKey,
	rating types.UserVote,
) error {
	query := `
		INSERT INTO advisor_ratings
		(user_id, org_id, rule_id, error_key, rated_at, last_updated_at, rating)
		VALUES
		($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, org_id, rule_id, error_key) DO UPDATE SET
		last_updated_at = $6, rating = $7
	`
	statement, err := storage.connection.Prepare(query)
	if err != nil {
		log.Error().Err(err).Msg("Unable")
	}

	defer func() {
		err := statement.Close()
		if err != nil {
			log.Error().Err(err).Msg(closeStatementError)
		}
	}()

	now := time.Now()
	_, err = statement.Exec(userID, orgID, ruleID, errorKey, now, now, rating)
	err = types.ConvertDBError(err, nil)
	if err != nil {
		log.Error().Err(err).Msg("RateOnRule")
		return err
	}

	metrics.RatingOnRules.Inc()

	return nil
}
