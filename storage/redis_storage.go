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
	"database/sql"
	"time"

	"github.com/RedHatInsights/insights-content-service/content"
	"github.com/Shopify/sarama"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-operator-utils/redis"
	ctypes "github.com/RedHatInsights/insights-results-types"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// RedisStorage represents a storage which does nothing (for benchmarking without a storage)
type RedisStorage struct {
	Client redis.Client
}

// Init noop
func (*RedisStorage) Init() error {
	log.Info().Msg("Redis database Init()")
	return nil
}

// Close noop
func (*RedisStorage) Close() error {
	return nil
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

// GetLatestKafkaOffset noop
func (*RedisStorage) GetLatestKafkaOffset() (types.KafkaOffset, error) {
	return 0, nil
}

// WriteReportForCluster noop
func (*RedisStorage) WriteReportForCluster(
	types.OrgID, types.ClusterName, types.ClusterReport, []types.ReportItem, time.Time, time.Time, time.Time, types.KafkaOffset,
) error {
	return nil
}

// WriteReportInfoForCluster noop
func (*RedisStorage) WriteReportInfoForCluster(
	orgID types.OrgID,
	clusterName types.ClusterName,
	info []types.InfoItem,
	lastCheckedTime time.Time,
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
func (*RedisStorage) ReadOrgIDsForClusters(clusterNames []types.ClusterName) ([]types.OrgID, error) {
	return nil, nil
}

// ReadReportsForClusters function reads reports for given list of cluster
// names.
func (*RedisStorage) ReadReportsForClusters(clusterNames []types.ClusterName) (map[types.ClusterName]types.ClusterReport, error) {
	return nil, nil
}

// ListOfDisabledRules function returns list of all rules disabled from a
// specified account (noop).
func (*RedisStorage) ListOfDisabledRules(orgID types.OrgID) ([]ctypes.DisabledRule, error) {
	return nil, nil
}

// ListOfReasons function returns list of reasons for all rules disabled from a
// specified account (noop).
func (*RedisStorage) ListOfReasons(userID types.UserID) ([]DisabledRuleReason, error) {
	return nil, nil
}

// ListOfDisabledClusters function returns list of all clusters disabled for a rule from a
// specified account (noop).
func (*RedisStorage) ListOfDisabledClusters(
	orgID types.OrgID,
	ruleID types.RuleID,
	errorKey types.ErrorKey,
) ([]ctypes.DisabledClusterInfo, error) {
	return nil, nil
}

// ListOfDisabledRulesForClusters function returns list of disabled rules for given clusters from a
// specified account (noop).
func (*RedisStorage) ListOfDisabledRulesForClusters(
	clusterList []string,
	orgID types.OrgID,
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
	orgID types.OrgID,
	ruleSelector types.RuleSelector,
) (
	ruleRating types.RuleRating,
	err error,
) {
	return
}

// DisableRuleSystemWide disables the selected rule for all clusters visible to
// given user
func (*RedisStorage) DisableRuleSystemWide(
	orgID types.OrgID, ruleID types.RuleID,
	errorKey types.ErrorKey, justification string,
) error {
	return nil
}

// EnableRuleSystemWide enables the selected rule for all clusters visible to
// given user
func (*RedisStorage) EnableRuleSystemWide(
	orgID types.OrgID, ruleID types.RuleID, errorKey types.ErrorKey,
) error {
	return nil
}

// UpdateDisabledRuleJustification change justification for already disabled rule
func (*RedisStorage) UpdateDisabledRuleJustification(
	orgID types.OrgID,
	ruleID types.RuleID,
	errorKey types.ErrorKey,
	justification string,
) error {
	return nil
}

// ReadDisabledRule function returns disabled rule (if disabled) from database
func (*RedisStorage) ReadDisabledRule(
	orgID types.OrgID, ruleID types.RuleID, errorKey types.ErrorKey,
) (ctypes.SystemWideRuleDisable, bool, error) {
	return ctypes.SystemWideRuleDisable{}, true, nil
}

// ListOfSystemWideDisabledRules function returns list of all rules that have been
// disabled for all clusters by given user
func (*RedisStorage) ListOfSystemWideDisabledRules(
	orgID types.OrgID,
) ([]ctypes.SystemWideRuleDisable, error) {
	return nil, nil
}

// ReadRecommendationsForClusters reads all recommendations from recommendation table for given organization
func (*RedisStorage) ReadRecommendationsForClusters(
	clusterList []string,
	orgID types.OrgID,
) (ctypes.RecommendationImpactedClusters, error) {
	return nil, nil
}

// ListOfClustersForOrgSpecificRule returns list of all clusters for
// given organization that are affected by given rule
func (*RedisStorage) ListOfClustersForOrgSpecificRule(
	orgID types.OrgID, ruleID types.RuleSelector, activeClusters []string,
) ([]ctypes.HittingClustersData, error) {
	return nil, nil
}

// ReadClusterListRecommendations retrieves cluster IDs and a list of hitting rules for each one
func (*RedisStorage) ReadClusterListRecommendations(
	clusterList []string, orgID types.OrgID,
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
