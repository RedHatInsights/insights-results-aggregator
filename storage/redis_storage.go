// Copyright 2023 Red Hat, Inc
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
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/RedHatInsights/insights-content-service/content"
	"github.com/Shopify/sarama"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-operator-utils/redis"
	ctypes "github.com/RedHatInsights/insights-results-types"

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// ExpirationDuration set to keys stored into Redis
const ExpirationDuration = "24h"

// DefaultValue represents value that is stored under some key. We don't care
// about the value right now so it can be empty.
const DefaultValue = ""

// RedisStorage represents a storage which does nothing (for benchmarking without a storage)
type RedisStorage struct {
	Client     redis.Client
	Expiration time.Duration
}

// Init method initializes Redis storage
func (storage *RedisStorage) Init() error {
	log.Info().Msg("Redis database Init()")

	// try to parse expiration duration in runtime
	expiration, err := time.ParseDuration(ExpirationDuration)
	if err != nil {
		log.Error().Err(err).Str("time", ExpirationDuration).Msg("Error parsing time")
		return err
	}

	storage.Expiration = expiration
	return nil
}

// Close noop
func (storage *RedisStorage) Close() error {
	log.Info().Msg("Redis database Close()")

	if storage.Client.Connection != nil {
		return storage.Client.Connection.Close()
	}
	return nil
}

// GetMigrations noop
func (storage *RedisStorage) GetMigrations() []migration.Migration {
	return nil
}

// GetMaxVersion noop
func (storage *RedisStorage) GetMaxVersion() migration.Version {
	return migration.Version(0)
}

// ListOfOrgs noop
func (*RedisStorage) ListOfOrgs() ([]types.OrgID, error) {
	return nil, nil
}

// ListOfClustersForOrg noop
func (*RedisStorage) ListOfClustersForOrg(types.OrgID, time.Time) ([]types.ClusterName, error) {
	return nil, nil
}

// ReadReportForCluster noop
func (*RedisStorage) ReadReportForCluster(types.OrgID, types.ClusterName) ([]types.RuleOnReport, types.Timestamp, types.Timestamp, types.Timestamp, error) {
	return []types.RuleOnReport{}, "", "", "", nil
}

// ReadReportInfoForCluster noop
func (*RedisStorage) ReadReportInfoForCluster(types.OrgID, types.ClusterName) (types.Version, error) {
	return "", nil
}

// ReadClusterVersionsForClusterList noop
func (*RedisStorage) ReadClusterVersionsForClusterList(
	types.OrgID, []string,
) (map[types.ClusterName]types.Version, error) {
	return nil, nil
}

// ReadSingleRuleTemplateData noop
func (*RedisStorage) ReadSingleRuleTemplateData(types.OrgID, types.ClusterName, types.RuleID, types.ErrorKey) (interface{}, error) {
	return "", nil
}

// ReadReportForClusterByClusterName noop
func (*RedisStorage) ReadReportForClusterByClusterName(
	types.ClusterName,
) ([]types.RuleOnReport, types.Timestamp, error) {
	return []types.RuleOnReport{}, "", nil
}

