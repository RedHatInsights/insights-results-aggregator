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
	"time"

	"github.com/RedHatInsights/insights-content-service/content"
	"github.com/Shopify/sarama"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// NoopStorage represents a storage which does nothing (for benchmarking without a storage)
type NoopStorage struct{}

// Init noop
func (*NoopStorage) Init() error {
	return nil
}

// Close noop
func (*NoopStorage) Close() error {
	return nil
}

// ListOfOrgs noop
func (*NoopStorage) ListOfOrgs() ([]types.OrgID, error) {
	return nil, nil
}

// ListOfClustersForOrg noop
func (*NoopStorage) ListOfClustersForOrg(types.OrgID, time.Time) ([]types.ClusterName, error) {
	return nil, nil
}

// ReadReportForCluster noop
func (*NoopStorage) ReadReportForCluster(types.OrgID, types.ClusterName) ([]types.RuleOnReport, types.Timestamp, error) {
	return []types.RuleOnReport{}, "", nil
}

// ReadSingleRuleTemplateData noop
func (*NoopStorage) ReadSingleRuleTemplateData(types.OrgID, types.ClusterName, types.RuleID, types.ErrorKey) (interface{}, error) {
	return "", nil
}

// ReadReportForClusterByClusterName noop
func (*NoopStorage) ReadReportForClusterByClusterName(
	types.ClusterName,
) ([]types.RuleOnReport, types.Timestamp, error) {
	return []types.RuleOnReport{}, "", nil
}

// GetLatestKafkaOffset noop
func (*NoopStorage) GetLatestKafkaOffset() (types.KafkaOffset, error) {
	return 0, nil
}

// WriteReportForCluster noop
func (*NoopStorage) WriteReportForCluster(
	types.OrgID, types.ClusterName, types.ClusterReport, []types.ReportItem, time.Time, types.KafkaOffset,
) error {
	return nil
}

// WriteRecommendationsForCluster noop
func (*NoopStorage) WriteRecommendationsForCluster(
	types.OrgID, types.ClusterName, types.ClusterReport,
) error {
	return nil
}

// ReportsCount noop
func (*NoopStorage) ReportsCount() (int, error) {
	return 0, nil
}

// VoteOnRule noop
func (*NoopStorage) VoteOnRule(types.ClusterName, types.RuleID, types.ErrorKey, types.UserID, types.UserVote, string) error {
	return nil
}

// AddOrUpdateFeedbackOnRule noop
func (*NoopStorage) AddOrUpdateFeedbackOnRule(
	types.ClusterName, types.RuleID, types.ErrorKey, types.UserID, string,
) error {
	return nil
}

// AddFeedbackOnRuleDisable noop
func (*NoopStorage) AddFeedbackOnRuleDisable(
	types.ClusterName, types.RuleID, types.ErrorKey, types.UserID, string,
) error {
	return nil
}

// GetUserFeedbackOnRuleDisable noop
func (*NoopStorage) GetUserFeedbackOnRuleDisable(
	types.ClusterName, types.RuleID, types.ErrorKey, types.UserID,
) (*UserFeedbackOnRule, error) {
	return nil, nil
}

// GetUserFeedbackOnRule noop
func (*NoopStorage) GetUserFeedbackOnRule(
	types.ClusterName, types.RuleID, types.ErrorKey, types.UserID,
) (*UserFeedbackOnRule, error) {
	return nil, nil
}

// DeleteReportsForOrg noop
func (*NoopStorage) DeleteReportsForOrg(types.OrgID) error {
	return nil
}

// DeleteReportsForCluster noop
func (*NoopStorage) DeleteReportsForCluster(types.ClusterName) error {
	return nil
}

// LoadRuleContent noop
func (*NoopStorage) LoadRuleContent(content.RuleContentDirectory) error {
	return nil
}

// GetRuleByID noop
func (*NoopStorage) GetRuleByID(types.RuleID) (*types.Rule, error) {
	return nil, nil
}

// GetOrgIDByClusterID noop
func (*NoopStorage) GetOrgIDByClusterID(types.ClusterName) (types.OrgID, error) {
	return 0, nil
}

// CreateRule noop
func (*NoopStorage) CreateRule(types.Rule) error {
	return nil
}

// DeleteRule noop
func (*NoopStorage) DeleteRule(types.RuleID) error {
	return nil
}

// CreateRuleErrorKey noop
func (*NoopStorage) CreateRuleErrorKey(types.RuleErrorKey) error {
	return nil
}

// DeleteRuleErrorKey noop
func (*NoopStorage) DeleteRuleErrorKey(types.RuleID, types.ErrorKey) error {
	return nil
}

// WriteConsumerError noop
func (*NoopStorage) WriteConsumerError(*sarama.ConsumerMessage, error) error {
	return nil
}

// ToggleRuleForCluster noop
func (*NoopStorage) ToggleRuleForCluster(
	types.ClusterName, types.RuleID, types.ErrorKey, types.UserID, RuleToggle,
) error {
	return nil
}

// DeleteFromRuleClusterToggle noop
func (*NoopStorage) DeleteFromRuleClusterToggle(
	types.ClusterName, types.RuleID) error {
	return nil
}

// GetFromClusterRuleToggle noop
func (*NoopStorage) GetFromClusterRuleToggle(
	types.ClusterName,
	types.RuleID,
) (*ClusterRuleToggle, error) {
	return nil, nil
}

// GetTogglesForRules noop
func (*NoopStorage) GetTogglesForRules(
	types.ClusterName,
	[]types.RuleOnReport,
) (map[types.RuleID]bool, error) {
	return nil, nil
}

// GetUserFeedbackOnRules noop
func (*NoopStorage) GetUserFeedbackOnRules(
	types.ClusterName,
	[]types.RuleOnReport,
	types.UserID,
) (map[types.RuleID]types.UserVote, error) {
	return nil, nil
}

// GetRuleWithContent noop
func (*NoopStorage) GetRuleWithContent(
	types.RuleID, types.ErrorKey,
) (*types.RuleWithContent, error) {
	return nil, nil
}

// GetUserDisableFeedbackOnRules noop
func (*NoopStorage) GetUserDisableFeedbackOnRules(
	types.ClusterName, []types.RuleOnReport, types.UserID,
) (map[types.RuleID]UserFeedbackOnRule, error) {
	return nil, nil
}

// DoesClusterExist noop
func (*NoopStorage) DoesClusterExist(types.ClusterName) (bool, error) {
	return false, nil
}

// ReadOrgIDsForClusters read organization IDs for given list of cluster names.
func (*NoopStorage) ReadOrgIDsForClusters(clusterNames []types.ClusterName) ([]types.OrgID, error) {
	return nil, nil
}

// ReadReportsForClusters function reads reports for given list of cluster
// names.
func (*NoopStorage) ReadReportsForClusters(clusterNames []types.ClusterName) (map[types.ClusterName]types.ClusterReport, error) {
	return nil, nil
}

// ListOfDisabledRules function returns list of all rules disabled from a
// specified account (noop).
func (*NoopStorage) ListOfDisabledRules(userID types.UserID) ([]DisabledRule, error) {
	return nil, nil
}
