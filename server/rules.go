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
	clusterID, ruleID, errorKey, successful := server.readClusterRuleParams(writer, request)
	if !successful {
		// everything has been handled already
		return
	}

	successful = server.checkUserClusterPermissions(writer, request, clusterID)
	if !successful {
		// everything has been handled already
		return
	}

	userID, succesful := readUserID(writer, request)
	if !succesful {
		// everything has been handled already
		return
	}

	err := server.Storage.ToggleRuleForCluster(clusterID, ruleID, errorKey, userID, toggleRule)
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

// listOfDisabledRules returns list of rules disabled from an account
func (server HTTPServer) listOfDisabledRules(writer http.ResponseWriter, request *http.Request) {
	log.Info().Msg("Lisf of disabled rules")

	// retrieve account (user) ID
	userID, succesful := readUserID(writer, request)
	if !succesful {
		// everything has been handled already
		return
	}
	log.Info().Str("account", string(userID)).Msg("disabled rules for account")

	// try to read list of disabled rules by an account/user from database
	disabledRules, err := server.Storage.ListOfDisabledRules(userID)
	if err != nil {
		log.Error().Err(err).Msg("Unable to read list of disabled rules")
		handleServerError(writer, err)
		return
	}
	log.Info().Int("disabled rules", len(disabledRules)).Msg("list of disabled rules")

	// try to send JSON payload to the client in a HTTP response
	err = responses.SendOK(writer,
		responses.BuildOkResponseWithData("rules", disabledRules))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// listOfReasons returns list of reasons why rule(s) have been disabled from an
// account
func (server HTTPServer) listOfReasons(writer http.ResponseWriter, request *http.Request) {
	log.Info().Msg("Lisf of reasons")

	// retrieve account (user) ID
	userID, succesful := readUserID(writer, request)
	if !succesful {
		// everything has been handled already
		return
	}
	log.Info().Str("account", string(userID)).Msg("reasons for disabling rules")

	// try to read list of reasons by an account/user from database
	reasons, err := server.Storage.ListOfReasons(userID)
	if err != nil {
		log.Error().Err(err).Msg("Unable to read list of reasons")
		handleServerError(writer, err)
		return
	}
	log.Info().Int("reasons", len(reasons)).Msg("list of reasons")

	// try to send JSON payload to the client in a HTTP response
	err = responses.SendOK(writer,
		responses.BuildOkResponseWithData("reasons", reasons))
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
	togglesRules, err := server.Storage.GetTogglesForRules(clusterName, rules)
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
	clusterID, ruleID, errorKey, successful := server.readClusterRuleParams(writer, request)
	if !successful {
		// everything has been handled already
		return
	}

	userID, succesful := readUserID(writer, request)
	if !succesful {
		// everything has been handled already
		return
	}

	successful = server.checkUserClusterPermissions(writer, request, clusterID)
	if !successful {
		// everything has been handled already
		return
	}

	feedback, err := server.getFeedbackMessageFromBody(request)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	err = server.Storage.AddFeedbackOnRuleDisable(clusterID, ruleID, errorKey, userID, feedback)
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
	ruleToggle, err := server.Storage.GetFromClusterRuleToggle(clusterName, rule.Module)
	if err != nil {
		log.Error().Err(err).Msg("Rule toggle was not found")
		rule.Disabled = false
	} else {
		rule.Disabled = ruleToggle.Disabled == storage.RuleToggleDisable
		rule.DisabledAt = types.Timestamp(ruleToggle.DisabledAt.Time.UTC().Format(time.RFC3339))
	}

	disableFeedback, err := server.Storage.GetUserFeedbackOnRuleDisable(clusterName, rule.Module, rule.ErrorKey, userID)
	if err != nil {
		log.Error().Err(err).Msg("Feedback for rule was not found")
		rule.DisableFeedback = ""
	} else {
		log.Info().Msgf("feedback Message: '%v'", disableFeedback.Message)
		rule.DisableFeedback = disableFeedback.Message
	}

	userVote, err := server.Storage.GetUserFeedbackOnRule(clusterName, rule.Module, rule.ErrorKey, userID)
	if err != nil {
		log.Error().Err(err).Msg("User vote for rule was not found")
		rule.UserVote = types.UserVoteNone
	} else {
		rule.UserVote = userVote.UserVote
	}

	return rule
}

// enableRuleSystemWide method re-enables a rule for all clusters
func (server HTTPServer) enableRuleSystemWide(writer http.ResponseWriter, request *http.Request) {
	log.Info().Msg("enableRuleSystemWide")

	// read unique rule+user selector
	selector, successful := readSystemWideRuleSelectors(writer, request)
	if !successful {
		// everything has been handled
		return
	}

	// try to enable rule
	err := server.Storage.EnableRuleSystemWide(
		selector.OrgID, selector.UserID,
		selector.RuleID, selector.ErrorKey)

	// handle any storage error
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// try to send JSON payload to the client in a HTTP response
	err = responses.SendOK(writer, responses.BuildOkResponseWithData(
		"status", "rule enabled",
	))

	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}

}

