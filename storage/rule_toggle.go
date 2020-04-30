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
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// RuleToggle is a type for user's vote
type RuleToggle int

const (
	// RuleToggleDisable indicates the rule has been disabled
	RuleToggleDisable RuleToggle = 1
	// RuleToggleEnable indicates the rule has been (re)enabled
	RuleToggleEnable RuleToggle = 0
)

// ClusterRuleToggle represents a record from rule_cluster_toggle
type ClusterRuleToggle struct {
	ClusterID  types.ClusterName
	RuleID     types.RuleID
	UserID     types.UserID
	Disabled   RuleToggle
	DisabledAt sql.NullTime
	EnabledAt  sql.NullTime
	UpdatedAt  sql.NullTime
}

// ToggleRuleForCluster toggles rule for specified cluster
func (storage DBStorage) ToggleRuleForCluster(
	clusterID types.ClusterName, ruleID types.RuleID, userID types.UserID, ruleToggle RuleToggle,
) error {

	var query string
	var enabledAt, disabledAt sql.NullTime

	now := time.Now()

	switch ruleToggle {
	case RuleToggleDisable:
		disabledAt = sql.NullTime{Time: now, Valid: true}
	case RuleToggleEnable:
		enabledAt = sql.NullTime{Time: now, Valid: true}
	default:
		return fmt.Errorf("Unexpected rule toggle value")
	}

	switch storage.dbDriverType {
	case DBDriverSQLite3, DBDriverPostgres:
		query = `
			INSERT INTO cluster_rule_toggle(
				cluster_id, rule_id, user_id, disabled, disabled_at, enabled_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (cluster_id, rule_id, user_id) DO UPDATE SET
				disabled = $4,
				disabled_at = $5,
				enabled_at = $6
		`
	default:
		return fmt.Errorf("DB driver %v is not supported", storage.dbDriverType)
	}

	_, err := storage.connection.Exec(
		query,
		clusterID,
		ruleID,
		userID,
		ruleToggle,
		disabledAt,
		enabledAt,
		now,
	)
	if err != nil {
		log.Error().Err(err).Msg("Error during execution SQL exec for cluster rule toggle")
		return err
	}

	return nil
}

// ListDisabledRulesForCluster retrieves disabled rules for specified cluster
func (storage DBStorage) ListDisabledRulesForCluster(
	clusterID types.ClusterName, userID types.UserID,
) ([]types.DisabledRuleResponse, error) {

	rules := make([]types.DisabledRuleResponse, 0)

	query := `
	SELECT
		rek.rule_module,
		rek.description,
		rek.generic,
		crt.disabled_at
	FROM
		cluster_rule_toggle crt
	INNER JOIN
		rule_error_key rek ON crt.rule_id = rek.rule_module
	WHERE
		crt.disabled = $1 AND
		crt.cluster_id = $2 AND
		crt.user_id = $3
	`

	rows, err := storage.connection.Query(query, RuleToggleDisable, clusterID, userID)
	if err != nil {
		return rules, err
	}
	defer closeRows(rows)

	for rows.Next() {
		var rule types.DisabledRuleResponse

		err = rows.Scan(
			&rule.RuleModule,
			&rule.Description,
			&rule.Generic,
			&rule.DisabledAt,
		)
		if err == nil {
			rules = append(rules, rule)
		} else {
			log.Error().Err(err).Msg("ListDisabledRulesForCluster")
		}
	}
	return rules, nil
}

// GetFromClusterRuleToggle gets a rule from cluster_rule_toggle
func (storage DBStorage) GetFromClusterRuleToggle(
	clusterID types.ClusterName, ruleID types.RuleID, userID types.UserID,
) (*ClusterRuleToggle, error) {
	var disabledRule ClusterRuleToggle

	query := `
	SELECT
		cluster_id,
		rule_id,
		user_id,
		disabled,
		disabled_at,
		enabled_at,
		updated_at
	FROM
		cluster_rule_toggle
	WHERE
		cluster_id = $1 AND
		rule_id = $2 AND
		user_id = $3
	`

	err := storage.connection.QueryRow(
		query,
		clusterID,
		ruleID,
		userID,
	).Scan(
		&disabledRule.ClusterID,
		&disabledRule.RuleID,
		&disabledRule.UserID,
		&disabledRule.Disabled,
		&disabledRule.DisabledAt,
		&disabledRule.EnabledAt,
		&disabledRule.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, &ItemNotFoundError{ItemID: ruleID}
	}

	return &disabledRule, err
}

// DeleteFromRuleClusterToggle deletes a record from the table rule_cluster_toggle. Only exposed in debug mode.
func (storage DBStorage) DeleteFromRuleClusterToggle(
	clusterID types.ClusterName, ruleID types.RuleID, userID types.UserID,
) error {
	query := `
	DELETE FROM
		cluster_rule_toggle
	WHERE
		cluster_id = $1 AND
		rule_id = $2 AND
		user_id = $3
	`
	_, err := storage.connection.Exec(query, clusterID, ruleID, userID)
	return err
}
