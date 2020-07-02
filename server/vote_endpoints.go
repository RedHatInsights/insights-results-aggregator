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
	"github.com/rs/zerolog/log"
	"io"
	"net/http"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// readFeedbackRequestBody parse request body and return object with message in it
func (server *HTTPServer) readFeedbackRequestBody(writer http.ResponseWriter, request *http.Request) (types.FeedbackRequest, bool) {
	var feedback types.FeedbackRequest

	err := json.NewDecoder(request.Body).Decode(&feedback)
	switch {
	case err == io.EOF:
		feedback.Message = ""
		return feedback, true
	case err != nil:
		handleServerError(writer, err)
		return feedback, false
	}

	if len(feedback.Message) > 250 {
		handleServerError(writer, &types.ValidationError{
			ErrString:  "String is longer then 250",
			ParamName:  "message",
			ParamValue: feedback.Message,
		})
		return feedback, false
	}
	return feedback, true
}

// likeRule likes the rule for current user
func (server *HTTPServer) likeRule(writer http.ResponseWriter, request *http.Request) {
	server.voteOnRule(writer, request, types.UserVoteLike)
}

// dislikeRule dislikes the rule for current user
func (server *HTTPServer) dislikeRule(writer http.ResponseWriter, request *http.Request) {
	server.voteOnRule(writer, request, types.UserVoteDislike)
}

// resetVoteOnRule resets vote for the rule for current user
func (server *HTTPServer) resetVoteOnRule(writer http.ResponseWriter, request *http.Request) {
	server.voteOnRule(writer, request, types.UserVoteNone)
}

func (server *HTTPServer) voteOnRule(writer http.ResponseWriter, request *http.Request, userVote types.UserVote) {
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

	voteMessage, successful := server.readFeedbackRequestBody(writer, request)
	if !successful {
		// everything has been handled already
		return
	}

	err = server.Storage.VoteOnRule(clusterID, ruleID, userID, userVote, voteMessage.Message)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	err = responses.SendOK(writer, responses.BuildOkResponse())
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func (server *HTTPServer) getVoteOnRule(writer http.ResponseWriter, request *http.Request) {
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

	userFeedbackOnRule, err := server.Storage.GetUserFeedbackOnRule(clusterID, ruleID, userID)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	err = responses.SendOK(writer, responses.BuildOkResponseWithData("vote", userFeedbackOnRule.UserVote))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}
