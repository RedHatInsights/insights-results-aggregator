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

package server

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// disableRuleForCluster disables a rule for specified cluster, excluding it from reports
func (server *HTTPServer) disableRuleForCluster(writer http.ResponseWriter, request *http.Request) {
	server.toggleRuleForCluster(writer, request, storage.RuleToggleDisable)
}

// enableRuleForCluster enables a previously disabled rule, showing it on reports again
func (server *HTTPServer) enableRuleForCluster(writer http.ResponseWriter, request *http.Request) {
	server.toggleRuleForCluster(writer, request, storage.RuleToggleEnable)
}

// toggleRuleForCluster contains shared functionality for enable/disable
func (server *HTTPServer) toggleRuleForCluster(writer http.ResponseWriter, request *http.Request, toggleRule storage.RuleToggle) {
	clusterID, ruleID, userID, successful := server.readClusterRuleUserParams(writer, request)
	if !successful {
		// everything has been handled already
		return
	}

	err := server.checkUserClusterPermissions(writer, request, clusterID)
	if err != nil {
		// everything has been handled already
		return
	}

	err = server.Storage.ToggleRuleForCluster(clusterID, ruleID, userID, toggleRule)
	if err != nil {
		log.Error().Err(err).Msg("Unable to toggle rule for selected cluster")
		handleServerError(writer, err)
		return
	}

	err = responses.SendOK(writer, responses.BuildOkResponse())
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// getFeedbackAndTogglesOnRules
func (server HTTPServer) getFeedbackAndTogglesOnRules(
	clusterName types.ClusterName,
	userID types.UserID,
	rules []types.RuleOnReport,
) ([]types.RuleOnReport, error) {
	togglesRules, err := server.Storage.GetTogglesForRules(clusterName, rules, userID)
	if err != nil {
		log.Error().Err(err).Msg("Unable to retrieve disabled status from database")
		return nil, err
	}

	feedbacks, err := server.Storage.GetUserFeedbackOnRules(clusterName, rules, userID)
	if err != nil {
		log.Error().Err(err).Msg("Unable to retrieve feedback results from database")
		return nil, err
	}

	disableFeedbacks, err := server.Storage.GetUserDisableFeedbackOnRules(clusterName, rules, userID)
	if err != nil {
		log.Error().Err(err).Msg("Unable to retrieve disable feedback results from database")
		return nil, err
	}

	for i := range rules {
		ruleID := rules[i].Module
		if vote, found := feedbacks[ruleID]; found {
			rules[i].UserVote = vote
		} else {
			rules[i].UserVote = types.UserVoteNone
		}

		if disabled, found := togglesRules[ruleID]; found {
			rules[i].Disabled = disabled
		} else {
			rules[i].Disabled = false
		}

		if disableFeedback, found := disableFeedbacks[ruleID]; found {
			rules[i].DisableFeedback = disableFeedback.Message
			rules[i].DisabledAt = types.Timestamp(disableFeedback.UpdatedAt.Format(time.RFC3339))
		}
	}

	return rules, nil
}

func (server HTTPServer) saveDisableFeedback(writer http.ResponseWriter, request *http.Request) {
	clusterID, ruleID, userID, successful := server.readClusterRuleUserParams(writer, request)
	if !successful {
		// everything has been handled already
		return
	}

	err := server.checkUserClusterPermissions(writer, request, clusterID)
	if err != nil {
		// everything has been handled already
		return
	}

	feedback, err := server.getFeedbackMessageFromBody(request)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	err = server.Storage.AddFeedbackOnRuleDisable(clusterID, ruleID, userID, feedback)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	err = responses.SendOK(writer, responses.BuildOkResponseWithData(
		"message", feedback,
	))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// getFeedbackAndTogglesOnRule
func (server HTTPServer) getFeedbackAndTogglesOnRule(
	clusterName types.ClusterName,
	userID types.UserID,
	rule types.RuleOnReport,
) types.RuleOnReport {
	ruleToggle, err := server.Storage.GetFromClusterRuleToggle(clusterName, rule.Module, userID)
	if err != nil {
		log.Error().Err(err).Msg("Rule toggle was not found")
		rule.Disabled = false
	} else {
		rule.Disabled = ruleToggle.Disabled == storage.RuleToggleDisable
	}

	feedback, err := server.Storage.GetUserFeedbackOnRule(clusterName, rule.Module, userID)
	if err != nil {
		log.Error().Err(err).Msg("Feedback for rule was not found")
		rule.UserVote = types.UserVoteNone
	} else {
		rule.UserVote = feedback.UserVote
	}

	return rule
}