// WriteReportForCluster method writes rule hits and other information about
// new report into Redis storage
func (storage *RedisStorage) WriteReportForCluster(
	orgID types.OrgID, clusterName types.ClusterName, _ types.ClusterReport,
	reportItems []types.ReportItem,
	_ time.Time, gatheredAtTime time.Time, storedAtTime time.Time,
	requestID types.RequestID,
) error {
	// retrieve context
	ctx := context.Background()

	// construct key to be stored in Redis
	key := fmt.Sprintf("organization:%d:cluster:%s:request:%s",
		int(orgID),
		string(clusterName),
		string(requestID))
	log.Info().Str("key", key).Msg("Storing key into Redis")

	// try to store key
	err := storage.Client.Connection.Set(ctx, key, DefaultValue, storage.Expiration).Err()
	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("Error storing key into Redis")
		return err
	}

	data := ctypes.SimplifiedReport{
		OrgID:              int(orgID),
		RequestID:          string(requestID),
		ClusterID:          string(clusterName),
		ReceivedTimestamp:  gatheredAtTime,
		ProcessedTimestamp: storedAtTime,
		RuleHitsCSV:        getRuleHitsCSV(reportItems),
	}

	// update hash with reports
	reportsKey := key + ":reports"
	log.Info().Str("reportsKey", reportsKey).Msg("Storing hash into Redis")

	err = storage.Client.Connection.HSet(ctx, reportsKey, data).Err()
	if err != nil {
		log.Error().Err(err).Str("reportsKey", reportsKey).Msg("Error storing hash into Redis")
		return err
	}

	// set EXPIRE on new key
	err = storage.Client.Connection.Expire(ctx, reportsKey, storage.Expiration).Err()
	if err != nil {
		log.Error().Err(err).Msg("Error updating expiration (TLL)")
		return err
	}

	// everything seems to be ok
	metrics.WrittenReports.Inc()
	log.Info().Msgf("Added data for request %v", requestID)
	return nil
}

// WriteReportInfoForCluster noop
func (*RedisStorage) WriteReportInfoForCluster(
	_ types.OrgID,
	_ types.ClusterName,
	_ []types.InfoItem,
	_ time.Time,
) error {
	return nil
}

// WriteRecommendationsForCluster noop
func (*RedisStorage) WriteRecommendationsForCluster(
	types.OrgID, types.ClusterName, types.ClusterReport, types.Timestamp,
) error {
	return nil
}

// ReportsCount noop
func (*RedisStorage) ReportsCount() (int, error) {
	return 0, nil
}

// VoteOnRule noop
func (*RedisStorage) VoteOnRule(types.ClusterName, types.RuleID, types.ErrorKey, types.OrgID, types.UserID, types.UserVote, string) error {
	return nil
}

// AddOrUpdateFeedbackOnRule noop
func (*RedisStorage) AddOrUpdateFeedbackOnRule(
	types.ClusterName, types.RuleID, types.ErrorKey, types.OrgID, types.UserID, string,
) error {
	return nil
}

// AddFeedbackOnRuleDisable noop
func (*RedisStorage) AddFeedbackOnRuleDisable(
	types.ClusterName, types.RuleID, types.ErrorKey, types.OrgID, types.UserID, string,
) error {
	return nil
}

// GetUserFeedbackOnRuleDisable noop
func (*RedisStorage) GetUserFeedbackOnRuleDisable(
	types.ClusterName, types.RuleID, types.ErrorKey, types.UserID,
) (*UserFeedbackOnRule, error) {
	return nil, nil
}

// GetUserFeedbackOnRule noop
func (*RedisStorage) GetUserFeedbackOnRule(
	types.ClusterName, types.RuleID, types.ErrorKey, types.UserID,
) (*UserFeedbackOnRule, error) {
	return nil, nil
}

// DeleteReportsForOrg noop
func (*RedisStorage) DeleteReportsForOrg(types.OrgID) error {
	return nil
}

// DeleteReportsForCluster noop
func (*RedisStorage) DeleteReportsForCluster(types.ClusterName) error {
	return nil
}

// LoadRuleContent noop
func (*RedisStorage) LoadRuleContent(content.RuleContentDirectory) error {
	return nil
}

// GetRuleByID noop
func (*RedisStorage) GetRuleByID(types.RuleID) (*types.Rule, error) {
	return nil, nil
}

// GetOrgIDByClusterID noop
func (*RedisStorage) GetOrgIDByClusterID(types.ClusterName) (types.OrgID, error) {
	return 0, nil
}

// CreateRule noop
func (*RedisStorage) CreateRule(types.Rule) error {
	return nil
}

// DeleteRule noop
func (*RedisStorage) DeleteRule(types.RuleID) error {
	return nil
}

