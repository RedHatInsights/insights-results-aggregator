// Copyright 2021 Red Hat, Inc
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

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/rs/zerolog/log"
)

func (server *HTTPServer) setRuleRating(writer http.ResponseWriter, request *http.Request) {
	// Read&parse rating JSON from body content
	rating, ok := readRuleRatingFromBody(writer, request)
	if !ok {
		// all errors handled inside
		return
	}

	// Extract user_id from URL
	userID, ok := readUserID(writer, request)
	if !ok {
		// everything is handled
		return
	}

	// extract org_id from URL
	orgID, ok := readOrgID(writer, request)
	if !ok {
		// everything is handled
		return
	}

	// Split the rule_fqdn (RuleID) and ErrorKey from the received rule
	ruleID, errorKey, err := getRuleAndErrorKeyFromRuleID(rating.Rule)
	if err != nil {
		log.Error().Err(err).Msg("Unable to parse rule identifier")
		handleServerError(writer, err)
		return
	}

	// Store to the db
	err = server.Storage.RateOnRule(userID, orgID, ruleID, errorKey, rating.Rating)
	if err != nil {
		log.Error().Err(err).Msg("Unable to store rating")
		handleServerError(writer, err)
		return
	}

	// If everythig goes fine, we should send the same ratings as response to the client
	err = responses.SendOK(writer, responses.BuildOkResponseWithData("ratings", rating))
	if err != nil {
		log.Error().Err(err).Msg("Errors sending response back to client")
		handleServerError(writer, err)
		return
	}
}

// getRuleRating handles getting the rating for a specific rule and user
func (server *HTTPServer) getRuleRating(writer http.ResponseWriter, request *http.Request) {
	log.Info().Msg("getRuleRating")

	userID, ok := readUserID(writer, request)
	if !ok {
		return
	}

	orgID, ok := readOrgID(writer, request)
	if !ok {
		return
	}

	ruleSelector, ok := readAndTrimRuleSelector(writer, request)
	if !ok {
		return
	}

	rating, err := server.Storage.GetRuleRating(userID, orgID, ruleSelector)
	if err != nil {
		log.Error().Err(err).Msg("Unable to get rating")
		handleServerError(writer, err)
		return
	}

	// If everything goes fine, we should send the same ratings as response to the client
	err = responses.SendOK(writer, responses.BuildOkResponseWithData("rating", rating))
	if err != nil {
		log.Error().Err(err).Msg("Errors sending response back to client")
		handleServerError(writer, err)
		return
	}
}