// disableRuleSystemWide method disables a rule for all clusters
func (server HTTPServer) disableRuleSystemWide(writer http.ResponseWriter, request *http.Request) {
	log.Info().Msg("disableRuleSystemWide")

	// read unique rule+user selector
	selector, successful := readSystemWideRuleSelectors(writer, request)
	if !successful {
		// everything has been handled
		return
	}

	// read justification from request body
	justification, err := server.getJustificationFromBody(request)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// try to disable rule
	err = server.Storage.DisableRuleSystemWide(
		selector.OrgID, selector.UserID,
		selector.RuleID, selector.ErrorKey,
		justification)

	// handle any storage error
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// try to send JSON payload to the client in a HTTP response
	err = responses.SendOK(writer, responses.BuildOkResponseWithData(
		"justification", justification,
	))

	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// updateRuleSystemWide method updates disable justification of a rule for all clusters
func (server HTTPServer) updateRuleSystemWide(writer http.ResponseWriter, request *http.Request) {
	log.Info().Msg("updateRuleSystemWide")

	// read unique rule+user selector
	selector, successful := readSystemWideRuleSelectors(writer, request)
	if !successful {
		// everything has been handled
		return
	}

	// read justification from request body
	justification, err := server.getJustificationFromBody(request)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// try to update rule disable justification
	err = server.Storage.UpdateDisabledRuleJustification(
		selector.OrgID, selector.UserID,
		selector.RuleID, selector.ErrorKey,
		justification)

	// handle any storage error
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// try to send JSON payload to the client in a HTTP response
	err = responses.SendOK(writer, responses.BuildOkResponseWithData(
		"justification", justification,
	))

	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// readRuleSystemWide method returns information about rule that has been
// disabled for all systems. In case such rule does not exists or was not
// disabled, HTTP code 404/Not Found is returned instead
func (server HTTPServer) readRuleSystemWide(writer http.ResponseWriter, request *http.Request) {
	log.Info().Msg("readRuleSystemWide")

	// read unique rule+user selector
	selector, successful := readSystemWideRuleSelectors(writer, request)
	if !successful {
		// everything has been handled
		return
	}

	// try to retrieve disabled rule from storage
	disabledRule, found, err := server.Storage.ReadDisabledRule(
		selector.OrgID, selector.UserID,
		selector.RuleID, selector.ErrorKey)

	// handle any storage error
	if err != nil {
		log.Error().Err(err).Msg("System-wide rule disable not found")
		handleServerError(writer, err)
		return
	}

	// handle situation when rule was not disabled ie. found in the storage
	if !found {
		const message = "Rule was not disabled"
		log.Info().Msg(message)
		responses.SendNotFound(writer, message)
		return
	}

	// try to send JSON payload to the client in a HTTP response
	err = responses.SendOK(writer,
		responses.BuildOkResponseWithData("disabledRule", disabledRule))

	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// listOfDisabledRulesSystemWide returns a list of rules disabled from current account
func (server HTTPServer) listOfDisabledRulesSystemWide(writer http.ResponseWriter, request *http.Request) {
	log.Info().Msg("listOfDisabledRulesSystemWide")

	orgID, successful := readOrgID(writer, request)
	if !successful {
		return
	}

	userID, successful := readUserID(writer, request)
	if !successful {
		return
	}

	// try to retrieve list of disabled rules from storage
	disabledRules, err := server.Storage.ListOfSystemWideDisabledRules(
		orgID, userID)

	// handle any storage error
	if err != nil {
		log.Error().Err(err).Msg("System-wide rules disable not found")
		handleServerError(writer, err)
		return
	}

	// try to send JSON payload to the client in a HTTP response
	err = responses.SendOK(writer,
		responses.BuildOkResponseWithData("disabledRules", disabledRules))

	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// SystemWideRuleSelector contains all four fields that are used to select
// system-wide rule disable
type SystemWideRuleSelector struct {
	OrgID    types.OrgID
	UserID   types.UserID
	RuleID   types.RuleID
	ErrorKey types.ErrorKey
}

// readSystemWideRuleSelectors helper function read all four parameters that
// are used to select system-wide rule disable
func readSystemWideRuleSelectors(writer http.ResponseWriter, request *http.Request) (SystemWideRuleSelector, bool) {
	var selector SystemWideRuleSelector = SystemWideRuleSelector{}
	var successful bool

	selector.OrgID, successful = readOrgID(writer, request)
	if !successful {
		return selector, false
	}

	selector.UserID, successful = readUserID(writer, request)
	if !successful {
		return selector, false
	}

	selector.RuleID, successful = readRuleID(writer, request)
	if !successful {
		return selector, false
	}

	selector.ErrorKey, successful = readErrorKey(writer, request)
	if !successful {
		return selector, false
	}

	log.Info().Msgf(
		"System-wide disabled rule selector: org: %v  user: %v  rule ID: %v  error key: %v",
		selector.OrgID, selector.UserID,
		selector.RuleID, selector.ErrorKey)

	return selector, true
}
