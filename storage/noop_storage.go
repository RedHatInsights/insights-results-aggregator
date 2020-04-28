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

	"github.com/RedHatInsights/insights-results-aggregator/content"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/Shopify/sarama"
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
func (*NoopStorage) ListOfClustersForOrg(_ types.OrgID) ([]types.ClusterName, error) {
	return nil, nil
}

// ReadReportForCluster noop
func (*NoopStorage) ReadReportForCluster(
	_ types.OrgID, _ types.ClusterName,
) (types.ClusterReport, types.Timestamp, error) {
	return "", "", nil
}

// ReadReportForClusterByClusterName noop
func (*NoopStorage) ReadReportForClusterByClusterName(
	_ types.ClusterName,
) (types.ClusterReport, types.Timestamp, error) {
	return "", "", nil
}

// WriteReportForCluster noop
func (*NoopStorage) WriteReportForCluster(
	_ types.OrgID, _ types.ClusterName, _ types.ClusterReport, _ time.Time,
) error {
	return nil
}

// ReportsCount noop
func (*NoopStorage) ReportsCount() (int, error) {
	return 0, nil
}

// VoteOnRule noop
func (*NoopStorage) VoteOnRule(_ types.ClusterName, _ types.RuleID, _ types.UserID, _ UserVote) error {
	return nil
}

// AddOrUpdateFeedbackOnRule noop
func (*NoopStorage) AddOrUpdateFeedbackOnRule(
	_ types.ClusterName, _ types.RuleID, _ types.UserID, _ string,
) error {
	return nil
}

// GetUserFeedbackOnRule noop
func (*NoopStorage) GetUserFeedbackOnRule(
	_ types.ClusterName, _ types.RuleID, _ types.UserID,
) (*UserFeedbackOnRule, error) {
	return nil, nil
}

// GetContentForRules noop
func (*NoopStorage) GetContentForRules(_ types.ReportRules) ([]types.RuleContentResponse, error) {
	return nil, nil
}

// DeleteReportsForOrg noop
func (*NoopStorage) DeleteReportsForOrg(_ types.OrgID) error {
	return nil
}

// DeleteReportsForCluster noop
func (*NoopStorage) DeleteReportsForCluster(_ types.ClusterName) error {
	return nil
}

// LoadRuleContent noop
func (*NoopStorage) LoadRuleContent(_ content.RuleContentDirectory) error {
	return nil
}

// GetRuleByID noop
func (*NoopStorage) GetRuleByID(_ types.RuleID) (*types.Rule, error) {
	return nil, nil
}

// GetOrgIDByClusterID noop
func (*NoopStorage) GetOrgIDByClusterID(_ types.ClusterName) (types.OrgID, error) {
	return 0, nil
}

// CreateRule noop
func (*NoopStorage) CreateRule(_ types.Rule) error {
	return nil
}

// DeleteRule noop
func (*NoopStorage) DeleteRule(_ types.RuleID) error {
	return nil
}

// CreateRuleErrorKey noop
func (*NoopStorage) CreateRuleErrorKey(_ types.RuleErrorKey) error {
	return nil
}

// DeleteRuleErrorKey noop
func (*NoopStorage) DeleteRuleErrorKey(_ types.RuleID, _ types.ErrorKey) error {
	return nil
}

// WriteConsumerError noop
func (*NoopStorage) WriteConsumerError(_ *sarama.ConsumerMessage, _ error) error {
	return nil
}