// CreateRuleErrorKey noop
func (*RedisStorage) CreateRuleErrorKey(types.RuleErrorKey) error {
	return nil
}

// DeleteRuleErrorKey noop
func (*RedisStorage) DeleteRuleErrorKey(types.RuleID, types.ErrorKey) error {
	return nil
}

// WriteConsumerError noop
func (*RedisStorage) WriteConsumerError(*sarama.ConsumerMessage, error) error {
	return nil
}

// ToggleRuleForCluster noop
func (*RedisStorage) ToggleRuleForCluster(
	types.ClusterName, types.RuleID, types.ErrorKey, types.OrgID, RuleToggle,
) error {
	return nil
}

// DeleteFromRuleClusterToggle noop
func (*RedisStorage) DeleteFromRuleClusterToggle(
	types.ClusterName, types.RuleID) error {
	return nil
}

// GetFromClusterRuleToggle noop
func (*RedisStorage) GetFromClusterRuleToggle(
	types.ClusterName,
	types.RuleID,
) (*ClusterRuleToggle, error) {
	return nil, nil
}

// GetTogglesForRules noop
func (*RedisStorage) GetTogglesForRules(
	types.ClusterName,
	[]types.RuleOnReport,
	types.OrgID,
) (map[types.RuleID]bool, error) {
	return nil, nil
}

// GetUserFeedbackOnRules noop
func (*RedisStorage) GetUserFeedbackOnRules(
	types.ClusterName,
	[]types.RuleOnReport,
	types.UserID,
) (map[types.RuleID]types.UserVote, error) {
	return nil, nil
}

// GetRuleWithContent noop
func (*RedisStorage) GetRuleWithContent(
	types.RuleID, types.ErrorKey,
) (*types.RuleWithContent, error) {
	return nil, nil
}

// GetUserDisableFeedbackOnRules noop
func (*RedisStorage) GetUserDisableFeedbackOnRules(
	types.ClusterName, []types.RuleOnReport, types.UserID,
) (map[types.RuleID]UserFeedbackOnRule, error) {
	return nil, nil
}

// DoesClusterExist noop
func (*RedisStorage) DoesClusterExist(types.ClusterName) (bool, error) {
	return false, nil
}

// ReadOrgIDsForClusters read organization IDs for given list of cluster names.
func (*RedisStorage) ReadOrgIDsForClusters(_ []types.ClusterName) ([]types.OrgID, error) {
	return nil, nil
}

// ReadReportsForClusters function reads reports for given list of cluster
// names.
func (*RedisStorage) ReadReportsForClusters(_ []types.ClusterName) (map[types.ClusterName]types.ClusterReport, error) {
	return nil, nil
}

// ListOfDisabledRules function returns list of all rules disabled from a
// specified account (noop).
func (*RedisStorage) ListOfDisabledRules(_ types.OrgID) ([]ctypes.DisabledRule, error) {
	return nil, nil
}

// ListOfReasons function returns list of reasons for all rules disabled from a
// specified account (noop).
func (*RedisStorage) ListOfReasons(_ types.UserID) ([]DisabledRuleReason, error) {
	return nil, nil
}

// ListOfDisabledClusters function returns list of all clusters disabled for a rule from a
// specified account (noop).
func (*RedisStorage) ListOfDisabledClusters(
	_ types.OrgID,
	_ types.RuleID,
	_ types.ErrorKey,
) ([]ctypes.DisabledClusterInfo, error) {
	return nil, nil
}

// ListOfDisabledRulesForClusters function returns list of disabled rules for given clusters from a
// specified account (noop).
func (*RedisStorage) ListOfDisabledRulesForClusters(
	_ []string,
	_ types.OrgID,
) ([]ctypes.DisabledRule, error) {
	return nil, nil
}

// RateOnRule function stores the vote (rating) given by an user to a rule+error key
func (*RedisStorage) RateOnRule(
	types.OrgID,
	types.RuleID,
	types.ErrorKey,
	types.UserVote,
) error {
	return nil
}

