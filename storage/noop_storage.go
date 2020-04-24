package storage

import (
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/content"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/Shopify/sarama"
)

// NoopStorage represents a storage which does nothing (for benchmarking without a storage)
type NoopStorage struct{}

func (*NoopStorage) Init() error {
	return nil
}

func (*NoopStorage) Close() error {
	return nil
}

func (*NoopStorage) ListOfOrgs() ([]types.OrgID, error) {
	return nil, nil
}

func (*NoopStorage) ListOfClustersForOrg(_ types.OrgID) ([]types.ClusterName, error) {
	return nil, nil
}

func (*NoopStorage) ReadReportForCluster(
	_ types.OrgID, _ types.ClusterName,
) (types.ClusterReport, types.Timestamp, error) {
	return "", "", nil
}

func (*NoopStorage) ReadReportForClusterByClusterName(
	_ types.ClusterName,
) (types.ClusterReport, types.Timestamp, error) {
	return "", "", nil
}

func (*NoopStorage) WriteReportForCluster(
	_ types.OrgID, _ types.ClusterName, _ types.ClusterReport, _ time.Time,
) error {
	return nil
}

func (*NoopStorage) ReportsCount() (int, error) {
	return 0, nil
}

func (*NoopStorage) VoteOnRule(_ types.ClusterName, _ types.RuleID, _ types.UserID, _ UserVote) error {
	return nil
}

func (*NoopStorage) AddOrUpdateFeedbackOnRule(
	_ types.ClusterName, _ types.RuleID, _ types.UserID, _ string,
) error {
	return nil
}

func (*NoopStorage) GetUserFeedbackOnRule(
	_ types.ClusterName, _ types.RuleID, _ types.UserID,
) (*UserFeedbackOnRule, error) {
	return nil, nil
}
func (*NoopStorage) GetContentForRules(_ types.ReportRules) ([]types.RuleContentResponse, error) {
	return nil, nil
}

func (*NoopStorage) DeleteReportsForOrg(_ types.OrgID) error {
	return nil
}

func (*NoopStorage) DeleteReportsForCluster(_ types.ClusterName) error {
	return nil
}

func (*NoopStorage) LoadRuleContent(_ content.RuleContentDirectory) error {
	return nil
}

func (*NoopStorage) GetRuleByID(_ types.RuleID) (*types.Rule, error) {
	return nil, nil
}

func (*NoopStorage) GetOrgIDByClusterID(_ types.ClusterName) (types.OrgID, error) {
	return 0, nil
}

func (*NoopStorage) CreateRule(_ types.Rule) error {
	return nil
}

func (*NoopStorage) DeleteRule(_ types.RuleID) error {
	return nil
}

func (*NoopStorage) CreateRuleErrorKey(_ types.RuleErrorKey) error {
	return nil
}

func (*NoopStorage) DeleteRuleErrorKey(_ types.RuleID, _ types.ErrorKey) error {
	return nil
}

func (*NoopStorage) WriteConsumerError(_ *sarama.ConsumerMessage, _ error) error {
	return nil
}
