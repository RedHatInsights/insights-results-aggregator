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

package storage_test

import (
	"testing"
	"time"

	"github.com/RedHatInsights/insights-content-service/content"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// Don't decrease code coverage by non-functional and not covered code.

// TestNoopStorageEmptyMethods1 calls empty methods that just needs to be
// defined in order for NoopStorage to satisfy Storage interface.
func TestNoopStorageEmptyMethods1(_ *testing.T) {
	noopStorage := storage.NoopOCPStorage{}
	orgID := types.OrgID(1)
	clusterName := types.ClusterName("")

	_ = noopStorage.Init()
	_ = noopStorage.Close()

	_, _ = noopStorage.ListOfOrgs()
	_, _ = noopStorage.ListOfClustersForOrg(orgID, time.Now())
	_, _ = noopStorage.ReadOrgIDsForClusters([]types.ClusterName{clusterName})
	_, _ = noopStorage.ReportsCount()
	_, _ = noopStorage.ReadReportsForClusters([]types.ClusterName{clusterName})
	_, _ = noopStorage.DoesClusterExist(clusterName)

	_, _, _, _, _ = noopStorage.ReadReportForCluster(orgID, clusterName)
	_, _ = noopStorage.ReadReportInfoForCluster(orgID, clusterName)
	_, _ = noopStorage.ReadClusterVersionsForClusterList(orgID, []string{string(clusterName)})
	_, _, _ = noopStorage.ReadReportForClusterByClusterName(clusterName)
	_ = noopStorage.DeleteReportsForOrg(orgID)
	_ = noopStorage.DeleteReportsForCluster(clusterName)

	_ = noopStorage.MigrateToLatest()
	_ = noopStorage.GetConnection()
	noopStorage.PrintRuleDisableDebugInfo()
	_ = noopStorage.GetDBDriverType()
}

// TestNoopStorageEmptyMethods2 calls empty methods that just needs to be
// defined in order for NoopStorage to satisfy Storage interface.
func TestNoopStorageEmptyMethods2(_ *testing.T) {
	noopStorage := storage.NoopOCPStorage{}
	orgID := types.OrgID(1)
	clusterName := types.ClusterName("")
	ruleID := types.RuleID("")
	errorKey := types.ErrorKey("")
	userID := types.UserID("")
	ruleSelector := types.RuleSelector("")

	_, _, _ = noopStorage.ReadDisabledRule(orgID, ruleID, errorKey)
	_ = noopStorage.VoteOnRule(clusterName, ruleID, errorKey, orgID, userID, 0, "some message")
	_ = noopStorage.AddOrUpdateFeedbackOnRule(clusterName, ruleID, errorKey, orgID, userID, "")
	_ = noopStorage.AddFeedbackOnRuleDisable(clusterName, ruleID, errorKey, orgID, userID, "")
	_, _ = noopStorage.GetUserFeedbackOnRuleDisable(clusterName, ruleID, errorKey, userID)
	_, _ = noopStorage.GetUserFeedbackOnRule(clusterName, ruleID, errorKey, userID)
	_ = noopStorage.LoadRuleContent(content.RuleContentDirectory{})
	_, _ = noopStorage.GetRuleByID(ruleID)
	_, _ = noopStorage.GetOrgIDByClusterID(clusterName)
	_, _ = noopStorage.ListOfSystemWideDisabledRules(orgID)
	_, _ = noopStorage.ListOfClustersForOrgSpecificRule(orgID, ruleSelector, nil)
	_, _ = noopStorage.ListOfClustersForOrgSpecificRule(orgID, ruleSelector, []string{"a"})
	_, _ = noopStorage.ReadRecommendationsForClusters([]string{}, types.OrgID(1))
	_, _ = noopStorage.ReadClusterListRecommendations([]string{}, types.OrgID(1))
	_, _ = noopStorage.ListOfDisabledClusters(orgID, ruleID, errorKey)
	_, _ = noopStorage.ReadSingleRuleTemplateData(orgID, clusterName, ruleID, errorKey)
}

// TestNoopStorageEmptyMethods3 calls empty methods that just needs to be
// defined in order for NoopStorage to satisfy Storage interface.
func TestNoopStorageEmptyMethods3(_ *testing.T) {
	noopStorage := storage.NoopOCPStorage{}
	orgID := types.OrgID(1)
	clusterName := types.ClusterName("")
	rule := types.Rule{}
	ruleID := types.RuleID("")
	ruleErrorKey := types.RuleErrorKey{}
	errorKey := types.ErrorKey("")
	userID := types.UserID("")

	_ = noopStorage.DisableRuleSystemWide(orgID, ruleID, errorKey, "justification#1")
	_ = noopStorage.EnableRuleSystemWide(orgID, ruleID, errorKey)
	_ = noopStorage.UpdateDisabledRuleJustification(orgID, ruleID, errorKey, "justification#2")
	_ = noopStorage.WriteReportInfoForCluster(orgID, clusterName, nil, time.Time{})
	_ = noopStorage.WriteRecommendationsForCluster(orgID, clusterName, "", types.Timestamp(""))
	_ = noopStorage.CreateRule(rule)
	_ = noopStorage.DeleteRule(ruleID)
	_ = noopStorage.CreateRuleErrorKey(ruleErrorKey)
	_ = noopStorage.DeleteRuleErrorKey(ruleID, errorKey)
	_ = noopStorage.ToggleRuleForCluster(clusterName, ruleID, errorKey, orgID, storage.RuleToggle(0))
	_ = noopStorage.DeleteFromRuleClusterToggle(clusterName, ruleID)
	_ = noopStorage.RateOnRule(orgID, ruleID, errorKey, types.UserVote(1))
	_, _ = noopStorage.GetFromClusterRuleToggle(clusterName, ruleID)
	_, _ = noopStorage.GetTogglesForRules(clusterName, nil, orgID)
	_, _ = noopStorage.GetUserFeedbackOnRules(clusterName, nil, userID)
	_, _ = noopStorage.GetUserDisableFeedbackOnRules(clusterName, nil, userID)
	_, _ = noopStorage.GetRuleWithContent(ruleID, errorKey)
	_, _ = noopStorage.ListOfDisabledRules(orgID)
	_, _ = noopStorage.GetRuleRating(orgID, types.RuleSelector(""))
	_, _ = noopStorage.ListOfReasons(userID)
	_, _ = noopStorage.ListOfDisabledRulesForClusters([]string{""}, orgID)
	_ = noopStorage.WriteConsumerError(nil, nil)
	_ = noopStorage.WriteReportForCluster(0, "", "", []types.ReportItem{}, time.Now(), time.Now(), time.Now(), "")
	_, _ = noopStorage.ReadReportsForClusters([]types.ClusterName{})
}