// GetRuleRating retrieves rating for given rule and user
func (*RedisStorage) GetRuleRating(
	_ types.OrgID,
	_ types.RuleSelector,
) (
	ruleRating types.RuleRating,
	err error,
) {
	return
}

// DisableRuleSystemWide disables the selected rule for all clusters visible to
// given user
func (*RedisStorage) DisableRuleSystemWide(
	_ types.OrgID, _ types.RuleID,
	_ types.ErrorKey, _ string,
) error {
	return nil
}

// EnableRuleSystemWide enables the selected rule for all clusters visible to
// given user
func (*RedisStorage) EnableRuleSystemWide(
	_ types.OrgID, _ types.RuleID, _ types.ErrorKey,
) error {
	return nil
}

// UpdateDisabledRuleJustification change justification for already disabled rule
func (*RedisStorage) UpdateDisabledRuleJustification(
	_ types.OrgID,
	_ types.RuleID,
	_ types.ErrorKey,
	_ string,
) error {
	return nil
}

// ReadDisabledRule function returns disabled rule (if disabled) from database
func (*RedisStorage) ReadDisabledRule(
	_ types.OrgID, _ types.RuleID, _ types.ErrorKey,
) (ctypes.SystemWideRuleDisable, bool, error) {
	return ctypes.SystemWideRuleDisable{}, true, nil
}

// ListOfSystemWideDisabledRules function returns list of all rules that have been
// disabled for all clusters by given user
func (*RedisStorage) ListOfSystemWideDisabledRules(
	_ types.OrgID,
) ([]ctypes.SystemWideRuleDisable, error) {
	return nil, nil
}

// ReadRecommendationsForClusters reads all recommendations from recommendation table for given organization
func (*RedisStorage) ReadRecommendationsForClusters(
	_ []string,
	_ types.OrgID,
) (ctypes.RecommendationImpactedClusters, error) {
	return nil, nil
}

// ListOfClustersForOrgSpecificRule returns list of all clusters for
// given organization that are affected by given rule
func (*RedisStorage) ListOfClustersForOrgSpecificRule(
	_ types.OrgID, _ types.RuleSelector, _ []string,
) ([]ctypes.HittingClustersData, error) {
	return nil, nil
}

// ReadClusterListRecommendations retrieves cluster IDs and a list of hitting rules for each one
func (*RedisStorage) ReadClusterListRecommendations(
	_ []string, _ types.OrgID,
) (ctypes.ClusterRecommendationMap, error) {
	return nil, nil
}

// MigrateToLatest migrates the database to the latest available
// migration version. This must be done before an Init() call.
func (*RedisStorage) MigrateToLatest() error {
	return nil
}

// GetConnection returns db connection(useful for testing)
func (*RedisStorage) GetConnection() *sql.DB {
	return nil
}

// PrintRuleDisableDebugInfo is a temporary helper function used to print form
// cluster rule toggle related tables
func (*RedisStorage) PrintRuleDisableDebugInfo() {
	log.Info().Msg("PrintRuleDisableDebugInfo() skipped for Redis storage")
}

// GetDBDriverType returns db driver type
func (*RedisStorage) GetDBDriverType() types.DBDriver {
	return types.DBDriverGeneral
}

func getRuleHitsCSV(reportItems []types.ReportItem) string {
	// usage of strings.Builder is more efficient than consecutive string
	// concatenation
	var output strings.Builder

	for i, reportItem := range reportItems {
		// rule separator
		if i > 0 {
			output.WriteRune(',')
		}
		// strip .report suffix from rule module added automatically by running insights evaluation
		output.WriteString(strings.TrimSuffix(string(reportItem.Module), ReportSuffix))
		output.WriteRune('|')
		output.WriteString(string(reportItem.ErrorKey))
	}

	// convert back to string
	return output.String()
}
