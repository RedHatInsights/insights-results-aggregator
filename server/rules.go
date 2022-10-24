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
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-operator-utils/generators"
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

	orgID, successful := readOrgID(writer, request)
	if !successful {
		// everything has been handled already
		return
	}

	err := server.Storage.ToggleRuleForCluster(clusterID, ruleID, errorKey, orgID, toggleRule)
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

// listOfDisabledRules returns list of rules disabled for given organization
func (server HTTPServer) listOfDisabledRules(writer http.ResponseWriter, request *http.Request) {
	log.Info().Msg("Lisf of disabled rules")

	// retrieve org ID
	orgID, successful := readOrgID(writer, request)
	if !successful {
		// everything has been handled already
		return
	}
	log.Info().Int(orgIDStr, int(orgID)).Msg("disabled rules for org_id")

	// try to read list of disabled rules by an organization from database
	disabledRules, err := server.Storage.ListOfDisabledRules(orgID)
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

// listOfReasons returns list of reasons why rule(s) have been disabled from
// a given organization
func (server HTTPServer) listOfReasons(writer http.ResponseWriter, request *http.Request) {
	log.Info().Msg("Lisf of reasons")

	// retrieve orgID
	orgID, successful := readOrgID(writer, request)
	if !successful {
		// everything has been handled already
		return
	}
	log.Info().Int(orgIDStr, int(orgID)).Msg("reasons for disabling rules")

	// try to read list of reasons from database
	reasons, err := server.Storage.ListOfReasons(orgID)
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

// listOfDisabledRulesForClusters returns list of rules disabled from an organization for given clusters
func (server HTTPServer) listOfDisabledRulesForClusters(writer http.ResponseWriter, request *http.Request) {
	// Extract org_id from URL
	orgID, ok := readOrgID(writer, request)
	if !ok {
		// everything has been handled
		return
	}
	log.Info().Int(orgIDStr, int(orgID)).Msg("listOfDisabledRulesForClusters")

	var listOfClusters []string
	err := json.NewDecoder(request.Body).Decode(&listOfClusters)
	if err != nil {
		handleServerError(writer, err)
		return
	}
	log.Info().Int(orgIDStr, int(orgID)).Msgf("listOfDisabledRulesForClusters number of clusters: %d", len(listOfClusters))

	// try to read list of disabled rules by an organization from database for given list of clusters
	disabledRules, err := server.Storage.ListOfDisabledRulesForClusters(listOfClusters, orgID)
	if err != nil {
		log.Error().Err(err).Msg("Unable to read list of disabled rules")
		handleServerError(writer, err)
		return
	}
	log.Info().Int(orgIDStr, int(orgID)).Int("#disabled rules", len(disabledRules)).Msg("listOfDisabledRulesForClusters")

	// try to send JSON payload to the client in a HTTP response
	err = responses.SendOK(writer,
		responses.BuildOkResponseWithData("rules", disabledRules))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// listOfDisabledClusters returns list of clusters disabled for a rule and user
func (server HTTPServer) listOfDisabledClusters(writer http.ResponseWriter, request *http.Request) {
	log.Info().Msg("Lisf of disabled clusters")

	orgID, successful := readOrgID(writer, request)
	if !successful {
		// everything has been handled already
		return
	}
	log.Info().Int(orgIDStr, int(orgID)).Msg("disabled clusters for organization")

	ruleID, successful := readRuleID(writer, request)
	if !successful {
		// everything has been handled already
		return
	}

	errorKey, successful := readErrorKey(writer, request)
	if !successful {
		// everything has been handled already
		return
	}
	log.Info().Int(orgIDStr, int(orgID)).Msgf("disabled clusters for rule ID %v|%v", ruleID, errorKey)

	// get disabled rules from DB
	disabledClusters, err := server.Storage.ListOfDisabledClusters(orgID, ruleID, errorKey)
	if err != nil {
		log.Error().Err(err).Msg("Unable to read list of disabled clusters")
		handleServerError(writer, err)
		return
	}
	log.Info().Int(orgIDStr, int(orgID)).Int("number of disabled clusters", len(disabledClusters)).Msg("list of disabled clusters")

	err = responses.SendOK(writer, responses.BuildOkResponseWithData("clusters", disabledClusters))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// getRuleToggleMapForCluster retrieves list of disabled rules and returns a map of rule_ids
// with disabled status.
func (server HTTPServer) getRuleToggleMapForCluster(
	clusterName types.ClusterName,
	orgID types.OrgID,
) (map[types.RuleID]bool, error) {
	toggleMap := make(map[types.RuleID]bool)

	disabledRules, err := server.Storage.ListOfDisabledRules(orgID)
	if err != nil {
		return toggleMap, err
	}

	for _, rule := range disabledRules {
		if rule.ClusterID != clusterName {
			continue
		}

		compositeRuleID, err := generators.GenerateCompositeRuleID(types.RuleFQDN(rule.RuleID), rule.ErrorKey)
		if err != nil {
			log.Error().Err(err).Msgf("error generating composite rule ID for rule [%+v]", rule)
			continue
		}

		toggleMap[compositeRuleID] = true
	}

	return toggleMap, nil
}

// getFeedbackAndTogglesOnRules fills in rule toggles and user feedbacks on the rule reports
func (server HTTPServer) getFeedbackAndTogglesOnRules(
	clusterName types.ClusterName,
	userID types.UserID,
	orgID types.OrgID,
	rules []types.RuleOnReport,
) ([]types.RuleOnReport, error) {
	togglesRules, err := server.getRuleToggleMapForCluster(clusterName, orgID)
	if err != nil {
		log.Error().Err(err).Msg("Unable to retrieve disabled status from database")
		return nil, err
	}

	feedbacks, err := server.Storage.GetUserFeedbackOnRules(clusterName, rules, orgID)
	if err != nil {
		log.Error().Err(err).Msg("Unable to retrieve feedback results from database")
		return nil, err
	}

	disableFeedbacks, err := server.Storage.GetUserDisableFeedbackOnRules(clusterName, rules, orgID)
	if err != nil {
		log.Error().Err(err).Msg("Unable to retrieve disable feedback results from database")
		return nil, err
	}

	for i := range rules {
		ruleID := rules[i].Module
		compositeRuleID, err := generators.GenerateCompositeRuleID(types.RuleFQDN(ruleID), rules[i].ErrorKey)
		if err != nil {
			log.Error().Err(err).Msgf("error generating composite rule ID for rule [%+v]", rules[i])
			return nil, err
		}

		if vote, found := feedbacks[ruleID]; found {
			rules[i].UserVote = vote
		} else {
			rules[i].UserVote = types.UserVoteNone
		}

		if disabled, found := togglesRules[compositeRuleID]; found {
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

	orgID, successful := readOrgID(writer, request)
	if !successful {
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

	err = server.Storage.AddFeedbackOnRuleDisable(clusterID, ruleID, errorKey, orgID, feedback)
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
	orgID types.OrgID,
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

	disableFeedback, err := server.Storage.GetUserFeedbackOnRuleDisable(clusterName, rule.Module, rule.ErrorKey, orgID)
	if err != nil {
		log.Error().Err(err).Msg("Feedback for rule was not found")
		rule.DisableFeedback = ""
	} else {
		log.Info().Msgf("feedback Message: '%v'", disableFeedback.Message)
		rule.DisableFeedback = disableFeedback.Message
	}

	userVote, err := server.Storage.GetUserFeedbackOnRule(clusterName, rule.Module, rule.ErrorKey, orgID)
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
		selector.OrgID, selector.RuleID, selector.ErrorKey,
	)

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
		selector.OrgID, selector.RuleID, selector.ErrorKey, justification,
	)

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
		selector.OrgID, selector.RuleID,
		selector.ErrorKey, justification,
	)

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
		selector.OrgID, selector.RuleID, selector.ErrorKey,
	)

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
		err := responses.SendNotFound(writer, message)
		if err != nil {
			log.Error().Err(err).Msg("Unable to send response data")
		}
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

	// try to retrieve list of disabled rules from storage
	disabledRules, err := server.Storage.ListOfSystemWideDisabledRules(orgID)

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
