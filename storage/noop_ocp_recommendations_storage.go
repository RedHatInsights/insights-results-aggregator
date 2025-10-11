// Copyright 2020, 2021, 2022, 2023 Red Hat, Inc
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

	"github.com/IBM/sarama"

	ctypes "github.com/RedHatInsights/insights-results-types"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// NoopOCPStorage represents a storage which does nothing (for benchmarking without a storage)
type NoopOCPStorage struct{}

// Init noop
func (*NoopOCPStorage) Init() error {
	return nil
}

// Close noop
func (*NoopOCPStorage) Close() error {
	return nil
}

// GetMigrations noop
func (*NoopOCPStorage) GetMigrations() []migration.Migration {
	return nil
}

// GetDBSchema noop
func (*NoopOCPStorage) GetDBSchema() migration.Schema {
	return migration.Schema("")
}

// GetMaxVersion noop
func (*NoopOCPStorage) GetMaxVersion() migration.Version {
	return migration.Version(0)
}

// ListOfOrgs noop
func (*NoopOCPStorage) ListOfOrgs() ([]types.OrgID, error) {
	return nil, nil
}

// ListOfClustersForOrg noop
func (*NoopOCPStorage) ListOfClustersForOrg(types.OrgID, time.Time) ([]types.ClusterName, error) {
	return nil, nil
}

// ReadReportForCluster noop
func (*NoopOCPStorage) ReadReportForCluster(types.OrgID, types.ClusterName) ([]types.RuleOnReport, types.Timestamp, types.Timestamp, types.Timestamp, error) {
	return []types.RuleOnReport{}, "", "", "", nil
}

// ReadReportInfoForCluster noop
func (*NoopOCPStorage) ReadReportInfoForCluster(types.OrgID, types.ClusterName) (types.Version, error) {
	return "", nil
}

// ReadClusterVersionsForClusterList noop
func (*NoopOCPStorage) ReadClusterVersionsForClusterList(
	types.OrgID, []string,
) (map[types.ClusterName]types.Version, error) {
	return nil, nil
}

// ReadSingleRuleTemplateData noop
func (*NoopOCPStorage) ReadSingleRuleTemplateData(types.OrgID, types.ClusterName, types.RuleID, types.ErrorKey) (interface{}, error) {
	return "", nil
}

// ReadReportForClusterByClusterName noop
func (*NoopOCPStorage) ReadReportForClusterByClusterName(
	types.ClusterName,
) ([]types.RuleOnReport, types.Timestamp, error) {
	return []types.RuleOnReport{}, "", nil
}

// WriteReportForCluster noop
func (*NoopOCPStorage) WriteReportForCluster(
	types.OrgID, types.ClusterName, types.ClusterReport, []types.ReportItem, time.Time, time.Time, time.Time,
	types.RequestID,
) error {
	return nil
}

// WriteReportInfoForCluster noop
func (*NoopOCPStorage) WriteReportInfoForCluster(
	_ types.OrgID,
	_ types.ClusterName,
	_ []types.InfoItem,
	_ time.Time,
) error {
	return nil
}

// WriteRecommendationsForCluster noop
func (*NoopOCPStorage) WriteRecommendationsForCluster(
	types.OrgID, types.ClusterName, types.ClusterReport, types.Timestamp,
) error {
	return nil
}

// ReportsCount noop
func (*NoopOCPStorage) ReportsCount() (int, error) {
	return 0, nil
}

// VoteOnRule noop
func (*NoopOCPStorage) VoteOnRule(types.ClusterName, types.RuleID, types.ErrorKey, types.OrgID, types.UserID, types.UserVote, string) error {
	return nil
}

// AddOrUpdateFeedbackOnRule noop
func (*NoopOCPStorage) AddOrUpdateFeedbackOnRule(
	types.ClusterName, types.RuleID, types.ErrorKey, types.OrgID, types.UserID, string,
) error {
	return nil
}

// AddFeedbackOnRuleDisable noop
func (*NoopOCPStorage) AddFeedbackOnRuleDisable(
	types.ClusterName, types.RuleID, types.ErrorKey, types.OrgID, types.UserID, string,
) error {
	return nil
}

