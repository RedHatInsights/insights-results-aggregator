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

package storage_test

import (
	"testing"
	"time"

	"github.com/RedHatInsights/insights-content-service/content"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// Don't decrease code coverage by non-functional and not covered code.

func TestNoopStorage_Methods(t *testing.T) {
	noopStorage := storage.NoopStorage{}

	_ = noopStorage.Init()
	_ = noopStorage.Close()
	_, _ = noopStorage.ListOfOrgs()
	_, _ = noopStorage.ListOfClustersForOrg(0, time.Now())
	_, _, _, _ = noopStorage.ReadReportForCluster(0, "")
	_, _, _ = noopStorage.ReadReportForClusterByClusterName("")
	_, _ = noopStorage.GetLatestKafkaOffset()
	_ = noopStorage.WriteReportForCluster(0, "", "", []types.ReportItem{}, time.Now(), time.Now(), 0)
	_, _ = noopStorage.ReportsCount()
	_ = noopStorage.VoteOnRule("", "", "", "", 0, "")
	_ = noopStorage.AddOrUpdateFeedbackOnRule("", "", "", "", "")
	_ = noopStorage.AddFeedbackOnRuleDisable("", "", "", "", "")
	_, _ = noopStorage.GetUserFeedbackOnRuleDisable("", "", "", "")
	_, _ = noopStorage.GetUserFeedbackOnRule("", "", "", "")
	_ = noopStorage.DeleteReportsForOrg(0)
	_ = noopStorage.DeleteReportsForCluster("")
	_ = noopStorage.LoadRuleContent(content.RuleContentDirectory{})
	_, _ = noopStorage.GetRuleByID("")
	_, _ = noopStorage.GetOrgIDByClusterID("")
}

func TestNoopStorage_Methods_Cont(t *testing.T) {
	noopStorage := storage.NoopStorage{}

	_ = noopStorage.CreateRule(types.Rule{})
	_ = noopStorage.DeleteRule("")
	_ = noopStorage.CreateRuleErrorKey(types.RuleErrorKey{})
	_ = noopStorage.DeleteRuleErrorKey("", "")
	_ = noopStorage.WriteConsumerError(nil, nil)
	_ = noopStorage.ToggleRuleForCluster("", "", "", "", 0)
	_ = noopStorage.DeleteFromRuleClusterToggle("", "")
	_, _ = noopStorage.GetFromClusterRuleToggle("", "")
	_, _ = noopStorage.GetTogglesForRules("", nil)
	_, _ = noopStorage.GetUserFeedbackOnRules("", nil, "")
	_, _ = noopStorage.GetRuleWithContent("", "")
	_, _ = noopStorage.ReadOrgIDsForClusters([]types.ClusterName{})
	_, _ = noopStorage.ReadReportsForClusters([]types.ClusterName{})
	_, _ = noopStorage.ReadSingleRuleTemplateData(0, "", "", "")
	_, _ = noopStorage.GetUserDisableFeedbackOnRules("",
		[]types.RuleOnReport{}, "")
	_, _ = noopStorage.DoesClusterExist("")
	_, _ = noopStorage.ListOfDisabledRules("")
	_, _ = noopStorage.ListOfReasons("")
	_ = noopStorage.WriteRecommendationsForCluster(0, "", "")
	_ = noopStorage.RateOnRule(types.UserID("99"), types.OrgID(1), "", "",
		types.UserVote(1))
	_, _ = noopStorage.GetRuleRating(types.UserID("99"), types.OrgID(1), "id")
}

func TestNoopStorage_Methods_Cont2(t *testing.T) {
	noopStorage := storage.NoopStorage{}
	orgID := types.OrgID(1)
	userID := types.UserID("1")

	_ = noopStorage.DisableRuleSystemWide(orgID, userID, "", "", "")
	_ = noopStorage.EnableRuleSystemWide(orgID, userID, "", "")
	_ = noopStorage.UpdateDisabledRuleJustification(orgID, userID, "", "", "justification")
	_, _, _ = noopStorage.ReadDisabledRule(orgID, userID, "", "")
	_, _ = noopStorage.ListOfSystemWideDisabledRules(orgID, userID)
	_, _ = noopStorage.ListOfClustersForOrgSpecificRule(0, "", nil)
	_, _ = noopStorage.ListOfClustersForOrgSpecificRule(0, "", []string{"a"})
	_, _ = noopStorage.ReadRecommendationsForClusters([]string{}, types.OrgID(1))
	_, _ = noopStorage.ReadClusterListRecommendations([]string{}, types.OrgID(1))
}
