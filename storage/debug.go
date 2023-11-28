// Copyright 2020 Red Hat, Inc
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
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// PrintRuleDisableDebugInfo is a temporary helper function used to print form cluster rule toggle related tables
func (storage OCPRecommendationsDBStorage) PrintRuleDisableDebugInfo() {
	err := storage.PrintRuleToggles()
	if err != nil {
		log.Error().Err(err).Msg("unable to print records from cluster_rule_toggle")
	}

	err = storage.PrintRuleDisableFeedbacks()
	if err != nil {
		log.Error().Err(err).Msg("unable to print records from cluster_user_rule_disable_feedback")
	}
}

// PrintRuleToggles prints enable/disable counts for all rules
// TEMPORARY because we currently don't have access to stage database when testing migrations.
func (storage OCPRecommendationsDBStorage) PrintRuleToggles() error {
	log.Info().Msg("PrintRuleToggles start")

	query := `
	SELECT
		rule_id,
		count(*)
	FROM
		cluster_rule_toggle
	GROUP BY
		rule_id
	`

	rows, err := storage.connection.Query(query)
	if err != nil {
		return err
	}
	defer closeRows(rows)

	for rows.Next() {
		var (
			ruleID types.RuleID
			count  int
		)

		err = rows.Scan(&ruleID, &count)

		if err != nil {
			log.Error().Err(err).Msg("PrintRuleToggles")
			return err
		}

		log.Info().Msgf("PrintRuleToggles rule_id '%v': count %d", ruleID, count)
	}

	return nil
}

// PrintRuleDisableFeedbacks prints enable/disable feedback counts for all rules
// TEMPORARY because we currently don't have access to stage database when testing migrations.
func (storage OCPRecommendationsDBStorage) PrintRuleDisableFeedbacks() error {
	log.Info().Msg("PrintRuleDisableFeedbacks start")

	query := `
	SELECT
		rule_id,
		count(*)
	FROM
		cluster_user_rule_disable_feedback
	GROUP BY
		rule_id
	`

	rows, err := storage.connection.Query(query)
	if err != nil {
		return err
	}
	defer closeRows(rows)

	for rows.Next() {
		var (
			ruleID types.RuleID
			count  int
		)

		err = rows.Scan(&ruleID, &count)

		if err != nil {
			log.Error().Err(err).Msg("PrintRuleDisableFeedbacks")
			return err
		}
		log.Info().Msgf("PrintRuleDisableFeedbacks rule_id '%v': %d", ruleID, count)
	}

	return nil
}