// GetUserFeedbackOnRuleDisable noop
func (*NoopOCPStorage) GetUserFeedbackOnRuleDisable(
	types.ClusterName, types.RuleID, types.ErrorKey, types.UserID,
) (*UserFeedbackOnRule, error) {
	return nil, nil
}

// GetUserFeedbackOnRule noop
func (*NoopOCPStorage) GetUserFeedbackOnRule(
	types.ClusterName, types.RuleID, types.ErrorKey, types.UserID,
) (*UserFeedbackOnRule, error) {
	return nil, nil
}

// DeleteReportsForOrg noop
func (*NoopOCPStorage) DeleteReportsForOrg(types.OrgID) error {
	return nil
}

// DeleteReportsForCluster noop
func (*NoopOCPStorage) DeleteReportsForCluster(types.ClusterName) error {
	return nil
}

// GetRuleByID noop
func (*NoopOCPStorage) GetRuleByID(types.RuleID) (*types.Rule, error) {
	return nil, nil
}

// GetOrgIDByClusterID noop
func (*NoopOCPStorage) GetOrgIDByClusterID(types.ClusterName) (types.OrgID, error) {
	return 0, nil
}

// CreateRule noop
func (*NoopOCPStorage) CreateRule(types.Rule) error {
	return nil
}

// DeleteRule noop
func (*NoopOCPStorage) DeleteRule(types.RuleID) error {
	return nil
}

// CreateRuleErrorKey noop
func (*NoopOCPStorage) CreateRuleErrorKey(types.RuleErrorKey) error {
	return nil
}

// DeleteRuleErrorKey noop
func (*NoopOCPStorage) DeleteRuleErrorKey(types.RuleID, types.ErrorKey) error {
	return nil
}

// WriteConsumerError noop
func (*NoopOCPStorage) WriteConsumerError(*sarama.ConsumerMessage, error) error {
	return nil
}

// ToggleRuleForCluster noop
func (*NoopOCPStorage) ToggleRuleForCluster(
	types.ClusterName, types.RuleID, types.ErrorKey, types.OrgID, RuleToggle,
) error {
	return nil
}

// DeleteFromRuleClusterToggle noop
func (*NoopOCPStorage) DeleteFromRuleClusterToggle(
	types.ClusterName, types.RuleID) error {
	return nil
}

// GetFromClusterRuleToggle noop
func (*NoopOCPStorage) GetFromClusterRuleToggle(
	types.ClusterName,
	types.RuleID,
) (*ClusterRuleToggle, error) {
	return nil, nil
}

// GetTogglesForRules noop
func (*NoopOCPStorage) GetTogglesForRules(
	types.ClusterName,
	[]types.RuleOnReport,
	types.OrgID,
) (map[types.RuleID]bool, error) {
	return nil, nil
}

// GetUserFeedbackOnRules noop
func (*NoopOCPStorage) GetUserFeedbackOnRules(
	types.ClusterName,
	[]types.RuleOnReport,
	types.UserID,
) (map[types.RuleID]types.UserVote, error) {
	return nil, nil
}

// GetRuleWithContent noop
func (*NoopOCPStorage) GetRuleWithContent(
	types.RuleID, types.ErrorKey,
) (*types.RuleWithContent, error) {
	return nil, nil
}

// GetUserDisableFeedbackOnRules noop
func (*NoopOCPStorage) GetUserDisableFeedbackOnRules(
	types.ClusterName, []types.RuleOnReport, types.UserID,
) (map[types.RuleID]UserFeedbackOnRule, error) {
	return nil, nil
}

// DoesClusterExist noop
func (*NoopOCPStorage) DoesClusterExist(types.ClusterName) (bool, error) {
	return false, nil
}

// ReadOrgIDsForClusters read organization IDs for given list of cluster names.
func (*NoopOCPStorage) ReadOrgIDsForClusters(_ []types.ClusterName) ([]types.OrgID, error) {
	return nil, nil
}

// ReadReportsForClusters function reads reports for given list of cluster
// names.
func (*NoopOCPStorage) ReadReportsForClusters(_ []types.ClusterName) (map[types.ClusterName]types.ClusterReport, error) {
	return nil, nil
}

// ListOfDisabledRules function returns list of all rules disabled from a
// specified account (noop).
func (*NoopOCPStorage) ListOfDisabledRules(_ types.OrgID) ([]ctypes.DisabledRule, error) {
	return nil, nil
}

// ListOfReasons function returns list of reasons for all rules disabled from a
// specified account (noop).
func (*NoopOCPStorage) ListOfReasons(_ types.UserID) ([]DisabledRuleReason, error) {
	return nil, nil
}

// ListOfDisabledClusters function returns list of all clusters disabled for a rule from a
// specified account (noop).
func (*NoopOCPStorage) ListOfDisabledClusters(
	_ types.OrgID,
	_ types.RuleID,
	_ types.ErrorKey,
) ([]ctypes.DisabledClusterInfo, error) {
	return nil, nil
}

// ListOfDisabledRulesForClusters function returns list of disabled rules for given clusters from a
// specified account (noop).
func (*NoopOCPStorage) ListOfDisabledRulesForClusters(
	_ []string,
	_ types.OrgID,
) ([]ctypes.DisabledRule, error) {
	return nil, nil
}

// RateOnRule function stores the vote (rating) given by an user to a rule+error key
func (*NoopOCPStorage) RateOnRule(
	types.OrgID,
	types.RuleID,
	types.ErrorKey,
	types.UserVote,
) error {
	return nil
}

// GetRuleRating retrieves rating for given rule and user
func (*NoopOCPStorage) GetRuleRating(
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
func (*NoopOCPStorage) DisableRuleSystemWide(
	_ types.OrgID, _ types.RuleID,
	_ types.ErrorKey, _ string,
) error {
	return nil
}

// EnableRuleSystemWide enables the selected rule for all clusters visible to
// given user
func (*NoopOCPStorage) EnableRuleSystemWide(
	_ types.OrgID, _ types.RuleID, _ types.ErrorKey,
) error {
	return nil
}

// UpdateDisabledRuleJustification change justification for already disabled rule
func (*NoopOCPStorage) UpdateDisabledRuleJustification(
	_ types.OrgID,
	_ types.RuleID,
	_ types.ErrorKey,
	_ string,
) error {
	return nil
}

// ReadDisabledRule function returns disabled rule (if disabled) from database
func (*NoopOCPStorage) ReadDisabledRule(
	_ types.OrgID, _ types.RuleID, _ types.ErrorKey,
) (ctypes.SystemWideRuleDisable, bool, error) {
	return ctypes.SystemWideRuleDisable{}, true, nil
}

// ListOfSystemWideDisabledRules function returns list of all rules that have been
// disabled for all clusters by given user
func (*NoopOCPStorage) ListOfSystemWideDisabledRules(
	_ types.OrgID,
) ([]ctypes.SystemWideRuleDisable, error) {
	return nil, nil
}

// ReadRecommendationsForClusters reads all recommendations from recommendation table for given organization
func (*NoopOCPStorage) ReadRecommendationsForClusters(
	_ []string,
	_ types.OrgID,
) (ctypes.RecommendationImpactedClusters, error) {
	return nil, nil
}

// ListOfClustersForOrgSpecificRule returns list of all clusters for
// given organization that are affected by given rule
func (*NoopOCPStorage) ListOfClustersForOrgSpecificRule(
	_ types.OrgID, _ types.RuleSelector, _ []string,
) ([]ctypes.HittingClustersData, error) {
	return nil, nil
}

// ReadClusterListRecommendations retrieves cluster IDs and a list of hitting rules for each one
func (*NoopOCPStorage) ReadClusterListRecommendations(
	_ []string, _ types.OrgID,
) (ctypes.ClusterRecommendationMap, error) {
	return nil, nil
}

// MigrateToLatest migrates the database to the latest available
// migration version. This must be done before an Init() call.
func (*NoopOCPStorage) MigrateToLatest() error {
	return nil
}

// GetConnection returns db connection(useful for testing)
func (*NoopOCPStorage) GetConnection() *sql.DB {
	return nil
}

// PrintRuleDisableDebugInfo is a temporary helper function used to print form
// cluster rule toggle related tables
func (*NoopOCPStorage) PrintRuleDisableDebugInfo() {
}

// GetDBDriverType returns db driver type
func (*NoopOCPStorage) GetDBDriverType() types.DBDriver {
	return types.DBDriverGeneral
}

// UpdateOrgIDForCluster noop
func (*NoopOCPStorage) UpdateOrgIDForCluster(types.OrgID, types.ClusterName) error {
	return nil
}
